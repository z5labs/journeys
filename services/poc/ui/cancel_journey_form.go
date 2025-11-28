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

//go:embed templates/create_journey_button.html
var createJourneyButtonTemplateForHandler string

type cancelJourneyFormHandler struct {
	template *template.Template
}

func CancelJourneyForm(dgraph *dgo.Dgraph) rest.ApiOption {
	tmpl, err := template.New("create_journey_button.html").Parse(createJourneyButtonTemplateForHandler)
	if err != nil {
		panic(err) // Configuration error - fail fast
	}

	h := &cancelJourneyFormHandler{
		template: tmpl,
	}

	return rest.Operation(
		http.MethodGet,
		rest.BasePath("/app/journey/cancel"),
		h,
	)
}

func (h *cancelJourneyFormHandler) Handle(ctx context.Context, req *rest.EmptyRequest) (*HtmlResponse, error) {
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
