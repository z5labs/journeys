package app

import (
	"context"

	"github.com/z5labs/humus/rest"
	"github.com/z5labs/journeys/api/endpoint"
)

type Config struct {
	rest.Config `config:",squash"`
}

func Init(ctx context.Context, cfg Config) (*rest.Api, error) {
	api := rest.NewApi(
		cfg.OpenApi.Title,
		cfg.OpenApi.Version,
	)

	endpoint.CreateJourney(api)

	return api, nil
}
