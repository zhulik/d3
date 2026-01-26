package folder

import (
	"context"
	"path/filepath"

	"github.com/redis/rueidis"
	"github.com/zhulik/d3/internal/core"

	"github.com/redis/rueidis/om"
)

type ObjectMetadataValue struct {
	Key      string               `json:"key" redis:",key"` // the redis:",key" is required to indicate which field is the ULID key
	Ver      int64                `json:"ver" redis:",ver"` // the redis:",ver" is required to do optimistic locking to prevent lost update
	Metadata *core.ObjectMetadata `json:"metadata"`
}

type MetadataRepository struct {
	Config *core.Config

	repo om.Repository[ObjectMetadataValue]
}

func (m *MetadataRepository) Init(_ context.Context) error {
	c, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{"127.0.0.1:6379"}})
	if err != nil {
		panic(err)
	}
	m.repo = om.NewHashRepository("d3", ObjectMetadataValue{}, c)

	return nil
}

func (m *MetadataRepository) Save(ctx context.Context, bucket, key string, metadata *core.ObjectMetadata) error {
	err := m.repo.Save(ctx, &ObjectMetadataValue{
		Key:      filepath.Join(bucket, key),
		Ver:      1,
		Metadata: metadata,
	})
	return err
}

func (m *MetadataRepository) Get(ctx context.Context, bucket, key string) (*core.ObjectMetadata, error) {
	metadata, err := m.repo.Fetch(ctx, filepath.Join(bucket, key))
	if err != nil {
		return nil, err
	}
	return metadata.Metadata, nil
}

func (m *MetadataRepository) Delete(ctx context.Context, bucket, key string) error {
	return m.repo.Remove(ctx, filepath.Join(bucket, key))
}
