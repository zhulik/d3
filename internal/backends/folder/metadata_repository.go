package folder

import (
	"context"
	"encoding/json"
	"path/filepath"
	"time"

	"github.com/redis/rueidis"
	"github.com/zhulik/d3/internal/core"

	"github.com/redis/rueidis/om"
)

type Metadata struct {
	ContentType  string            `json:"str"`
	LastModified time.Time         `json:"last_modified"`
	SHA256       string            `json:"sha256"`
	Size         int64             `json:"size"`
	Metadata     map[string]string `json:"metadata"`
}

type marshallableMetadata Metadata

func (m marshallableMetadata) MarshalJSON() ([]byte, error) {
	return json.Marshal(m)
}

func (m *marshallableMetadata) UnmarshalJSON(data []byte) error {
	var metadata Metadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return err
	}
	*m = marshallableMetadata(metadata)
	return nil
}

type ObjectMetadataValue struct {
	Key      string   `json:"key" redis:",key"` // the redis:",key" is required to indicate which field is the ULID key
	Ver      int64    `json:"ver" redis:",ver"` // the redis:",ver" is required to do optimistic locking to prevent lost update
	Metadata Metadata `json:"metadata"`
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
	m.repo = om.NewHashRepository("my_prefix", ObjectMetadataValue{}, c)

	return nil
}

func (m *MetadataRepository) Save(ctx context.Context, bucket, key string, metadata Metadata) error {
	err := m.repo.Save(ctx, &ObjectMetadataValue{
		Key:      filepath.Join(bucket, key),
		Ver:      1,
		Metadata: metadata,
	})
	return err
}

func (m *MetadataRepository) Get(ctx context.Context, bucket, key string) (Metadata, error) {
	metadata, err := m.repo.Fetch(ctx, filepath.Join(bucket, key))
	if err != nil {
		return Metadata{}, err
	}
	return metadata.Metadata, nil
}
