package ui

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"html/template"
	"net/http"

	"github.com/dgraph-io/dgo/v240"
	"github.com/z5labs/humus/rest"
)

//go:embed templates/journeys_list.html
var journeysListTemplate string

//go:embed templates/journey_item.html
var journeyItemTemplate string

//go:embed templates/create_journey_button.html
var createJourneyButtonTemplate string

//go:embed templates/create_journey_form.html
var createJourneyFormTemplate string

type getJourneysHandler struct {
	dgraph   *dgo.Dgraph
	template *template.Template
}

func GetJourneys(dgraph *dgo.Dgraph) rest.ApiOption {
	// Parse all templates
	tmpl, err := template.New("journeys_list.html").Parse(journeysListTemplate)
	if err != nil {
		panic(err) // Configuration error - fail fast
	}

	tmpl, err = tmpl.New("journey_item.html").Parse(journeyItemTemplate)
	if err != nil {
		panic(err)
	}

	tmpl, err = tmpl.New("create_journey_button.html").Parse(createJourneyButtonTemplate)
	if err != nil {
		panic(err)
	}

	tmpl, err = tmpl.New("create_journey_form.html").Parse(createJourneyFormTemplate)
	if err != nil {
		panic(err)
	}

	h := &getJourneysHandler{
		dgraph:   dgraph,
		template: tmpl,
	}

	return rest.Operation(
		http.MethodGet,
		rest.BasePath("/app"),
		h,
	)
}

func (h *getJourneysHandler) Handle(ctx context.Context, req *rest.EmptyRequest) (*HtmlResponse, error) {
	// Query Dgraph for all journeys
	query := `
	{
		journeys(func: type(Journey)) {
			journey.id
			journey.title
		}
	}
	`

	// Create read-only transaction
	txn := h.dgraph.NewReadOnlyTxn()
	defer txn.Discard(ctx)

	// Execute query
	resp, err := txn.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	// Parse response
	var result struct {
		Journeys []struct {
			ID    string `json:"journey.id"`
			Title string `json:"journey.title"`
		} `json:"journeys"`
	}

	if err = json.Unmarshal(resp.Json, &result); err != nil {
		return nil, err
	}

	// Convert to journey slice
	journeys := make([]journey, len(result.Journeys))
	for i, j := range result.Journeys {
		journeys[i] = journey{
			ID:    j.ID,
			Title: j.Title,
		}
	}

	// Prepare template data
	data := struct {
		Journeys []journey
	}{
		Journeys: journeys,
	}

	// Execute template
	var buf bytes.Buffer
	err = h.template.ExecuteTemplate(&buf, "journeys_list.html", data)
	if err != nil {
		return nil, err
	}

	// Return HTML response
	return &HtmlResponse{
		ContentType: "text/html; charset=utf-8",
		Body:        buf.Bytes(),
	}, nil
}
