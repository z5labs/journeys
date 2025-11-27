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
	Name         string   `config:"name"`
	ClientID     string   `config:"client_id"`
	ClientSecret string   `config:"client_secret"`
	AuthURL      string   `config:"auth_url"`
	TokenURL     string   `config:"token_url"`
	Scopes       []string `config:"scopes"`
}

type Config struct {
	rest.Config `config:",squash"`
	Providers   []ProviderConfig `config:"providers"`
}

func Init(ctx context.Context, cfg Config) (*rest.Api, error) {
	var opts []endpoint.AuthProviderOption
	for _, provider := range cfg.Providers {
		opts = append(opts, endpoint.WithProvider(
			provider.Name,
			provider.ClientID,
			provider.ClientSecret,
			provider.AuthURL,
			provider.TokenURL,
			provider.Scopes,
		))
	}
	
	operation := endpoint.RegisterAuthProviderGetEndpoint(opts...)
	
	api := rest.NewApi(
		cfg.OpenApi.Title,
		cfg.OpenApi.Version,
		operation,
	)
	
	return api, nil
}
