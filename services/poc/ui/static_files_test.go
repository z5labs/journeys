package ui

import (
	"context"
	"testing"

	"github.com/z5labs/humus/rest"
)

func TestServeStaticFiles_JourneyMapJS(t *testing.T) {
	// Create handler
	opt := ServeStaticFiles()
	if opt == nil {
		t.Fatal("ServeStaticFiles() returned nil")
	}

	// Create a mock API to register the handler
	api := rest.NewApi("test", "v1", opt)
	if api == nil {
		t.Fatal("Failed to create API with static file handler")
	}

	// Note: Full HTTP testing would require starting the server
	// This test just verifies the handler can be registered without panicking
}

func TestStaticFileHandler_Handle(t *testing.T) {
	h := &staticFileHandler{
		content:     []byte("test content"),
		contentType: "application/javascript; charset=utf-8",
	}

	resp, err := h.Handle(context.Background(), &rest.EmptyRequest{})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if resp == nil {
		t.Fatal("Handle() returned nil response")
	}

	if resp.ContentType != "application/javascript; charset=utf-8" {
		t.Errorf("ContentType = %q, want %q", resp.ContentType, "application/javascript; charset=utf-8")
	}

	if string(resp.Body) != "test content" {
		t.Errorf("Body = %q, want %q", string(resp.Body), "test content")
	}
}

func TestGetContentType(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"script.js", "application/javascript; charset=utf-8"},
		{"style.css", "text/css; charset=utf-8"},
		{"index.html", "text/html; charset=utf-8"},
		{"data.json", "application/json; charset=utf-8"},
		{"file.txt", "application/octet-stream"},
		{"unknown", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := getContentType(tt.filename)
			if got != tt.want {
				t.Errorf("getContentType(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}
