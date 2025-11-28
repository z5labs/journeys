// Copyright (c) 2025 Z5labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/z5labs/journeys/services/poc/storage"

	"github.com/dgraph-io/dgo/v240"
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

func (h *contentPreviewHandler) Handle(ctx context.Context, req *rest.EmptyRequest) (*HtmlResponse, error) {
	contentID := rest.PathParamValue(ctx, "contentID")
	if contentID == "" {
		return nil, fmt.Errorf("content ID is required")
	}

	query := `query getContent($contentID: string) {
		content(func: eq(content.id, $contentID)) {
			content.minio_key
		}
	}`

	vars := map[string]string{"contentID": contentID}
	resp, err := h.dgraph.NewReadOnlyTxn().QueryWithVars(ctx, query, vars)
	if err != nil {
		return nil, fmt.Errorf("failed to query content: %w", err)
	}

	var result struct {
		Content []struct {
			MinioKey string `json:"content.minio_key"`
		} `json:"content"`
	}
	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal content: %w", err)
	}

	if len(result.Content) == 0 {
		return nil, fmt.Errorf("content not found")
	}

	minioKey := result.Content[0].MinioKey

	presignedURL, err := h.minio.GetFileURL(ctx, minioKey, 15*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed to get presigned URL: %w", err)
	}

	redirectHTML := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta http-equiv="refresh" content="0; url=%s">
</head>
<body>
    <p>Redirecting...</p>
</body>
</html>`, presignedURL)

	return &HtmlResponse{
		ContentType: "text/html; charset=utf-8",
		Body:        []byte(redirectHTML),
	}, nil
}
