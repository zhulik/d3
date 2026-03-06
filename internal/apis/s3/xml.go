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
	IsTruncated           bool            `xml:"IsTruncated"`
	Contents              []*types.Object `xml:"Contents"`
	Name                  string          `xml:"Name"`
	Prefix                string          `xml:"Prefix"`
	Delimiter             string          `xml:"Delimiter,omitempty"`
	MaxKeys               int             `xml:"MaxKeys"`
	KeyCount              int             `xml:"KeyCount"`
	NextContinuationToken *string         `xml:"NextContinuationToken,omitempty"`

	CommonPrefixes []prefixEntry `xml:"CommonPrefixes,omitempty"`
}

type listBucketResult struct {
	XMLName        xml.Name        `xml:"http://s3.amazonaws.com/doc/2006-03-01/ ListBucketResult"`
	IsTruncated    bool            `xml:"IsTruncated"`
	Marker         string          `xml:"Marker,omitempty"`
	NextMarker     *string         `xml:"NextMarker,omitempty"`
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

type copyObjectResultXML struct {
	XMLName      xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ CopyObjectResult"`
	ETag         string   `xml:"ETag"`
	LastModified string   `xml:"LastModified"`
}

type listPartXML struct {
	PartNumber   int    `xml:"PartNumber"`
	LastModified string `xml:"LastModified"`
	ETag         string `xml:"ETag"`
	Size         int64  `xml:"Size"`
}

type listPartsResultXML struct {
	XMLName              xml.Name         `xml:"http://s3.amazonaws.com/doc/2006-03-01/ ListPartsResult"`
	Bucket               string           `xml:"Bucket"`
	Key                  string           `xml:"Key"`
	UploadID             string           `xml:"UploadId"`
	PartNumberMarker     int              `xml:"PartNumberMarker,omitempty"`
	NextPartNumberMarker int              `xml:"NextPartNumberMarker,omitempty"`
	MaxParts             int              `xml:"MaxParts"`
	IsTruncated          bool             `xml:"IsTruncated"`
	Parts                []listPartXML    `xml:"Part,omitempty"`
	Owner                *types.Owner     `xml:"Owner,omitempty"`
	Initiator            *types.Initiator `xml:"Initiator,omitempty"`
	StorageClass         string           `xml:"StorageClass,omitempty"`
}

type listMultipartUploadEntryXML struct {
	Key       string `xml:"Key"`
	UploadID  string `xml:"UploadId"`
	Initiated string `xml:"Initiated"`
}

type listMultipartUploadsResultXML struct {
	XMLName            xml.Name                      `xml:"http://s3.amazonaws.com/doc/2006-03-01/ ListMultipartUploadsResult"` //nolint:lll
	Bucket             string                        `xml:"Bucket"`
	KeyMarker          string                        `xml:"KeyMarker,omitempty"`
	UploadIDMarker     string                        `xml:"UploadIdMarker,omitempty"` // AWS uses UploadIdMarker
	NextKeyMarker      string                        `xml:"NextKeyMarker,omitempty"`
	NextUploadIDMarker string                        `xml:"NextUploadIdMarker,omitempty"` // AWS uses NextUploadIdMarker
	Prefix             string                        `xml:"Prefix,omitempty"`
	Delimiter          string                        `xml:"Delimiter,omitempty"`
	MaxUploads         int                           `xml:"MaxUploads"`
	IsTruncated        bool                          `xml:"IsTruncated"`
	Uploads            []listMultipartUploadEntryXML `xml:"Upload,omitempty"`
	CommonPrefixes     []prefixEntry                 `xml:"CommonPrefixes,omitempty"`
}
