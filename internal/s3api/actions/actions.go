package actions

type Action string

const (
	// Bucket actions
	CreateBucket      Action = "s3:CreateBucket"
	HeadBucket        Action = "s3:HeadBucket"
	ListBuckets       Action = "s3:ListBuckets"
	DeleteBucket      Action = "s3:DeleteBucket"
	GetBucketLocation Action = "s3:GetBucketLocation"

	// Object actions
	PutObject               Action = "s3:PutObject"
	GetObject               Action = "s3:GetObject"
	DeleteObject            Action = "s3:DeleteObject"
	ListObjectsV2           Action = "s3:ListObjectsV2"
	DeleteObjects           Action = "s3:DeleteObjects"
	CreateMultipartUpload   Action = "s3:CreateMultipartUpload"
	UploadPart              Action = "s3:UploadPart"
	CompleteMultipartUpload Action = "s3:CompleteMultipartUpload"
	AbortMultipartUpload    Action = "s3:AbortMultipartUpload"
)
