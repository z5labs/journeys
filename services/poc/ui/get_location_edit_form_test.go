// Copyright (c) 2025 Z5labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package ui

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dgraph-io/dgo/v240"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/z5labs/humus/rest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestGetLocationEditForm_Success_WithLocation(t *testing.T) {
	ctx := context.Background()

	// Setup container
	container, err := setupDgraphContainer(ctx)
	require.NoError(t, err, "failed to setup dgraph container")
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	})

	// Create Dgraph client
	dgraph, err := dgo.NewClient(
		container.grpcEndpoint,
		dgo.WithGrpcOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	require.NoError(t, err, "failed to create dgraph client")

	// Initialize schema
	err = initializeGetJourneySchema(ctx, dgraph)
	require.NoError(t, err, "failed to initialize schema")

	// Seed content with location
	lat, lon, alt := 34.0522, -118.2437, 100.0
	contents := []testContent{
		{
			ID:          "photo-1",
			Type:        "photo",
			MinioKey:    "journeys/test/photo-1.jpg",
			Title:       "LA Photo",
			Description: "Photo from LA",
			MimeType:    "image/jpeg",
			FileSize:    102400,
			Latitude:    &lat,
			Longitude:   &lon,
			Altitude:    &alt,
		},
	}

	journey := testJourney{ID: "test-journey", Title: "Trip"}
	err = seedJourneyWithContent(ctx, dgraph, journey, contents)
	require.NoError(t, err, "failed to seed journey with content")

	// Create API and test server
	api := rest.NewApi(
		"Test API",
		"v1.0.0",
		GetLocationEditForm(dgraph),
	)
	server := httptest.NewServer(api)
	defer server.Close()

	// Make GET request
	resp, err := http.Get(server.URL + "/app/content/photo-1/location/form")
	require.NoError(t, err, "failed to make request")
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed to read response body")
	bodyStr := string(body)

	// Assertions
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/html; charset=utf-8", resp.Header.Get("Content-Type"))
	
	// Verify form has pre-populated values
	assert.Contains(t, bodyStr, "34.0522")
	assert.Contains(t, bodyStr, "-118.2437")
	assert.Contains(t, bodyStr, "100")
	
	// Verify HTMX attributes for form submission
	assert.Contains(t, bodyStr, `hx-put="/app/content/photo-1/location"`)
}

func TestGetLocationEditForm_Success_NoLocation(t *testing.T) {
	ctx := context.Background()

	// Setup container
	container, err := setupDgraphContainer(ctx)
	require.NoError(t, err, "failed to setup dgraph container")
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	})

	// Create Dgraph client
	dgraph, err := dgo.NewClient(
		container.grpcEndpoint,
		dgo.WithGrpcOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	require.NoError(t, err, "failed to create dgraph client")

	// Initialize schema
	err = initializeGetJourneySchema(ctx, dgraph)
	require.NoError(t, err, "failed to initialize schema")

	// Seed content without location
	contents := []testContent{
		{
			ID:          "photo-1",
			Type:        "photo",
			MinioKey:    "journeys/test/photo-1.jpg",
			Title:       "Photo",
			Description: "Test photo",
			MimeType:    "image/jpeg",
			FileSize:    102400,
		},
	}

	journey := testJourney{ID: "test-journey", Title: "Trip"}
	err = seedJourneyWithContent(ctx, dgraph, journey, contents)
	require.NoError(t, err, "failed to seed journey with content")

	// Create API and test server
	api := rest.NewApi(
		"Test API",
		"v1.0.0",
		GetLocationEditForm(dgraph),
	)
	server := httptest.NewServer(api)
	defer server.Close()

	// Make GET request
	resp, err := http.Get(server.URL + "/app/content/photo-1/location/form")
	require.NoError(t, err, "failed to make request")
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed to read response body")
	bodyStr := string(body)

	// Assertions - endpoint should work even with nil location values
	if resp.StatusCode != http.StatusOK {
		t.Logf("Response body: %s", bodyStr)
	}
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/html; charset=utf-8", resp.Header.Get("Content-Type"))
	
	// Verify form is rendered (with empty location fields)
	assert.Contains(t, bodyStr, `hx-put="/app/content/photo-1/location"`)
}

func TestGetLocationEditForm_NotFound(t *testing.T) {
	ctx := context.Background()

	// Setup container
	container, err := setupDgraphContainer(ctx)
	require.NoError(t, err, "failed to setup dgraph container")
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	})

	// Create Dgraph client
	dgraph, err := dgo.NewClient(
		container.grpcEndpoint,
		dgo.WithGrpcOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	require.NoError(t, err, "failed to create dgraph client")

	// Initialize schema
	err = initializeGetJourneySchema(ctx, dgraph)
	require.NoError(t, err, "failed to initialize schema")

	// Create API and test server
	api := rest.NewApi(
		"Test API",
		"v1.0.0",
		GetLocationEditForm(dgraph),
	)
	server := httptest.NewServer(api)
	defer server.Close()

	// Make GET request for non-existent content
	resp, err := http.Get(server.URL + "/app/content/non-existent-id/location/form")
	require.NoError(t, err, "failed to make request")
	defer resp.Body.Close()

	// Assertions - should return error
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}
