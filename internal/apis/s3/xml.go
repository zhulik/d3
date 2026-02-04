package s3

import (
	"encoding/xml"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type bucketsResult struct {
	XMLName xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ ListAllMyBucketsResult"`
	Owner   *types.Owner
	Buckets []*types.Bucket `xml:"Buckets>Bucket"`
}

type locationConstraintResponse struct {
	XMLName  xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ LocationConstraint"`
	Location string   `xml:",chardata"`
}

type prefixEntry struct {
	Prefix string `xml:"Prefix"`
}

type listObjectsV2Result struct {
	IsTruncated    bool            `xml:"IsTruncated"`
	Contents       []*types.Object `xml:"Contents"`
	Name           string          `xml:"Name"`
	Prefix         string          `xml:"Prefix"`
	Delimiter      string          `xml:"Delimiter,omitempty"`
	MaxKeys        int             `xml:"MaxKeys"`
	CommonPrefixes []prefixEntry   `xml:"CommonPrefixes,omitempty"`
}

type taggingXML struct {
	XMLName xml.Name  `xml:"Tagging"`
	TagSet  tagSetXML `xml:"TagSet"`
}

type tagSetXML struct {
	Tags []tagXML `xml:"Tag"`
}

type tagXML struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

type deleteRequestXML struct {
	XMLName xml.Name          `xml:"http://s3.amazonaws.com/doc/2006-03-01/ Delete"`
	Objects []deleteObjectXML `xml:"Object"`
	Quiet   *bool             `xml:"Quiet"`
}

type deleteObjectXML struct {
	ETag             *string `xml:"ETag"`
	Key              string  `xml:"Key"`
	LastModifiedTime *string `xml:"LastModifiedTime"`
	Size             *int64  `xml:"Size"`
	VersionID        *string `xml:"VersionId"`
}

type deleteResultXML struct {
	XMLName xml.Name          `xml:"http://s3.amazonaws.com/doc/2006-03-01/ DeleteResult"`
	Deleted []deletedEntryXML `xml:"Deleted,omitempty"`
	Errors  []errorEntryXML   `xml:"Error,omitempty"`
}

type deletedEntryXML struct {
	DeleteMarker          *bool   `xml:"DeleteMarker,omitempty"`
	DeleteMarkerVersionID *string `xml:"DeleteMarkerVersionId,omitempty"`
	Key                   string  `xml:"Key"`
	VersionID             *string `xml:"VersionId,omitempty"`
}

type errorEntryXML struct {
	Code      string  `xml:"Code"`
	Key       string  `xml:"Key"`
	Message   string  `xml:"Message"`
	VersionID *string `xml:"VersionId,omitempty"`
}

type initiateMultipartUploadResultXML struct {
	XMLName  xml.Name `xml:"InitiateMultipartUploadResult"`
	Bucket   string   `xml:"Bucket"`
	Key      string   `xml:"Key"`
	UploadID string   `xml:"UploadId"`
}

type completeMultipartUploadRequestXML struct {
	XMLName xml.Name  `xml:"http://s3.amazonaws.com/doc/2006-03-01/ CompleteMultipartUpload"`
	Parts   []partXML `xml:"Part"`
}

type partXML struct {
	ChecksumCRC32     *string `xml:"ChecksumCRC32,omitempty"`
	ChecksumCRC32C    *string `xml:"ChecksumCRC32C,omitempty"`
	ChecksumCRC64NVME *string `xml:"ChecksumCRC64NVME,omitempty"`
	ChecksumSHA1      *string `xml:"ChecksumSHA1,omitempty"`
	ChecksumSHA256    *string `xml:"ChecksumSHA256,omitempty"`
	ETag              string  `xml:"ETag"`
	PartNumber        int     `xml:"PartNumber"`
}

type completeMultipartUploadResultXML struct {
	XMLName  xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ CompleteMultipartUploadResult"`
	Location string   `xml:"Location"`
	Bucket   string   `xml:"Bucket"`
	Key      string   `xml:"Key"`
	ETag     string   `xml:"ETag"`
}
