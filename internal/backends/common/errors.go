package common

import "errors"

var (
	ErrBucketAlreadyExists = errors.New("bucket already exists")
	ErrBucketNotFound      = errors.New("bucket not found")

	ErrObjectNotFound = errors.New("object not found")
	ErrObjectTooLarge = errors.New("object too large")
)
