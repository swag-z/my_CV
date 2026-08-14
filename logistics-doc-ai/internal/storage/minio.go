package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Storage interface for object storage operations
type Storage interface {
	Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	PresignedGetURL(ctx context.Context, key string, expiry time.Duration) (string, error)
	EnsureBucket(ctx context.Context, bucket string) error
}

// MinIOClient implements Storage interface
type MinIOClient struct {
	client *minio.Client
	bucket string
}

// NewMinIOClient creates a new MinIO client
func NewMinIOClient(endpoint, accessKey, secretKey string, useSSL bool) (*MinIOClient, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	return &MinIOClient{
		client: client,
	}, nil
}

// SetBucket sets the default bucket
func (m *MinIOClient) SetBucket(bucket string) {
	m.bucket = bucket
}

// EnsureBucket creates bucket if it doesn't exist
func (m *MinIOClient) EnsureBucket(ctx context.Context, bucket string) error {
	exists, err := m.client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("check bucket exists: %w", err)
	}

	if !exists {
		err = m.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("create bucket: %w", err)
		}
	}

	return nil
}

// Upload uploads a file to MinIO
func (m *MinIOClient) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	bucket := m.bucket
	if bucket == "" {
		return fmt.Errorf("bucket not set")
	}

	_, err := m.client.PutObject(ctx, bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("upload object: %w", err)
	}

	return nil
}

// Download downloads a file from MinIO
func (m *MinIOClient) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	bucket := m.bucket
	if bucket == "" {
		return nil, fmt.Errorf("bucket not set")
	}

	obj, err := m.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object: %w", err)
	}

	return obj, nil
}

// PresignedGetURL generates a presigned URL for GET operation
func (m *MinIOClient) PresignedGetURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	bucket := m.bucket
	if bucket == "" {
		return "", fmt.Errorf("bucket not set")
	}

	url, err := m.client.PresignedGetObject(ctx, bucket, key, expiry, "")
	if err != nil {
		return "", fmt.Errorf("generate presigned URL: %w", err)
	}

	return url.String(), nil
}
