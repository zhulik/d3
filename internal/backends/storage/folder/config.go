package folder

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/zhulik/d3/internal/core"
)

const (
	configYamlFilename   = "d3.yaml"
	bucketsFolder        = "buckets"
	objectsFolder        = "objects"
	TmpFolder            = "tmp"
	uploadsFolder        = "uploads"
	regularUploadsFolder = "regular"
	multipartFolder      = "multiplart"
	metadataYamlFilename = "metadata.yaml"
	uploadYamlFilename   = "upload.yaml"
	blobFilename         = "blob"
	binFolder            = "bin"
)

type Config struct {
	*core.Config
}

func (c *Config) bucketPath(bucket string) (string, error) {
	path := filepath.Join(c.FolderStorageBackendPath, bucketsFolder, bucket)

	return path, EnsureContained(path, c.bucketsPath())
}

func (c *Config) objectPath(bucket, key string) (string, error) {
	bucketRoot, err := c.bucketPath(bucket)
	if err != nil {
		return "", err
	}

	objectsRoot := filepath.Join(bucketRoot, objectsFolder)

	path := filepath.Join(objectsRoot, key)

	return path, EnsureContained(path, objectsRoot)
}

func (c *Config) bucketsPath() string {
	return filepath.Join(c.FolderStorageBackendPath, bucketsFolder)
}

func (c *Config) bucketUploadsPath(bucket string) (string, error) {
	bucketRoot, err := c.bucketPath(bucket)
	if err != nil {
		return "", err
	}

	uploadsRoot := filepath.Join(bucketRoot, uploadsFolder)

	return uploadsRoot, EnsureContained(uploadsRoot, bucketRoot)
}

func (c *Config) binPath() string {
	return filepath.Join(c.FolderStorageBackendPath, TmpFolder, binFolder)
}

func (c *Config) newBinPath() string {
	return filepath.Join(c.binPath(), uuid.NewString())
}

func (c *Config) configYamlPath() string {
	return filepath.Join(c.FolderStorageBackendPath, configYamlFilename)
}

func (c *Config) newUploadPath(bucket string) (string, error) {
	uploadsRoot, err := c.bucketUploadsPath(bucket)
	if err != nil {
		return "", err
	}

	return filepath.Join(uploadsRoot, regularUploadsFolder, uuid.NewString()), nil
}

func (c *Config) multipartUploadPath(bucket, uploadID string) (string, error) {
	uploadsRoot, err := c.bucketUploadsPath(bucket)
	if err != nil {
		return "", err
	}

	path := filepath.Join(uploadsRoot, multipartFolder, "multipart-"+uploadID)

	return path, EnsureContained(path, uploadsRoot)
}

func (c *Config) newMultipartUploadPath(bucket string) (string, string, error) {
	uploadsRoot, err := c.bucketUploadsPath(bucket)
	if err != nil {
		return "", "", err
	}

	id := uuid.NewString()

	path := filepath.Join(uploadsRoot, multipartFolder, "multipart-"+id)

	return id, path, EnsureContained(path, uploadsRoot)
}

func EnsureContained(path, parent string) error {
	cleanPath := filepath.Clean(path) + string(filepath.Separator)
	cleanParent := filepath.Clean(parent) + string(filepath.Separator)

	if !strings.HasPrefix(cleanPath, cleanParent) {
		return fmt.Errorf("%w: %q escapes %q", core.ErrPathTraversal, path, parent)
	}

	return nil
}
