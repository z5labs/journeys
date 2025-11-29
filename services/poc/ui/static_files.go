package ui

import (
	"context"
	_ "embed"
	"net/http"
	"strings"

	"github.com/z5labs/humus/rest"
)

//go:embed static/js/journey-map.js
var journeyMapJS []byte

type staticFileHandler struct {
	content     []byte
	contentType string
}

func ServeStaticFiles() rest.ApiOption {
	jsHandler := &staticFileHandler{
		content:     journeyMapJS,
		contentType: "application/javascript; charset=utf-8",
	}

	return rest.Operation(
		http.MethodGet,
		rest.BasePath("/static/js/journey-map.js"),
		jsHandler,
	)
}

func (h *staticFileHandler) Handle(ctx context.Context, req *rest.EmptyRequest) (*HtmlResponse, error) {
	return &HtmlResponse{
		ContentType: h.contentType,
		Body:        h.content,
	}, nil
}

// GetContentType extracts the content type from filename
func getContentType(filename string) string {
	switch {
	case strings.HasSuffix(filename, ".js"):
		return "application/javascript; charset=utf-8"
	case strings.HasSuffix(filename, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(filename, ".html"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(filename, ".json"):
		return "application/json; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}
