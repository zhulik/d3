package core

const (
	MaxKeys       = 1000
	MaxPartNumber = 10000 // AWS S3 limit for multipart upload part numbers
	Delimiter     = "/"

	// SizeLimit5Gb is the max request body size for S3 API (AWS S3 single PUT and multipart part max).
	SizeLimit5Gb = 5 * 1024 * 1024 * 1024
	// SizeLimit1Mb is the max request body size for Management API (JSON/YAML payloads).
	SizeLimit1Mb = 1024 * 1024
)
