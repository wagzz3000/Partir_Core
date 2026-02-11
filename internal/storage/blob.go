package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// BlobStore handles S3/MinIO object storage
type BlobStore struct {
	client *minio.Client
	bucket string
}

// NewBlobStore creates a new S3/MinIO blob store
func NewBlobStore(endpoint, bucket, accessKey, secretKey string) (*BlobStore, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false, // Use true for production with TLS
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	return &BlobStore{
		client: client,
		bucket: bucket,
	}, nil
}

// EnsureBucket creates the bucket if it doesn't exist
func (b *BlobStore) EnsureBucket(ctx context.Context) error {
	exists, err := b.client.BucketExists(ctx, b.bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket: %w", err)
	}
	if !exists {
		if err := b.client.MakeBucket(ctx, b.bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}
	return nil
}

// PutBlob stores a blob by content hash
func (b *BlobStore) PutBlob(ctx context.Context, hash string, data []byte, contentType string) (string, error) {
	objectName := fmt.Sprintf("artifacts/%s", hash)

	reader := bytes.NewReader(data)
	_, err := b.client.PutObject(ctx, b.bucket, objectName, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to put object: %w", err)
	}

	return fmt.Sprintf("s3://%s/%s", b.bucket, objectName), nil
}

// GetBlob retrieves a blob by hash
func (b *BlobStore) GetBlob(ctx context.Context, hash string) ([]byte, error) {
	objectName := fmt.Sprintf("artifacts/%s", hash)

	obj, err := b.client.GetObject(ctx, b.bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer obj.Close()

	return io.ReadAll(obj)
}

// PutFailed stores a failed artifact with TTL prefix
func (b *BlobStore) PutFailed(ctx context.Context, hash string, data []byte, contentType string) (string, error) {
	objectName := fmt.Sprintf("failed/%s", hash)

	reader := bytes.NewReader(data)
	_, err := b.client.PutObject(ctx, b.bucket, objectName, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to put failed artifact: %w", err)
	}

	return fmt.Sprintf("s3://%s/%s", b.bucket, objectName), nil
}

// PutLog stores execution logs
func (b *BlobStore) PutLog(ctx context.Context, runID string, data []byte) (string, error) {
	objectName := fmt.Sprintf("logs/%s/execution.log", runID)

	reader := bytes.NewReader(data)
	_, err := b.client.PutObject(ctx, b.bucket, objectName, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: "text/plain",
	})
	if err != nil {
		return "", fmt.Errorf("failed to put log: %w", err)
	}

	return fmt.Sprintf("s3://%s/%s", b.bucket, objectName), nil
}

// DeleteBlob removes a blob by hash
func (b *BlobStore) DeleteBlob(ctx context.Context, hash string) error {
	objectName := fmt.Sprintf("artifacts/%s", hash)
	return b.client.RemoveObject(ctx, b.bucket, objectName, minio.RemoveObjectOptions{})
}

// CleanupFailed removes failed artifacts older than the given duration
func (b *BlobStore) CleanupFailed(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	prefix := "failed/"

	objectCh := b.client.ListObjects(ctx, b.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for obj := range objectCh {
		if obj.Err != nil {
			return fmt.Errorf("failed to list objects: %w", obj.Err)
		}
		if obj.LastModified.Before(cutoff) {
			if err := b.client.RemoveObject(ctx, b.bucket, obj.Key, minio.RemoveObjectOptions{}); err != nil {
				return fmt.Errorf("failed to remove old failed artifact: %w", err)
			}
		}
	}

	return nil
}
