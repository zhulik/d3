package core

import (
	"context"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type ObjectMetadata struct {
	ContentType  string            `yaml:"content_type"`
	LastModified time.Time         `yaml:"last_modified"`
	SHA256       string            `yaml:"sha256"`
	SHA256Base64 string            `yaml:"sha256_base64"` // only when ObjectMetadata is returned by the backend
	Size         int64             `yaml:"size"`
	Tags         map[string]string `yaml:"tags"`
	Meta         map[string]string `yaml:"meta"`
}

type ObjectContent struct {
	Reader   io.ReadCloser
	Metadata *ObjectMetadata
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
	GetObjectTagging(ctx context.Context, bucket, key string) (map[string]string, error)
	GetObject(ctx context.Context, bucket, key string) (*ObjectContent, error)
	ListObjectsV2(ctx context.Context, bucket, prefix string) ([]*types.Object, error)
	DeleteObject(ctx context.Context, bucket, key string) error
}
