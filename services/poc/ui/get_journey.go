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

func (h *getJourneyHandler) Handle(ctx context.Context, req *rest.EmptyRequest) (*HtmlResponse, error) {
	// Extract journey ID from path parameter
	// Note: PathParamValue can panic if the parameter wasn't injected into context
	journeyID := rest.PathParamValue(ctx, "id")
	if journeyID == "" {
		return nil, fmt.Errorf("journey ID is required")
	}

	// Query Dgraph for the specific journey
	query := `
	query getJourney($id: string) {
		journey(func: eq(journey.id, $id)) @filter(type(Journey)) {
			journey.id
			journey.title
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

	// Parse response
	var result struct {
		Journey []struct {
			ID    string `json:"journey.id"`
			Title string `json:"journey.title"`
		} `json:"journey"`
	}

	if err = json.Unmarshal(resp.Json, &result); err != nil {
		return nil, fmt.Errorf("failed to parse query result: %w", err)
	}

	// Check if journey was found
	if len(result.Journey) == 0 {
		return nil, fmt.Errorf("journey not found: %s", journeyID)
	}

	// Create journey model for template
	journeyModel := journey{
		ID:    result.Journey[0].ID,
		Title: result.Journey[0].Title,
	}

	// Execute template
	var buf bytes.Buffer
	err = h.template.Execute(&buf, journeyModel)
	if err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	return &HtmlResponse{
		ContentType: "text/html; charset=utf-8",
		Body:        buf.Bytes(),
	}, nil
}
