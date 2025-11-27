package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/z5labs/humus/rest"
)

// ErrorResponse represents a standard API error response.
// All API endpoints should use this format for consistency.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// NewErrorResponse creates a new ErrorResponse with the given code and message.
func NewErrorResponse(code, message string) *ErrorResponse {
	return &ErrorResponse{
		Error:   code,
		Message: message,
	}
}

// ErrorHandler implements rest.ErrorHandler to map errors to ErrorResponse format.
type ErrorHandler struct {
	log *slog.Logger
}

// NewErrorHandler creates a new ErrorHandler with the given logger.
func NewErrorHandler(log *slog.Logger) *ErrorHandler {
	return &ErrorHandler{log: log}
}

// OnError handles errors by mapping them to ErrorResponse and writing JSON.
func (h *ErrorHandler) OnError(ctx context.Context, w http.ResponseWriter, err error) {
	// Check if error implements HttpResponseWriter
	var httpWriter rest.HttpResponseWriter
	if errors.As(err, &httpWriter) {
		httpWriter.WriteHttpResponse(ctx, w)
		return
	}

	// Map known error types to ErrorResponse
	var (
		statusCode int
		errCode    string
		errMsg     string
	)

	var badReq rest.BadRequestError
	var unauthorized rest.UnauthorizedError
	var invalidParam rest.InvalidParameterValueError
	var missingParam rest.MissingRequiredParameterError

	switch {
	case errors.As(err, &badReq):
		statusCode = http.StatusBadRequest
		errCode = "bad_request"
		errMsg = err.Error()
	case errors.As(err, &unauthorized):
		statusCode = http.StatusUnauthorized
		errCode = "unauthorized"
		errMsg = err.Error()
	case errors.As(err, &invalidParam):
		statusCode = http.StatusBadRequest
		errCode = "invalid_parameter"
		errMsg = err.Error()
	case errors.As(err, &missingParam):
		statusCode = http.StatusBadRequest
		errCode = "missing_parameter"
		errMsg = err.Error()
	default:
		// Internal server error for unknown errors
		statusCode = http.StatusInternalServerError
		errCode = "internal_error"
		errMsg = "An unexpected error occurred"
		h.log.ErrorContext(ctx, "unhandled error", "error", err)
	}

	// Write ErrorResponse as JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := NewErrorResponse(errCode, errMsg)
	if encErr := json.NewEncoder(w).Encode(resp); encErr != nil {
		h.log.ErrorContext(ctx, "failed to encode error response", "error", encErr)
	}
}
