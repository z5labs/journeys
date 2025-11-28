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
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/z5labs/journeys/services/poc/storage"

	"github.com/dgraph-io/dgo/v240"
	"github.com/dgraph-io/dgo/v240/protos/api"
	"github.com/google/uuid"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/z5labs/humus/rest"
)

//go:embed templates/content_item.html
var contentItemTemplate string

const (
	maxFormSize = 10 * 1024 * 1024 * 1024 // 10GB for multipart form parsing
)

var allowedMimeTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/heic":      true,
	"video/mp4":       true,
	"video/quicktime": true,
}

type contentNode struct {
	UID         string    `json:"uid,omitempty"`
	DType       []string  `json:"dgraph.type"`
	ContentID   string    `json:"content.id"`
	Type        string    `json:"content.type"`
	MinioKey    string    `json:"content.minio_key"`
	Title       string    `json:"content.title"`
	Description string    `json:"content.description"`
	UploadedAt  time.Time `json:"content.uploaded_at"`
	FileSize    int64     `json:"content.file_size"`
	MimeType    string    `json:"content.mime_type"`
}

type UploadContentRequest struct {
	JourneyID   string
	Files       []*multipart.FileHeader
	Type        string
	Title       string
	Description string
}

func (r *UploadContentRequest) ReadRequest(ctx context.Context, req *http.Request) error {
	if err := req.ParseMultipartForm(maxFormSize); err != nil {
		return fmt.Errorf("failed to parse multipart form: %w", err)
	}

	r.JourneyID = rest.PathParamValue(ctx, "journeyID")
	if r.JourneyID == "" {
		return fmt.Errorf("journey ID is required")
	}

	r.Type = strings.TrimSpace(req.FormValue("type"))
	r.Title = strings.TrimSpace(req.FormValue("title"))
	r.Description = strings.TrimSpace(req.FormValue("description"))

	if r.Type != "photo" && r.Type != "video" {
		return fmt.Errorf("invalid content type: must be 'photo' or 'video'")
	}

	files := req.MultipartForm.File["files"]
	if len(files) == 0 {
		return fmt.Errorf("at least one file is required")
	}

	r.Files = files
	return nil
}

func (r *UploadContentRequest) Spec() (openapi3.RequestBodyOrRef, error) {
	desc := "Multipart form data for uploading content"
	return openapi3.RequestBodyOrRef{
		RequestBody: &openapi3.RequestBody{
			Description: &desc,
			Content: map[string]openapi3.MediaType{
				"multipart/form-data": {},
			},
		},
	}, nil
}

type uploadContentHandler struct {
	dgraph   *dgo.Dgraph
	minio    *storage.MinioClient
	template *template.Template
}

func UploadContent(dgraph *dgo.Dgraph, minio *storage.MinioClient) rest.ApiOption {
	tmpl, err := template.New("content_item.html").Parse(contentItemTemplate)
	if err != nil {
		panic(err)
	}

	h := &uploadContentHandler{
		dgraph:   dgraph,
		minio:    minio,
		template: tmpl,
	}

	return rest.Operation(
		http.MethodPost,
		rest.BasePath("/app/journey").Param("journeyID").Segment("content"),
		h,
	)
}

func (h *uploadContentHandler) Handle(ctx context.Context, req *UploadContentRequest) (*HtmlResponse, error) {
	query := `query getJourney($journeyID: string) {
		journey(func: eq(journey.id, $journeyID)) {
			uid
		}
	}`

	vars := map[string]string{"$journeyID": req.JourneyID}
	resp, err := h.dgraph.NewReadOnlyTxn().QueryWithVars(ctx, query, vars)
	if err != nil {
		return nil, fmt.Errorf("failed to query journey: %w", err)
	}

	var result struct {
		Journey []struct {
			UID string `json:"uid"`
		} `json:"journey"`
	}
	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal journey: %w", err)
	}

	if len(result.Journey) == 0 {
		return nil, fmt.Errorf("journey not found")
	}

	journeyUID := result.Journey[0].UID

	var htmlFragments bytes.Buffer

	for _, fileHeader := range req.Files {
		if err := h.uploadSingleFile(ctx, req, fileHeader, journeyUID, &htmlFragments); err != nil {
			return nil, err
		}
	}

	htmlFragments.WriteString(`<div id="content-form-modal" hx-swap-oob="innerHTML"></div>`)

	return &HtmlResponse{
		ContentType: "text/html; charset=utf-8",
		Body:        htmlFragments.Bytes(),
	}, nil
}

func (h *uploadContentHandler) uploadSingleFile(ctx context.Context, req *UploadContentRequest, fileHeader *multipart.FileHeader, journeyUID string, htmlFragments *bytes.Buffer) error {
	contentType := fileHeader.Header.Get("Content-Type")
	if !allowedMimeTypes[contentType] {
		return fmt.Errorf("unsupported file type: %s", contentType)
	}

	file, err := fileHeader.Open()
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	contentID := uuid.NewString()
	ext := filepath.Ext(fileHeader.Filename)
	minioKey := fmt.Sprintf("%s/%s%s", req.JourneyID, contentID, ext)

	fileData, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	if err := h.minio.UploadFile(ctx, minioKey, bytes.NewReader(fileData), fileHeader.Size, contentType); err != nil {
		return fmt.Errorf("failed to upload to MinIO: %w", err)
	}

	contentTitle := req.Title
	if contentTitle == "" {
		contentTitle = fileHeader.Filename
	}

	node := &contentNode{
		UID:         "_:content",
		DType:       []string{"Content"},
		ContentID:   contentID,
		Type:        req.Type,
		MinioKey:    minioKey,
		Title:       contentTitle,
		Description: req.Description,
		UploadedAt:  time.Now().UTC(),
		FileSize:    fileHeader.Size,
		MimeType:    contentType,
	}

	jsonData, err := json.Marshal(node)
	if err != nil {
		return fmt.Errorf("failed to marshal content: %w", err)
	}

	txn := h.dgraph.NewTxn()
	defer txn.Discard(ctx)

	mu := &api.Mutation{
		SetJson: jsonData,
	}

	assigned, err := txn.Mutate(ctx, mu)
	if err != nil {
		return fmt.Errorf("failed to create content in Dgraph: %w", err)
	}

	contentUID := assigned.Uids["content"]

	edgeMutation := fmt.Sprintf(`<%s> <journey.content> <%s> .`, journeyUID, contentUID)
	mu = &api.Mutation{
		SetNquads: []byte(edgeMutation),
		CommitNow: true,
	}

	if _, err := txn.Mutate(ctx, mu); err != nil {
		return fmt.Errorf("failed to link content to journey: %w", err)
	}

	contentModel := Content{
		ID:          contentID,
		Type:        req.Type,
		MinioKey:    minioKey,
		Title:       contentTitle,
		Description: req.Description,
		UploadedAt:  time.Now().UTC(),
		FileSize:    fileHeader.Size,
		MimeType:    contentType,
	}

	if err := h.template.Execute(htmlFragments, contentModel); err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	return nil
}
