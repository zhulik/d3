package folder

import "time"

type Bucket struct {
	name         string
	creationDate time.Time
}

func (b Bucket) Name() string {
	return b.name
}

func (b Bucket) ARN() string {
	return "arn:aws:s3:::" + b.Name()
}

func (b Bucket) Region() string {
	return "local"
}

func (b Bucket) CreationDate() time.Time {
	return b.creationDate
}
