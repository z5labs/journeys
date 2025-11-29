// Copyright (c) 2025 Z5labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package ui

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/z5labs/humus"
)

// MetadataScanConfig controls metadata extraction behavior
type MetadataScanConfig struct {
	// UseStreamingScanner enables concurrent metadata extraction with io.Pipe
	UseStreamingScanner bool
	// ScanBudget is the maximum bytes to scan for metadata (0 = unlimited).
	// For videos with mvhd at the end, this may need to be the full file size.
	// Setting to 0 allows scanning the entire stream regardless of size.
	ScanBudget int64
	// ScanTimeout is the maximum time to spend scanning.
	// Should be generous for multi-GB files on slow connections.
	ScanTimeout time.Duration
}

var defaultScanConfig = MetadataScanConfig{
	UseStreamingScanner: true,
	ScanBudget:          0,               // 0 = unlimited, scan entire file if needed
	ScanTimeout:         5 * time.Minute, // Allow time for multi-GB files
}

// MetadataResult carries extracted metadata from concurrent scanner
type MetadataResult struct {
	CapturedAt    *time.Time
	BytesScanned  int64
	ScanDuration  time.Duration
	FoundMetadata bool
}

// MP4 box types that can contain child boxes
var containerBoxTypes = map[string]bool{
	"moov": true,
	"trak": true,
	"mdia": true,
	"minf": true,
	"stbl": true,
	"mvex": true,
	"edts": true,
	"udta": true,
	"meta": true,
}

// scanForMvhd scans an MP4 stream for the mvhd box and extracts creation time.
// It reads box headers sequentially, recurses into containers, and skips non-relevant boxes.
// Returns creation time and bytes scanned, or error if mvhd not found within limit.
func scanForMvhd(ctx context.Context, r io.Reader, limit int64) (time.Time, int64, error) {
	log := humus.Logger("mp4-scanner")
	const epoch1904 = 1904

	// If limit is 0 or negative, treat as unlimited
	unlimited := limit <= 0
	remaining := limit
	if unlimited {
		remaining = 1<<63 - 1 // max int64
	}
	totalScanned := int64(0)

	for remaining > 0 {
		select {
		case <-ctx.Done():
			return time.Time{}, totalScanned, ctx.Err()
		default:
		}

		// Need at least 8 bytes for box header (size + type)
		if remaining < 8 {
			return time.Time{}, totalScanned, errors.New("scan budget exhausted")
		}

		var hdr [8]byte
		if _, err := io.ReadFull(r, hdr[:]); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return time.Time{}, totalScanned, errors.New("mvhd not found")
			}
			return time.Time{}, totalScanned, err
		}
		remaining -= 8
		totalScanned += 8

		size32 := binary.BigEndian.Uint32(hdr[0:4])
		boxType := string(hdr[4:8])

		var boxSize int64
		headerLen := int64(8)

		// Handle extended size (size == 1 means 64-bit size follows)
		if size32 == 1 {
			if remaining < 8 {
				return time.Time{}, totalScanned, errors.New("truncated extended size")
			}
			var ext [8]byte
			if _, err := io.ReadFull(r, ext[:]); err != nil {
				return time.Time{}, totalScanned, err
			}
			remaining -= 8
			totalScanned += 8
			boxSize = int64(binary.BigEndian.Uint64(ext[:]))
			headerLen = 16
		} else if size32 == 0 {
			// Box extends to EOF
			boxSize = -1
		} else {
			boxSize = int64(size32)
		}

		var payloadSize int64
		if boxSize == -1 {
			// To EOF - use remaining budget
			payloadSize = remaining
		} else {
			payloadSize = boxSize - headerLen
			if payloadSize < 0 {
				payloadSize = 0
			}
			if payloadSize > remaining {
				// Truncated relative to scan limit
				payloadSize = remaining
			}
		}

		log.Debug("scanning box",
			slog.String("type", boxType),
			slog.Int64("size", boxSize),
			slog.Int64("payload", payloadSize))

		// Found mvhd - parse creation time
		if boxType == "mvhd" {
			if payloadSize < 5 {
				return time.Time{}, totalScanned, errors.New("mvhd payload too small")
			}

			// Read mvhd payload (it's small, safe to buffer)
			payload := make([]byte, payloadSize)
			if _, err := io.ReadFull(r, payload); err != nil && err != io.ErrUnexpectedEOF {
				return time.Time{}, totalScanned, err
			}
			totalScanned += int64(len(payload))

			// mvhd format: version(1) flags(3) creation_time(32 or 64) ...
			version := payload[0]
			off := 4 // after version + flags

			var creationTime uint64
			if version == 0 {
				if len(payload) < off+4 {
					return time.Time{}, totalScanned, errors.New("mvhd v0 too small")
				}
				creationTime = uint64(binary.BigEndian.Uint32(payload[off : off+4]))
			} else if version == 1 {
				if len(payload) < off+8 {
					return time.Time{}, totalScanned, errors.New("mvhd v1 too small")
				}
				creationTime = binary.BigEndian.Uint64(payload[off : off+8])
			} else {
				return time.Time{}, totalScanned, fmt.Errorf("unknown mvhd version: %d", version)
			}

			t := time.Date(epoch1904, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(creationTime) * time.Second)
			log.Info("found mvhd",
				slog.Int("version", int(version)),
				slog.Time("creation_time", t),
				slog.Int64("bytes_scanned", totalScanned))
			return t, totalScanned, nil
		}

		// Container box - recurse into it
		if containerBoxTypes[boxType] {
			lr := &io.LimitedReader{R: r, N: payloadSize}
			t, scanned, err := scanForMvhd(ctx, lr, payloadSize)
			totalScanned += scanned

			// Drain any unread bytes from the container
			if lr.N > 0 {
				drained, _ := io.CopyN(io.Discard, lr, lr.N)
				totalScanned += drained
			}
			remaining -= payloadSize

			if err == nil {
				return t, totalScanned, nil
			}
			// Continue scanning siblings
			continue
		}

		// Skip non-container, non-mvhd payload
		if payloadSize > 0 {
			skipped, err := io.CopyN(io.Discard, r, payloadSize)
			totalScanned += skipped
			remaining -= skipped
			if err != nil && err != io.EOF {
				return time.Time{}, totalScanned, err
			}
		}
	}

	if unlimited {
		return time.Time{}, totalScanned, errors.New("mvhd not found")
	}
	return time.Time{}, totalScanned, errors.New("mvhd not found within scan limit")
}

// drainReader copies from r to io.Discard until EOF, logging progress.
// This prevents TeeReader from blocking the upload goroutine.
func drainReader(ctx context.Context, r io.Reader, label string) error {
	log := humus.Logger("metadata-scanner")
	start := time.Now()

	n, err := io.Copy(io.Discard, r)

	log.Debug("drained reader",
		slog.String("label", label),
		slog.Int64("bytes", n),
		slog.Duration("duration", time.Since(start)))

	return err
}

// extractMP4MetadataStreaming extracts metadata from an MP4 stream concurrently.
// It reads from the provided reader, scans for mvhd with the given budget,
// and then drains the remaining stream to prevent blocking upstream writers.
func extractMP4MetadataStreaming(ctx context.Context, r io.Reader, cfg MetadataScanConfig) (*MetadataResult, error) {
	log := humus.Logger("metadata-scanner")
	start := time.Now()

	result := &MetadataResult{}

	// Apply timeout if configured
	if cfg.ScanTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.ScanTimeout)
		defer cancel()
	}

	// Scan for mvhd
	t, bytesScanned, err := scanForMvhd(ctx, r, cfg.ScanBudget)
	result.BytesScanned = bytesScanned
	result.ScanDuration = time.Since(start)

	if err == nil {
		result.CapturedAt = &t
		result.FoundMetadata = true
		log.Info("metadata extraction succeeded",
			slog.Int64("bytes_scanned", bytesScanned),
			slog.Duration("duration", result.ScanDuration),
			slog.Time("captured_at", t))
	} else {
		log.Info("metadata extraction failed",
			slog.Int64("bytes_scanned", bytesScanned),
			slog.Duration("duration", result.ScanDuration),
			slog.String("error", err.Error()))
	}

	// CRITICAL: Always drain the reader to prevent blocking the TeeReader/upload
	drainStart := time.Now()
	if drainErr := drainReader(ctx, r, "post-scan"); drainErr != nil && drainErr != io.EOF {
		log.Warn("error draining reader", slog.String("error", drainErr.Error()))
	}
	log.Debug("post-scan drain completed", slog.Duration("drain_duration", time.Since(drainStart)))

	return result, err
}
