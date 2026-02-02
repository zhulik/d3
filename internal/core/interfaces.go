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
	Reader   io.ReadSeekCloser
	Metadata *ObjectMetadata
}

type PutObjectInput struct {
	Reader   io.Reader
	Metadata ObjectMetadata
}

type ListObjectsV2Input struct {
	Prefix  string
	MaxKeys int
}

type DeleteResult struct {
	Key   string
	Error error
}

type CompletePart struct {
	PartNumber int
	ETag       string
}

type User struct {
	Name            string
	AccessKeyID     string
	SecretAccessKey string
}

func (u User) ARN() string {
	return "arn:aws:iam:::user/" + u.Name
}

type Bucket interface {
	Name() string
	ARN() string
	Region() string
	CreationDate() time.Time
}

type Backend interface { //nolint:interfacebloat
	ListBuckets(ctx context.Context) ([]Bucket, error)
	CreateBucket(ctx context.Context, name string) error
	DeleteBucket(ctx context.Context, name string) error
	HeadBucket(ctx context.Context, name string) (Bucket, error)

	HeadObject(ctx context.Context, bucket, key string) (*ObjectMetadata, error)
	PutObject(ctx context.Context, bucket, key string, input PutObjectInput) error
	GetObjectTagging(ctx context.Context, bucket, key string) (map[string]string, error)
	GetObject(ctx context.Context, bucket, key string) (*ObjectContent, error)
	ListObjectsV2(ctx context.Context, bucket string, input ListObjectsV2Input) ([]*types.Object, error)
	DeleteObjects(ctx context.Context, bucket string, quiet bool, keys ...string) ([]DeleteResult, error)

	CreateMultipartUpload(ctx context.Context, bucket, key string, metadata ObjectMetadata) (string, error)
	UploadPart(ctx context.Context, bucket, key string, uploadID string, partNumber int, body io.Reader) error
	CompleteMultipartUpload(ctx context.Context, bucket, key string, uploadID string, parts []CompletePart) error
	AbortMultipartUpload(ctx context.Context, bucket, key string, uploadID string) error
}

type UserRepository interface {
	AdminCredentials() (string, string)

	GetUserByName(ctx context.Context, name string) (*User, error)
	GetUserByAccessKeyID(ctx context.Context, accessKeyID string) (*User, error)
	CreateUser(ctx context.Context, user User) error
	DeleteUser(ctx context.Context, name string) error
}
