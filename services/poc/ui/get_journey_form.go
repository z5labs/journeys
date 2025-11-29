// Copyright (c) 2025 Z5labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package ui

import (
	"bytes"
	"context"
	_ "embed"
	"html/template"
	"net/http"

	"github.com/dgraph-io/dgo/v240"
	"github.com/z5labs/humus/rest"
)

//go:embed templates/create_journey_form.html
var createJourneyFormTemplateForHandler string

type getJourneyFormHandler struct {
	template *template.Template
}

func GetJourneyForm(dgraph *dgo.Dgraph) rest.ApiOption {
	tmpl, err := template.New("create_journey_form.html").Parse(createJourneyFormTemplateForHandler)
	if err != nil {
		panic(err) // Configuration error - fail fast
	}

	h := &getJourneyFormHandler{
		template: tmpl,
	}

	return rest.Operation(
		http.MethodGet,
		rest.BasePath("/app/journey/form"),
		h,
	)
}

func (h *getJourneyFormHandler) Handle(ctx context.Context, req *rest.EmptyRequest) (*HtmlResponse, error) {
	// Execute template
	var buf bytes.Buffer
	err := h.template.Execute(&buf, nil)
	if err != nil {
		return nil, err
	}

	// Return HTML response
	return &HtmlResponse{
		ContentType: "text/html; charset=utf-8",
		Body:        buf.Bytes(),
	}, nil
}
