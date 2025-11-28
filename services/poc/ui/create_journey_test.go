// Copyright (c) 2025 Z5labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package ui

import (
	"context"
	"encoding/json"
	"fmt"
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

func TestCreateJourney_Success(t *testing.T) {
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
	err = initializeDgraphSchema(ctx, dgraph)
	require.NoError(t, err, "failed to initialize schema")

	// Create API and test server
	api := rest.NewApi(
		"Test API",
		"v1.0.0",
		CreateJourney(dgraph),
	)
	server := httptest.NewServer(api)
	defer server.Close()

	// Make POST request with form data
	formData := url.Values{}
	formData.Set("title", "Summer Road Trip 2024")

	resp, err := http.Post(
		server.URL+"/app/journey",
		"application/x-www-form-urlencoded",
		strings.NewReader(formData.Encode()),
	)
	require.NoError(t, err, "failed to make request")
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed to read response body")
	bodyStr := string(body)

	// Assertions
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/html; charset=utf-8", resp.Header.Get("Content-Type"))
	assert.Contains(t, bodyStr, "Summer Road Trip 2024")
	assert.Contains(t, bodyStr, `class="journey-item"`)

	// Verify journey was created in Dgraph
	query := `{
		journey(func: type(Journey)) @filter(eq(journey.title, "Summer Road Trip 2024")) {
			journey.id
			journey.title
			journey.created_at
		}
	}`

	txn := dgraph.NewReadOnlyTxn()
	defer txn.Discard(ctx)

	queryResp, err := txn.Query(ctx, query)
	require.NoError(t, err, "failed to query created journey")

	var result struct {
		Journey []struct {
			ID        string `json:"journey.id"`
			Title     string `json:"journey.title"`
			CreatedAt string `json:"journey.created_at"`
		} `json:"journey"`
	}

	err = json.Unmarshal(queryResp.Json, &result)
	require.NoError(t, err, "failed to unmarshal query result")

	assert.Len(t, result.Journey, 1)
	assert.Equal(t, "Summer Road Trip 2024", result.Journey[0].Title)
	assert.NotEmpty(t, result.Journey[0].ID)
	assert.NotEmpty(t, result.Journey[0].CreatedAt)
}

func TestCreateJourney_ValidationError_EmptyTitle(t *testing.T) {
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
	err = initializeDgraphSchema(ctx, dgraph)
	require.NoError(t, err, "failed to initialize schema")

	// Create API and test server
	api := rest.NewApi(
		"Test API",
		"v1.0.0",
		CreateJourney(dgraph),
	)
	server := httptest.NewServer(api)
	defer server.Close()

	// Make POST request with empty title
	formData := url.Values{}
	formData.Set("title", "")

	resp, err := http.Post(
		server.URL+"/app/journey",
		"application/x-www-form-urlencoded",
		strings.NewReader(formData.Encode()),
	)
	require.NoError(t, err, "failed to make request")
	defer resp.Body.Close()

	// Assertions - should return error
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestCreateJourney_ValidationError_TitleTooLong(t *testing.T) {
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
	err = initializeDgraphSchema(ctx, dgraph)
	require.NoError(t, err, "failed to initialize schema")

	// Create API and test server
	api := rest.NewApi(
		"Test API",
		"v1.0.0",
		CreateJourney(dgraph),
	)
	server := httptest.NewServer(api)
	defer server.Close()

	// Make POST request with title > 100 characters
	longTitle := strings.Repeat("a", 101)
	formData := url.Values{}
	formData.Set("title", longTitle)

	resp, err := http.Post(
		server.URL+"/app/journey",
		"application/x-www-form-urlencoded",
		strings.NewReader(formData.Encode()),
	)
	require.NoError(t, err, "failed to make request")
	defer resp.Body.Close()

	// Assertions - should return error
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestCreateJourney_MultipleJourneys(t *testing.T) {
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
	err = initializeDgraphSchema(ctx, dgraph)
	require.NoError(t, err, "failed to initialize schema")

	// Create API and test server
	api := rest.NewApi(
		"Test API",
		"v1.0.0",
		CreateJourney(dgraph),
	)
	server := httptest.NewServer(api)
	defer server.Close()

	// Create multiple journeys
	titles := []string{"Journey 1", "Journey 2", "Journey 3"}
	for _, title := range titles {
		formData := url.Values{}
		formData.Set("title", title)

		resp, err := http.Post(
			server.URL+"/app/journey",
			"application/x-www-form-urlencoded",
			strings.NewReader(formData.Encode()),
		)
		require.NoError(t, err, "failed to make request")
		resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	// Verify all journeys were created
	query := `{
		journeys(func: type(Journey)) {
			journey.id
			journey.title
		}
	}`

	txn := dgraph.NewReadOnlyTxn()
	defer txn.Discard(ctx)

	queryResp, err := txn.Query(ctx, query)
	require.NoError(t, err, "failed to query journeys")

	var result struct {
		Journeys []struct {
			ID    string `json:"journey.id"`
			Title string `json:"journey.title"`
		} `json:"journeys"`
	}

	err = json.Unmarshal(queryResp.Json, &result)
	require.NoError(t, err, "failed to unmarshal query result")

	assert.Len(t, result.Journeys, 3)

	// Verify each journey has a unique ID and correct title
	foundTitles := make(map[string]bool)
	ids := make(map[string]bool)
	for _, j := range result.Journeys {
		assert.NotEmpty(t, j.ID, "journey should have an ID")
		assert.False(t, ids[j.ID], "journey IDs should be unique")
		ids[j.ID] = true
		foundTitles[j.Title] = true
	}

	for _, title := range titles {
		assert.True(t, foundTitles[title], fmt.Sprintf("should have created journey with title %s", title))
	}
}
