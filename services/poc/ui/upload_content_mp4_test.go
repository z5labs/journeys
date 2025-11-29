// Copyright (c) 2025 Z5labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package ui

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildMvhdBox creates a minimal valid mvhd box for testing
func buildMvhdBox(version byte, creationTime uint64) []byte {
	var buf bytes.Buffer

	if version == 0 {
		// Version 0: 32-bit timestamps
		boxSize := uint32(108) // Standard mvhd v0 size
		buf.Write(binary.BigEndian.AppendUint32(nil, boxSize))
		buf.WriteString("mvhd")
		buf.WriteByte(version)
		buf.Write([]byte{0, 0, 0}) // flags

		// Creation time (32-bit)
		buf.Write(binary.BigEndian.AppendUint32(nil, uint32(creationTime)))
		// Modification time (32-bit)
		buf.Write(binary.BigEndian.AppendUint32(nil, uint32(creationTime)))
		// Timescale
		buf.Write(binary.BigEndian.AppendUint32(nil, 1000))
		// Duration (32-bit)
		buf.Write(binary.BigEndian.AppendUint32(nil, 0))

		// Padding to reach standard size (rate, volume, reserved, matrix, preview, etc.)
		for buf.Len() < int(boxSize) {
			buf.WriteByte(0)
		}
	} else {
		// Version 1: 64-bit timestamps
		boxSize := uint32(120) // Standard mvhd v1 size
		buf.Write(binary.BigEndian.AppendUint32(nil, boxSize))
		buf.WriteString("mvhd")
		buf.WriteByte(version)
		buf.Write([]byte{0, 0, 0}) // flags

		// Creation time (64-bit)
		buf.Write(binary.BigEndian.AppendUint64(nil, creationTime))
		// Modification time (64-bit)
		buf.Write(binary.BigEndian.AppendUint64(nil, creationTime))
		// Timescale
		buf.Write(binary.BigEndian.AppendUint32(nil, 1000))
		// Duration (64-bit)
		buf.Write(binary.BigEndian.AppendUint64(nil, 0))

		// Padding to reach standard size
		for buf.Len() < int(boxSize) {
			buf.WriteByte(0)
		}
	}

	return buf.Bytes()
}

// buildMoovBox wraps mvhd in a moov container
func buildMoovBox(mvhdData []byte) []byte {
	var buf bytes.Buffer

	moovSize := uint32(8 + len(mvhdData))
	buf.Write(binary.BigEndian.AppendUint32(nil, moovSize))
	buf.WriteString("moov")
	buf.Write(mvhdData)

	return buf.Bytes()
}

func TestParseMP4CreationTime_Version0(t *testing.T) {
	// Use a known timestamp: 2024-01-01 00:00:00 UTC
	// Seconds from 1904-01-01 to 2024-01-01
	expected := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	epoch1904 := time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC)
	creationTime := uint64(expected.Sub(epoch1904).Seconds())

	mvhd := buildMvhdBox(0, creationTime)
	moov := buildMoovBox(mvhd)

	result, err := parseMP4CreationTime(moov)
	require.NoError(t, err)
	assert.Equal(t, expected.Unix(), result.Unix())
}

func TestParseMP4CreationTime_Version1(t *testing.T) {
	// Use a known timestamp: 2024-06-15 12:30:45 UTC
	expected := time.Date(2024, 6, 15, 12, 30, 45, 0, time.UTC)
	epoch1904 := time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC)
	creationTime := uint64(expected.Sub(epoch1904).Seconds())

	mvhd := buildMvhdBox(1, creationTime)
	moov := buildMoovBox(mvhd)

	result, err := parseMP4CreationTime(moov)
	require.NoError(t, err)
	assert.Equal(t, expected.Unix(), result.Unix())
}

func TestParseMP4CreationTime_NoMvhd(t *testing.T) {
	// Create a minimal moov without mvhd
	var buf bytes.Buffer
	moovSize := uint32(8)
	buf.Write(binary.BigEndian.AppendUint32(nil, moovSize))
	buf.WriteString("moov")

	_, err := parseMP4CreationTime(buf.Bytes())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mvhd not found")
}

func TestParseMP4CreationTime_FromDownloads(t *testing.T) {
	// Temporary test: iteratively increase probe size starting at 16MB to find where mvhd is present.
	path := "~/Downloads/bb363e05-5ec6-4a1b-8dec-1a25a92e25c7.mp4"
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		require.NoError(t, err)
		path = filepath.Join(home, path[2:])
	}
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	t.Logf("file size: %d bytes", len(data))

	limit := 16 * 1024 * 1024     // start at 16MB
	maxLimit := 256 * 1024 * 1024 // cap at 256MB to avoid runaway
	found := false
	for limit <= maxLimit {
		probe := data
		if len(data) > limit {
			probe = data[:limit]
			t.Logf("trying limit %d bytes", limit)
		} else {
			t.Logf("trying full file size %d bytes", len(data))
		}

		result, err := parseMP4CreationTime(probe)
		if err == nil {
			t.Logf("mvhd found at %d bytes: %s", len(probe), result.UTC().Format(time.RFC3339))
			found = true
			break
		}
		t.Logf("limit %d: parse error: %v", limit, err)

		// If we've already used the full file and failed, break
		if len(data) <= limit {
			break
		}

		limit *= 2
	}

	if !found {
		// try full file as last attempt
		result, err := parseMP4CreationTime(data)
		if err == nil {
			t.Logf("mvhd found in full file: %s", result.UTC().Format(time.RFC3339))
			found = true
		} else {
			t.Fatalf("mvhd not found up to %d bytes and full file parse failed: %v", maxLimit, err)
		}
	}
}
