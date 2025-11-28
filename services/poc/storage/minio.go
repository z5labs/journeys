// Copyright (c) 2025 Z5labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package storage

import (
	"context"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioClient struct {
	client *minio.Client
	bucket string
}

func NewMinioClient(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*MinioClient, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	return &MinioClient{
		client: client,
		bucket: bucket,
	}, nil
}

func (m *MinioClient) EnsureBucket(ctx context.Context) error {
	exists, err := m.client.BucketExists(ctx, m.bucket)
	if err != nil {
		return err
	}

	if !exists {
		err = m.client.MakeBucket(ctx, m.bucket, minio.MakeBucketOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *MinioClient) UploadFile(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) error {
	_, err := m.client.PutObject(ctx, m.bucket, objectKey, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func (m *MinioClient) GetFileURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	url, err := m.client.PresignedGetObject(ctx, m.bucket, objectKey, expiry, nil)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

func (m *MinioClient) DeleteFile(ctx context.Context, objectKey string) error {
	return m.client.RemoveObject(ctx, m.bucket, objectKey, minio.RemoveObjectOptions{})
}
