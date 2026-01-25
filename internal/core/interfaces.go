package core

import (
	"context"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type HeadObjectResult struct {
	LastModified  time.Time
	ContentLength int64
}

type ObjectContent struct {
	io.ReadCloser
	LastModified time.Time
	Size         int64
}

type Backend interface {
	ListBuckets(ctx context.Context) ([]*types.Bucket, error)
	CreateBucket(ctx context.Context, name string) error
	DeleteBucket(ctx context.Context, name string) error
	HeadBucket(ctx context.Context, name string) error

	HeadObject(ctx context.Context, bucket, key string) (*HeadObjectResult, error)
	PutObject(ctx context.Context, bucket, key string, reader io.Reader) error
	GetObject(ctx context.Context, bucket, key string) (*ObjectContent, error)
	ListObjects(ctx context.Context, bucket, prefix string) ([]*types.Object, error)
}
