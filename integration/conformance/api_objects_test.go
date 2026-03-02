package conformance_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/minio/minio-go/v7"
	"github.com/zhulik/d3/integration/testhelpers"

	"github.com/samber/lo"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	objectMetadata = map[string]string{
		"foo": "bar",
	}
	objectKeyAWS   = lo.ToPtr("hello.txt")
	objectKeyMinio = "hello-minio.txt"
	objectData     = "hello world"
)

var _ = Describe("Objects API", Label("conformance"), Label("api-objects"), Ordered, func() {
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

	Describe("PutObject", func() {
		When("object does not exist", func() {
			It("creates an object with AWS SDK", func(ctx context.Context) {
				_, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
					Bucket:      &bucketName,
					Key:         objectKeyAWS,
					Body:        strings.NewReader(objectData),
					ContentType: lo.ToPtr("text/plain"),
					Metadata:    objectMetadata,
					Tagging:     lo.ToPtr("bar=baz"),
				})
				Expect(err).NotTo(HaveOccurred())
			})

			It("creates an object with Minio SDK", func(ctx context.Context) {
				_, err := minioClient.PutObject(ctx, bucketName, objectKeyMinio, strings.NewReader(objectData), int64(len(objectData)), minio.PutObjectOptions{
					ContentType:  "text/plain",
					UserMetadata: objectMetadata,
					UserTags:     map[string]string{"bar": "baz"},
				})
				Expect(err).NotTo(HaveOccurred())
			})
		})

		When("object already exists", func() {
			overwriteKey := "overwrite-test.txt"

			AfterAll(func(ctx context.Context) {
				lo.Must(s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
					Bucket: &bucketName,
					Key:    &overwriteKey,
				}))
			})

			It("overwrites the object", func(ctx context.Context) {
				_, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
					Bucket: &bucketName,
					Key:    &overwriteKey,
					Body:   strings.NewReader("original data"),
				})
				Expect(err).NotTo(HaveOccurred())

				newData := "overwritten data"
				_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
					Bucket: &bucketName,
					Key:    &overwriteKey,
					Body:   strings.NewReader(newData),
				})
				Expect(err).NotTo(HaveOccurred())

				result, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
					Bucket: &bucketName,
					Key:    &overwriteKey,
				})
				Expect(err).NotTo(HaveOccurred())

				defer result.Body.Close()

				body, err := io.ReadAll(result.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(Equal(newData))
			})
		})

		When("If-None-Match is *", func() {
			ifNoneMatchKey := "if-none-match-new.txt"

			AfterAll(func(ctx context.Context) {
				lo.Must(s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
					Bucket: &bucketName,
					Key:    &ifNoneMatchKey,
				}))
			})

			It("creates a new object", func(ctx context.Context) {
				_, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
					Bucket:      &bucketName,
					Key:         &ifNoneMatchKey,
					Body:        strings.NewReader(objectData),
					IfNoneMatch: lo.ToPtr("*"),
				})
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns precondition failed when object exists", func(ctx context.Context) {
				_, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
					Bucket:      &bucketName,
					Key:         objectKeyAWS,
					Body:        strings.NewReader(objectData),
					IfNoneMatch: lo.ToPtr("*"),
				})
				Expect(err).To(HaveOccurred())

				var httpErr interface{ HTTPStatusCode() int }
				Expect(errors.As(err, &httpErr)).To(BeTrue())
				Expect(httpErr.HTTPStatusCode()).To(Equal(412))
			})
		})

		When("bucket does not exist", func() {
			It("returns an error", func(ctx context.Context) {
				_, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
					Bucket: lo.ToPtr("does-not-exist"),
					Key:    objectKeyAWS,
				})
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("ListObjectsV2", func() {
		BeforeAll(func(ctx context.Context) {
			lo.Must(s3Client.PutObject(ctx, &s3.PutObjectInput{
				Bucket: &bucketName,
				Key:    lo.ToPtr("dir/hello.txt"),
				Body:   strings.NewReader(objectData),
			}))
		})

		When("prefix is specified", func() {
			When("there are objects matching the prefix", func() {
				It("lists objects", func(ctx context.Context) {
					listObjectsOutput, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
						Bucket: &bucketName,
						Prefix: lo.ToPtr("h"),
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(listObjectsOutput.Contents).To(HaveLen(2))
					Expect(listObjectsOutput.Contents[0].Key).To(Equal(lo.ToPtr("hello-minio.txt")))
					Expect(listObjectsOutput.Contents[1].Key).To(Equal(lo.ToPtr("hello.txt")))
				})
			})

			When("there are no objects matching the prefix", func() {
				It("returnsan empty list", func(ctx context.Context) {
					listObjectsOutput, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
						Bucket: &bucketName,
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
			It("lists all objects with AWS SDK", func(ctx context.Context) {
				listObjectsOutput, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
					Bucket: &bucketName,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(listObjectsOutput.Contents).To(HaveLen(3))
				Expect(listObjectsOutput.Contents[0].Key).To(Equal(lo.ToPtr("dir/hello.txt")))
				Expect(listObjectsOutput.Contents[1].Key).To(Equal(lo.ToPtr("hello-minio.txt")))
				Expect(listObjectsOutput.Contents[2].Key).To(Equal(lo.ToPtr("hello.txt")))
			})

			It("lists all objects with Minio SDK", func(ctx context.Context) {
				objectsChan := minioClient.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Recursive: true})
				objects := lo.ChannelToSlice(objectsChan)

				Expect(objects).To(HaveLen(3))
				Expect(objects[0].Key).To(Equal("dir/hello.txt"))
				Expect(objects[1].Key).To(Equal("hello-minio.txt"))
				Expect(objects[2].Key).To(Equal("hello.txt"))
			})
		})

		Context("pagination", func() {
			const paginatePrefix = "paginate/"

			paginateKeys := []string{"paginate/a", "paginate/b", "paginate/c", "paginate/d", "paginate/e"}

			BeforeAll(func(ctx context.Context) {
				lo.ForEach(paginateKeys, func(key string, _ int) {
					lo.Must(s3Client.PutObject(ctx, &s3.PutObjectInput{
						Bucket: &bucketName,
						Key:    lo.ToPtr(key),
						Body:   strings.NewReader("data"),
					}))
				})
			})

			When("max-keys is specified", func() {
				It("returns requested number of objects", func(ctx context.Context) {
					output, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
						Bucket:  &bucketName,
						Prefix:  lo.ToPtr(paginatePrefix),
						MaxKeys: lo.ToPtr(int32(2)),
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(output.Contents).To(HaveLen(2))
					Expect(*output.IsTruncated).To(BeTrue())
					Expect(output.Contents[0].Key).To(Equal(lo.ToPtr("paginate/a")))
					Expect(output.Contents[1].Key).To(Equal(lo.ToPtr("paginate/b")))
				})
			})

			When("continuation-token is provided", func() {
				It("returns next page of results", func(ctx context.Context) {
					firstPage, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
						Bucket:  &bucketName,
						Prefix:  lo.ToPtr(paginatePrefix),
						MaxKeys: lo.ToPtr(int32(2)),
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(firstPage.NextContinuationToken).NotTo(BeNil())

					secondPage, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
						Bucket:            &bucketName,
						Prefix:            lo.ToPtr(paginatePrefix),
						MaxKeys:           lo.ToPtr(int32(2)),
						ContinuationToken: firstPage.NextContinuationToken,
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(secondPage.Contents).To(HaveLen(2))
					Expect(secondPage.Contents[0].Key).To(Equal(lo.ToPtr("paginate/c")))
					Expect(secondPage.Contents[1].Key).To(Equal(lo.ToPtr("paginate/d")))
				})
			})

			When("iterating through all pages", func() {
				It("returns all objects in order", func(ctx context.Context) {
					var (
						allKeys           []string
						continuationToken *string
					)

					for {
						output, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
							Bucket:            &bucketName,
							Prefix:            lo.ToPtr(paginatePrefix),
							MaxKeys:           lo.ToPtr(int32(2)),
							ContinuationToken: continuationToken,
						})
						Expect(err).NotTo(HaveOccurred())

						for _, obj := range output.Contents {
							allKeys = append(allKeys, *obj.Key)
						}

						if output.IsTruncated == nil || !*output.IsTruncated {
							break
						}

						continuationToken = output.NextContinuationToken
					}

					Expect(allKeys).To(Equal(paginateKeys))
				})
			})
		})

		Context("delimiter", func() {
			delimKeys := []string{
				"delim/sub/another.txt",
				"delim/sub/nested.txt",
				"delim/sub2/deep/file.txt",
				"delim/top.txt",
			}

			BeforeAll(func(ctx context.Context) {
				lo.ForEach(delimKeys, func(key string, _ int) {
					lo.Must(s3Client.PutObject(ctx, &s3.PutObjectInput{
						Bucket: &bucketName,
						Key:    lo.ToPtr(key),
						Body:   strings.NewReader("data"),
					}))
				})
			})

			When("delimiter is / at root level", func() {
				It("returns root objects in Contents and directory prefixes in CommonPrefixes", func(ctx context.Context) {
					output, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
						Bucket:    &bucketName,
						Delimiter: lo.ToPtr("/"),
					})
					Expect(err).NotTo(HaveOccurred())

					keys := lo.Map(output.Contents, func(o types.Object, _ int) string { return *o.Key })
					Expect(keys).To(ContainElements("hello-minio.txt", "hello.txt"))

					prefixes := lo.Map(output.CommonPrefixes, func(p types.CommonPrefix, _ int) string { return *p.Prefix })
					Expect(prefixes).To(ContainElements("delim/", "dir/", "paginate/"))

					Expect(*output.KeyCount).To(Equal(int32(len(output.Contents) + len(output.CommonPrefixes))))
				})
			})

			When("delimiter is / with prefix", func() {
				It("returns direct objects and sub-directory prefixes under the prefix", func(ctx context.Context) {
					output, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
						Bucket:    &bucketName,
						Prefix:    lo.ToPtr("delim/"),
						Delimiter: lo.ToPtr("/"),
					})
					Expect(err).NotTo(HaveOccurred())

					keys := lo.Map(output.Contents, func(o types.Object, _ int) string { return *o.Key })
					Expect(keys).To(Equal([]string{"delim/top.txt"}))

					prefixes := lo.Map(output.CommonPrefixes, func(p types.CommonPrefix, _ int) string { return *p.Prefix })
					Expect(prefixes).To(Equal([]string{"delim/sub/", "delim/sub2/"}))

					Expect(*output.KeyCount).To(Equal(int32(3)))
				})
			})

			When("delimiter is / with nested prefix", func() {
				It("returns objects directly under nested prefix with no further common prefixes", func(ctx context.Context) {
					output, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
						Bucket:    &bucketName,
						Prefix:    lo.ToPtr("delim/sub/"),
						Delimiter: lo.ToPtr("/"),
					})
					Expect(err).NotTo(HaveOccurred())

					keys := lo.Map(output.Contents, func(o types.Object, _ int) string { return *o.Key })
					Expect(keys).To(Equal([]string{"delim/sub/another.txt", "delim/sub/nested.txt"}))
					Expect(output.CommonPrefixes).To(BeEmpty())
				})
			})

			When("delimiter is / with prefix that has nested subdirectories", func() {
				It("returns the intermediate common prefix", func(ctx context.Context) {
					output, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
						Bucket:    &bucketName,
						Prefix:    lo.ToPtr("delim/sub2/"),
						Delimiter: lo.ToPtr("/"),
					})
					Expect(err).NotTo(HaveOccurred())

					Expect(output.Contents).To(BeEmpty())

					prefixes := lo.Map(output.CommonPrefixes, func(p types.CommonPrefix, _ int) string { return *p.Prefix })
					Expect(prefixes).To(Equal([]string{"delim/sub2/deep/"}))
				})
			})

			When("no delimiter is specified", func() {
				It("returns all objects flat with no common prefixes", func(ctx context.Context) {
					output, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
						Bucket: &bucketName,
						Prefix: lo.ToPtr("delim/"),
					})
					Expect(err).NotTo(HaveOccurred())

					keys := lo.Map(output.Contents, func(o types.Object, _ int) string { return *o.Key })
					Expect(keys).To(Equal([]string{
						"delim/sub/another.txt",
						"delim/sub/nested.txt",
						"delim/sub2/deep/file.txt",
						"delim/top.txt",
					}))
					Expect(output.CommonPrefixes).To(BeEmpty())
				})
			})

			When("delimiter with max-keys limits combined count", func() {
				It("counts both contents and common prefixes toward max-keys", func(ctx context.Context) {
					output, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
						Bucket:    &bucketName,
						Prefix:    lo.ToPtr("delim/"),
						Delimiter: lo.ToPtr("/"),
						MaxKeys:   lo.ToPtr(int32(2)),
					})
					Expect(err).NotTo(HaveOccurred())

					total := len(output.Contents) + len(output.CommonPrefixes)
					Expect(total).To(Equal(2))
					Expect(*output.IsTruncated).To(BeTrue())
				})
			})

			When("paginating with delimiter", func() {
				It("returns all entries across pages", func(ctx context.Context) {
					var (
						allKeys           []string
						allPrefixes       []string
						continuationToken *string
					)

					for {
						output, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
							Bucket:            &bucketName,
							Prefix:            lo.ToPtr("delim/"),
							Delimiter:         lo.ToPtr("/"),
							MaxKeys:           lo.ToPtr(int32(1)),
							ContinuationToken: continuationToken,
						})
						Expect(err).NotTo(HaveOccurred())

						for _, obj := range output.Contents {
							allKeys = append(allKeys, *obj.Key)
						}

						for _, p := range output.CommonPrefixes {
							allPrefixes = append(allPrefixes, *p.Prefix)
						}

						if output.IsTruncated == nil || !*output.IsTruncated {
							break
						}

						continuationToken = output.NextContinuationToken
					}

					Expect(allKeys).To(Equal([]string{"delim/top.txt"}))
					Expect(allPrefixes).To(Equal([]string{"delim/sub/", "delim/sub2/"}))
				})
			})
		})
	})

	Describe("HeadObject", func() {
		When("object exists", func() {
			It("returns object metadata with AWS SDK", func(ctx context.Context) {
				headObjectOutput, err := s3Client.HeadObject(ctx, &s3.HeadObjectInput{
					Bucket: &bucketName,
					Key:    objectKeyAWS,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(headObjectOutput.ContentLength).NotTo(BeNil())
				Expect(*headObjectOutput.ContentLength).To(Equal(int64(len(objectData))))
				Expect(*headObjectOutput.ContentType).To(Equal("text/plain"))
				Expect(headObjectOutput.Metadata).To(Equal(objectMetadata))
				Expect(*headObjectOutput.TagCount).To(Equal(int32(1)))
			})
		})

		When("object does not exist", func() {
			It("returnserror", func(ctx context.Context) {
				_, err := s3Client.HeadObject(ctx, &s3.HeadObjectInput{
					Bucket: &bucketName,
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
					Bucket: &bucketName,
					Key:    objectKeyAWS,
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(output.TagSet).To(HaveLen(1))
			})
		})

		When("object does not exist", func() {
			It("returnserror", func(ctx context.Context) {
				_, err := s3Client.GetObjectTagging(ctx, &s3.GetObjectTaggingInput{
					Bucket: &bucketName,
					Key:    lo.ToPtr("does-not-exist.txt"),
				})
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("GetObject", func() {
		When("object exists", func() {
			It("returns object uploaded with AWS SDK and verifies content matches", func(ctx context.Context) {
				getObjectOutput, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
					Bucket: &bucketName,
					Key:    objectKeyAWS,
				})
				Expect(err).NotTo(HaveOccurred())

				defer getObjectOutput.Body.Close()

				bodyBytes, err := io.ReadAll(getObjectOutput.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(bodyBytes)).To(Equal(objectData))
				Expect(*getObjectOutput.ContentType).To(Equal("text/plain"))
				Expect(getObjectOutput.Metadata).To(Equal(objectMetadata))
				Expect(*getObjectOutput.TagCount).To(Equal(int32(1)))
			})

			It("returns object uploaded with Minio SDK and verifies content matches", func(ctx context.Context) {
				getObjectOutput, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
					Bucket: &bucketName,
					Key:    &objectKeyMinio,
				})
				Expect(err).NotTo(HaveOccurred())

				defer getObjectOutput.Body.Close()

				bodyBytes, err := io.ReadAll(getObjectOutput.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(bodyBytes)).To(Equal(objectData))
				Expect(*getObjectOutput.ContentType).To(Equal("text/plain"))
				Expect(getObjectOutput.Metadata).To(Equal(objectMetadata))
				Expect(*getObjectOutput.TagCount).To(Equal(int32(1)))
			})
		})

		When("fetching a range of the object", func() {
			It("returns the range of the object", func(ctx context.Context) {
				getObjectOutput, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
					Bucket: &bucketName,
					Key:    objectKeyAWS,
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
					Bucket: &bucketName,
					Key:    lo.ToPtr("does-not-exist.txt"),
				})
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("CopyObject", Ordered, func() {
		copySourceKey := "copy-source.txt"
		copyDestKey := "copy-dest.txt"

		BeforeAll(func(ctx context.Context) {
			_, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
				Bucket:      &bucketName,
				Key:         &copySourceKey,
				Body:        strings.NewReader(objectData),
				ContentType: lo.ToPtr("text/plain"),
				Metadata:    objectMetadata,
				Tagging:     lo.ToPtr("bar=baz"),
			})
			Expect(err).NotTo(HaveOccurred())
		})

		AfterAll(func(ctx context.Context) {
			lo.Must(s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: &bucketName,
				Key:    &copyDestKey,
			}))
		})

		When("source object exists", func() {
			It("copies the object", func(ctx context.Context) {
				_, err := s3Client.CopyObject(ctx, &s3.CopyObjectInput{
					Bucket:     &bucketName,
					CopySource: lo.ToPtr(bucketName + "/" + copySourceKey),
					Key:        &copyDestKey,
				})
				Expect(err).NotTo(HaveOccurred())
			})

			It("copy survives source deletion", func(ctx context.Context) {
				_, err := s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
					Bucket: &bucketName,
					Key:    &copySourceKey,
				})
				Expect(err).NotTo(HaveOccurred())

				result, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
					Bucket: &bucketName,
					Key:    &copyDestKey,
				})
				Expect(err).NotTo(HaveOccurred())

				defer result.Body.Close()

				body, err := io.ReadAll(result.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(Equal(objectData))
				Expect(*result.ContentType).To(Equal("text/plain"))
			})
		})

		When("copying object onto itself", func() {
			It("updates metadata without losing the object", func(ctx context.Context) {
				_, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
					Bucket:      &bucketName,
					Key:         &copySourceKey,
					Body:        strings.NewReader(objectData),
					ContentType: lo.ToPtr("text/plain"),
					Metadata:    objectMetadata,
				})
				Expect(err).NotTo(HaveOccurred())

				_, err = s3Client.CopyObject(ctx, &s3.CopyObjectInput{
					Bucket:            &bucketName,
					CopySource:        lo.ToPtr(bucketName + "/" + copySourceKey),
					Key:               &copySourceKey,
					MetadataDirective: types.MetadataDirectiveReplace,
					Metadata: map[string]string{
						"foo": "baz",
					},
				})
				Expect(err).NotTo(HaveOccurred())

				head, err := s3Client.HeadObject(ctx, &s3.HeadObjectInput{
					Bucket: &bucketName,
					Key:    &copySourceKey,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(head.Metadata).To(HaveKeyWithValue("foo", "baz"))
			})
		})
	})

	Describe("DeleteObject", func() {
		When("object exists", func() {
			It("deletes object", func(ctx context.Context) {
				_, err := s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
					Bucket: &bucketName,
					Key:    objectKeyAWS,
				})
				Expect(err).NotTo(HaveOccurred())

				_, err = s3Client.HeadObject(ctx, &s3.HeadObjectInput{
					Bucket: &bucketName,
					Key:    objectKeyAWS,
				})
				Expect(err).To(HaveOccurred())
			})
		})

		When("object does not exist", func() {
			It("returnserror", func(ctx context.Context) {
				_, err := s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
					Bucket: &bucketName,
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
					Bucket: &bucketName,
					Key:    lo.ToPtr(key),
					Body:   strings.NewReader(objectData),
				}))
			})
		})

		When("objects exist", func() {
			It("deletes objects", func(ctx context.Context) {
				_, err := s3Client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
					Bucket: &bucketName,
					Delete: &types.Delete{
						Objects: lo.Map(keys, func(key string, _ int) types.ObjectIdentifier {
							return types.ObjectIdentifier{Key: lo.ToPtr(key)}
						}),
					},
				})
				Expect(err).NotTo(HaveOccurred())

				for _, key := range keys {
					_, err := s3Client.HeadObject(ctx, &s3.HeadObjectInput{
						Bucket: &bucketName,
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
					Bucket: &bucketName,
					Key:    objectKeyAWS,
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
						Bucket:     &bucketName,
						Key:        objectKeyAWS,
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
					Bucket:   &bucketName,
					Key:      objectKeyAWS,
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
					Bucket: &bucketName,
					Key:    objectKeyAWS,
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
					Bucket: &bucketName,
					Key:    objectKeyAWS,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(createMultipartUploadOutput.UploadId).NotTo(BeNil())
				uploadID = createMultipartUploadOutput.UploadId
			})
		})

		Describe("AbortMultipartUpload", func() {
			It("aborts multipart upload", func(ctx context.Context) {
				_, err := s3Client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
					Bucket:   &bucketName,
					Key:      objectKeyAWS,
					UploadId: uploadID,
				})
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
