package d3

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/samber/lo"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	endpoint = "http://localhost:8080"
)

var objectMetadata = map[string]string{
	"foo": "bar",
}

var _ = Describe("Core conformance", Label("conformance"), Ordered, func() {
	var s3Client *s3.Client
	var bucketName string

	BeforeAll(func(ctx context.Context) {
		bucketName = fmt.Sprintf("conformance-bucket-%d", time.Now().Unix())

		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithRegion("local"),
			config.WithBaseEndpoint(endpoint),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "test")),
		)
		Expect(err).NotTo(HaveOccurred())

		s3Client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.UsePathStyle = true
		})

		_, err = s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(bucketName),
		})
		Expect(err).NotTo(HaveOccurred())
	})

	AfterAll(func(ctx context.Context) {
		_, err := s3Client.DeleteBucket(ctx, &s3.DeleteBucketInput{
			Bucket: aws.String(bucketName),
		})
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("ListBuckets", func() {
		It("should list buckets and include our bucket", func(ctx context.Context) {
			listBucketsOutput, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
			Expect(err).NotTo(HaveOccurred())

			found := lo.ContainsBy(listBucketsOutput.Buckets, func(bucket types.Bucket) bool {
				return *bucket.Name == bucketName
			})

			Expect(found).To(BeTrue(), "bucket should be in list")
		})
	})

	Describe("PutObject", func() {
		It("should put object", func(ctx context.Context) {
			content := "hello world"
			_, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
				Bucket:      aws.String(bucketName),
				Key:         aws.String("hello.txt"),
				Body:        strings.NewReader(content),
				ContentType: aws.String("text/plain"),
				Metadata:    objectMetadata,
				Tagging:     aws.String("bar=baz"),
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("HeadObject", func() {
		It("should head object", func(ctx context.Context) {
			content := "hello world"
			headObjectOutput, err := s3Client.HeadObject(ctx, &s3.HeadObjectInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String("hello.txt"),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(headObjectOutput.ContentLength).NotTo(BeNil())
			Expect(*headObjectOutput.ContentLength).To(Equal(int64(len(content))))
			Expect(*headObjectOutput.ContentType).To(Equal("text/plain"))
			Expect(headObjectOutput.Metadata).To(Equal(objectMetadata))
			Expect(*headObjectOutput.TagCount).To(Equal(int32(1)))
		})
	})

	Describe("GetObjectTagging", func() {
		It("should get object and verify content matches", func(ctx context.Context) {
			output, err := s3Client.GetObjectTagging(ctx, &s3.GetObjectTaggingInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String("hello.txt"),
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(output.TagSet).To(HaveLen(1))
		})
	})

	Describe("GetObject", func() {
		It("should get object and verify content matches", func(ctx context.Context) {
			content := "hello world"
			getObjectOutput, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String("hello.txt"),
			})
			Expect(err).NotTo(HaveOccurred())

			defer getObjectOutput.Body.Close()

			bodyBytes, err := io.ReadAll(getObjectOutput.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(bodyBytes)).To(Equal(content))
			Expect(*getObjectOutput.ContentType).To(Equal("text/plain"))
			Expect(getObjectOutput.Metadata).To(Equal(objectMetadata))
			Expect(*getObjectOutput.TagCount).To(Equal(int32(1)))
		})
	})

	Describe("DeleteObject", func() {
		It("should delete object", func(ctx context.Context) {
			_, err := s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String("hello.txt"),
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
