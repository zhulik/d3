package conformance_test

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/minio/minio-go/v7"
	"github.com/samber/lo"
	"github.com/zhulik/d3/integration/testhelpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Buckets API", Label("conformance"), Label("api-buckets"), Ordered, func() {
	var (
		app         *testhelpers.App
		s3Client    *s3.Client
		minioClient *minio.Client
		bucketName  string
	)

	BeforeAll(func(ctx context.Context) {
		app = testhelpers.NewApp() //nolint:contextcheck
		s3Client = app.S3Client(ctx, "admin")
		minioClient = app.MinioClient(ctx, "admin")
		bucketName = app.BucketName()
	})

	AfterAll(func(ctx context.Context) {
		app.Stop(ctx)
	})

	Describe("CreateBucket", func() {
		When("bucket already exists", func() {
			It("returns error", func(ctx context.Context) {
				_, err := s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
					Bucket: &bucketName,
				})
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("ListBuckets", func() {
		It("resturns a list of buckets", func(ctx context.Context) {
			listBucketsOutput, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
			Expect(err).NotTo(HaveOccurred())

			found := lo.ContainsBy(listBucketsOutput.Buckets, func(bucket types.Bucket) bool {
				return *bucket.Name == bucketName
			})

			Expect(found).To(BeTrue())
		})
	})

	Describe("HeadBuckets", func() {
		It("resturns a bucket using AWS SDK", func(ctx context.Context) {
			headBucketOutput, err := s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
				Bucket: &bucketName,
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(*headBucketOutput.BucketRegion).To(Equal("local"))
		})

		XIt("resturns a bucket using Minio SDK", func(ctx context.Context) {
			headBucketOutput, err := minioClient.BucketExists(ctx, bucketName)
			Expect(err).NotTo(HaveOccurred())
			Expect(headBucketOutput).To(BeTrue())
		})
	})

	Describe("DeleteBucket", func() {
		When("bucket is not empty", func() {
			BeforeAll(func(ctx context.Context) {
				_, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
					Bucket: &bucketName,
					Key:    lo.ToPtr("hello.txt"),
					Body:   strings.NewReader("hello world"),
				})
				Expect(err).NotTo(HaveOccurred())
			})

			It("returnserror", func(ctx context.Context) {
				_, err := s3Client.DeleteBucket(ctx, &s3.DeleteBucketInput{
					Bucket: &bucketName,
				})
				Expect(err).To(HaveOccurred())
			})
		})

		When("bucket does not exist", func() {
			It("returnserror", func(ctx context.Context) {
				_, err := s3Client.DeleteBucket(ctx, &s3.DeleteBucketInput{
					Bucket: lo.ToPtr("does-not-exist"),
				})
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
