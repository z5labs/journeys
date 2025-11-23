// Copyright (c) 2025 Z5labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package endpoint

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/z5labs/humus"
	"github.com/z5labs/humus/rest"
	"github.com/z5labs/humus/rest/rpc"
)

type providerConfig struct {
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	Scopes       []string
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
	log       *slog.Logger
	providers map[string]providerConfig
}

type AuthProviderOption func(*authProviderHandler)

func WithProvider(name, clientID, clientSecret, authURL, tokenURL string, scopes []string) AuthProviderOption {
	return func(h *authProviderHandler) {
		h.providers[name] = providerConfig{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			AuthURL:      authURL,
			TokenURL:     tokenURL,
			Scopes:       scopes,
		}
	}
}

func RegisterAuthProviderGetEndpoint(opts ...AuthProviderOption) rest.ApiOption {
	h := &authProviderHandler{
		log:       humus.Logger("auth.provider.get"),
		providers: make(map[string]providerConfig),
	}
	
	for _, opt := range opts {
		opt(h)
	}
	
	return rest.Handle(
		http.MethodGet,
		rest.BasePath("/v1/auth").Param("provider"),
		rpc.ProduceJson(h),
		rest.QueryParam("redirect_uri", rest.Required()),
		rest.QueryParam("state"),
	)
}

func (h *authProviderHandler) Produce(ctx context.Context) (*AuthProviderResponse, error) {
	provider := rest.PathParamValue(ctx, "provider")
	
	redirectURIVals := rest.QueryParamValue(ctx, "redirect_uri")
	var redirectURI string
	if len(redirectURIVals) > 0 {
		redirectURI = redirectURIVals[0]
	}
	
	stateVals := rest.QueryParamValue(ctx, "state")
	var state string
	if len(stateVals) > 0 {
		state = stateVals[0]
	}
	
	h.log.InfoContext(ctx, "handling auth provider request", "provider", provider)
	
	resp, err := h.handle(ctx, provider, redirectURI, state)
	if err != nil {
		h.log.ErrorContext(ctx, "failed to handle request", "error", err)
		return nil, err
	}
	
	return resp, nil
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
	
	if _, ok := h.providers[provider]; !ok {
		return NewInvalidProviderError(provider)
	}
	return nil
}

func (h *authProviderHandler) getProviderConfig(provider string) (providerConfig, error) {
	provider = strings.ToLower(provider)
	
	cfg, ok := h.providers[provider]
	if !ok {
		return providerConfig{}, NewInvalidProviderError(provider)
	}
	return cfg, nil
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

func buildAuthorizationURL(cfg providerConfig, redirectURI, state string) (string, error) {
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
