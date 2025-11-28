// Copyright (c) 2025 Z5labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package ui

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"net/http"

	"github.com/dgraph-io/dgo/v240"
	"github.com/z5labs/humus/rest"
)

//go:embed templates/content_upload_form.html
var contentUploadFormTemplate string

type contentUploadFormHandler struct {
	dgraph   *dgo.Dgraph
	template *template.Template
}

func GetContentUploadForm(dgraph *dgo.Dgraph) rest.ApiOption {
	tmpl, err := template.New("content_upload_form.html").Parse(contentUploadFormTemplate)
	if err != nil {
		panic(err)
	}

	h := &contentUploadFormHandler{
		dgraph:   dgraph,
		template: tmpl,
	}

	return rest.Operation(
		http.MethodGet,
		rest.BasePath("/app/journey").Param("journeyID").Segment("content").Segment("form"),
		h,
	)
}

func (h *contentUploadFormHandler) Handle(ctx context.Context, req *rest.EmptyRequest) (*HtmlResponse, error) {
	journeyID := rest.PathParamValue(ctx, "journeyID")
	if journeyID == "" {
		return nil, fmt.Errorf("journey ID is required")
	}

	var buf bytes.Buffer
	err := h.template.Execute(&buf, map[string]string{
		"JourneyID": journeyID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	return &HtmlResponse{
		ContentType: "text/html; charset=utf-8",
		Body:        buf.Bytes(),
	}, nil
}
