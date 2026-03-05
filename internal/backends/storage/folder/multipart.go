package folder

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/yaml"
)

// IncompleteMultipartUpload represents an in-progress multipart upload.
type IncompleteMultipartUpload struct {
	bucket    *Bucket
	path      string
	key       string
	uploadID  string
	initiated time.Time
}

func (u *IncompleteMultipartUpload) Key() string {
	return u.key
}

func (u *IncompleteMultipartUpload) UploadID() string {
	return u.uploadID
}

func (u *IncompleteMultipartUpload) Initiated() time.Time {
	return u.initiated
}

// MultipartUploadFromPath builds an IncompleteMultipartUpload from a path under the multipart root.
// The path must be multipartRoot/key/uploadId with a valid metadata.yaml (Initiated = LastModified).
// Returns nil, nil if the path is not a valid multipart upload directory.
func MultipartUploadFromPath(bucket *Bucket, multipartRoot, path string) (*IncompleteMultipartUpload, error) {
	if err := rejectSymlinkInPath(path); err != nil {
		return nil, err
	}

	rel, err := filepath.Rel(multipartRoot, path)
	if err != nil {
		return nil, err
	}

	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, nil //nolint:nilnil
	}

	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) < 2 {
		return nil, nil //nolint:nilnil
	}

	key := filepath.ToSlash(strings.Join(parts[:len(parts)-1], string(filepath.Separator)))
	uploadID := parts[len(parts)-1]

	metadataPath := filepath.Join(path, metadataYamlFilename)

	info, err := os.Lstat(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil //nolint:nilnil
		}

		return nil, err
	}

	if !info.Mode().IsRegular() {
		return nil, nil //nolint:nilnil
	}

	metadata, err := yaml.UnmarshalFromFile[core.ObjectMetadata](metadataPath)
	if err != nil {
		return nil, err
	}

	return &IncompleteMultipartUpload{
		bucket:    bucket,
		path:      path,
		key:       key,
		uploadID:  uploadID,
		initiated: metadata.LastModified,
	}, nil
}

// IsMultipartUploadPath reports whether path is a valid multipart upload directory under multipartRoot.
func IsMultipartUploadPath(multipartRoot, path string) (bool, error) {
	rel, err := filepath.Rel(multipartRoot, path)
	if err != nil {
		return false, err
	}

	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false, nil
	}

	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) < 2 {
		return false, nil
	}

	metadataPath := filepath.Join(path, metadataYamlFilename)

	exists, err := existsAndIsFile(metadataPath)
	if err != nil {
		return false, err
	}

	return exists, nil
}
