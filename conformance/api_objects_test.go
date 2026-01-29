package conformance_test

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/samber/lo"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var objectMetadata = map[string]string{
	"foo": "bar",
}

var _ = Describe("Core conformance", Label("conformance"), Ordered, func() {
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

	Describe("PutObject", func() {
		It("should put object", func(ctx context.Context) {
			content := "hello world"
			_, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
				Bucket:      bucketName,
				Key:         lo.ToPtr("hello.txt"),
				Body:        strings.NewReader(content),
				ContentType: lo.ToPtr("text/plain"),
				Metadata:    objectMetadata,
				Tagging:     lo.ToPtr("bar=baz"),
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ListObjectsV2", func() {
		BeforeAll(func(ctx context.Context) {
			lo.Must(s3Client.PutObject(ctx, &s3.PutObjectInput{
				Bucket: bucketName,
				Key:    lo.ToPtr("dir/hello.txt"),
				Body:   strings.NewReader("hello world"),
			}))
		})

		Context("when prefix is specified", func() {
			Context("when there are objects matching the prefix", func() {
				It("should list objects", func(ctx context.Context) {
					listObjectsOutput, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
						Bucket: bucketName,
						Prefix: lo.ToPtr("h"),
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(listObjectsOutput.Contents).To(HaveLen(1))
					Expect(listObjectsOutput.Contents[0].Key).To(Equal(lo.ToPtr("hello.txt")))
				})
			})

			Context("when there are no objects matching the prefix", func() {
				It("should return an empty list", func(ctx context.Context) {
					listObjectsOutput, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
						Bucket: bucketName,
						Prefix: lo.ToPtr("does-not-exist"),
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(listObjectsOutput.Contents).To(BeEmpty())
				})
			})

			Context("when listing non-existent bucket", func() {
				It("should return error", func(ctx context.Context) {
					_, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
						Bucket: lo.ToPtr("does-not-exist"),
					})
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("when prefix is not specified", func() {
			It("should list objects", func(ctx context.Context) {
				listObjectsOutput, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
					Bucket: bucketName,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(listObjectsOutput.Contents).To(HaveLen(2))
				Expect(listObjectsOutput.Contents[0].Key).To(Equal(lo.ToPtr("dir/hello.txt")))
				Expect(listObjectsOutput.Contents[1].Key).To(Equal(lo.ToPtr("hello.txt")))
			})
		})
	})

	Describe("HeadObject", func() {
		Context("when object exists", func() {
			It("should head object", func(ctx context.Context) {
				content := "hello world"
				headObjectOutput, err := s3Client.HeadObject(ctx, &s3.HeadObjectInput{
					Bucket: bucketName,
					Key:    lo.ToPtr("hello.txt"),
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(headObjectOutput.ContentLength).NotTo(BeNil())
				Expect(*headObjectOutput.ContentLength).To(Equal(int64(len(content))))
				Expect(*headObjectOutput.ContentType).To(Equal("text/plain"))
				Expect(headObjectOutput.Metadata).To(Equal(objectMetadata))
				Expect(*headObjectOutput.TagCount).To(Equal(int32(1)))
			})
		})

		Context("when object does not exist", func() {
			It("should return error", func(ctx context.Context) {
				_, err := s3Client.HeadObject(ctx, &s3.HeadObjectInput{
					Bucket: bucketName,
					Key:    lo.ToPtr("does-not-exist.txt"),
				})
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("GetObjectTagging", func() {
		Context("when object exists", func() {
			It("should get object tagging", func(ctx context.Context) {
				output, err := s3Client.GetObjectTagging(ctx, &s3.GetObjectTaggingInput{
					Bucket: bucketName,
					Key:    lo.ToPtr("hello.txt"),
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(output.TagSet).To(HaveLen(1))
			})
		})

		Context("when object does not exist", func() {
			It("should return error", func(ctx context.Context) {
				_, err := s3Client.GetObjectTagging(ctx, &s3.GetObjectTaggingInput{
					Bucket: bucketName,
					Key:    lo.ToPtr("does-not-exist.txt"),
				})
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("GetObject", func() {
		Context("when object exists", func() {
			It("should get object and verify content matches", func(ctx context.Context) {
				content := "hello world"
				getObjectOutput, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
					Bucket: bucketName,
					Key:    lo.ToPtr("hello.txt"),
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

		Context("when object does not exist", func() {
			It("should return error", func(ctx context.Context) {
				_, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
					Bucket: bucketName,
					Key:    lo.ToPtr("does-not-exist.txt"),
				})
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("DeleteObject", func() {
		Context("when object exists", func() {
			It("should delete object", func(ctx context.Context) {
				_, err := s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
					Bucket: bucketName,
					Key:    lo.ToPtr("hello.txt"),
				})
				Expect(err).NotTo(HaveOccurred())

				_, err = s3Client.HeadObject(ctx, &s3.HeadObjectInput{
					Bucket: bucketName,
					Key:    lo.ToPtr("hello.txt"),
				})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when object does not exist", func() {
			It("should return error", func(ctx context.Context) {
				_, err := s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
					Bucket: bucketName,
					Key:    lo.ToPtr("does-not-exist.txt"),
				})
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("DeleteObjects", func() {
		var keys = []string{"hello.txt", "hello2.txt"}

		BeforeEach(func(ctx context.Context) {
			lo.ForEach(keys, func(key string, _ int) {
				lo.Must(s3Client.PutObject(ctx, &s3.PutObjectInput{
					Bucket: bucketName,
					Key:    lo.ToPtr(key),
					Body:   strings.NewReader("hello world"),
				}))
			})
		})

		Context("when objects exist", func() {
			It("should delete objects", func(ctx context.Context) {
				_, err := s3Client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
					Bucket: bucketName,
					Delete: &types.Delete{
						Objects: lo.Map(keys, func(key string, _ int) types.ObjectIdentifier {
							return types.ObjectIdentifier{Key: lo.ToPtr(key)}
						}),
					},
				})
				Expect(err).NotTo(HaveOccurred())

				for _, key := range keys {
					_, err := s3Client.HeadObject(ctx, &s3.HeadObjectInput{
						Bucket: bucketName,
						Key:    lo.ToPtr(key),
					})
					Expect(err).To(HaveOccurred())
				}
			})
		})
	})

	Describe("Multiplart Upload full cycle", func() {
		var uploadID *string

		Describe("CreateMultipartUpload", func() {
			It("should create multipart upload", func(ctx context.Context) {
				createMultipartUploadOutput, err := s3Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
					Bucket: bucketName,
					Key:    lo.ToPtr("hello.txt"),
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(createMultipartUploadOutput.UploadId).NotTo(BeNil())
				uploadID = createMultipartUploadOutput.UploadId
			})
		})

		Describe("UploadPart", func() {
			for i := 1; i <= 10; i++ {
				It(fmt.Sprintf("should upload part %d", i), func(ctx context.Context) {
					_, err := s3Client.UploadPart(ctx, &s3.UploadPartInput{
						Bucket:     bucketName,
						Key:        lo.ToPtr("hello.txt"),
						PartNumber: lo.ToPtr(int32(i)),
						UploadId:   uploadID,
						Body:       strings.NewReader(fmt.Sprintf("hello %d\n", i)),
					})
					Expect(err).NotTo(HaveOccurred())
				})
			}
		})

		Describe("CompleteMultipartUpload", func() {
			It("should complete multipart upload", func(ctx context.Context) {
				_, err := s3Client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
					Bucket:   bucketName,
					Key:      lo.ToPtr("hello.txt"),
					UploadId: uploadID,
					MultipartUpload: &types.CompletedMultipartUpload{
						Parts: []types.CompletedPart{
							{PartNumber: lo.ToPtr(int32(1))},
							{PartNumber: lo.ToPtr(int32(2))},
							{PartNumber: lo.ToPtr(int32(3))},
							{PartNumber: lo.ToPtr(int32(4))},
							{PartNumber: lo.ToPtr(int32(5))},
							{PartNumber: lo.ToPtr(int32(6))},
							{PartNumber: lo.ToPtr(int32(7))},
							{PartNumber: lo.ToPtr(int32(8))},
							{PartNumber: lo.ToPtr(int32(9))},
							{PartNumber: lo.ToPtr(int32(10))},
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())

			})
		})

		Describe("GetObject", func() {
			It("should get object", func(ctx context.Context) {
				getObjectOutput, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
					Bucket: bucketName,
					Key:    lo.ToPtr("hello.txt"),
				})
				Expect(err).NotTo(HaveOccurred())

				defer getObjectOutput.Body.Close()

				bodyBytes, err := io.ReadAll(getObjectOutput.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(bodyBytes)).To(Equal("hello 1\nhello 2\nhello 3\nhello 4\nhello 5\nhello 6\nhello 7\nhello 8\nhello 9\nhello 10\n"))
			})
		})
	})

	Describe("Multiplart Upload abort", func() {
		var uploadID *string

		Describe("CreateMultipartUpload", func() {
			It("should create multipart upload", func(ctx context.Context) {
				createMultipartUploadOutput, err := s3Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
					Bucket: bucketName,
					Key:    lo.ToPtr("hello.txt"),
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(createMultipartUploadOutput.UploadId).NotTo(BeNil())
				uploadID = createMultipartUploadOutput.UploadId
			})
		})

		Describe("AbortMultipartUpload", func() {
			It("should abort multipart upload", func(ctx context.Context) {
				_, err := s3Client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
					Bucket:   bucketName,
					Key:      lo.ToPtr("hello.txt"),
					UploadId: uploadID,
				})
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
