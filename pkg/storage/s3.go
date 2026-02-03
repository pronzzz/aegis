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

type S3Backend struct {
	client     *minio.Client
	bucketName string
}

// NewS3Backend creates a new S3 storage backend
func NewS3Backend(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*S3Backend, error) {
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	// Check bucket existence
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := minioClient.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if !exists {
		// Auto-create? Policy decision due to permissions.
		// For now, fail if not exists to be safe/explicit.
		// Or try to create.
		if err := minioClient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("bucket %s does not exist and creation failed: %w", bucket, err)
		}
	}

	return &S3Backend{
		client:     minioClient,
		bucketName: bucket,
	}, nil
}

func (s *S3Backend) objectKey(key string) string {
	// Use same hierarchy? objects/ab/cdef...
	// Object storage handles flat namespaces well, but hierarchy is good for structure.
	if len(key) < 2 {
		return "objects/" + key
	}
	return fmt.Sprintf("objects/%s/%s", key[:2], key[2:])
}

func (s *S3Backend) Put(key string, data []byte) error {
	ctx := context.Background()
	objectName := s.objectKey(key)

	// Upload
	// PutObject takes an io.Reader
	reader := bytes.NewReader(data)
	_, err := s.client.PutObject(ctx, s.bucketName, objectName, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	return err
}

func (s *S3Backend) Get(key string) ([]byte, error) {
	ctx := context.Background()
	objectName := s.objectKey(key)

	obj, err := s.client.GetObject(ctx, s.bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer obj.Close()

	// Read all
	return io.ReadAll(obj)
}

func (s *S3Backend) Has(key string) (bool, error) {
	ctx := context.Background()
	objectName := s.objectKey(key)

	_, err := s.client.StatObject(ctx, s.bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *S3Backend) Close() error {
	return nil
}
