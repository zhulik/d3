package folder //nolint:testpackage

import (
	"context"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/yaml"
)

type noopLocker struct{}

func (noopLocker) Lock(ctx context.Context, _ string) (context.Context, context.CancelFunc, error) {
	return ctx, func() {}, nil
}

var _ = Describe("ListMultipartUploads", func() {
	var (
		tmpDir        string
		cfg           *core.Config
		config        *Config
		bucket        *Bucket
		multipartRoot string
	)

	BeforeEach(func() {
		tmpDir = lo.Must(os.MkdirTemp("", "list-multipart-*"))

		DeferCleanup(func() { _ = os.RemoveAll(tmpDir) })

		lo.Must0(os.MkdirAll(filepath.Join(tmpDir, bucketsFolder), 0755))
		bucketPath := filepath.Join(tmpDir, bucketsFolder, "testbucket")
		lo.Must0(os.MkdirAll(bucketPath, 0755))
		lo.Must0(os.MkdirAll(filepath.Join(bucketPath, uploadsFolder, multipartFolder), 0755))

		cfg = &core.Config{FolderStorageBackendPath: tmpDir}
		config = &Config{Config: cfg}
		multipartRoot = filepath.Join(bucketPath, uploadsFolder, multipartFolder)
		bucket = &Bucket{
			name:         "testbucket",
			creationDate: time.Now(),
			config:       config,
			Locker:       noopLocker{},
		}
	})

	createUploadDir := func(key, uploadID string, initiated time.Time) {
		uploadPath := filepath.Join(multipartRoot, filepath.FromSlash(key), uploadID)
		lo.Must0(os.MkdirAll(uploadPath, 0755))

		meta := core.ObjectMetadata{LastModified: initiated}
		lo.Must0(yaml.MarshalToFile(meta, filepath.Join(uploadPath, metadataYamlFilename)))
	}

	When("no uploads exist", func() {
		It("returns empty result", func(ctx context.Context) {
			result, err := bucket.ListMultipartUploads(ctx, core.ListMultipartUploadsInput{})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Uploads).To(BeEmpty())
			Expect(result.CommonPrefixes).To(BeEmpty())
			Expect(result.IsTruncated).To(BeFalse())
		})
	})

	When("uploads exist", func() {
		BeforeEach(func() {
			t1 := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
			t2 := time.Date(2025, 1, 1, 11, 0, 0, 0, time.UTC)

			createUploadDir("a", "id1", t1)
			createUploadDir("a", "id2", t2)
			createUploadDir("b", "id3", t1)
		})

		It("returns all uploads sorted by key then initiated", func(ctx context.Context) {
			result, err := bucket.ListMultipartUploads(ctx, core.ListMultipartUploadsInput{})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Uploads).To(HaveLen(3))
			Expect(result.Uploads[0].Key).To(Equal("a"))
			Expect(result.Uploads[0].UploadID).To(Equal("id1"))
			Expect(result.Uploads[1].Key).To(Equal("a"))
			Expect(result.Uploads[1].UploadID).To(Equal("id2"))
			Expect(result.Uploads[2].Key).To(Equal("b"))
			Expect(result.Uploads[2].UploadID).To(Equal("id3"))
		})

		It("filters by prefix", func(ctx context.Context) {
			result, err := bucket.ListMultipartUploads(ctx, core.ListMultipartUploadsInput{
				Prefix: "a",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Uploads).To(HaveLen(2))
			Expect(result.Uploads[0].Key).To(Equal("a"))
			Expect(result.Uploads[1].Key).To(Equal("a"))
		})

		It("returns common prefixes when delimiter is set", func(ctx context.Context) {
			createUploadDir("p/q", "id4", time.Now())

			result, err := bucket.ListMultipartUploads(ctx, core.ListMultipartUploadsInput{
				Prefix:    "",
				Delimiter: "/",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.CommonPrefixes).To(ContainElement("p/"))
			Expect(result.Uploads).To(HaveLen(3)) // a, b (p/q is under common prefix p/)
		})

		It("truncates with max-uploads and returns next markers", func(ctx context.Context) {
			result, err := bucket.ListMultipartUploads(ctx, core.ListMultipartUploadsInput{
				MaxUploads: 2,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Uploads).To(HaveLen(2))
			Expect(result.IsTruncated).To(BeTrue())
			Expect(result.NextKeyMarker).NotTo(BeNil())
			Expect(result.NextUploadIDMarker).NotTo(BeNil())
			Expect(*result.NextKeyMarker).To(Equal("a"))
			Expect(*result.NextUploadIDMarker).To(Equal("id2"))
		})

		It("resumes from key-marker and upload-id-marker", func(ctx context.Context) {
			first, err := bucket.ListMultipartUploads(ctx, core.ListMultipartUploadsInput{
				MaxUploads: 1,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(first.Uploads).To(HaveLen(1))
			Expect(first.IsTruncated).To(BeTrue())

			second, err := bucket.ListMultipartUploads(ctx, core.ListMultipartUploadsInput{
				KeyMarker:      *first.NextKeyMarker,
				UploadIDMarker: *first.NextUploadIDMarker,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(second.Uploads).To(HaveLen(2))
			// Sorted order: (a, id2) then (b, id3)
			Expect(second.Uploads[0].Key).To(Equal("a"))
			Expect(second.Uploads[0].UploadID).To(Equal("id2"))
			Expect(second.Uploads[1].Key).To(Equal("b"))
		})
	})
})
