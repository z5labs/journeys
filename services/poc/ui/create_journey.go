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
	"strings"
	"time"

	"github.com/dgraph-io/dgo/v240"
	"github.com/dgraph-io/dgo/v240/protos/api"
	"github.com/google/uuid"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/z5labs/humus/rest"
)

//go:embed templates/journey_item.html
var journeyItemTemplateForCreate string

// journeyNode represents the Dgraph node structure for a journey
type journeyNode struct {
	UID       string    `json:"uid,omitempty"`
	DType     []string  `json:"dgraph.type"`
	JourneyID string    `json:"journey.id"`
	Title     string    `json:"journey.title"`
	CreatedAt time.Time `json:"journey.created_at"`
}

// FormRequest represents the form data for creating a journey
type FormRequest struct {
	Title string
}

// ReadRequest parses the form-encoded request body
func (r *FormRequest) ReadRequest(ctx context.Context, req *http.Request) error {
	if err := req.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	r.Title = strings.TrimSpace(req.FormValue("title"))

	// Validation
	if r.Title == "" {
		return fmt.Errorf("title is required")
	}
	if len(r.Title) > 100 {
		return fmt.Errorf("title must be less than 100 characters")
	}

	return nil
}

// Spec returns the OpenAPI specification for the form request
func (r *FormRequest) Spec() (openapi3.RequestBodyOrRef, error) {
	typeObject := openapi3.SchemaTypeObject
	typeString := openapi3.SchemaTypeString
	desc := "Form data for creating a journey"
	titleDesc := "Journey title"

	titleSchema := openapi3.Schema{}
	titleSchema.Type = &typeString
	titleSchema.Description = &titleDesc

	schema := openapi3.Schema{}
	schema.Type = &typeObject
	schema.Properties = map[string]openapi3.SchemaOrRef{
		"title": {Schema: &titleSchema},
	}

	return openapi3.RequestBodyOrRef{
		RequestBody: &openapi3.RequestBody{
			Description: &desc,
			Content: map[string]openapi3.MediaType{
				"application/x-www-form-urlencoded": {
					Schema: &openapi3.SchemaOrRef{
						Schema: &schema,
					},
				},
			},
		},
	}, nil
}

type createJourneyHandler struct {
	dgraph   *dgo.Dgraph
	template *template.Template
}

func CreateJourney(dgraph *dgo.Dgraph) rest.ApiOption {
	tmpl, err := template.New("journey_item.html").Parse(journeyItemTemplateForCreate)
	if err != nil {
		panic(err) // Configuration error - fail fast
	}

	h := &createJourneyHandler{
		dgraph:   dgraph,
		template: tmpl,
	}

	return rest.Operation(
		http.MethodPost,
		rest.BasePath("/app/journey"),
		h,
	)
}

// Handle processes the create journey request
func (h *createJourneyHandler) Handle(ctx context.Context, req *FormRequest) (*HtmlResponse, error) {
	// Generate UUID for journey
	journeyID := uuid.NewString()

	// Create journey node for Dgraph
	node := &journeyNode{
		UID:       "_:journey",
		DType:     []string{"Journey"},
		JourneyID: journeyID,
		Title:     req.Title,
		CreatedAt: time.Now().UTC(),
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(node)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal journey: %w", err)
	}

	// Create transaction
	txn := h.dgraph.NewTxn()
	defer txn.Discard(ctx)

	// Execute mutation
	mu := &api.Mutation{
		SetJson:   jsonData,
		CommitNow: true,
	}

	_, err = txn.Mutate(ctx, mu)
	if err != nil {
		return nil, fmt.Errorf("failed to create journey in Dgraph: %w", err)
	}

	// Create journey model for template
	journeyModel := journey{
		ID:    journeyID,
		Title: req.Title,
	}

	// Render template
	var buf bytes.Buffer
	err = h.template.Execute(&buf, journeyModel)
	if err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	// Return HTML response
	return &HtmlResponse{
		ContentType: "text/html; charset=utf-8",
		Body:        buf.Bytes(),
	}, nil
}
