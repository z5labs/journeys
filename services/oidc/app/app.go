// Copyright (c) 2025 Z5labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package app

import (
	"context"

	"github.com/z5labs/humus/rest"
	"github.com/z5labs/journeys/services/oidc/endpoint"
)

type ProviderConfig struct {
	ClientID     string   `config:"client_id"`
	ClientSecret string   `config:"client_secret"`
	AuthURL      string   `config:"auth_url"`
	TokenURL     string   `config:"token_url"`
	Scopes       []string `config:"scopes"`
}

type OAuthConfig struct {
	Google   ProviderConfig `config:"google"`
	Facebook ProviderConfig `config:"facebook"`
	Apple    ProviderConfig `config:"apple"`
}

type Config struct {
	rest.Config `config:",squash"`
	OAuth       OAuthConfig `config:"oauth"`
}

func Init(ctx context.Context, cfg Config) (*rest.Api, error) {
	oauthCfg := endpoint.OAuthConfig{
		Google: endpoint.ProviderConfig{
			ClientID:     cfg.OAuth.Google.ClientID,
			ClientSecret: cfg.OAuth.Google.ClientSecret,
			AuthURL:      cfg.OAuth.Google.AuthURL,
			TokenURL:     cfg.OAuth.Google.TokenURL,
			Scopes:       cfg.OAuth.Google.Scopes,
		},
		Facebook: endpoint.ProviderConfig{
			ClientID:     cfg.OAuth.Facebook.ClientID,
			ClientSecret: cfg.OAuth.Facebook.ClientSecret,
			AuthURL:      cfg.OAuth.Facebook.AuthURL,
			TokenURL:     cfg.OAuth.Facebook.TokenURL,
			Scopes:       cfg.OAuth.Facebook.Scopes,
		},
		Apple: endpoint.ProviderConfig{
			ClientID:     cfg.OAuth.Apple.ClientID,
			ClientSecret: cfg.OAuth.Apple.ClientSecret,
			AuthURL:      cfg.OAuth.Apple.AuthURL,
			TokenURL:     cfg.OAuth.Apple.TokenURL,
			Scopes:       cfg.OAuth.Apple.Scopes,
		},
	}
	
	operation := endpoint.RegisterAuthProviderGetEndpoint(oauthCfg)
	
	api := rest.NewApi(
		cfg.OpenApi.Title,
		cfg.OpenApi.Version,
		operation,
	)
	
	return api, nil
}
