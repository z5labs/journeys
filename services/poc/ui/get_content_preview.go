// Copyright (c) 2025 Z5labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/z5labs/journeys/services/poc/storage"

	"github.com/dgraph-io/dgo/v240"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/z5labs/humus/rest"
)

type contentPreviewHandler struct {
	dgraph *dgo.Dgraph
	minio  *storage.MinioClient
}

func GetContentPreview(dgraph *dgo.Dgraph, minio *storage.MinioClient) rest.ApiOption {
	h := &contentPreviewHandler{
		dgraph: dgraph,
		minio:  minio,
	}

	return rest.Operation(
		http.MethodGet,
		rest.BasePath("/app/content").Param("contentID").Segment("preview"),
		h,
	)
}

func (h *contentPreviewHandler) Handle(ctx context.Context, req *rest.EmptyRequest) (*streamResponse, error) {
	contentID := rest.PathParamValue(ctx, "contentID")
	if contentID == "" {
		return nil, fmt.Errorf("content ID is required")
	}

	query := `query getContent($contentID: string) {
		content(func: eq(content.id, $contentID)) {
			content.minio_key
			content.mime_type
		}
	}`

	vars := map[string]string{"$contentID": contentID}
	resp, err := h.dgraph.NewReadOnlyTxn().QueryWithVars(ctx, query, vars)
	if err != nil {
		return nil, fmt.Errorf("failed to query content: %w", err)
	}

	var result struct {
		Content []struct {
			MinioKey string `json:"content.minio_key"`
			MimeType string `json:"content.mime_type"`
		} `json:"content"`
	}
	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal content: %w", err)
	}

	if len(result.Content) == 0 {
		return nil, fmt.Errorf("content not found")
	}

	minioKey := result.Content[0].MinioKey
	mimeType := result.Content[0].MimeType

	object, err := h.minio.GetFile(ctx, minioKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get file from MinIO: %w", err)
	}

	info, err := object.Stat()
	if err != nil {
		object.Close()
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return &streamResponse{
		contentType:   mimeType,
		contentLength: info.Size,
		reader:        object,
	}, nil
}

type streamResponse struct {
	contentType   string
	contentLength int64
	reader        io.ReadCloser
}

func (r *streamResponse) WriteResponse(ctx context.Context, w http.ResponseWriter) error {
	defer r.reader.Close()
	w.Header().Set("Content-Type", r.contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", r.contentLength))
	_, err := io.Copy(w, r.reader)
	return err
}

func (r *streamResponse) Spec() (int, openapi3.ResponseOrRef, error) {
	schema := openapi3.Schema{}
	schema.WithType("string").WithFormat("binary").WithDescription("Binary content stream")

	resp := openapi3.Response{
		Description: "Content file stream",
		Content: map[string]openapi3.MediaType{
			"image/*": {
				Schema: &openapi3.SchemaOrRef{
					Schema: &schema,
				},
			},
			"video/*": {
				Schema: &openapi3.SchemaOrRef{
					Schema: &schema,
				},
			},
		},
	}

	return http.StatusOK, openapi3.ResponseOrRef{Response: &resp}, nil
}
