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
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/dgraph-io/dgo/v240"
	"github.com/dgraph-io/dgo/v240/protos/api"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/z5labs/humus"
	"github.com/z5labs/humus/rest"
)

//go:embed templates/location_section.html
var locationSectionTemplate string

type UpdateLocationRequest struct {
	ContentID string
	Latitude  *float64
	Longitude *float64
	Altitude  *float64
}

func (r *UpdateLocationRequest) ReadRequest(ctx context.Context, req *http.Request) error {
	if err := req.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	r.ContentID = rest.PathParamValue(ctx, "contentID")
	if r.ContentID == "" {
		return fmt.Errorf("content ID is required")
	}

	// Parse latitude (optional - if empty string, set to nil)
	latStr := strings.TrimSpace(req.FormValue("latitude"))
	if latStr != "" {
		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			return fmt.Errorf("invalid latitude: must be a number")
		}
		if lat < -90 || lat > 90 {
			return fmt.Errorf("invalid latitude: must be between -90 and 90")
		}
		r.Latitude = &lat
	}

	// Parse longitude (optional - if empty string, set to nil)
	lonStr := strings.TrimSpace(req.FormValue("longitude"))
	if lonStr != "" {
		lon, err := strconv.ParseFloat(lonStr, 64)
		if err != nil {
			return fmt.Errorf("invalid longitude: must be a number")
		}
		if lon < -180 || lon > 180 {
			return fmt.Errorf("invalid longitude: must be between -180 and 180")
		}
		r.Longitude = &lon
	}

	// Parse altitude (optional)
	altStr := strings.TrimSpace(req.FormValue("altitude"))
	if altStr != "" {
		alt, err := strconv.ParseFloat(altStr, 64)
		if err != nil {
			return fmt.Errorf("invalid altitude: must be a number")
		}
		r.Altitude = &alt
	}

	// Validation: both lat and lon must be present or both absent
	if (r.Latitude == nil) != (r.Longitude == nil) {
		return fmt.Errorf("latitude and longitude must both be provided or both be empty")
	}

	return nil
}

func (r *UpdateLocationRequest) Spec() (openapi3.RequestBodyOrRef, error) {
	desc := "Form data for updating content location"
	return openapi3.RequestBodyOrRef{
		RequestBody: &openapi3.RequestBody{
			Description: &desc,
			Content: map[string]openapi3.MediaType{
				"application/x-www-form-urlencoded": {},
			},
		},
	}, nil
}

type updateLocationHandler struct {
	dgraph   *dgo.Dgraph
	template *template.Template
}

func UpdateLocation(dgraph *dgo.Dgraph) rest.ApiOption {
	tmpl, err := template.New("location_section.html").Funcs(template.FuncMap{
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
	}).Parse(locationSectionTemplate)
	if err != nil {
		panic(err)
	}

	h := &updateLocationHandler{
		dgraph:   dgraph,
		template: tmpl,
	}

	return rest.Operation(
		http.MethodPut,
		rest.BasePath("/app/content").Param("contentID").Segment("location"),
		h,
	)
}

func (h *updateLocationHandler) Handle(ctx context.Context, req *UpdateLocationRequest) (*HtmlResponse, error) {
	log := humus.Logger("update_location")

	// First, get the content UID and all other fields
	query := `query getContent($contentID: string) {
		content(func: eq(content.id, $contentID)) @filter(type(Content)) {
			uid
			content.id
			content.type
			content.minio_key
			content.title
			content.description
			content.uploaded_at
			content.file_size
			content.mime_type
			content.captured_at
		}
	}`

	vars := map[string]string{"$contentID": req.ContentID}
	resp, err := h.dgraph.NewReadOnlyTxn().QueryWithVars(ctx, query, vars)
	if err != nil {
		return nil, fmt.Errorf("failed to query content: %w", err)
	}

	var result struct {
		Content []contentNode `json:"content"`
	}
	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal content: %w", err)
	}

	if len(result.Content) == 0 {
		return nil, fmt.Errorf("content not found")
	}

	contentUID := result.Content[0].UID
	existingContent := result.Content[0]

	log.Info("updating location",
		slog.String("content_id", req.ContentID),
		slog.String("uid", contentUID),
		slog.Any("latitude", req.Latitude),
		slog.Any("longitude", req.Longitude),
		slog.Any("altitude", req.Altitude))

	// Update the location fields
	updateNode := map[string]interface{}{
		"uid":               contentUID,
		"content.latitude":  req.Latitude,
		"content.longitude": req.Longitude,
		"content.altitude":  req.Altitude,
	}

	jsonData, err := json.Marshal(updateNode)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal update: %w", err)
	}

	txn := h.dgraph.NewTxn()
	defer txn.Discard(ctx)

	mu := &api.Mutation{
		SetJson:   jsonData,
		CommitNow: true,
	}

	_, err = txn.Mutate(ctx, mu)
	if err != nil {
		return nil, fmt.Errorf("failed to update location in Dgraph: %w", err)
	}

	log.Info("successfully updated location")

	// Build the updated Content model for template
	contentModel := Content{
		ID:          existingContent.ContentID,
		Type:        existingContent.Type,
		MinioKey:    existingContent.MinioKey,
		Title:       existingContent.Title,
		Description: existingContent.Description,
		UploadedAt:  existingContent.UploadedAt,
		FileSize:    existingContent.FileSize,
		MimeType:    existingContent.MimeType,
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
		Altitude:    req.Altitude,
		CapturedAt:  existingContent.CapturedAt,
	}

	var htmlBuf bytes.Buffer
	if err := h.template.Execute(&htmlBuf, contentModel); err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	// Close the modal by clearing its content
	htmlBuf.WriteString(`<div id="location-form-modal" hx-swap-oob="innerHTML"></div>`)

	return &HtmlResponse{
		ContentType: "text/html; charset=utf-8",
		Body:        htmlBuf.Bytes(),
	}, nil
}
