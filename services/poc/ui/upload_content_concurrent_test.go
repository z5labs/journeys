// Copyright (c) 2025 Z5labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package ui

import (
	"bytes"
	"context"
	"io"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPart simulates partReader for testing
type mockPart struct {
	io.Reader
	hdr      textproto.MIMEHeader
	filename string
	formName string
}

func (m *mockPart) GetHeader(key string) string {
	return m.hdr.Get(key)
}

func (m *mockPart) FileName() string {
	return m.filename
}

func (m *mockPart) FormName() string {
	return m.formName
}

func (m *mockPart) Close() error {
	if closer, ok := m.Reader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (m *mockPart) Read(p []byte) (int, error) {
	return m.Reader.Read(p)
}

func newMockMP4Part(data []byte, filename string) *mockPart {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Type", "video/mp4")
	h.Set("Content-Disposition", `form-data; name="files"; filename="`+filename+`"`)

	return &mockPart{
		Reader:   bytes.NewReader(data),
		hdr:      h,
		filename: filename,
		formName: "files",
	}
}

func TestStreamingFileUpload_ConcurrentExtraction_RealFile(t *testing.T) {
	data := loadTestMP4(t)

	part := newMockMP4Part(data, "test-video.mp4")
	upload, err := newStreamingFileUpload(part)
	require.NoError(t, err)

	assert.True(t, upload.isVideo)
	assert.NotNil(t, upload.pipeReader, "pipe reader should be initialized")
	assert.NotNil(t, upload.pipeWriter, "pipe writer should be initialized")
	assert.NotNil(t, upload.metadataResult, "metadata result channel should be initialized")
}

func TestStreamingFileUpload_ConcurrentExtraction_FullPipeline(t *testing.T) {
	data := loadTestMP4(t)

	part := newMockMP4Part(data, "test-video.mp4")
	upload, err := newStreamingFileUpload(part)
	require.NoError(t, err)

	// This test file has mvhd at the end (~146MB), so use appropriate budget
	upload.scanConfig.ScanBudget = int64(len(data)) + 1024

	ctx := context.Background()

	// Start metadata extraction
	upload.startMetadataExtraction(ctx)

	// Simulate MinIO upload by consuming the reader
	reader := upload.getReader()
	uploadedBytes, err := io.Copy(io.Discard, reader)
	require.NoError(t, err)
	assert.Equal(t, int64(len(data)), uploadedBytes, "all bytes should be uploaded")

	// Close writer to signal EOF to metadata goroutine
	err = upload.closeWriter()
	require.NoError(t, err)

	// Extract metadata (waits for goroutine to complete)
	meta, err := upload.extractMetadata()
	require.NoError(t, err)

	// Verify metadata was extracted
	assert.NotNil(t, meta.CapturedAt, "captured_at should be extracted")
	if meta.CapturedAt != nil {
		t.Logf("extracted creation time: %s", meta.CapturedAt.UTC().Format(time.RFC3339))
		assert.True(t, meta.CapturedAt.After(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)))
	}
}

func TestStreamingFileUpload_ConcurrentExtraction_WithSmallBudget(t *testing.T) {
	data := loadTestMP4(t)

	part := newMockMP4Part(data, "test-video.mp4")
	upload, err := newStreamingFileUpload(part)
	require.NoError(t, err)

	// Override with small budget that won't find mvhd in this file
	upload.scanConfig.ScanBudget = 8 * 1024 * 1024 // 8MB

	ctx := context.Background()

	// Start metadata extraction
	upload.startMetadataExtraction(ctx)

	// Simulate MinIO upload
	reader := upload.getReader()
	uploadedBytes, err := io.Copy(io.Discard, reader)
	require.NoError(t, err)
	assert.Equal(t, int64(len(data)), uploadedBytes)

	// Close writer
	err = upload.closeWriter()
	require.NoError(t, err)

	// Extract metadata - should complete but not find metadata
	meta, err := upload.extractMetadata()
	require.NoError(t, err)

	// With small budget, mvhd won't be found in this file
	assert.Nil(t, meta.CapturedAt, "captured_at should not be found with small budget")
}

func TestStreamingFileUpload_ConcurrentExtraction_LargeBudget(t *testing.T) {
	data := loadTestMP4(t)

	part := newMockMP4Part(data, "test-video.mp4")
	upload, err := newStreamingFileUpload(part)
	require.NoError(t, err)

	// Use large budget that will find mvhd
	upload.scanConfig.ScanBudget = int64(len(data)) + 1024

	ctx := context.Background()

	// Start metadata extraction
	upload.startMetadataExtraction(ctx)

	// Simulate MinIO upload
	reader := upload.getReader()
	uploadedBytes, err := io.Copy(io.Discard, reader)
	require.NoError(t, err)
	assert.Equal(t, int64(len(data)), uploadedBytes)

	// Close writer
	err = upload.closeWriter()
	require.NoError(t, err)

	// Extract metadata - should find it
	meta, err := upload.extractMetadata()
	require.NoError(t, err)

	assert.NotNil(t, meta.CapturedAt, "captured_at should be found with large budget")
	t.Logf("extracted creation time: %s", meta.CapturedAt.UTC().Format(time.RFC3339))
}

func TestStreamingFileUpload_ImageBuffering(t *testing.T) {
	// Test that images still use buffer-based approach (no change)
	imageData := []byte{0xFF, 0xD8, 0xFF, 0xE0} // JPEG header

	h := make(textproto.MIMEHeader)
	h.Set("Content-Type", "image/jpeg")

	part := &mockPart{
		Reader:   bytes.NewReader(imageData),
		hdr:      h,
		filename: "test.jpg",
		formName: "files",
	}

	upload, err := newStreamingFileUpload(part)
	require.NoError(t, err)

	assert.True(t, upload.isImage)
	assert.NotNil(t, upload.buffer, "buffer should be initialized for images")
	assert.Nil(t, upload.pipeReader, "pipe should not be used for images")
}

func TestStreamingFileUpload_DisabledStreamingScanner(t *testing.T) {
	data := loadTestMP4(t)

	part := newMockMP4Part(data, "test-video.mp4")
	upload, err := newStreamingFileUpload(part)
	require.NoError(t, err)

	// Disable streaming scanner
	upload.scanConfig.UseStreamingScanner = false

	// Re-initialize as if constructor ran with disabled config
	upload.pipeReader = nil
	upload.pipeWriter = nil
	upload.metadataResult = nil
	upload.buffer = &bytes.Buffer{}
	upload.probeWriter = &cappedWriter{buf: upload.buffer, limit: maxVideoProbeSize}

	ctx := context.Background()

	// Start metadata extraction should be a no-op
	upload.startMetadataExtraction(ctx)

	// Simulate upload
	reader := upload.getReader()
	_, err = io.Copy(io.Discard, reader)
	require.NoError(t, err)

	// Should have buffered up to maxVideoProbeSize
	assert.LessOrEqual(t, upload.buffer.Len(), maxVideoProbeSize)
	t.Logf("buffered %d bytes (limit: %d)", upload.buffer.Len(), maxVideoProbeSize)

	// Legacy extraction won't find metadata (mvhd beyond 8MB in this file)
	meta, err := upload.extractMetadata()
	require.NoError(t, err)
	assert.Nil(t, meta.CapturedAt)
}

func BenchmarkStreamingFileUpload_ConcurrentExtraction(b *testing.B) {
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
		part := newMockMP4Part(data, "bench-video.mp4")
		upload, err := newStreamingFileUpload(part)
		if err != nil {
			b.Fatal(err)
		}

		upload.startMetadataExtraction(ctx)

		reader := upload.getReader()
		_, err = io.Copy(io.Discard, reader)
		if err != nil {
			b.Fatal(err)
		}

		upload.closeWriter()

		_, err = upload.extractMetadata()
		if err != nil {
			b.Fatal(err)
		}
	}
}
