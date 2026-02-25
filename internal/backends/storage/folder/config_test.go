package folder_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/zhulik/d3/internal/backends/storage/folder"
	"github.com/zhulik/d3/internal/core"
)

func TestFolder(t *testing.T) {
	t.Parallel()

	RegisterFailHandler(Fail)
	RunSpecs(t, "Folder Suite")
}

var _ = Describe("EnsureContained", func() {
	DescribeTable("contained paths",
		func(path, parent string) {
			Expect(folder.EnsureContained(path, parent)).To(Succeed())
		},
		Entry("direct child", "/data/buckets/mybucket", "/data/buckets"),
		Entry("nested child", "/data/buckets/mybucket/subdir", "/data/buckets"),
		Entry("path equals parent", "/data/buckets", "/data/buckets"),
		Entry("with trailing separators", "/data/buckets/", "/data/buckets/"),
	)

	DescribeTable("escaped paths",
		func(path, parent string) {
			err := folder.EnsureContained(path, parent)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring(core.ErrPathTraversal.Error())))
		},
		Entry("parent traversal", "/data/buckets/../secret", "/data/buckets"),
		Entry("sibling directory", "/data/other", "/data/buckets"),
		Entry("prefix trick", "/data/buckets-evil", "/data/buckets"),
		Entry("double traversal", "/data/buckets/../../etc", "/data/buckets"),
		Entry("completely unrelated", "/etc/passwd", "/data/buckets"),
	)
})
