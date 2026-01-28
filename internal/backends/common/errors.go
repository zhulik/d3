package common //nolint:revive

import "errors"

var (
	ErrBucketAlreadyExists = errors.New("bucket already exists")
	ErrBucketNotFound      = errors.New("bucket not found")
	ErrBucketNotEmpty      = errors.New("bucket is not empty")

	ErrObjectNotFound         = errors.New("object not found")
	ErrObjectAlreadyExists    = errors.New("object already exists")
	ErrObjectChecksumMismatch = errors.New("object checksum mismatch")
)
