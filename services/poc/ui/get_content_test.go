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
	"time"

	"github.com/dgraph-io/dgo/v240"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/z5labs/humus/rest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestGetContent_Success_WithGeoData(t *testing.T) {
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

	// Seed test data with content that has geo data
	lat, lon, alt := 37.7749, -122.4194, 100.0
	capturedAt := time.Now().Add(-24 * time.Hour)
	contents := []testContent{
		{
			ID:           "photo-1",
			Type:         "photo",
			MinioKey:     "journeys/test-journey/photo-1.jpg",
			Title:        "Golden Gate Bridge",
			Description:  "View of the bridge at sunset",
			MimeType:     "image/jpeg",
			FileSize:     102400,
			Latitude:     &lat,
			Longitude:    &lon,
			Altitude:     &alt,
			LocationName: "San Francisco, CA",
			CapturedAt:   &capturedAt,
		},
	}

	journey := testJourney{ID: "test-journey", Title: "San Francisco Trip"}
	err = seedJourneyWithContent(ctx, dgraph, journey, contents)
	require.NoError(t, err, "failed to seed journey with content")

	// Create API and test server
	api := rest.NewApi(
		"Test API",
		"v1.0.0",
		GetContent(dgraph),
	)
	server := httptest.NewServer(api)
	defer server.Close()

	// Make request
	resp, err := http.Get(server.URL + "/app/content/photo-1")
	require.NoError(t, err, "failed to make request")
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed to read response body")
	bodyStr := string(body)

	// Assertions
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/html; charset=utf-8", resp.Header.Get("Content-Type"))
	assert.Contains(t, bodyStr, "Golden Gate Bridge")
	assert.Contains(t, bodyStr, "View of the bridge at sunset")
	assert.Contains(t, bodyStr, "San Francisco, CA")
	assert.Contains(t, bodyStr, "San Francisco Trip") // Parent journey title
	assert.Contains(t, bodyStr, "photo-1")
}

func TestGetContent_Success_NoGeoData(t *testing.T) {
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

	// Seed test data without geo data
	contents := []testContent{
		{
			ID:          "video-1",
			Type:        "video",
			MinioKey:    "journeys/test-journey/video-1.mp4",
			Title:       "Vacation Video",
			Description: "A short clip",
			MimeType:    "video/mp4",
			FileSize:    2048000,
		},
	}

	journey := testJourney{ID: "test-journey", Title: "Vacation 2024"}
	err = seedJourneyWithContent(ctx, dgraph, journey, contents)
	require.NoError(t, err, "failed to seed journey with content")

	// Create API and test server
	api := rest.NewApi(
		"Test API",
		"v1.0.0",
		GetContent(dgraph),
	)
	server := httptest.NewServer(api)
	defer server.Close()

	// Make request
	resp, err := http.Get(server.URL + "/app/content/video-1")
	require.NoError(t, err, "failed to make request")
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed to read response body")
	bodyStr := string(body)

	// Assertions
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, bodyStr, "Vacation Video")
	assert.Contains(t, bodyStr, "A short clip")
	assert.Contains(t, bodyStr, "Vacation 2024")
}

func TestGetContent_NotFound(t *testing.T) {
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
		GetContent(dgraph),
	)
	server := httptest.NewServer(api)
	defer server.Close()

	// Make request for non-existent content
	resp, err := http.Get(server.URL + "/app/content/non-existent-id")
	require.NoError(t, err, "failed to make request")
	defer resp.Body.Close()

	// Assertions - should return error
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestGetContent_EmptyID(t *testing.T) {
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
		GetContent(dgraph),
	)
	server := httptest.NewServer(api)
	defer server.Close()

	// Make request with empty content ID
	resp, err := http.Get(server.URL + "/app/content/")
	require.NoError(t, err, "failed to make request")
	defer resp.Body.Close()

	// Assertions - should return error
	assert.NotEqual(t, http.StatusOK, resp.StatusCode)
}
