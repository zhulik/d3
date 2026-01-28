package folder

import (
	"context"
	"path/filepath"

	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/yaml"
)

type MetadataRepository struct {
	Config *core.Config
}

func (m *MetadataRepository) SaveTmp(ctx context.Context, folderPath string, metadata *core.ObjectMetadata) error {
	return yaml.MarshalToFile(metadata, filepath.Join(folderPath, metadataYamlFilename))
}

func (m *MetadataRepository) Get(_ context.Context, bucket, key string) (*core.ObjectMetadata, error) {
	path := filepath.Join(m.Config.FolderBackendPath, bucketsFolder, bucket, key)

	metadata, err := yaml.UnmarshalFromFile[core.ObjectMetadata](filepath.Join(path, metadataYamlFilename))
	if err != nil {
		return nil, err
	}
	return &metadata, nil
}
