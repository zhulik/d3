package conformance_test

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/samber/lo"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Buckets API", Label("conformance"), Label("api-buckets"), Ordered, func() {
	var s3Client *s3.Client
	var bucketName *string

	var cancelApp context.CancelFunc

	BeforeAll(func(ctx context.Context) {
		s3Client, bucketName, cancelApp = prepareConformanceTests(ctx)
	})

	AfterAll(func(ctx context.Context) {
		cleanupS3(ctx, s3Client, bucketName)

		cancelApp()
	})

	Describe("CreateBucket", func() {
		Context("when bucket already exists", func() {
			It("should return error", func(ctx context.Context) {
				_, err := s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
					Bucket: bucketName,
				})
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("ListBuckets", func() {
		It("should list buckets", func(ctx context.Context) {
			listBucketsOutput, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
			Expect(err).NotTo(HaveOccurred())

			found := lo.ContainsBy(listBucketsOutput.Buckets, func(bucket types.Bucket) bool {
				return *bucket.Name == *bucketName
			})

			Expect(found).To(BeTrue())
		})
	})

	Describe("DeleteBucket", func() {
		Context("when bucket is not empty", func() {
			BeforeAll(func(ctx context.Context) {
				_, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
					Bucket: bucketName,
					Key:    lo.ToPtr("hello.txt"),
					Body:   strings.NewReader("hello world"),
				})
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return error", func(ctx context.Context) {
				_, err := s3Client.DeleteBucket(ctx, &s3.DeleteBucketInput{
					Bucket: bucketName,
				})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when bucket does not exist", func() {
			It("should return error", func(ctx context.Context) {
				_, err := s3Client.DeleteBucket(ctx, &s3.DeleteBucketInput{
					Bucket: lo.ToPtr("does-not-exist"),
				})
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
