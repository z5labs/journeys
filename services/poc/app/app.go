// Copyright (c) 2025 Z5labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package app

import (
	"context"

	"github.com/z5labs/journeys/services/poc/endpoint"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/dgraph-io/dgo/v240"
	"github.com/z5labs/humus/rest"
)

type Config struct {
	rest.Config `config:",squash"`

	Dgraph struct {
		Address string `config:"address"`
	} `config:"dgraph"`
}

func Init(ctx context.Context, cfg Config) (*rest.Api, error) {
	dgraph, err := dgo.NewClient(
		cfg.Dgraph.Address,
		dgo.WithGrpcOption(
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		),
	)
	if err != nil {
		return nil, err
	}

	api := rest.NewApi(
		cfg.OpenApi.Title,
		cfg.OpenApi.Version,
		endpoint.NewJourney(dgraph),
	)

	return api, nil
}
