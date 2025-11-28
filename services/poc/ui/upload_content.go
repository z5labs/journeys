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
	"log/slog"
	"math"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/z5labs/journeys/services/poc/storage"

	"github.com/dgraph-io/dgo/v240"
	"github.com/dgraph-io/dgo/v240/protos/api"
	"github.com/google/uuid"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/z5labs/humus"
	"github.com/z5labs/humus/rest"
)

//go:embed templates/content_item.html
var contentItemTemplate string

const (
	maxImageBufferSize = 100 * 1024 * 1024 // 100MB max for image buffering
)

var allowedMimeTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/heic":      true,
	"video/mp4":       true,
	"video/quicktime": true,
}

// determineContentType categorizes content based on MIME type
func determineContentType(mimeType string) string {
	if strings.HasPrefix(mimeType, "image/") {
		return "photo"
	}
	if strings.HasPrefix(mimeType, "video/") {
		return "video"
	}
	return "unknown"
}

type GeoMetadata struct {
	Latitude   *float64
	Longitude  *float64
	Altitude   *float64
	CapturedAt *time.Time
}

func extractGeoMetadata(fileData []byte, mimeType string) (*GeoMetadata, error) {
	log := humus.Logger("upload")
	meta := &GeoMetadata{}

	if !strings.HasPrefix(mimeType, "image/") {
		log.Info("skipping geo extraction for non-image", slog.String("mime_type", mimeType))
		return meta, nil
	}

	reader := bytes.NewReader(fileData)
	exifData, err := exif.Decode(reader)
	if err != nil {
		log.Info("no exif data found or decode error", slog.String("error", err.Error()))
		return meta, nil
	}

	lat, lon, err := exifData.LatLong()
	if err == nil && !math.IsNaN(lat) && !math.IsNaN(lon) {
		meta.Latitude = &lat
		meta.Longitude = &lon
		log.Info("extracted GPS coordinates", slog.Float64("lat", lat), slog.Float64("lon", lon))
	} else if err != nil {
		log.Info("no GPS coordinates in exif", slog.String("error", err.Error()))
	} else {
		log.Info("invalid GPS coordinates (NaN) in exif")
	}

	if altTag, err := exifData.Get(exif.GPSAltitude); err == nil {
		if alt, err := altTag.Float(0); err == nil && !math.IsNaN(alt) {
			meta.Altitude = &alt
			log.Info("extracted altitude", slog.Float64("altitude", alt))
		}
	}

	if dtTag, err := exifData.DateTime(); err == nil {
		meta.CapturedAt = &dtTag
		log.Info("extracted capture datetime", slog.Time("captured_at", dtTag))
	}

	return meta, nil
}

// streamingFileUpload encapsulates streaming upload logic
type streamingFileUpload struct {
	part        *multipart.Part
	contentType string
	filename    string
	isImage     bool
	buffer      *bytes.Buffer // Only for images
}

func newStreamingFileUpload(part *multipart.Part) (*streamingFileUpload, error) {
	contentType := part.Header.Get("Content-Type")

	// Validate MIME type before upload
	if !allowedMimeTypes[contentType] {
		return nil, fmt.Errorf("unsupported file type: %s", contentType)
	}

	isImage := strings.HasPrefix(contentType, "image/")

	upload := &streamingFileUpload{
		part:        part,
		contentType: contentType,
		filename:    part.FileName(),
		isImage:     isImage,
	}

	if isImage {
		upload.buffer = &bytes.Buffer{}
	}

	return upload, nil
}

func (s *streamingFileUpload) getReader() io.Reader {
	if s.isImage {
		// TeeReader streams to MinIO while capturing for EXIF
		return io.TeeReader(s.part, s.buffer)
	}
	// Videos: direct streaming, no buffering
	return s.part
}

func (s *streamingFileUpload) extractMetadata() (*GeoMetadata, error) {
	if !s.isImage || s.buffer.Len() == 0 {
		return &GeoMetadata{}, nil
	}
	return extractGeoMetadata(s.buffer.Bytes(), s.contentType)
}

type contentNode struct {
	UID          string     `json:"uid,omitempty"`
	DType        []string   `json:"dgraph.type"`
	ContentID    string     `json:"content.id"`
	Type         string     `json:"content.type"`
	MinioKey     string     `json:"content.minio_key"`
	Title        string     `json:"content.title"`
	Description  string     `json:"content.description"`
	UploadedAt   time.Time  `json:"content.uploaded_at"`
	FileSize     int64      `json:"content.file_size"`
	MimeType     string     `json:"content.mime_type"`
	Latitude     *float64   `json:"content.latitude,omitempty"`
	Longitude    *float64   `json:"content.longitude,omitempty"`
	Altitude     *float64   `json:"content.altitude,omitempty"`
	LocationName string     `json:"content.location_name,omitempty"`
	CapturedAt   *time.Time `json:"content.captured_at,omitempty"`
}

type UploadContentRequest struct {
	JourneyID   string
	rawRequest  *http.Request // Store for streaming access
	Title       string
	Description string
}

func (r *UploadContentRequest) ReadRequest(ctx context.Context, req *http.Request) error {
	r.JourneyID = rest.PathParamValue(ctx, "journeyID")
	if r.JourneyID == "" {
		return fmt.Errorf("journey ID is required")
	}

	r.rawRequest = req
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
	tmpl, err := template.New("content_item.html").Funcs(template.FuncMap{
		"deref": func(ptr interface{}) interface{} {
			if ptr == nil {
				return nil
			}
			switch v := ptr.(type) {
			case *float64:
				if v == nil {
					return nil
				}
				return *v
			case *int:
				if v == nil {
					return nil
				}
				return *v
			}
			return ptr
		},
	}).Parse(contentItemTemplate)
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
	// Extract multipart boundary from Content-Type header
	contentType := req.rawRequest.Header.Get("Content-Type")
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Content-Type: %w", err)
	}

	boundary := params["boundary"]
	if boundary == "" {
		return nil, fmt.Errorf("multipart boundary not found")
	}

	// Get journey UID first (before processing multipart stream)
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

	// Create streaming multipart reader
	mr := multipart.NewReader(req.rawRequest.Body, boundary)
	defer req.rawRequest.Body.Close()

	formFields := make(map[string]string)
	var htmlFragments bytes.Buffer
	fileCount := 0

	// Process parts as they arrive - upload files immediately
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read multipart part: %w", err)
		}

		formName := part.FormName()

		// Handle text form fields
		if part.FileName() == "" {
			value, err := io.ReadAll(part)
			if err != nil {
				part.Close()
				return nil, fmt.Errorf("failed to read form field: %w", err)
			}
			formFields[formName] = string(value)
			part.Close()
			continue
		}

		// Handle file uploads - upload immediately while reader is valid
		if formName == "files" {
			fileCount++
			upload, err := newStreamingFileUpload(part)
			if err != nil {
				part.Close()
				return nil, err
			}

			// Extract form fields for this upload
			req.Title = strings.TrimSpace(formFields["title"])
			req.Description = strings.TrimSpace(formFields["description"])

			// Upload immediately while part reader is valid
			if err := h.uploadSingleFileStreaming(ctx, req, upload, journeyUID, &htmlFragments); err != nil {
				part.Close()
				return nil, err
			}
			part.Close()
		} else {
			part.Close()
		}
	}

	// Validate at least one file
	if fileCount == 0 {
		return nil, fmt.Errorf("at least one file is required")
	}

	htmlFragments.WriteString(`<div id="content-form-modal" hx-swap-oob="innerHTML"></div>`)

	return &HtmlResponse{
		ContentType: "text/html; charset=utf-8",
		Body:        htmlFragments.Bytes(),
	}, nil
}

func (h *uploadContentHandler) uploadSingleFileStreaming(
	ctx context.Context,
	req *UploadContentRequest,
	upload *streamingFileUpload,
	journeyUID string,
	htmlFragments *bytes.Buffer,
) error {
	// Generate MinIO key
	contentID := uuid.NewString()
	ext := filepath.Ext(upload.filename)
	minioKey := fmt.Sprintf("%s/%s%s", req.JourneyID, contentID, ext)

	// Stream to MinIO with unknown size (-1)
	reader := upload.getReader()
	err := h.minio.UploadFile(ctx, minioKey, reader, -1, upload.contentType)
	if err != nil {
		return fmt.Errorf("failed to upload to MinIO: %w", err)
	}

	// For images, validate buffer size didn't exceed limit
	if upload.isImage && upload.buffer.Len() >= maxImageBufferSize {
		// Image exceeded buffer limit - clean up and reject
		_ = h.minio.DeleteFile(ctx, minioKey)
		return fmt.Errorf("image file exceeds maximum size of %d bytes", maxImageBufferSize)
	}

	// Get actual file size from MinIO
	fileInfo, err := h.minio.GetFileInfo(ctx, minioKey)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Extract EXIF metadata (only buffered for images)
	geoMeta, _ := upload.extractMetadata()

	log := humus.Logger("upload")
	if geoMeta.Latitude != nil && geoMeta.Longitude != nil {
		log.Info("storing content with geolocation",
			slog.Float64("lat", *geoMeta.Latitude),
			slog.Float64("lon", *geoMeta.Longitude))
	} else {
		log.Info("storing content without geolocation")
	}

	contentTitle := req.Title
	if contentTitle == "" {
		contentTitle = upload.filename
	}

	detectedType := determineContentType(upload.contentType)

	// Create Dgraph node
	node := &contentNode{
		UID:         "_:content",
		DType:       []string{"Content"},
		ContentID:   contentID,
		Type:        detectedType,
		MinioKey:    minioKey,
		Title:       contentTitle,
		Description: req.Description,
		UploadedAt:  time.Now().UTC(),
		FileSize:    fileInfo.Size, // Use actual size from MinIO
		MimeType:    upload.contentType,
		Latitude:    geoMeta.Latitude,
		Longitude:   geoMeta.Longitude,
		Altitude:    geoMeta.Altitude,
		CapturedAt:  geoMeta.CapturedAt,
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

	// Render HTML fragment
	contentModel := Content{
		ID:           contentID,
		Type:         detectedType,
		MinioKey:     minioKey,
		Title:        contentTitle,
		Description:  req.Description,
		UploadedAt:   time.Now().UTC(),
		FileSize:     fileInfo.Size,
		MimeType:     upload.contentType,
		Latitude:     geoMeta.Latitude,
		Longitude:    geoMeta.Longitude,
		Altitude:     geoMeta.Altitude,
		LocationName: "",
		CapturedAt:   geoMeta.CapturedAt,
	}

	if err := h.template.Execute(htmlFragments, contentModel); err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	return nil
}
