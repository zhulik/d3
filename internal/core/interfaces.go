package core

import (
	"context"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type ObjectMetadata struct {
	ContentType  string            `json:"str"`
	LastModified time.Time         `json:"last_modified"`
	SHA256       string            `json:"sha256"`
	SHA256Base64 string            `json:"sha256_base64"` // only when ObjectMetadata is returned by the backend
	Size         int64             `json:"size"`
	Tags         map[string]string `json:"tags"`
	Meta         map[string]string `json:"meta"`
}

type ObjectContent struct {
	io.ReadCloser
	*ObjectMetadata
}

type PutObjectInput struct {
	Reader   io.Reader
	Metadata ObjectMetadata
}

type Backend interface {
	ListBuckets(ctx context.Context) ([]*types.Bucket, error)
	CreateBucket(ctx context.Context, name string) error
	DeleteBucket(ctx context.Context, name string) error
	HeadBucket(ctx context.Context, name string) error

	HeadObject(ctx context.Context, bucket, key string) (*ObjectMetadata, error)
	PutObject(ctx context.Context, bucket, key string, input PutObjectInput) error
	GetObject(ctx context.Context, bucket, key string) (*ObjectContent, error)
	ListObjects(ctx context.Context, bucket, prefix string) ([]*types.Object, error)
	DeleteObject(ctx context.Context, bucket, key string) error
}
