package core

import (
	"context"
	"io"
	"time"
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

type Bucket interface { //nolint:interfacebloat
	Name() string
	ARN() string
	Region() string
	CreationDate() time.Time

	HeadObject(ctx context.Context, key string) (Object, error)
	PutObject(ctx context.Context, key string, input PutObjectInput) error
	GetObject(ctx context.Context, key string) (Object, error)
	ListObjectsV2(ctx context.Context, input ListObjectsV2Input) ([]Object, error)
	DeleteObjects(ctx context.Context, quiet bool, keys ...string) ([]DeleteResult, error)

	CreateMultipartUpload(ctx context.Context, key string, metadata ObjectMetadata) (string, error)
	UploadPart(ctx context.Context, key string, uploadID string, partNumber int, body io.Reader) error
	CompleteMultipartUpload(ctx context.Context, key string, uploadID string, parts []CompletePart) error
	AbortMultipartUpload(ctx context.Context, key string, uploadID string) error
}

type Object interface {
	io.ReadSeekCloser

	Key() string
	LastModified() time.Time
	Size() int64
	Metadata() *ObjectMetadata
}

type StorageBackend interface {
	ListBuckets(ctx context.Context) ([]Bucket, error)
	CreateBucket(ctx context.Context, name string) error
	DeleteBucket(ctx context.Context, name string) error
	HeadBucket(ctx context.Context, name string) (Bucket, error)
}

type ManagementBackend interface {
	AdminCredentials() (string, string)

	GetUsers(ctx context.Context) ([]string, error)
	GetUserByName(ctx context.Context, name string) (*User, error)
	GetUserByAccessKeyID(ctx context.Context, accessKeyID string) (*User, error)
	CreateUser(ctx context.Context, user User) error
	UpdateUser(ctx context.Context, user User) error
	DeleteUser(ctx context.Context, name string) error
}
