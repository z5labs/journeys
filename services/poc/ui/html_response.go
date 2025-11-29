// Copyright (c) 2025 Z5labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package ui

import (
	"context"
	"net/http"

	"github.com/swaggest/openapi-go/openapi3"
)

// HtmlResponse is a custom response type that writes HTML content directly
// to the HTTP response writer. It implements the rest.TypedResponse[HtmlResponse]
// interface, allowing it to be used as a return type from humus handler functions.
type HtmlResponse struct {
	ContentType string
	Body        []byte
}

// WriteResponse writes the HTML response to the provided http.ResponseWriter.
// It sets the Content-Type header and writes the body content.
func (r *HtmlResponse) WriteResponse(ctx context.Context, w http.ResponseWriter) error {
	w.Header().Set("Content-Type", r.ContentType)
	_, err := w.Write(r.Body)
	return err
}

// Spec returns the OpenAPI 3.0 specification for this HTML response.
// It describes the response as returning text/html content with a 200 status code.
func (r *HtmlResponse) Spec() (int, openapi3.ResponseOrRef, error) {
	schema := openapi3.Schema{}
	schema.WithType("string").WithDescription("HTML content")

	resp := openapi3.Response{
		Description: "HTML page response",
		Content: map[string]openapi3.MediaType{
			"text/html": {
				Schema: &openapi3.SchemaOrRef{
					Schema: &schema,
				},
			},
		},
	}

	return http.StatusOK, openapi3.ResponseOrRef{Response: &resp}, nil
}
