// Copyright (c) 2025 Z5labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package ui

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/z5labs/humus/rest"
)

func TestGetJourney(t *testing.T) {
	// Create a test API with the GetJourney endpoint
	// Note: We can't easily mock Dgraph since it's a concrete type,
	// so we pass nil and focus on testing the routing/parameter extraction
	api := rest.NewApi(
		"Test API",
		"v1.0.0",
		GetJourney(nil),
	)

	// Create test server
	server := httptest.NewServer(api)
	defer server.Close()

	// Make request to the journey endpoint
	t.Logf("Making request to: %s/app/journey/test-journey-id", server.URL)
	resp, err := http.Get(server.URL + "/app/journey/test-journey-id")
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	t.Logf("Response status: %d", resp.StatusCode)

	// We expect 500 because dgraph is nil, but if we get a panic about path params,
	// we'll see it in the error
	if resp.StatusCode == http.StatusOK {
		t.Errorf("expected error status due to nil dgraph, got 200")
	}
}

func TestGetJourney_NotFound(t *testing.T) {
	// Create a test API with the GetJourney endpoint
	api := rest.NewApi(
		"Test API",
		"v1.0.0",
		GetJourney(nil),
	)

	// Create test server
	server := httptest.NewServer(api)
	defer server.Close()

	// Make request to the journey endpoint
	t.Logf("Making request to: %s/app/journey/non-existent-id", server.URL)
	resp, err := http.Get(server.URL + "/app/journey/non-existent-id")
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	t.Logf("Response status: %d", resp.StatusCode)

	// Should return error (500 since we passed nil Dgraph)
	if resp.StatusCode == http.StatusOK {
		t.Errorf("expected error status, got 200")
	}
}
