package folder //nolint:testpackage

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/zhulik/d3/internal/core"
)

var _ = Describe("Backend bucket creation date", func() {
	var (
		tmpDir  string
		backend *Backend
	)

	BeforeEach(func(ctx SpecContext) {
		tmpDir = lo.Must(os.MkdirTemp("", "bucket-creation-date-*"))

		DeferCleanup(func() { _ = os.RemoveAll(tmpDir) })

		backend = &Backend{
			Cfg: &core.Config{
				FolderStorageBackendPath: tmpDir,
			},
			Locker: noopLocker{},
		}

		lo.Must0(backend.Init(ctx))
	})

	When("a bucket exists with metadata", func() {
		It("keeps creation date stable when directory mtime changes", func(ctx SpecContext) {
			lo.Must0(backend.CreateBucket(ctx, "stable"))

			initialBucket := lo.Must(backend.HeadBucket(ctx, "stable"))
			initialCreationDate := initialBucket.CreationDate()

			bucketPath := lo.Must(backend.config.bucketPath("stable"))
			touched := initialCreationDate.Add(2 * time.Hour)
			lo.Must0(os.Chtimes(bucketPath, touched, touched))

			// Ensure test setup actually changed the directory mtime.
			stat := lo.Must(os.Lstat(bucketPath))
			Expect(stat.ModTime()).NotTo(Equal(initialCreationDate))

			afterHead := lo.Must(backend.HeadBucket(ctx, "stable"))
			Expect(afterHead.CreationDate()).To(Equal(initialCreationDate))

			listed := lo.Must(backend.ListBuckets(ctx))
			Expect(listed).To(HaveLen(1))
			Expect(listed[0].Name()).To(Equal("stable"))
			Expect(listed[0].CreationDate()).To(Equal(initialCreationDate))
		})
	})

	When("a bucket metadata file is missing", func() {
		It("falls back to directory mtime", func(ctx SpecContext) {
			lo.Must0(backend.CreateBucket(ctx, "legacy"))

			metadataPath := filepath.Join(tmpDir, bucketsFolder, "legacy", "bucket.yaml")
			lo.Must0(os.Remove(metadataPath))

			bucketPath := lo.Must(backend.config.bucketPath("legacy"))
			expected := time.Now().Add(3 * time.Hour).Truncate(time.Second)
			lo.Must0(os.Chtimes(bucketPath, expected, expected))

			got := lo.Must(backend.HeadBucket(ctx, "legacy"))
			Expect(got.CreationDate().Unix()).To(Equal(expected.Unix()))
		})
	})

	When("deleting an empty bucket with metadata", func() {
		It("removes bucket successfully", func(ctx SpecContext) {
			lo.Must0(backend.CreateBucket(ctx, "to-delete"))
			lo.Must0(backend.DeleteBucket(ctx, "to-delete"))

			_, err := backend.HeadBucket(ctx, "to-delete")
			Expect(err).To(Equal(core.ErrBucketNotFound))
		})
	})
})
