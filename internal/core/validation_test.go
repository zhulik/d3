package core_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/google/uuid"
	"github.com/zhulik/d3/internal/core"
)

func TestCore(t *testing.T) {
	t.Parallel()

	RegisterFailHandler(Fail)
	RunSpecs(t, "Core Suite")
}

var _ = Describe("ValidateBucketName", func() {
	DescribeTable("valid names",
		func(name string) {
			Expect(core.ValidateBucketName(name)).To(Succeed())
		},
		Entry("simple lowercase", "mybucket"),
		Entry("with hyphens", "my-bucket"),
		Entry("with dots", "my.bucket"),
		Entry("with numbers", "bucket123"),
		Entry("minimum length (3)", "abc"),
		Entry("maximum length (63)", "aaa012345678901234567890123456789012345678901234567890123456789"),
		Entry("numbers only", "123"),
		Entry("mixed", "my-bucket.v2"),
	)

	DescribeTable("invalid names",
		func(name string) {
			err := core.ValidateBucketName(name)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("invalid bucket name")))
		},
		Entry("empty", ""),
		Entry("too short (2)", "ab"),
		Entry("too short (1)", "a"),
		Entry("uppercase letters", "MyBucket"),
		Entry("starts with hyphen", "-bucket"),
		Entry("ends with hyphen", "bucket-"),
		Entry("starts with dot", ".bucket"),
		Entry("ends with dot", "bucket."),
		Entry("contains double dots", "my..bucket"),
		Entry("contains underscore", "my_bucket"),
		Entry("contains space", "my bucket"),
		Entry("contains slash", "my/bucket"),
		Entry("contains backslash", "my\\bucket"),
		Entry("path traversal attempt", "../etc"),
	)
})

var _ = Describe("ValidateObjectKey", func() {
	DescribeTable("valid keys",
		func(key string) {
			Expect(core.ValidateObjectKey(key)).To(Succeed())
		},
		Entry("simple key", "file.txt"),
		Entry("nested path", "path/to/file.txt"),
		Entry("deeply nested", "a/b/c/d/e/file"),
		Entry("with dots in name", "archive.tar.gz"),
		Entry("trailing slash", "prefix/"),
		Entry("single char", "x"),
		Entry("literal dots not as segment", "my...file"),
		Entry("dot as segment", "path/./file.txt"),
	)

	DescribeTable("invalid keys",
		func(key string) {
			err := core.ValidateObjectKey(key)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("invalid object key")))
		},
		Entry("empty", ""),
		Entry("dot-dot segment at start", "../etc/passwd"),
		Entry("dot-dot segment in middle", "path/../secret"),
		Entry("dot-dot segment at end", "path/to/.."),
		Entry("only dot-dot", ".."),
		Entry("multiple traversals", "a/../../b"),
	)
})

var _ = Describe("ValidateUploadID", func() {
	DescribeTable("valid upload IDs",
		func(id string) {
			Expect(core.ValidateUploadID(id)).To(Succeed())
		},
		Entry("v4 UUID", uuid.NewString()),
		Entry("zeroed UUID", "00000000-0000-0000-0000-000000000000"),
	)

	DescribeTable("invalid upload IDs",
		func(id string) {
			err := core.ValidateUploadID(id)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("invalid upload ID")))
		},
		Entry("empty", ""),
		Entry("random string", "not-a-uuid"),
		Entry("path traversal attempt", "../../../etc/passwd"),
		Entry("too short", "abcd"),
		Entry("contains spaces", "00000000 0000 0000 0000 000000000000"),
	)
})
