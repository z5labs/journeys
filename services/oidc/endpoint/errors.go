// Copyright (c) 2025 Z5labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package endpoint

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidProvider   = errors.New("invalid provider")
	ErrMissingParameter  = errors.New("missing required parameter")
	ErrInvalidRedirectURI = errors.New("invalid redirect_uri")
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func NewInvalidProviderError(provider string) error {
	return fmt.Errorf("%w: provider '%s' is not supported. Valid providers: google, facebook, apple", ErrInvalidProvider, provider)
}

func NewMissingParameterError(param string) error {
	return fmt.Errorf("%w: required query parameter '%s' is missing", ErrMissingParameter, param)
}

func NewInvalidRedirectURIError(uri string) error {
	return fmt.Errorf("%w: '%s' is not a valid URL", ErrInvalidRedirectURI, uri)
}
