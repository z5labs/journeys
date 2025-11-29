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
	"time"

	"github.com/dgraph-io/dgo/v240"
	"github.com/z5labs/humus/rest"
)

//go:embed templates/content_view.html
var contentViewTemplate string

type getContentHandler struct {
	dgraph   *dgo.Dgraph
	template *template.Template
}

// GetContent creates a REST operation for viewing a single content item by ID
func GetContent(dgraph *dgo.Dgraph) rest.ApiOption {
	tmpl, err := template.New("content_view.html").Funcs(template.FuncMap{
		"deref": func(ptr interface{}) interface{} {
			if ptr == nil {
				return nil
			}
			switch v := ptr.(type) {
			case *float64:
				if v == nil {
					return nil
				}
				return *v
			case *int:
				if v == nil {
					return nil
				}
				return *v
			}
			return ptr
		},
	}).Parse(contentViewTemplate)
	if err != nil {
		panic(err)
	}

	h := &getContentHandler{
		dgraph:   dgraph,
		template: tmpl,
	}

	return rest.Operation(
		http.MethodGet,
		rest.BasePath("/app/content").Param("id"),
		h,
	)
}

func (h *getContentHandler) Handle(ctx context.Context, req *rest.EmptyRequest) (*HtmlResponse, error) {
	contentID := rest.PathParamValue(ctx, "id")
	if contentID == "" {
		return nil, fmt.Errorf("content ID is required")
	}

	query := `
	query getContent($id: string) {
		content(func: eq(content.id, $id)) @filter(type(Content)) {
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
			~journey.content {
				journey.id
				journey.title
			}
		}
	}
	`

	txn := h.dgraph.NewReadOnlyTxn()
	defer txn.Discard(ctx)

	resp, err := txn.QueryWithVars(ctx, query, map[string]string{
		"$id": contentID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query content: %w", err)
	}

	var result struct {
		Content []struct {
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
			Journey      []struct {
				ID    string `json:"journey.id"`
				Title string `json:"journey.title"`
			} `json:"~journey.content"`
		} `json:"content"`
	}

	if err = json.Unmarshal(resp.Json, &result); err != nil {
		return nil, fmt.Errorf("failed to parse query result: %w", err)
	}

	if len(result.Content) == 0 {
		return nil, fmt.Errorf("content not found: %s", contentID)
	}

	c := result.Content[0]
	contentModel := Content{
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

	// Extract journey info for back navigation
	var journeyID, journeyTitle string
	if len(c.Journey) > 0 {
		journeyID = c.Journey[0].ID
		journeyTitle = c.Journey[0].Title
	}

	viewModel := struct {
		Content      Content
		JourneyID    string
		JourneyTitle string
	}{
		Content:      contentModel,
		JourneyID:    journeyID,
		JourneyTitle: journeyTitle,
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
