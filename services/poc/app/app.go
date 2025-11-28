// Copyright (c) 2025 Z5labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package app

import (
	"context"

	"github.com/z5labs/journeys/services/poc/storage"
	"github.com/z5labs/journeys/services/poc/ui"

	"github.com/dgraph-io/dgo/v240"
	"github.com/dgraph-io/dgo/v240/protos/api"
	"github.com/z5labs/humus/rest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Config struct {
	rest.Config `config:",squash"`

	Dgraph struct {
		Address string `config:"address"`
	} `config:"dgraph"`

	Minio struct {
		Endpoint  string `config:"endpoint"`
		AccessKey string `config:"access_key"`
		SecretKey string `config:"secret_key"`
		Bucket    string `config:"bucket"`
		UseSSL    bool   `config:"use_ssl"`
	} `config:"minio"`
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

	// Apply schema with indexes
	if err := dgraph.Alter(ctx, &api.Operation{Schema: dgraphSchema}); err != nil {
		return nil, err
	}

	minioClient, err := storage.NewMinioClient(
		cfg.Minio.Endpoint,
		cfg.Minio.AccessKey,
		cfg.Minio.SecretKey,
		cfg.Minio.Bucket,
		cfg.Minio.UseSSL,
	)
	if err != nil {
		return nil, err
	}

	if err := minioClient.EnsureBucket(ctx); err != nil {
		return nil, err
	}

	api := rest.NewApi(
		cfg.OpenApi.Title,
		cfg.OpenApi.Version,
		ui.GetJourneys(dgraph),
		ui.GetJourney(dgraph),
		ui.GetJourneyForm(dgraph),
		ui.CancelJourneyForm(dgraph),
		ui.CreateJourney(dgraph),
		ui.GetContent(dgraph),
		ui.GetContentUploadForm(dgraph),
		ui.UploadContent(dgraph, minioClient),
		ui.GetContentPreview(dgraph, minioClient),
		ui.GetLocationEditForm(dgraph),
		ui.UpdateLocation(dgraph),
	)

	return api, nil
}
