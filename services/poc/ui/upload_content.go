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
	maxFormSize = 10 * 1024 * 1024 * 1024 // 10GB for multipart form parsing
)

var allowedMimeTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/heic":      true,
	"video/mp4":       true,
	"video/quicktime": true,
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

	geoMeta, _ := extractGeoMetadata(fileData, contentType)

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
		contentTitle = fileHeader.Filename
	}

	node := &contentNode{
		UID:          "_:content",
		DType:        []string{"Content"},
		ContentID:    contentID,
		Type:         req.Type,
		MinioKey:     minioKey,
		Title:        contentTitle,
		Description:  req.Description,
		UploadedAt:   time.Now().UTC(),
		FileSize:     fileHeader.Size,
		MimeType:     contentType,
		Latitude:     geoMeta.Latitude,
		Longitude:    geoMeta.Longitude,
		Altitude:     geoMeta.Altitude,
		CapturedAt:   geoMeta.CapturedAt,
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
		ID:           contentID,
		Type:         req.Type,
		MinioKey:     minioKey,
		Title:        contentTitle,
		Description:  req.Description,
		UploadedAt:   time.Now().UTC(),
		FileSize:     fileHeader.Size,
		MimeType:     contentType,
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
