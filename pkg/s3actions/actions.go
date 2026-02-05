package s3actions

type Action string

const (
	All Action = "s3:*"

	CreateBucket      Action = "s3:CreateBucket"
	HeadBucket        Action = "s3:HeadBucket"
	ListBuckets       Action = "s3:ListBuckets"
	DeleteBucket      Action = "s3:DeleteBucket"
	GetBucketLocation Action = "s3:GetBucketLocation"

	PutObject               Action = "s3:PutObject"
	GetObject               Action = "s3:GetObject"
	HeadObject              Action = "s3:HeadObject"
	DeleteObject            Action = "s3:DeleteObject"
	ListObjectsV2           Action = "s3:ListObjectsV2"
	DeleteObjects           Action = "s3:DeleteObjects"
	CreateMultipartUpload   Action = "s3:CreateMultipartUpload"
	UploadPart              Action = "s3:UploadPart"
	CompleteMultipartUpload Action = "s3:CompleteMultipartUpload"
	AbortMultipartUpload    Action = "s3:AbortMultipartUpload"
	GetObjectTagging        Action = "s3:GetObjectTagging"
)

var (
	Actions = []Action{ //nolint:gochecknoglobals
		All,
		ListBuckets,
		HeadBucket,
		DeleteBucket,
		CreateBucket,
		GetBucketLocation,
		PutObject,
		GetObject,
		HeadObject,
		DeleteObject,
		ListObjectsV2,
		DeleteObjects,
		CreateMultipartUpload,
		UploadPart,
		CompleteMultipartUpload,
		AbortMultipartUpload,
		GetObjectTagging,
	}
)
