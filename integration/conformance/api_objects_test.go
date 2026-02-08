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

var (
	objectMetadata = map[string]string{
		"foo": "bar",
	}
	objectKey = lo.ToPtr("hello.txt")
)

var _ = Describe("Objects API", Label("conformance"), Label("api-objects"), Ordered, func() {
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
		When("object does not exist", func() {
			It("creats an object", func(ctx context.Context) {
				_, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
					Bucket:      bucketName,
					Key:         objectKey,
					Body:        strings.NewReader("hello world"),
					ContentType: lo.ToPtr("text/plain"),
					Metadata:    objectMetadata,
					Tagging:     lo.ToPtr("bar=baz"),
				})
				Expect(err).NotTo(HaveOccurred())
			})
		})

		When("object already exists", func() {
			It("returns an error", func(ctx context.Context) {
				_, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
					Bucket: bucketName,
					Key:    objectKey,
					Body:   strings.NewReader("hello world"),
				})
				Expect(err).To(HaveOccurred())
			})
		})

		When("bucket does not exist", func() {
			It("returns an error", func(ctx context.Context) {
				_, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
					Bucket: lo.ToPtr("does-not-exist"),
					Key:    objectKey,
				})
				Expect(err).To(HaveOccurred())
			})
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

		When("prefix is specified", func() {
			When("there are objects matching the prefix", func() {
				It("lists objects", func(ctx context.Context) {
					listObjectsOutput, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
						Bucket: bucketName,
						Prefix: lo.ToPtr("h"),
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(listObjectsOutput.Contents).To(HaveLen(1))
					Expect(listObjectsOutput.Contents[0].Key).To(Equal(lo.ToPtr("hello.txt")))
				})
			})

			When("there are no objects matching the prefix", func() {
				It("returnsan empty list", func(ctx context.Context) {
					listObjectsOutput, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
						Bucket: bucketName,
						Prefix: lo.ToPtr("does-not-exist"),
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(listObjectsOutput.Contents).To(BeEmpty())
				})
			})

			When("listing non-existent bucket", func() {
				It("returns an error", func(ctx context.Context) {
					_, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
						Bucket: lo.ToPtr("does-not-exist"),
					})
					Expect(err).To(HaveOccurred())
				})
			})
		})

		When("prefix is not specified", func() {
			It("lists all objects", func(ctx context.Context) {
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
		When("object exists", func() {
			It("returns object metadata", func(ctx context.Context) {
				content := "hello world"
				headObjectOutput, err := s3Client.HeadObject(ctx, &s3.HeadObjectInput{
					Bucket: bucketName,
					Key:    objectKey,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(headObjectOutput.ContentLength).NotTo(BeNil())
				Expect(*headObjectOutput.ContentLength).To(Equal(int64(len(content))))
				Expect(*headObjectOutput.ContentType).To(Equal("text/plain"))
				Expect(headObjectOutput.Metadata).To(Equal(objectMetadata))
				Expect(*headObjectOutput.TagCount).To(Equal(int32(1)))
			})
		})

		When("object does not exist", func() {
			It("returnserror", func(ctx context.Context) {
				_, err := s3Client.HeadObject(ctx, &s3.HeadObjectInput{
					Bucket: bucketName,
					Key:    lo.ToPtr("does-not-exist.txt"),
				})
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("GetObjectTagging", func() {
		When("object exists", func() {
			It("returns object tagging", func(ctx context.Context) {
				output, err := s3Client.GetObjectTagging(ctx, &s3.GetObjectTaggingInput{
					Bucket: bucketName,
					Key:    objectKey,
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(output.TagSet).To(HaveLen(1))
			})
		})

		When("object does not exist", func() {
			It("returnserror", func(ctx context.Context) {
				_, err := s3Client.GetObjectTagging(ctx, &s3.GetObjectTaggingInput{
					Bucket: bucketName,
					Key:    lo.ToPtr("does-not-exist.txt"),
				})
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("GetObject", func() {
		When("object exists", func() {
			It("returns object and verifies content matches", func(ctx context.Context) {
				content := "hello world"
				getObjectOutput, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
					Bucket: bucketName,
					Key:    objectKey,
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

		When("fetching a range of the object", func() {
			It("returns the range of the object", func(ctx context.Context) {
				getObjectOutput, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
					Bucket: bucketName,
					Key:    objectKey,
					Range:  lo.ToPtr("bytes=1-5"),
				})
				Expect(err).NotTo(HaveOccurred())

				defer getObjectOutput.Body.Close()

				bodyBytes, err := io.ReadAll(getObjectOutput.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(bodyBytes)).To(Equal("ello "))
			})
		})

		When("object does not exist", func() {
			It("returnserror", func(ctx context.Context) {
				_, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
					Bucket: bucketName,
					Key:    lo.ToPtr("does-not-exist.txt"),
				})
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("DeleteObject", func() {
		When("object exists", func() {
			It("deletes object", func(ctx context.Context) {
				_, err := s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
					Bucket: bucketName,
					Key:    objectKey,
				})
				Expect(err).NotTo(HaveOccurred())

				_, err = s3Client.HeadObject(ctx, &s3.HeadObjectInput{
					Bucket: bucketName,
					Key:    objectKey,
				})
				Expect(err).To(HaveOccurred())
			})
		})

		When("object does not exist", func() {
			It("returnserror", func(ctx context.Context) {
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

		When("objects exist", func() {
			It("deletes objects", func(ctx context.Context) {
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
			It("creates multipart upload", func(ctx context.Context) {
				createMultipartUploadOutput, err := s3Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
					Bucket: bucketName,
					Key:    objectKey,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(createMultipartUploadOutput.UploadId).NotTo(BeNil())
				uploadID = createMultipartUploadOutput.UploadId
			})
		})

		Describe("UploadPart", func() {
			for i := 1; i <= 10; i++ {
				It(fmt.Sprintf("uploads part %d", i), func(ctx context.Context) {
					_, err := s3Client.UploadPart(ctx, &s3.UploadPartInput{
						Bucket:     bucketName,
						Key:        objectKey,
						PartNumber: lo.ToPtr(int32(i)),
						UploadId:   uploadID,
						Body:       strings.NewReader(fmt.Sprintf("hello %d\n", i)),
					})
					Expect(err).NotTo(HaveOccurred())
				})
			}
		})

		Describe("CompleteMultipartUpload", func() {
			It("completes multipart upload", func(ctx context.Context) {
				_, err := s3Client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
					Bucket:   bucketName,
					Key:      objectKey,
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
			It("returns object", func(ctx context.Context) {
				getObjectOutput, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
					Bucket: bucketName,
					Key:    objectKey,
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
			It("creates multipart upload", func(ctx context.Context) {
				createMultipartUploadOutput, err := s3Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
					Bucket: bucketName,
					Key:    objectKey,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(createMultipartUploadOutput.UploadId).NotTo(BeNil())
				uploadID = createMultipartUploadOutput.UploadId
			})
		})

		Describe("AbortMultipartUpload", func() {
			It("aborts multipart upload", func(ctx context.Context) {
				_, err := s3Client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
					Bucket:   bucketName,
					Key:      objectKey,
					UploadId: uploadID,
				})
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
