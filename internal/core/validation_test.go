package core_test

import (
	"errors"
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

var _ = Describe("ValidatePartNumber", func() {
	DescribeTable("valid part numbers",
		func(partNumber int) {
			Expect(core.ValidatePartNumber(partNumber)).To(Succeed())
		},
		Entry("minimum (1)", 1),
		Entry("typical", 5),
		Entry("maximum (10000)", core.MaxPartNumber),
	)

	DescribeTable("invalid part numbers",
		func(partNumber int) {
			err := core.ValidatePartNumber(partNumber)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, core.ErrInvalidPartNumber)).To(BeTrue())
		},
		Entry("zero", 0),
		Entry("negative", -1),
		Entry("above maximum", core.MaxPartNumber+1),
	)
})

var _ = Describe("ValidateAdminUser", func() {
	When("user is valid", func() {
		It("succeeds with AWS-style credentials", func() {
			user := &core.User{
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			}
			Expect(core.ValidateAdminUser(user)).To(Succeed())
		})
	})

	When("user is invalid", func() {
		DescribeTable("validation failures",
			func(user *core.User, expectedSubstring string) {
				err := core.ValidateAdminUser(user)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring(expectedSubstring)))
			},
			Entry("nil user", nil, "admin user is nil"),
			Entry("empty access key", &core.User{
				AccessKeyID:     "",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			}, "access_key_id is empty"),
			Entry("empty secret key", &core.User{
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "",
			}, "secret_access_key is empty"),
			Entry("wrong access key length", &core.User{
				AccessKeyID:     "short",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			}, "access_key_id must be 20 characters"),
			Entry("wrong secret key length", &core.User{
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "short",
			}, "secret_access_key must be 40 characters"),
		)
	})
})
