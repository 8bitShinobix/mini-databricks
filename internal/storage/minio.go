package storage

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioClient struct {
	client     *minio.Client
	bucketName string
}

func NewMinioClient(endpoint, accessKey, secretKey, bucketName string) (*MinioClient, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	return &MinioClient{client: client, bucketName: bucketName}, nil
}

func (m *MinioClient) EnsureBucketExists(ctx context.Context) error {
	exists, err := m.client.BucketExists(ctx, m.bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket: %w", err)
	}

	if !exists {
		if err := m.client.MakeBucket(ctx, m.bucketName, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return nil
}

func (m *MinioClient) GenerateUploadURL(ctx context.Context, objectKey string) (string, error) {
	url, err := m.client.PresignedPutObject(ctx, m.bucketName, objectKey, 15*time.Minute)
	if err != nil {
		return "", fmt.Errorf("failed to generate upload url: %w", err)
	}

	return url.String(), nil
}

func (m *MinioClient) GenerateDownloadURL(ctx context.Context, objectKey string) (string, error) {
	url, err := m.client.PresignedGetObject(ctx, m.bucketName, objectKey, 15*time.Minute, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate download url: %w", err)
	}

	return url.String(), nil
}

func (m *MinioClient) UploadBytes(ctx context.Context, objectKey string, data []byte, contentType string) (int64, error) {
	reader := bytes.NewReader(data)
	info, err := m.client.PutObject(ctx, m.bucketName, objectKey, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to upload object: %w", err)
	}
	return info.Size, nil
}
