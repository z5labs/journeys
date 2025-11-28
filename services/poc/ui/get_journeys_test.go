// Copyright (c) 2025 Z5labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package ui

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dgraph-io/dgo/v240"
	"github.com/dgraph-io/dgo/v240/protos/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/z5labs/humus/rest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// testDgraphSchema is a minimal Dgraph schema containing only Journey type fields
// needed for testing GetJourneys functionality. This is a subset of the full
// schema defined in app/schema.go.
const testDgraphSchema = `
type Journey {
	journey.id: string
	journey.title: string
	journey.created_at: datetime
}

journey.id: string @index(exact) .
journey.title: string .
journey.created_at: datetime .
`

// Test fixture data representing sample journeys used across multiple tests
var (
	journey1 = testJourney{ID: "journey-1", Title: "Summer Road Trip"}
	journey2 = testJourney{ID: "journey-2", Title: "Winter Ski Adventure"}
	journey3 = testJourney{ID: "journey-3", Title: "Spring Hiking Trail"}
)

// dgraphContainer wraps a testcontainer with the gRPC endpoint for Dgraph connections
type dgraphContainer struct {
	testcontainers.Container
	grpcEndpoint string
}

// testJourney represents a minimal journey structure for test data seeding
type testJourney struct {
	ID    string `json:"journey.id"`
	Title string `json:"journey.title"`
}

// setupDgraphContainer creates and starts a Dgraph container for testing.
// The container uses the dgraph/standalone image and exposes the gRPC port (9080).
// Wait strategy ensures the container is ready by checking for the "Server is ready"
// log message and confirming the port is listening (30s timeout).
// Containers automatically get unique ports to avoid conflicts when running tests in parallel.
func setupDgraphContainer(ctx context.Context) (*dgraphContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "dgraph/standalone:latest",
		ExposedPorts: []string{"9080/tcp"},
		WaitingFor: wait.ForAll(
			wait.ForLog("Server is ready"),
			wait.ForListeningPort("9080/tcp"),
		).WithDeadline(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	mappedPort, err := container.MappedPort(ctx, "9080")
	if err != nil {
		return nil, fmt.Errorf("failed to get mapped port: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get host: %w", err)
	}

	endpoint := fmt.Sprintf("%s:%s", host, mappedPort.Port())

	return &dgraphContainer{
		Container:    container,
		grpcEndpoint: endpoint,
	}, nil
}

// initializeDgraphSchema applies the test schema to the Dgraph instance.
// This must be called before seeding data or running queries.
func initializeDgraphSchema(ctx context.Context, dgraph *dgo.Dgraph) error {
	return dgraph.Alter(ctx, &api.Operation{Schema: testDgraphSchema})
}

// seedJourneys inserts test journey data into Dgraph.
// All journeys are inserted in a single transaction for atomicity.
// The created_at timestamp is set to the current time for all journeys.
func seedJourneys(ctx context.Context, dgraph *dgo.Dgraph, journeys []testJourney) error {
	txn := dgraph.NewTxn()
	defer txn.Discard(ctx)

	for _, j := range journeys {
		mutation := &api.Mutation{
			SetJson: []byte(fmt.Sprintf(`{
				"dgraph.type": "Journey",
				"journey.id": %q,
				"journey.title": %q,
				"journey.created_at": %q
			}`, j.ID, j.Title, time.Now().Format(time.RFC3339))),
			CommitNow: false,
		}

		if _, err := txn.Mutate(ctx, mutation); err != nil {
			return fmt.Errorf("failed to mutate journey %s: %w", j.ID, err)
		}
	}

	return txn.Commit(ctx)
}

func TestGetJourneys_Success_MultipleJourneys(t *testing.T) {
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

	// Seed test data
	err = seedJourneys(ctx, dgraph, []testJourney{journey1, journey2, journey3})
	require.NoError(t, err, "failed to seed journeys")

	// Create API and test server
	api := rest.NewApi(
		"Test API",
		"v1.0.0",
		GetJourneys(dgraph),
	)
	server := httptest.NewServer(api)
	defer server.Close()

	// Make request
	resp, err := http.Get(server.URL + "/app")
	require.NoError(t, err, "failed to make request")
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed to read response body")
	bodyStr := string(body)

	// Assertions
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/html; charset=utf-8", resp.Header.Get("Content-Type"))
	assert.Contains(t, bodyStr, "Summer Road Trip")
	assert.Contains(t, bodyStr, "Winter Ski Adventure")
	assert.Contains(t, bodyStr, "Spring Hiking Trail")
	assert.Contains(t, bodyStr, "<h1>My Journeys</h1>")
	assert.Contains(t, bodyStr, `id="journeys-list"`)
	assert.Contains(t, bodyStr, "htmx.org")
}

func TestGetJourneys_Success_SingleJourney(t *testing.T) {
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

	// Seed single journey
	err = seedJourneys(ctx, dgraph, []testJourney{journey1})
	require.NoError(t, err, "failed to seed journey")

	// Create API and test server
	api := rest.NewApi(
		"Test API",
		"v1.0.0",
		GetJourneys(dgraph),
	)
	server := httptest.NewServer(api)
	defer server.Close()

	// Make request
	resp, err := http.Get(server.URL + "/app")
	require.NoError(t, err, "failed to make request")
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed to read response body")
	bodyStr := string(body)

	// Assertions
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, bodyStr, "Summer Road Trip")
	assert.Contains(t, bodyStr, "<h1>My Journeys</h1>")
}

func TestGetJourneys_Success_NoJourneys(t *testing.T) {
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

	// Initialize schema (but don't seed any journeys)
	err = initializeDgraphSchema(ctx, dgraph)
	require.NoError(t, err, "failed to initialize schema")

	// Create API and test server
	api := rest.NewApi(
		"Test API",
		"v1.0.0",
		GetJourneys(dgraph),
	)
	server := httptest.NewServer(api)
	defer server.Close()

	// Make request
	resp, err := http.Get(server.URL + "/app")
	require.NoError(t, err, "failed to make request")
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed to read response body")
	bodyStr := string(body)

	// Assertions
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, bodyStr, "<h1>My Journeys</h1>")
	assert.Contains(t, bodyStr, `id="journeys-list"`)
	assert.NotContains(t, bodyStr, `class="journey-item"`)
}

func TestGetJourneys_FiltersEmptyIDs(t *testing.T) {
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

	// Manually insert journey with empty ID
	txn := dgraph.NewTxn()
	mutation := &api.Mutation{
		SetJson: []byte(fmt.Sprintf(`{
			"dgraph.type": "Journey",
			"journey.id": "",
			"journey.title": "Journey With Empty ID",
			"journey.created_at": %q
		}`, time.Now().Format(time.RFC3339))),
		CommitNow: true,
	}
	_, err = txn.Mutate(ctx, mutation)
	require.NoError(t, err, "failed to insert journey with empty ID")

	// Seed valid journey
	err = seedJourneys(ctx, dgraph, []testJourney{journey1})
	require.NoError(t, err, "failed to seed valid journey")

	// Create API and test server
	api := rest.NewApi(
		"Test API",
		"v1.0.0",
		GetJourneys(dgraph),
	)
	server := httptest.NewServer(api)
	defer server.Close()

	// Make request
	resp, err := http.Get(server.URL + "/app")
	require.NoError(t, err, "failed to make request")
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed to read response body")
	bodyStr := string(body)

	// Assertions - should contain valid journey but not the one with empty ID
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, bodyStr, "Summer Road Trip")
	assert.NotContains(t, bodyStr, "Journey With Empty ID")
}
