// Copyright (c) 2025 Z5labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package ui

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/dgraph-io/dgo/v240"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/z5labs/humus/rest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestUpdateLocation_Success_SetLocation(t *testing.T) {
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
			Title:       "Beach Photo",
			Description: "A photo from the beach",
			MimeType:    "image/jpeg",
			FileSize:    102400,
		},
	}

	journey := testJourney{ID: "test-journey", Title: "Beach Trip"}
	err = seedJourneyWithContent(ctx, dgraph, journey, contents)
	require.NoError(t, err, "failed to seed journey with content")

	// Create API and test server
	api := rest.NewApi(
		"Test API",
		"v1.0.0",
		UpdateLocation(dgraph),
	)
	server := httptest.NewServer(api)
	defer server.Close()

	// Make PUT request with location data
	formData := url.Values{}
	formData.Set("latitude", "34.0522")
	formData.Set("longitude", "-118.2437")
	formData.Set("altitude", "100.5")

	req, err := http.NewRequest(
		http.MethodPut,
		server.URL+"/app/content/photo-1/location",
		strings.NewReader(formData.Encode()),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "failed to make request")
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed to read response body")
	bodyStr := string(body)

	// Assertions
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/html; charset=utf-8", resp.Header.Get("Content-Type"))
	assert.Contains(t, bodyStr, "34.052200")   // Latitude in formatted output
	assert.Contains(t, bodyStr, "-118.243700") // Longitude in formatted output
	assert.Contains(t, bodyStr, "100.5")       // Altitude in output

	// Verify location was updated in Dgraph
	query := `{
		content(func: eq(content.id, "photo-1")) @filter(type(Content)) {
			content.latitude
			content.longitude
			content.altitude
		}
	}`

	txn := dgraph.NewReadOnlyTxn()
	defer txn.Discard(ctx)

	queryResp, err := txn.Query(ctx, query)
	require.NoError(t, err, "failed to query updated content")

	var result struct {
		Content []struct {
			Latitude  *float64 `json:"content.latitude"`
			Longitude *float64 `json:"content.longitude"`
			Altitude  *float64 `json:"content.altitude"`
		} `json:"content"`
	}

	err = json.Unmarshal(queryResp.Json, &result)
	require.NoError(t, err, "failed to unmarshal query result")

	require.Len(t, result.Content, 1)
	assert.NotNil(t, result.Content[0].Latitude)
	assert.NotNil(t, result.Content[0].Longitude)
	assert.NotNil(t, result.Content[0].Altitude)
	assert.InDelta(t, 34.0522, *result.Content[0].Latitude, 0.0001)
	assert.InDelta(t, -118.2437, *result.Content[0].Longitude, 0.0001)
	assert.InDelta(t, 100.5, *result.Content[0].Altitude, 0.1)
}

func TestUpdateLocation_Success_UpdateExisting(t *testing.T) {
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

	// Seed content with initial location
	lat, lon, alt := 37.7749, -122.4194, 50.0
	contents := []testContent{
		{
			ID:          "photo-1",
			Type:        "photo",
			MinioKey:    "journeys/test/photo-1.jpg",
			Title:       "SF Photo",
			Description: "From SF",
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
		UpdateLocation(dgraph),
	)
	server := httptest.NewServer(api)
	defer server.Close()

	// Make PUT request to update location
	formData := url.Values{}
	formData.Set("latitude", "40.7128")
	formData.Set("longitude", "-74.0060")
	formData.Set("altitude", "10.0")

	req, err := http.NewRequest(
		http.MethodPut,
		server.URL+"/app/content/photo-1/location",
		strings.NewReader(formData.Encode()),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "failed to make request")
	defer resp.Body.Close()

	// Assertions
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify location was updated
	query := `{
		content(func: eq(content.id, "photo-1")) @filter(type(Content)) {
			content.latitude
			content.longitude
			content.altitude
		}
	}`

	txn := dgraph.NewReadOnlyTxn()
	defer txn.Discard(ctx)

	queryResp, err := txn.Query(ctx, query)
	require.NoError(t, err)

	var result struct {
		Content []struct {
			Latitude  *float64 `json:"content.latitude"`
			Longitude *float64 `json:"content.longitude"`
			Altitude  *float64 `json:"content.altitude"`
		} `json:"content"`
	}

	err = json.Unmarshal(queryResp.Json, &result)
	require.NoError(t, err)

	require.Len(t, result.Content, 1)
	assert.InDelta(t, 40.7128, *result.Content[0].Latitude, 0.0001)
	assert.InDelta(t, -74.0060, *result.Content[0].Longitude, 0.0001)
	assert.InDelta(t, 10.0, *result.Content[0].Altitude, 0.1)
}

func TestUpdateLocation_ValidationError_LatitudeOnly(t *testing.T) {
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

	// Seed content
	contents := []testContent{
		{
			ID:          "photo-1",
			Type:        "photo",
			MinioKey:    "journeys/test/photo-1.jpg",
			Title:       "Photo",
			Description: "Test",
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
		UpdateLocation(dgraph),
	)
	server := httptest.NewServer(api)
	defer server.Close()

	// Make PUT request with only latitude
	formData := url.Values{}
	formData.Set("latitude", "34.0522")

	req, err := http.NewRequest(
		http.MethodPut,
		server.URL+"/app/content/photo-1/location",
		strings.NewReader(formData.Encode()),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "failed to make request")
	defer resp.Body.Close()

	// Assertions - should return error
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestUpdateLocation_ValidationError_InvalidLatitude(t *testing.T) {
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

	// Seed content
	contents := []testContent{
		{
			ID:          "photo-1",
			Type:        "photo",
			MinioKey:    "journeys/test/photo-1.jpg",
			Title:       "Photo",
			Description: "Test",
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
		UpdateLocation(dgraph),
	)
	server := httptest.NewServer(api)
	defer server.Close()

	// Make PUT request with invalid latitude (> 90)
	formData := url.Values{}
	formData.Set("latitude", "91.0")
	formData.Set("longitude", "0.0")

	req, err := http.NewRequest(
		http.MethodPut,
		server.URL+"/app/content/photo-1/location",
		strings.NewReader(formData.Encode()),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "failed to make request")
	defer resp.Body.Close()

	// Assertions - should return error
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestUpdateLocation_NotFound(t *testing.T) {
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
		UpdateLocation(dgraph),
	)
	server := httptest.NewServer(api)
	defer server.Close()

	// Make PUT request for non-existent content
	formData := url.Values{}
	formData.Set("latitude", "34.0522")
	formData.Set("longitude", "-118.2437")

	req, err := http.NewRequest(
		http.MethodPut,
		server.URL+"/app/content/non-existent-id/location",
		strings.NewReader(formData.Encode()),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "failed to make request")
	defer resp.Body.Close()

	// Assertions - should return error
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}
