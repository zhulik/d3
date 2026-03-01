package core

import "errors"

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

	ErrInvalidAdminCredentials = errors.New("invalid admin credentials")

	ErrConfigVersionMismatch = errors.New("config version mismatch")

	ErrObjectMetadataNotReadable = errors.New("object metadata not readable")

	ErrPolicyNotFound      = errors.New("policy not found")
	ErrPolicyAlreadyExists = errors.New("policy already exists")

	ErrBindingNotFound      = errors.New("binding not found")
	ErrBindingAlreadyExists = errors.New("binding already exists")
	ErrBindingInvalid       = errors.New("invalid binding")

	ErrUnauthorized = errors.New("unauthorized")

	ErrInvalidBucketName = errors.New("invalid bucket name")
	ErrInvalidObjectKey  = errors.New("invalid object key")
	ErrInvalidUploadID   = errors.New("invalid upload ID")
	ErrInvalidMaxKeys    = errors.New("max-keys must be between 1 and 1000")
	ErrPathTraversal     = errors.New("path traversal detected")
	ErrSymlinkNotAllowed = errors.New("symlinks are not allowed")
)
