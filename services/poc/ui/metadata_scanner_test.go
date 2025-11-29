// Copyright (c) 2025 Z5labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package ui

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testMP4Path = "~/Downloads/bb363e05-5ec6-4a1b-8dec-1a25a92e25c7.mp4"

func loadTestMP4(t *testing.T) []byte {
	t.Helper()
	path := testMP4Path
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		require.NoError(t, err)
		path = filepath.Join(home, path[2:])
	}
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	t.Logf("loaded test MP4: %d bytes", len(data))
	return data
}

func TestScanForMvhd_RealFile_FullScan(t *testing.T) {
	data := loadTestMP4(t)
	ctx := context.Background()

	// Scan with unlimited budget
	result, bytesScanned, err := scanForMvhd(ctx, bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)

	assert.False(t, result.IsZero())
	assert.Greater(t, bytesScanned, int64(0))
	t.Logf("found mvhd at %d bytes: %s", bytesScanned, result.UTC().Format(time.RFC3339))

	// Verify the timestamp is reasonable (not in 1904 or far future)
	assert.True(t, result.After(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)))
	assert.True(t, result.Before(time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)))
}

func TestScanForMvhd_RealFile_InsufficientBudget(t *testing.T) {
	data := loadTestMP4(t)
	ctx := context.Background()

	// Based on previous test, we know mvhd is beyond 128MB
	smallBudget := int64(8 * 1024 * 1024) // 8MB

	_, bytesScanned, err := scanForMvhd(ctx, bytes.NewReader(data), smallBudget)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mvhd not found")
	assert.LessOrEqual(t, bytesScanned, smallBudget)
	t.Logf("scanned %d bytes with budget %d before failing", bytesScanned, smallBudget)
}

func TestScanForMvhd_RealFile_Progressive(t *testing.T) {
	data := loadTestMP4(t)
	ctx := context.Background()

	// Test progressive budget expansion
	budgets := []int64{
		16 * 1024 * 1024,  // 16MB
		32 * 1024 * 1024,  // 32MB
		64 * 1024 * 1024,  // 64MB
		128 * 1024 * 1024, // 128MB
		int64(len(data)),  // Full file
	}

	for _, budget := range budgets {
		result, bytesScanned, err := scanForMvhd(ctx, bytes.NewReader(data), budget)
		if err == nil {
			t.Logf("mvhd found with budget %d MB (scanned %d bytes): %s",
				budget/(1024*1024), bytesScanned, result.UTC().Format(time.RFC3339))
			assert.False(t, result.IsZero())
			return
		}
		t.Logf("budget %d MB insufficient (scanned %d bytes)", budget/(1024*1024), bytesScanned)
	}

	t.Fatal("mvhd not found even with full file budget")
}

func TestScanForMvhd_RealFile_UnlimitedBudget(t *testing.T) {
	data := loadTestMP4(t)
	ctx := context.Background()

	// Scan with unlimited budget (0 = no limit)
	result, bytesScanned, err := scanForMvhd(ctx, bytes.NewReader(data), 0)
	require.NoError(t, err)

	assert.False(t, result.IsZero())
	assert.Greater(t, bytesScanned, int64(0))
	t.Logf("unlimited budget: found mvhd after scanning %d bytes (%.1f MB): %s",
		bytesScanned, float64(bytesScanned)/(1024*1024), result.UTC().Format(time.RFC3339))

	// Verify the timestamp is reasonable
	assert.True(t, result.After(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)))
	assert.True(t, result.Before(time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)))
}

func TestScanForMvhd_ContextCancellation(t *testing.T) {
	data := loadTestMP4(t)
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	_, _, err := scanForMvhd(ctx, bytes.NewReader(data), int64(len(data)))
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestScanForMvhd_StreamingRead(t *testing.T) {
	data := loadTestMP4(t)
	ctx := context.Background()

	// Simulate streaming by using a pipe
	pr, pw := io.Pipe()

	errCh := make(chan error, 1)
	resultCh := make(chan struct {
		time  time.Time
		bytes int64
		err   error
	}, 1)

	// Writer goroutine - simulates upload stream
	go func() {
		defer pw.Close()
		_, err := io.Copy(pw, bytes.NewReader(data))
		errCh <- err
	}()

	// Scanner goroutine - reads from pipe
	go func() {
		t, b, err := scanForMvhd(ctx, pr, int64(len(data)))
		resultCh <- struct {
			time  time.Time
			bytes int64
			err   error
		}{t, b, err}
	}()

	// Wait for scanner result
	select {
	case result := <-resultCh:
		require.NoError(t, result.err)
		assert.False(t, result.time.IsZero())
		t.Logf("streaming scan found mvhd at %d bytes: %s",
			result.bytes, result.time.UTC().Format(time.RFC3339))
	case <-time.After(10 * time.Second):
		t.Fatal("streaming scan timed out")
	}

	// Wait for writer completion
	select {
	case err := <-errCh:
		// io.ErrClosedPipe is expected if scanner found mvhd and stopped reading
		if err != nil && err != io.ErrClosedPipe {
			t.Logf("writer error (may be expected): %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("writer goroutine hung")
	}
}

func TestDrainReader(t *testing.T) {
	data := []byte("test data for draining")
	ctx := context.Background()

	err := drainReader(ctx, bytes.NewReader(data), "test-drain")
	assert.NoError(t, err)
}

func TestDrainReader_LargeFile(t *testing.T) {
	data := loadTestMP4(t)
	ctx := context.Background()

	start := time.Now()
	err := drainReader(ctx, bytes.NewReader(data), "large-file-drain")
	duration := time.Since(start)

	assert.NoError(t, err)
	t.Logf("drained %d bytes in %v", len(data), duration)
}

func BenchmarkScanForMvhd_RealFile(b *testing.B) {
	// Load once
	path := testMP4Path
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[2:])
	}
	data, err := os.ReadFile(path)
	if err != nil {
		b.Skip("test file not available")
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := scanForMvhd(ctx, bytes.NewReader(data), int64(len(data)))
		if err != nil {
			b.Fatal(err)
		}
	}
}
