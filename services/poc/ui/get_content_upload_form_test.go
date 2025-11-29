// Copyright (c) 2025 Z5labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package ui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/z5labs/humus/rest"
)

func TestGetContentUploadForm_Success(t *testing.T) {
	// Create API with GetContentUploadForm endpoint
	api := rest.NewApi(
		"Test API",
		"v1.0.0",
		GetContentUploadForm(nil), // nil dgraph is fine - not used by this endpoint
	)
	server := httptest.NewServer(api)
	defer server.Close()

	// Make GET request
	resp, err := http.Get(server.URL + "/app/journey/test-journey-123/content/form")
	require.NoError(t, err, "failed to make request")
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed to read response body")
	bodyStr := string(body)

	// Assertions
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/html; charset=utf-8", resp.Header.Get("Content-Type"))

	// Verify HTMX attributes for form posting
	assert.Contains(t, bodyStr, `hx-post="/app/journey/test-journey-123/content"`)

	// Verify form has file input
	assert.Contains(t, bodyStr, `type="file"`)
	assert.Contains(t, bodyStr, `name="files"`)

	// Verify multipart form encoding (HTMX style)
	assert.Contains(t, bodyStr, `hx-encoding="multipart/form-data"`)
}
