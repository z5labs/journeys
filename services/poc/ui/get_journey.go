// Copyright (c) 2025 Z5labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package ui

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"time"

	"github.com/dgraph-io/dgo/v240"
	"github.com/z5labs/humus/rest"
)

//go:embed templates/journey_view.html
var journeyViewTemplate string

type getJourneyHandler struct {
	dgraph   *dgo.Dgraph
	template *template.Template
}

// GetJourney creates a REST operation for viewing a single journey by ID
func GetJourney(dgraph *dgo.Dgraph) rest.ApiOption {
	tmpl, err := template.New("journey_view.html").Parse(journeyViewTemplate)
	if err != nil {
		panic(err)
	}

	h := &getJourneyHandler{
		dgraph:   dgraph,
		template: tmpl,
	}

	return rest.Operation(
		http.MethodGet,
		rest.BasePath("/app/journey").Param("id"),
		h,
	)
}

// groupContentByDate groups and sorts content by captured date
func groupContentByDate(contents []Content) []ContentSection {
	// Map to accumulate content by date key
	sectionMap := make(map[string]*ContentSection)

	for _, content := range contents {
		var dateKey string
		var displayDate string
		var sortOrder int64

		if content.CapturedAt == nil {
			dateKey = "Unknown"
			displayDate = "Unknown Date"
			sortOrder = -9223372036854775807 // Very negative to sort first
		} else {
			// Normalize to UTC date for grouping
			utcDate := content.CapturedAt.UTC()
			dateKey = utcDate.Format("2006-01-02")
			displayDate = dateKey
			// Use negative timestamp for descending sort (newer dates have less negative values)
			sortOrder = -time.Date(
				utcDate.Year(), utcDate.Month(), utcDate.Day(),
				0, 0, 0, 0, time.UTC,
			).Unix()
		}

		// Get or create section
		section, exists := sectionMap[dateKey]
		if !exists {
			section = &ContentSection{
				DateKey:     dateKey,
				DisplayDate: displayDate,
				SortOrder:   sortOrder,
				Contents:    []Content{},
			}
			sectionMap[dateKey] = section
		}

		section.Contents = append(section.Contents, content)
	}

	// Convert map to slice
	sections := make([]ContentSection, 0, len(sectionMap))
	for _, section := range sectionMap {
		// Sort contents within section by time (earliest to latest)
		sort.Slice(section.Contents, func(i, j int) bool {
			a, b := section.Contents[i], section.Contents[j]

			// Items with CapturedAt come before those without
			if a.CapturedAt == nil && b.CapturedAt != nil {
				return false
			}
			if a.CapturedAt != nil && b.CapturedAt == nil {
				return true
			}
			if a.CapturedAt == nil && b.CapturedAt == nil {
				// Both nil: sort by upload time
				return a.UploadedAt.Before(b.UploadedAt)
			}

			// Both have CapturedAt: sort by captured time
			return a.CapturedAt.Before(*b.CapturedAt)
		})

		sections = append(sections, *section)
	}

	// Sort sections by SortOrder (Unknown=0 first, then negative timestamps)
	sort.Slice(sections, func(i, j int) bool {
		return sections[i].SortOrder < sections[j].SortOrder
	})

	return sections
}

func (h *getJourneyHandler) Handle(ctx context.Context, req *rest.EmptyRequest) (*HtmlResponse, error) {
	journeyID := rest.PathParamValue(ctx, "id")
	if journeyID == "" {
		return nil, fmt.Errorf("journey ID is required")
	}

	query := `
	query getJourney($id: string) {
		journey(func: eq(journey.id, $id)) @filter(type(Journey)) {
			journey.id
			journey.title
			journey.content {
				content.id
				content.type
				content.minio_key
				content.title
				content.description
				content.uploaded_at
				content.file_size
				content.mime_type
				content.latitude
				content.longitude
				content.altitude
				content.location_name
				content.captured_at
			}
		}
	}
	`

	txn := h.dgraph.NewReadOnlyTxn()
	defer txn.Discard(ctx)

	resp, err := txn.QueryWithVars(ctx, query, map[string]string{
		"$id": journeyID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query journey: %w", err)
	}

	var result struct {
		Journey []struct {
			ID       string `json:"journey.id"`
			Title    string `json:"journey.title"`
			Contents []struct {
				ID           string     `json:"content.id"`
				Type         string     `json:"content.type"`
				MinioKey     string     `json:"content.minio_key"`
				Title        string     `json:"content.title"`
				Description  string     `json:"content.description"`
				UploadedAt   time.Time  `json:"content.uploaded_at"`
				FileSize     int64      `json:"content.file_size"`
				MimeType     string     `json:"content.mime_type"`
				Latitude     *float64   `json:"content.latitude"`
				Longitude    *float64   `json:"content.longitude"`
				Altitude     *float64   `json:"content.altitude"`
				LocationName string     `json:"content.location_name"`
				CapturedAt   *time.Time `json:"content.captured_at"`
			} `json:"journey.content"`
		} `json:"journey"`
	}

	if err = json.Unmarshal(resp.Json, &result); err != nil {
		return nil, fmt.Errorf("failed to parse query result: %w", err)
	}

	if len(result.Journey) == 0 {
		return nil, fmt.Errorf("journey not found: %s", journeyID)
	}

	j := result.Journey[0]
	contents := make([]Content, len(j.Contents))
	for i, c := range j.Contents {
		contents[i] = Content{
			ID:           c.ID,
			Type:         c.Type,
			MinioKey:     c.MinioKey,
			Title:        c.Title,
			Description:  c.Description,
			UploadedAt:   c.UploadedAt,
			FileSize:     c.FileSize,
			MimeType:     c.MimeType,
			Latitude:     c.Latitude,
			Longitude:    c.Longitude,
			Altitude:     c.Altitude,
			LocationName: c.LocationName,
			CapturedAt:   c.CapturedAt,
		}
	}

	// Group content by date
	sections := groupContentByDate(contents)

	// Create view model
	viewModel := journeyViewModel{
		ID:       j.ID,
		Title:    j.Title,
		Sections: sections,
	}

	var buf bytes.Buffer
	err = h.template.Execute(&buf, viewModel)
	if err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	return &HtmlResponse{
		ContentType: "text/html; charset=utf-8",
		Body:        buf.Bytes(),
	}, nil
}
