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

	"github.com/abema/go-mp4"
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
	maxVideoProbeSize  = 8 * 1024 * 1024   // 8MB max for probing video metadata
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

// parseMP4CreationTime extracts the creation time from MP4/QuickTime mvhd box
// using the abema/go-mp4 library.
func parseMP4CreationTime(data []byte) (time.Time, error) {
	log := humus.Logger("mp4-parser")

	var foundTime time.Time
	foundMvhd := false

	r := bytes.NewReader(data)
	_, err := mp4.ReadBoxStructure(r, func(h *mp4.ReadHandle) (interface{}, error) {
		if h.BoxInfo.Type == mp4.BoxTypeMvhd() {
			box, _, err := h.ReadPayload()
			if err != nil {
				return nil, err
			}

			mvhd, ok := box.(*mp4.Mvhd)
			if !ok {
				return nil, fmt.Errorf("expected *mp4.Mvhd, got %T", box)
			}

			// mvhd stores creation_time as seconds since midnight, Jan 1, 1904 UTC
			var creationTime uint64
			if mvhd.GetVersion() == 0 {
				creationTime = uint64(mvhd.CreationTimeV0)
			} else {
				creationTime = mvhd.CreationTimeV1
			}

			foundTime = time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(creationTime) * time.Second)
			foundMvhd = true
			log.Info("extracted creation time from mvhd",
				slog.Int("version", int(mvhd.GetVersion())),
				slog.Time("creation_time", foundTime))

			return nil, nil
		}

		// Expand container boxes to traverse into them
		if h.BoxInfo.Type == mp4.BoxTypeMoov() {
			return h.Expand()
		}

		return nil, nil
	})

	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse MP4: %w", err)
	}

	if foundMvhd {
		return foundTime, nil
	}

	return time.Time{}, fmt.Errorf("mvhd not found")
}

// cappedWriter writes up to a limit into an internal buffer and discards the rest.
// This allows probing the first N bytes of large uploads without buffering entire file.
type cappedWriter struct {
	buf   *bytes.Buffer
	limit int
}

func (c *cappedWriter) Write(p []byte) (int, error) {
	remaining := c.limit - c.buf.Len()
	if remaining <= 0 {
		// pretend we wrote everything to avoid tee reader blocking
		return len(p), nil
	}
	toWrite := len(p)
	if toWrite > remaining {
		toWrite = remaining
	}
	c.buf.Write(p[:toWrite])
	// report full length consumed
	return len(p), nil
}

// streamingFileUpload encapsulates streaming upload logic with concurrent metadata extraction
type streamingFileUpload struct {
	part        io.Reader
	contentType string
	filename    string
	isImage     bool
	isVideo     bool
	buffer      *bytes.Buffer // buffer for probing metadata (images only)
	probeWriter io.Writer     // writer used by TeeReader to capture probe bytes

	// Concurrent metadata extraction (videos only)
	pipeReader     *io.PipeReader
	pipeWriter     *io.PipeWriter
	metadataResult chan *MetadataResult
	scanConfig     MetadataScanConfig
}

type partReader interface {
	io.Reader
	FileName() string
	GetHeader(key string) string
}

type multipartPartAdapter struct {
	*multipart.Part
}

func (a *multipartPartAdapter) GetHeader(key string) string {
	return a.Part.Header.Get(key)
}

func newStreamingFileUpload(part partReader) (*streamingFileUpload, error) {
	contentType := part.GetHeader("Content-Type")

	// Validate MIME type before upload
	if !allowedMimeTypes[contentType] {
		return nil, fmt.Errorf("unsupported file type: %s", contentType)
	}

	isImage := strings.HasPrefix(contentType, "image/")
	isVideo := strings.HasPrefix(contentType, "video/")

	upload := &streamingFileUpload{
		part:        part,
		contentType: contentType,
		filename:    part.FileName(),
		isImage:     isImage,
		isVideo:     isVideo,
		scanConfig:  defaultScanConfig,
	}

	// Images: buffer fully for EXIF extraction
	if isImage {
		upload.buffer = &bytes.Buffer{}
		upload.probeWriter = upload.buffer
	} else if isVideo && upload.scanConfig.UseStreamingScanner {
		// Videos: use concurrent streaming scanner with io.Pipe
		pr, pw := io.Pipe()
		upload.pipeReader = pr
		upload.pipeWriter = pw
		upload.metadataResult = make(chan *MetadataResult, 1)
	} else if isVideo {
		// Fallback: legacy capped buffer approach
		upload.buffer = &bytes.Buffer{}
		upload.probeWriter = &cappedWriter{buf: upload.buffer, limit: maxVideoProbeSize}
	}

	return upload, nil
}

func (s *streamingFileUpload) getReader() io.Reader {
	// Videos with streaming scanner: use TeeReader to pipe
	if s.isVideo && s.pipeWriter != nil {
		return io.TeeReader(s.part, s.pipeWriter)
	}

	// Legacy buffering approach
	if s.probeWriter != nil {
		return io.TeeReader(s.part, s.probeWriter)
	}

	return s.part
}

// startMetadataExtraction launches concurrent metadata extraction goroutine.
// Must be called before getReader() is consumed.
func (s *streamingFileUpload) startMetadataExtraction(ctx context.Context) {
	if !s.isVideo || s.pipeReader == nil {
		return
	}

	log := humus.Logger("upload")
	log.Debug("starting concurrent metadata extraction", slog.String("filename", s.filename))

	go func() {
		defer s.pipeReader.Close()

		result, err := extractMP4MetadataStreaming(ctx, s.pipeReader, s.scanConfig)
		if err != nil {
			log.Info("metadata extraction completed with error",
				slog.String("filename", s.filename),
				slog.String("error", err.Error()))
		} else {
			log.Info("metadata extraction completed successfully",
				slog.String("filename", s.filename),
				slog.Int64("bytes_scanned", result.BytesScanned))
		}

		s.metadataResult <- result
		close(s.metadataResult)
	}()
}

// closeWriter closes the pipe writer after upload completes.
// This signals EOF to the metadata extraction goroutine.
func (s *streamingFileUpload) closeWriter() error {
	if s.pipeWriter != nil {
		return s.pipeWriter.Close()
	}
	return nil
}

func (s *streamingFileUpload) extractMetadata() (*GeoMetadata, error) {
	meta := &GeoMetadata{}

	// Images: extract from buffer
	if s.isImage && s.buffer != nil && s.buffer.Len() > 0 {
		imgMeta, _ := extractGeoMetadata(s.buffer.Bytes(), s.contentType)
		meta.Latitude = imgMeta.Latitude
		meta.Longitude = imgMeta.Longitude
		meta.Altitude = imgMeta.Altitude
		meta.CapturedAt = imgMeta.CapturedAt
		return meta, nil
	}

	// Videos: wait for concurrent metadata extraction result
	if s.isVideo && s.metadataResult != nil {
		result := <-s.metadataResult
		if result != nil && result.FoundMetadata {
			meta.CapturedAt = result.CapturedAt
			return meta, nil
		}
		return meta, nil
	}

	// Fallback: legacy buffer-based extraction
	if s.isVideo && s.buffer != nil && s.buffer.Len() > 0 {
		if t, err := parseMP4CreationTime(s.buffer.Bytes()); err == nil {
			meta.CapturedAt = &t
			return meta, nil
		}
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
			upload, err := newStreamingFileUpload(&multipartPartAdapter{Part: part})
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
	log := humus.Logger("upload")

	// Generate MinIO key
	contentID := uuid.NewString()
	ext := filepath.Ext(upload.filename)
	minioKey := fmt.Sprintf("%s/%s%s", req.JourneyID, contentID, ext)

	// Start concurrent metadata extraction before consuming reader
	upload.startMetadataExtraction(ctx)

	// Stream to MinIO with unknown size (-1)
	reader := upload.getReader()
	err := h.minio.UploadFile(ctx, minioKey, reader, -1, upload.contentType)

	// Close pipe writer to signal EOF to metadata goroutine
	if closeErr := upload.closeWriter(); closeErr != nil {
		log.Warn("failed to close pipe writer", slog.String("error", closeErr.Error()))
	}

	if err != nil {
		return fmt.Errorf("failed to upload to MinIO: %w", err)
	}

	// For images, validate buffer size didn't exceed limit
	if upload.isImage && upload.buffer != nil && upload.buffer.Len() >= maxImageBufferSize {
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
