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

	"github.com/dgraph-io/dgo/v240"
	"github.com/z5labs/humus"
	"github.com/z5labs/humus/rest"
)

//go:embed templates/location_edit_form.html
var locationEditFormTemplate string

type locationEditFormHandler struct {
	dgraph   *dgo.Dgraph
	template *template.Template
}

func GetLocationEditForm(dgraph *dgo.Dgraph) rest.ApiOption {
	tmpl, err := template.New("location_edit_form.html").Parse(locationEditFormTemplate)
	if err != nil {
		panic(err)
	}

	h := &locationEditFormHandler{
		dgraph:   dgraph,
		template: tmpl,
	}

	return rest.Operation(
		http.MethodGet,
		rest.BasePath("/app/content").Param("contentID").Segment("location").Segment("form"),
		h,
	)
}

type LocationFormData struct {
	ContentID string
	Latitude  *float64
	Longitude *float64
	Altitude  *float64
}

func (h *locationEditFormHandler) Handle(ctx context.Context, req *rest.EmptyRequest) (*HtmlResponse, error) {
	log := humus.Logger("get_location_edit_form")

	contentID := rest.PathParamValue(ctx, "contentID")
	if contentID == "" {
		return nil, fmt.Errorf("content ID is required")
	}

	log.Info("fetching location form", slog.String("content_id", contentID))

	// Query current location data
	query := `query getContentLocation($contentID: string) {
		content(func: eq(content.id, $contentID)) @filter(type(Content)) {
			content.id
			content.latitude
			content.longitude
			content.altitude
		}
	}`

	vars := map[string]string{"$contentID": contentID}
	resp, err := h.dgraph.NewReadOnlyTxn().QueryWithVars(ctx, query, vars)
	if err != nil {
		log.Error("failed to query content", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to query content: %w", err)
	}

	log.Info("dgraph response", slog.String("json", string(resp.Json)))

	var result struct {
		Content []struct {
			ID        string   `json:"content.id"`
			Latitude  *float64 `json:"content.latitude"`
			Longitude *float64 `json:"content.longitude"`
			Altitude  *float64 `json:"content.altitude"`
		} `json:"content"`
	}
	if err := json.Unmarshal(resp.Json, &result); err != nil {
		log.Error("failed to unmarshal content", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to unmarshal content: %w", err)
	}

	if len(result.Content) == 0 {
		log.Error("content not found", slog.String("content_id", contentID))
		return nil, fmt.Errorf("content not found")
	}

	log.Info("found content",
		slog.String("content_id", contentID),
		slog.Any("latitude", result.Content[0].Latitude),
		slog.Any("longitude", result.Content[0].Longitude),
		slog.Any("altitude", result.Content[0].Altitude))

	formData := LocationFormData{
		ContentID: contentID,
		Latitude:  result.Content[0].Latitude,
		Longitude: result.Content[0].Longitude,
		Altitude:  result.Content[0].Altitude,
	}

	var buf bytes.Buffer
	err = h.template.Execute(&buf, formData)
	if err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	return &HtmlResponse{
		ContentType: "text/html; charset=utf-8",
		Body:        buf.Bytes(),
	}, nil
}
