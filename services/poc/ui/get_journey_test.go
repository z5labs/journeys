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
	"github.com/z5labs/humus/rest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Extended test schema including Content type for GetJourney tests
const getJourneyTestSchema = `
type Journey {
	journey.id: string
	journey.title: string
	journey.created_at: datetime
	journey.content: [uid]
}

type Content {
	content.id: string
	content.type: string
	content.minio_key: string
	content.title: string
	content.description: string
	content.uploaded_at: datetime
	content.file_size: int
	content.mime_type: string
	content.latitude: float
	content.longitude: float
	content.altitude: float
	content.location_name: string
	content.captured_at: datetime
}

journey.id: string @index(exact) .
journey.title: string .
journey.created_at: datetime .
journey.content: [uid] @reverse .
content.id: string @index(exact) .
content.type: string @index(exact) .
content.minio_key: string .
content.title: string .
content.description: string .
content.uploaded_at: datetime .
content.file_size: int .
content.mime_type: string .
content.latitude: float .
content.longitude: float .
content.altitude: float .
content.location_name: string .
content.captured_at: datetime .
`

type testContent struct {
	ID           string     `json:"content.id"`
	Type         string     `json:"content.type"`
	MinioKey     string     `json:"content.minio_key"`
	Title        string     `json:"content.title"`
	Description  string     `json:"content.description"`
	MimeType     string     `json:"content.mime_type"`
	FileSize     int64      `json:"content.file_size"`
	Latitude     *float64   `json:"content.latitude,omitempty"`
	Longitude    *float64   `json:"content.longitude,omitempty"`
	Altitude     *float64   `json:"content.altitude,omitempty"`
	LocationName string     `json:"content.location_name,omitempty"`
	CapturedAt   *time.Time `json:"content.captured_at,omitempty"`
}

func initializeGetJourneySchema(ctx context.Context, dgraph *dgo.Dgraph) error {
	return dgraph.Alter(ctx, &api.Operation{Schema: getJourneyTestSchema})
}

func seedJourneyWithContent(ctx context.Context, dgraph *dgo.Dgraph, journey testJourney, contents []testContent) error {
	txn := dgraph.NewTxn()
	defer txn.Discard(ctx)

	// Create journey first with a named UID
	journeyMutation := &api.Mutation{
		SetJson: []byte(fmt.Sprintf(`{
			"uid": "_:journey",
			"dgraph.type": "Journey",
			"journey.id": %q,
			"journey.title": %q,
			"journey.created_at": %q
		}`, journey.ID, journey.Title, time.Now().Format(time.RFC3339))),
		CommitNow: false,
	}

	journeyResp, err := txn.Mutate(ctx, journeyMutation)
	if err != nil {
		return fmt.Errorf("failed to create journey: %w", err)
	}

	journeyUID := journeyResp.Uids["journey"]
	if journeyUID == "" {
		return fmt.Errorf("failed to get journey UID")
	}

	// Create content items and link to journey
	for i, content := range contents {
		contentName := fmt.Sprintf("content%d", i)
		contentJSON := fmt.Sprintf(`{
			"uid": "_:%s",
			"dgraph.type": "Content",
			"content.id": %q,
			"content.type": %q,
			"content.minio_key": %q,
			"content.title": %q,
			"content.description": %q,
			"content.mime_type": %q,
			"content.file_size": %d,
			"content.uploaded_at": %q`,
			contentName,
			content.ID, content.Type, content.MinioKey, content.Title,
			content.Description, content.MimeType, content.FileSize,
			time.Now().Format(time.RFC3339))

		if content.Latitude != nil {
			contentJSON += fmt.Sprintf(`,
			"content.latitude": %f`, *content.Latitude)
		}
		if content.Longitude != nil {
			contentJSON += fmt.Sprintf(`,
			"content.longitude": %f`, *content.Longitude)
		}
		if content.Altitude != nil {
			contentJSON += fmt.Sprintf(`,
			"content.altitude": %f`, *content.Altitude)
		}
		if content.LocationName != "" {
			contentJSON += fmt.Sprintf(`,
			"content.location_name": %q`, content.LocationName)
		}
		if content.CapturedAt != nil {
			contentJSON += fmt.Sprintf(`,
			"content.captured_at": %q`, content.CapturedAt.Format(time.RFC3339))
		}

		contentJSON += "}"

		contentMutation := &api.Mutation{
			SetJson:   []byte(contentJSON),
			CommitNow: false,
		}

		contentResp, err := txn.Mutate(ctx, contentMutation)
		if err != nil {
			return fmt.Errorf("failed to create content %d: %w", i, err)
		}

		contentUID := contentResp.Uids[contentName]
		if contentUID == "" {
			return fmt.Errorf("failed to get content UID for content %d", i)
		}

		// Link content to journey
		linkMutation := &api.Mutation{
			SetNquads: []byte(fmt.Sprintf("<%s> <journey.content> <%s> .", journeyUID, contentUID)),
			CommitNow: false,
		}

		if _, err := txn.Mutate(ctx, linkMutation); err != nil {
			return fmt.Errorf("failed to link content %d to journey: %w", i, err)
		}
	}

	return txn.Commit(ctx)
}

func TestGetJourney_Success_WithContent(t *testing.T) {
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

	// Seed test data with content
	lat, lon, alt := 37.7749, -122.4194, 100.0
	capturedAt := time.Now().Add(-24 * time.Hour)
	contents := []testContent{
		{
			ID:           "photo-1",
			Type:         "photo",
			MinioKey:     "journeys/test-journey/photo-1.jpg",
			Title:        "Golden Gate Bridge",
			Description:  "View of the bridge",
			MimeType:     "image/jpeg",
			FileSize:     102400,
			Latitude:     &lat,
			Longitude:    &lon,
			Altitude:     &alt,
			LocationName: "San Francisco",
			CapturedAt:   &capturedAt,
		},
		{
			ID:          "video-1",
			Type:        "video",
			MinioKey:    "journeys/test-journey/video-1.mp4",
			Title:       "Bridge Timelapse",
			Description: "Sunset timelapse",
			MimeType:    "video/mp4",
			FileSize:    2048000,
		},
	}

	journey := testJourney{ID: "test-journey", Title: "San Francisco Trip"}
	err = seedJourneyWithContent(ctx, dgraph, journey, contents)
	require.NoError(t, err, "failed to seed journey with content")

	// Create API and test server
	api := rest.NewApi(
		"Test API",
		"v1.0.0",
		GetJourney(dgraph),
	)
	server := httptest.NewServer(api)
	defer server.Close()

	// Make request
	resp, err := http.Get(server.URL + "/app/journey/test-journey")
	require.NoError(t, err, "failed to make request")
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed to read response body")
	bodyStr := string(body)

	// Assertions
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/html; charset=utf-8", resp.Header.Get("Content-Type"))
	assert.Contains(t, bodyStr, "San Francisco Trip")
	assert.Contains(t, bodyStr, "Golden Gate Bridge")
	assert.Contains(t, bodyStr, "View of the bridge")
	assert.Contains(t, bodyStr, "Bridge Timelapse")
	assert.Contains(t, bodyStr, "photo-1")
	assert.Contains(t, bodyStr, "video-1")

	// Verify section headers appear
	assert.Contains(t, bodyStr, "Unknown Date") // for video-1 without CapturedAt

	// Verify date section appears (capturedAt was yesterday)
	yesterday := time.Now().Add(-24 * time.Hour).UTC().Format("2006-01-02")
	assert.Contains(t, bodyStr, yesterday)
}

func TestGetJourney_Success_NoContent(t *testing.T) {
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

	// Seed journey without content
	journey := testJourney{ID: "empty-journey", Title: "Empty Journey"}
	err = seedJourneys(ctx, dgraph, []testJourney{journey})
	require.NoError(t, err, "failed to seed journey")

	// Create API and test server
	api := rest.NewApi(
		"Test API",
		"v1.0.0",
		GetJourney(dgraph),
	)
	server := httptest.NewServer(api)
	defer server.Close()

	// Make request
	resp, err := http.Get(server.URL + "/app/journey/empty-journey")
	require.NoError(t, err, "failed to make request")
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed to read response body")
	bodyStr := string(body)

	// Assertions
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, bodyStr, "Empty Journey")
}

func TestGetJourney_NotFound(t *testing.T) {
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
		GetJourney(dgraph),
	)
	server := httptest.NewServer(api)
	defer server.Close()

	// Make request for non-existent journey
	resp, err := http.Get(server.URL + "/app/journey/non-existent-id")
	require.NoError(t, err, "failed to make request")
	defer resp.Body.Close()

	// Assertions - should return 500 error since journey not found
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestGetJourney_EmptyID(t *testing.T) {
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
		GetJourney(dgraph),
	)
	server := httptest.NewServer(api)
	defer server.Close()

	// Make request with empty journey ID
	resp, err := http.Get(server.URL + "/app/journey/")
	require.NoError(t, err, "failed to make request")
	defer resp.Body.Close()

	// Assertions - should return error
	assert.NotEqual(t, http.StatusOK, resp.StatusCode)
}

// Helper for tests
func ptrTime(t time.Time) *time.Time {
	return &t
}

func TestGroupContentByDate(t *testing.T) {
	tests := []struct {
		name     string
		contents []Content
		want     []ContentSection
	}{
		{
			name:     "empty content",
			contents: []Content{},
			want:     []ContentSection{},
		},
		{
			name: "all content without CapturedAt",
			contents: []Content{
				{ID: "1", Title: "Photo 1", UploadedAt: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)},
				{ID: "2", Title: "Photo 2", UploadedAt: time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC)},
			},
			want: []ContentSection{
				{
					DateKey:     "Unknown",
					DisplayDate: "Unknown Date",
					SortOrder:   -9223372036854775807,
					Contents: []Content{
						{ID: "1", Title: "Photo 1", UploadedAt: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)},
						{ID: "2", Title: "Photo 2", UploadedAt: time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC)},
					},
				},
			},
		},
		{
			name: "content grouped by two different dates",
			contents: []Content{
				{ID: "1", CapturedAt: ptrTime(time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC))},
				{ID: "2", CapturedAt: ptrTime(time.Date(2025, 1, 15, 14, 0, 0, 0, time.UTC))},
				{ID: "3", CapturedAt: ptrTime(time.Date(2025, 1, 16, 9, 0, 0, 0, time.UTC))},
			},
			want: []ContentSection{
				{
					DateKey:     "2025-01-16",
					DisplayDate: "2025-01-16",
					Contents:    []Content{{ID: "3", CapturedAt: ptrTime(time.Date(2025, 1, 16, 9, 0, 0, 0, time.UTC))}},
				},
				{
					DateKey:     "2025-01-15",
					DisplayDate: "2025-01-15",
					Contents: []Content{
						{ID: "1", CapturedAt: ptrTime(time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC))},
						{ID: "2", CapturedAt: ptrTime(time.Date(2025, 1, 15, 14, 0, 0, 0, time.UTC))},
					},
				},
			},
		},
		{
			name: "mixed: unknown date appears first",
			contents: []Content{
				{ID: "1", CapturedAt: ptrTime(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC))},
				{ID: "2", CapturedAt: nil, UploadedAt: time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)},
				{ID: "3", CapturedAt: ptrTime(time.Date(2025, 1, 16, 10, 0, 0, 0, time.UTC))},
			},
			want: []ContentSection{
				{
					DateKey:     "Unknown",
					DisplayDate: "Unknown Date",
					SortOrder:   -9223372036854775807,
					Contents:    []Content{{ID: "2", CapturedAt: nil, UploadedAt: time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)}},
				},
				{
					DateKey:     "2025-01-16",
					DisplayDate: "2025-01-16",
					Contents:    []Content{{ID: "3", CapturedAt: ptrTime(time.Date(2025, 1, 16, 10, 0, 0, 0, time.UTC))}},
				},
				{
					DateKey:     "2025-01-15",
					DisplayDate: "2025-01-15",
					Contents:    []Content{{ID: "1", CapturedAt: ptrTime(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC))}},
				},
			},
		},
		{
			name: "within section sorting: earliest to latest",
			contents: []Content{
				{ID: "1", CapturedAt: ptrTime(time.Date(2025, 1, 15, 14, 0, 0, 0, time.UTC))},
				{ID: "2", CapturedAt: ptrTime(time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC))},
				{ID: "3", CapturedAt: ptrTime(time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC))},
			},
			want: []ContentSection{
				{
					DateKey:     "2025-01-15",
					DisplayDate: "2025-01-15",
					Contents: []Content{
						{ID: "2", CapturedAt: ptrTime(time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC))},   // 9:00
						{ID: "3", CapturedAt: ptrTime(time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC))}, // 12:00
						{ID: "1", CapturedAt: ptrTime(time.Date(2025, 1, 15, 14, 0, 0, 0, time.UTC))}, // 14:00
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := groupContentByDate(tt.contents)

			assert.Equal(t, len(tt.want), len(got), "section count mismatch")

			for i := range tt.want {
				assert.Equal(t, tt.want[i].DateKey, got[i].DateKey, "DateKey mismatch at section %d", i)
				assert.Equal(t, tt.want[i].DisplayDate, got[i].DisplayDate, "DisplayDate mismatch at section %d", i)
				assert.Equal(t, len(tt.want[i].Contents), len(got[i].Contents), "Contents count mismatch at section %d", i)

				// Verify content IDs in correct order
				for j := range tt.want[i].Contents {
					assert.Equal(t, tt.want[i].Contents[j].ID, got[i].Contents[j].ID, "Content ID mismatch at section %d, item %d", i, j)
				}
			}
		})
	}
}
