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

func TestGetJourneyForm_Success(t *testing.T) {
	// Create API with GetJourneyForm endpoint
	api := rest.NewApi(
		"Test API",
		"v1.0.0",
		GetJourneyForm(nil), // nil dgraph is fine - not used by this endpoint
	)
	server := httptest.NewServer(api)
	defer server.Close()

	// Make GET request
	resp, err := http.Get(server.URL + "/app/journey/form")
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
	assert.Contains(t, bodyStr, `hx-post="/app/journey"`)
	assert.Contains(t, bodyStr, `hx-target="#journeys-list"`)
	assert.Contains(t, bodyStr, `hx-swap="afterbegin"`)

	// Verify form has input for title
	assert.Contains(t, bodyStr, `name="title"`)
	assert.Contains(t, bodyStr, `type="text"`)

	// Verify form has submit button
	assert.Contains(t, bodyStr, `type="submit"`)
}

func TestCancelJourneyForm_Success(t *testing.T) {
	// Create API with CancelJourneyForm endpoint
	api := rest.NewApi(
		"Test API",
		"v1.0.0",
		CancelJourneyForm(nil), // nil dgraph is fine - not used by this endpoint
	)
	server := httptest.NewServer(api)
	defer server.Close()

	// Make GET request
	resp, err := http.Get(server.URL + "/app/journey/cancel")
	require.NoError(t, err, "failed to make request")
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed to read response body")
	bodyStr := string(body)

	// Assertions
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/html; charset=utf-8", resp.Header.Get("Content-Type"))

	// Verify HTMX attributes for replacing form with button
	assert.Contains(t, bodyStr, `hx-get="/app/journey/form"`)
	assert.Contains(t, bodyStr, `hx-target="#create-journey-container"`)

	// Verify it's a button
	assert.Contains(t, bodyStr, `<button`)
}
