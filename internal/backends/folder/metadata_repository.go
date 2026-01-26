package folder

import (
	"context"
	"encoding/json"
	"path"

	"github.com/redis/rueidis"
	"github.com/zhulik/d3/internal/core"

	"github.com/redis/rueidis/om"
)

type Metadata struct {
	ContentType string            `json:"str"`
	Metadata    map[string]string `json:"metadata"`
	SHA256      string            `json:"sha256"`
}

func (m Metadata) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"content_type": m.ContentType,
		"metadata":     m.Metadata,
		"sha256":       m.SHA256,
	})
}

func (m *Metadata) UnmarshalJSON(data []byte) error {
	var tmp struct {
		ContentType string            `json:"content_type"`
		Metadata    map[string]string `json:"metadata"`
		SHA256      string            `json:"sha256"`
	}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	m.ContentType = tmp.ContentType
	m.Metadata = tmp.Metadata
	m.SHA256 = tmp.SHA256
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
		Key:      path.Join(bucket, key),
		Ver:      1,
		Metadata: metadata,
	})
	return err
}

func (m *MetadataRepository) Get(ctx context.Context, bucket, key string) (Metadata, error) {
	metadata, err := m.repo.Fetch(ctx, path.Join(bucket, key))
	if err != nil {
		return Metadata{}, err
	}
	return metadata.Metadata, nil
}
