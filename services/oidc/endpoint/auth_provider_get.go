// Copyright (c) 2025 Z5labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package endpoint

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/swaggest/jsonschema-go"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/z5labs/humus"
	"github.com/z5labs/humus/rest"
)

const (
	ProviderGoogle   = "google"
	ProviderFacebook = "facebook"
	ProviderApple    = "apple"
)

type ProviderConfig struct {
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	Scopes       []string
}

type OAuthConfig struct {
	Google   ProviderConfig
	Facebook   ProviderConfig
	Apple    ProviderConfig
}

type AuthProviderResponse struct {
	Provider         string   `json:"provider"`
	AuthorizationURL string   `json:"authorizationUrl"`
	ClientID         string   `json:"clientId"`
	Scopes           []string `json:"scopes"`
	ResponseType     string   `json:"responseType"`
	State            string   `json:"state"`
}

type authProviderHandler struct {
	log   *slog.Logger
	oauth OAuthConfig
}

func RegisterAuthProviderGetEndpoint(oauth OAuthConfig) rest.ApiOption {
	h := &authProviderHandler{
		log:   humus.Logger("auth.provider.get"),
		oauth: oauth,
	}
	
	return rest.Handle(
		http.MethodGet,
		rest.BasePath("/v1/auth").Param("provider"),
		h,
		rest.QueryParam("redirect_uri", rest.Required()),
		rest.QueryParam("state"),
	)
}

func (h *authProviderHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	provider := r.PathValue("provider")
	redirectURI := r.URL.Query().Get("redirect_uri")
	state := r.URL.Query().Get("state")
	
	h.log.InfoContext(ctx, "handling auth provider request", "provider", provider)
	
	resp, err := h.handle(ctx, provider, redirectURI, state)
	if err != nil {
		h.log.ErrorContext(ctx, "failed to handle request", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.log.ErrorContext(ctx, "failed to encode response", "error", err)
	}
}

func (h *authProviderHandler) RequestBody() openapi3.RequestBodyOrRef {
	return openapi3.RequestBodyOrRef{}
}

func (h *authProviderHandler) Responses() openapi3.Responses {
	var resp AuthProviderResponse
	var reflector jsonschema.Reflector

	jsonSchema, err := reflector.Reflect(resp, jsonschema.InlineRefs)
	if err != nil {
		return openapi3.Responses{}
	}

	var schemaOrRef openapi3.SchemaOrRef
	schemaOrRef.FromJSONSchema(jsonSchema.ToSchemaOrBool())

	return openapi3.Responses{
		MapOfResponseOrRefValues: map[string]openapi3.ResponseOrRef{
			"200": {
				Response: &openapi3.Response{
					Content: map[string]openapi3.MediaType{
						"application/json": {
							Schema: &schemaOrRef,
						},
					},
				},
			},
		},
	}
}

func (h *authProviderHandler) handle(ctx context.Context, provider, redirectURI, state string) (*AuthProviderResponse, error) {
	if err := h.validateProvider(provider); err != nil {
		h.log.WarnContext(ctx, "invalid provider", "provider", provider, "error", err)
		return nil, err
	}
	
	if redirectURI == "" {
		h.log.WarnContext(ctx, "missing redirect_uri parameter")
		return nil, NewMissingParameterError("redirect_uri")
	}
	
	if err := validateRedirectURI(redirectURI); err != nil {
		h.log.WarnContext(ctx, "invalid redirect_uri", "redirect_uri", redirectURI, "error", err)
		return nil, err
	}
	
	if state == "" {
		var err error
		state, err = generateState()
		if err != nil {
			h.log.ErrorContext(ctx, "failed to generate state", "error", err)
			return nil, fmt.Errorf("failed to generate state: %w", err)
		}
		h.log.InfoContext(ctx, "generated state parameter", "state", state)
	}
	
	providerCfg, err := h.getProviderConfig(provider)
	if err != nil {
		h.log.ErrorContext(ctx, "failed to get provider config", "provider", provider, "error", err)
		return nil, fmt.Errorf("failed to get provider configuration: %w", err)
	}
	
	authURL, err := buildAuthorizationURL(providerCfg, redirectURI, state)
	if err != nil {
		h.log.ErrorContext(ctx, "failed to build authorization URL", "provider", provider, "error", err)
		return nil, fmt.Errorf("failed to build authorization URL: %w", err)
	}
	
	resp := &AuthProviderResponse{
		Provider:         provider,
		AuthorizationURL: authURL,
		ClientID:         providerCfg.ClientID,
		Scopes:           providerCfg.Scopes,
		ResponseType:     "code",
		State:            state,
	}
	
	h.log.InfoContext(ctx, "successfully generated auth provider response", "provider", provider)
	return resp, nil
}

func (h *authProviderHandler) validateProvider(provider string) error {
	provider = strings.ToLower(provider)
	
	switch provider {
	case ProviderGoogle, ProviderFacebook, ProviderApple:
		return nil
	default:
		return NewInvalidProviderError(provider)
	}
}

func (h *authProviderHandler) getProviderConfig(provider string) (ProviderConfig, error) {
	provider = strings.ToLower(provider)
	
	switch provider {
	case ProviderGoogle:
		return h.oauth.Google, nil
	case ProviderFacebook:
		return h.oauth.Facebook, nil
	case ProviderApple:
		return h.oauth.Apple, nil
	default:
		return ProviderConfig{}, NewInvalidProviderError(provider)
	}
}

func validateRedirectURI(redirectURI string) error {
	if redirectURI == "" {
		return NewMissingParameterError("redirect_uri")
	}
	
	parsedURL, err := url.Parse(redirectURI)
	if err != nil {
		return NewInvalidRedirectURIError(redirectURI)
	}
	
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return NewInvalidRedirectURIError(redirectURI)
	}
	
	return nil
}

func generateState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b), nil
}

func buildAuthorizationURL(cfg ProviderConfig, redirectURI, state string) (string, error) {
	if cfg.AuthURL == "" {
		return "", errors.New("provider auth_url not configured")
	}
	
	params := url.Values{}
	params.Set("client_id", cfg.ClientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("response_type", "code")
	params.Set("scope", strings.Join(cfg.Scopes, " "))
	params.Set("state", state)
	
	return fmt.Sprintf("%s?%s", cfg.AuthURL, params.Encode()), nil
}
