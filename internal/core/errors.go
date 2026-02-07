package core

import "errors"

// TODO: move to core.
var (
	ErrBucketAlreadyExists = errors.New("bucket already exists")
	ErrBucketNotFound      = errors.New("bucket not found")
	ErrBucketNotEmpty      = errors.New("bucket is not empty")

	ErrObjectNotFound         = errors.New("object not found")
	ErrObjectAlreadyExists    = errors.New("object already exists")
	ErrObjectChecksumMismatch = errors.New("object checksum mismatch")

	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrUserInvalid       = errors.New("invalid user")

	ErrInvalidConfig = errors.New("invalid config")

	ErrConfigVersionMismatch = errors.New("config version mismatch")

	ErrObjectMetadataNotReadable = errors.New("object metadata not readable")

	ErrPolicyNotFound      = errors.New("policy not found")
	ErrPolicyAlreadyExists = errors.New("policy already exists")

	ErrUnauthorized = errors.New("unauthorized")
)
