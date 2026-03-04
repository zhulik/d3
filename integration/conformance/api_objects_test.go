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

		When("more than 10 tags are provided", func() {
			It("returns 400 InvalidTag", func(ctx context.Context) {
				tagPairs := make([]string, 11)
				for i := range 11 {
					tagPairs[i] = fmt.Sprintf("key%d=val%d", i, i)
				}

				_, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
					Bucket:  &bucketName,
					Key:     lo.ToPtr("too-many-tags.txt"),
					Body:    strings.NewReader("x"),
					Tagging: lo.ToPtr(strings.Join(tagPairs, "&")),
				})
				Expect(err).To(HaveOccurred())
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

	Describe("PutObjectTagging", func() {
		putTaggingKey := "put-tagging-target.txt"

		BeforeAll(func(ctx context.Context) {
			_, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
				Bucket: &bucketName,
				Key:    &putTaggingKey,
				Body:   strings.NewReader("content"),
			})
			Expect(err).NotTo(HaveOccurred())
		})

		When("object exists", func() {
			It("sets tags and GetObjectTagging returns them", func(ctx context.Context) {
				_, err := s3Client.PutObjectTagging(ctx, &s3.PutObjectTaggingInput{
					Bucket: &bucketName,
					Key:    &putTaggingKey,
					Tagging: &types.Tagging{
						TagSet: []types.Tag{
							{Key: lo.ToPtr("env"), Value: lo.ToPtr("test")},
							{Key: lo.ToPtr("team"), Value: lo.ToPtr("backend")},
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())

				output, err := s3Client.GetObjectTagging(ctx, &s3.GetObjectTaggingInput{
					Bucket: &bucketName,
					Key:    &putTaggingKey,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(output.TagSet).To(HaveLen(2))

				tagMap := make(map[string]string)
				for _, t := range output.TagSet {
					tagMap[lo.FromPtr(t.Key)] = lo.FromPtr(t.Value)
				}

				Expect(tagMap).To(HaveKeyWithValue("env", "test"))
				Expect(tagMap).To(HaveKeyWithValue("team", "backend"))
			})
		})

		When("object does not exist", func() {
			It("returns error", func(ctx context.Context) {
				_, err := s3Client.PutObjectTagging(ctx, &s3.PutObjectTaggingInput{
					Bucket: &bucketName,
					Key:    lo.ToPtr("does-not-exist.txt"),
					Tagging: &types.Tagging{
						TagSet: []types.Tag{{Key: lo.ToPtr("k"), Value: lo.ToPtr("v")}},
					},
				})
				Expect(err).To(HaveOccurred())
			})
		})

		When("more than 10 tags are provided", func() {
			It("returns 400 InvalidTag", func(ctx context.Context) {
				tagSet := make([]types.Tag, 11)
				for i := range 11 {
					tagSet[i] = types.Tag{Key: lo.ToPtr(fmt.Sprintf("k%d", i)), Value: lo.ToPtr(fmt.Sprintf("v%d", i))}
				}

				_, err := s3Client.PutObjectTagging(ctx, &s3.PutObjectTaggingInput{
					Bucket:  &bucketName,
					Key:     &putTaggingKey,
					Tagging: &types.Tagging{TagSet: tagSet},
				})
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("DeleteObjectTagging", func() {
		deleteTaggingKey := "delete-tagging-target.txt"

		BeforeAll(func(ctx context.Context) {
			_, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
				Bucket:  &bucketName,
				Key:     &deleteTaggingKey,
				Body:    strings.NewReader("content"),
				Tagging: lo.ToPtr("a=b"),
			})
			Expect(err).NotTo(HaveOccurred())
		})

		When("object exists with tags", func() {
			It("removes all tags", func(ctx context.Context) {
				_, err := s3Client.DeleteObjectTagging(ctx, &s3.DeleteObjectTaggingInput{
					Bucket: &bucketName,
					Key:    &deleteTaggingKey,
				})
				Expect(err).NotTo(HaveOccurred())

				output, err := s3Client.GetObjectTagging(ctx, &s3.GetObjectTaggingInput{
					Bucket: &bucketName,
					Key:    &deleteTaggingKey,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(output.TagSet).To(BeEmpty())
			})
		})

		When("object does not exist", func() {
			It("returns error", func(ctx context.Context) {
				_, err := s3Client.DeleteObjectTagging(ctx, &s3.DeleteObjectTaggingInput{
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
		var (
			uploadID  *string
			partETags []*string
		)

		Describe("CreateMultipartUpload", func() {
			It("creates multipart upload", func(ctx context.Context) {
				createMultipartUploadOutput, err := s3Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
					Bucket: &bucketName,
					Key:    objectKeyAWS,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(createMultipartUploadOutput.UploadId).NotTo(BeNil())
				uploadID = createMultipartUploadOutput.UploadId
				partETags = make([]*string, 11)
			})
		})

		Describe("UploadPart", func() {
			for i := 1; i <= 10; i++ {
				It(fmt.Sprintf("uploads part %d", i), func(ctx context.Context) {
					output, err := s3Client.UploadPart(ctx, &s3.UploadPartInput{
						Bucket:     &bucketName,
						Key:        objectKeyAWS,
						PartNumber: lo.ToPtr(int32(i)),
						UploadId:   uploadID,
						Body:       strings.NewReader(fmt.Sprintf("hello %d\n", i)),
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(output.ETag).NotTo(BeNil())
					Expect(*output.ETag).NotTo(BeEmpty())

					partETags[i] = output.ETag
				})
			}
		})

		Describe("CompleteMultipartUpload", func() {
			It("completes multipart upload", func(ctx context.Context) {
				Expect(partETags[1]).NotTo(BeNil())

				completeOutput, err := s3Client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
					Bucket:   &bucketName,
					Key:      objectKeyAWS,
					UploadId: uploadID,
					MultipartUpload: &types.CompletedMultipartUpload{
						Parts: []types.CompletedPart{
							{PartNumber: lo.ToPtr(int32(1)), ETag: partETags[1]},
							{PartNumber: lo.ToPtr(int32(2)), ETag: partETags[2]},
							{PartNumber: lo.ToPtr(int32(3)), ETag: partETags[3]},
							{PartNumber: lo.ToPtr(int32(4)), ETag: partETags[4]},
							{PartNumber: lo.ToPtr(int32(5)), ETag: partETags[5]},
							{PartNumber: lo.ToPtr(int32(6)), ETag: partETags[6]},
							{PartNumber: lo.ToPtr(int32(7)), ETag: partETags[7]},
							{PartNumber: lo.ToPtr(int32(8)), ETag: partETags[8]},
							{PartNumber: lo.ToPtr(int32(9)), ETag: partETags[9]},
							{PartNumber: lo.ToPtr(int32(10)), ETag: partETags[10]},
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(completeOutput.ETag).NotTo(BeNil())
				Expect(*completeOutput.ETag).NotTo(BeEmpty())

				headOutput, err := s3Client.HeadObject(ctx, &s3.HeadObjectInput{
					Bucket: &bucketName,
					Key:    objectKeyAWS,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(headOutput.ETag).To(Equal(completeOutput.ETag))
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

	Describe("ListParts", func() {
		listPartsKey := "list-parts-key.txt"

		var uploadID *string

		BeforeAll(func(ctx context.Context) {
			createOut, err := s3Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
				Bucket: &bucketName,
				Key:    &listPartsKey,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(createOut.UploadId).NotTo(BeNil())
			uploadID = createOut.UploadId

			for i := 1; i <= 5; i++ {
				_, err := s3Client.UploadPart(ctx, &s3.UploadPartInput{
					Bucket:     &bucketName,
					Key:        &listPartsKey,
					PartNumber: lo.ToPtr(int32(i)),
					UploadId:   uploadID,
					Body:       strings.NewReader(fmt.Sprintf("part %d data", i)),
				})
				Expect(err).NotTo(HaveOccurred())
			}
		})

		AfterAll(func(ctx context.Context) {
			if uploadID != nil {
				lo.Must(s3Client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
					Bucket:   &bucketName,
					Key:      &listPartsKey,
					UploadId: uploadID,
				}))
			}
		})

		When("multipart upload has parts", func() {
			It("returns all parts in order with PartNumber, ETag and Size", func(ctx context.Context) {
				output, err := s3Client.ListParts(ctx, &s3.ListPartsInput{
					Bucket:   &bucketName,
					Key:      &listPartsKey,
					UploadId: uploadID,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(output.Parts).To(HaveLen(5))

				for i, part := range output.Parts {
					Expect(part.PartNumber).NotTo(BeNil())
					Expect(*part.PartNumber).To(Equal(int32(i + 1)))
					Expect(part.ETag).NotTo(BeNil())
					Expect(*part.ETag).NotTo(BeEmpty())
					Expect(part.Size).NotTo(BeNil())
					Expect(*part.Size).To(Equal(int64(len(fmt.Sprintf("part %d data", i+1)))))
				}

				Expect(output.IsTruncated).NotTo(BeNil())
				Expect(*output.IsTruncated).To(BeFalse())
			})
		})

		When("max-parts limits the response", func() {
			It("returns first page and NextPartNumberMarker", func(ctx context.Context) {
				output, err := s3Client.ListParts(ctx, &s3.ListPartsInput{
					Bucket:   &bucketName,
					Key:      &listPartsKey,
					UploadId: uploadID,
					MaxParts: lo.ToPtr(int32(2)),
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(output.Parts).To(HaveLen(2))
				Expect(output.Parts[0].PartNumber).NotTo(BeNil())
				Expect(*output.Parts[0].PartNumber).To(Equal(int32(1)))
				Expect(output.Parts[1].PartNumber).NotTo(BeNil())
				Expect(*output.Parts[1].PartNumber).To(Equal(int32(2)))
				Expect(output.IsTruncated).NotTo(BeNil())
				Expect(*output.IsTruncated).To(BeTrue())
				Expect(output.NextPartNumberMarker).NotTo(BeNil())
				Expect(*output.NextPartNumberMarker).To(Equal("2"))
			})

			It("returns next page when part-number-marker is provided", func(ctx context.Context) {
				marker := "2"
				output, err := s3Client.ListParts(ctx, &s3.ListPartsInput{
					Bucket:           &bucketName,
					Key:              &listPartsKey,
					UploadId:         uploadID,
					MaxParts:         lo.ToPtr(int32(2)),
					PartNumberMarker: &marker,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(output.Parts).To(HaveLen(2))
				Expect(output.Parts[0].PartNumber).NotTo(BeNil())
				Expect(*output.Parts[0].PartNumber).To(Equal(int32(3)))
				Expect(output.Parts[1].PartNumber).NotTo(BeNil())
				Expect(*output.Parts[1].PartNumber).To(Equal(int32(4)))
				Expect(output.IsTruncated).NotTo(BeNil())
				Expect(*output.IsTruncated).To(BeTrue())
				Expect(output.NextPartNumberMarker).NotTo(BeNil())
				Expect(*output.NextPartNumberMarker).To(Equal("4"))
			})

			It("returns last page with remaining parts", func(ctx context.Context) {
				marker := "4"
				output, err := s3Client.ListParts(ctx, &s3.ListPartsInput{
					Bucket:           &bucketName,
					Key:              &listPartsKey,
					UploadId:         uploadID,
					MaxParts:         lo.ToPtr(int32(10)),
					PartNumberMarker: &marker,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(output.Parts).To(HaveLen(1))
				Expect(output.Parts[0].PartNumber).NotTo(BeNil())
				Expect(*output.Parts[0].PartNumber).To(Equal(int32(5)))
				Expect(output.IsTruncated).NotTo(BeNil())
				Expect(*output.IsTruncated).To(BeFalse())
			})
		})
	})

	Describe("Multipart Upload with invalid part etags", func() {
		testKey := lo.ToPtr("invalid-etag-test.txt")

		When("part etag is completely wrong", func() {
			var (
				uploadID  *string
				partETags []*string
			)

			BeforeEach(func(ctx context.Context) {
				createMultipartUploadOutput, err := s3Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
					Bucket: &bucketName,
					Key:    testKey,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(createMultipartUploadOutput.UploadId).NotTo(BeNil())
				uploadID = createMultipartUploadOutput.UploadId
				partETags = make([]*string, 4)

				for i := 1; i <= 3; i++ {
					output, err := s3Client.UploadPart(ctx, &s3.UploadPartInput{
						Bucket:     &bucketName,
						Key:        testKey,
						PartNumber: lo.ToPtr(int32(i)),
						UploadId:   uploadID,
						Body:       strings.NewReader(fmt.Sprintf("part %d data", i)),
					})
					Expect(err).NotTo(HaveOccurred())

					partETags[i] = output.ETag
				}
			})

			AfterEach(func(ctx context.Context) {
				if uploadID != nil {
					lo.Must(s3Client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
						Bucket:   &bucketName,
						Key:      testKey,
						UploadId: uploadID,
					}))
				}
			})

			It("returns an error", func(ctx context.Context) {
				wrongETag := lo.ToPtr("\"invalid-etag-value\"")
				_, err := s3Client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
					Bucket:   &bucketName,
					Key:      testKey,
					UploadId: uploadID,
					MultipartUpload: &types.CompletedMultipartUpload{
						Parts: []types.CompletedPart{
							{PartNumber: lo.ToPtr(int32(1)), ETag: partETags[1]},
							{PartNumber: lo.ToPtr(int32(2)), ETag: wrongETag},
							{PartNumber: lo.ToPtr(int32(3)), ETag: partETags[3]},
						},
					},
				})
				Expect(err).To(HaveOccurred())

				var httpErr interface{ HTTPStatusCode() int }
				Expect(errors.As(err, &httpErr)).To(BeTrue())
				Expect(httpErr.HTTPStatusCode()).To(Equal(400))
			})
		})

		When("part etag is from a different part number", func() {
			var (
				uploadID  *string
				partETags []*string
			)

			BeforeEach(func(ctx context.Context) {
				createMultipartUploadOutput, err := s3Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
					Bucket: &bucketName,
					Key:    testKey,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(createMultipartUploadOutput.UploadId).NotTo(BeNil())
				uploadID = createMultipartUploadOutput.UploadId
				partETags = make([]*string, 4)

				for i := 1; i <= 3; i++ {
					output, err := s3Client.UploadPart(ctx, &s3.UploadPartInput{
						Bucket:     &bucketName,
						Key:        testKey,
						PartNumber: lo.ToPtr(int32(i)),
						UploadId:   uploadID,
						Body:       strings.NewReader(fmt.Sprintf("part %d data", i)),
					})
					Expect(err).NotTo(HaveOccurred())

					partETags[i] = output.ETag
				}
			})

			AfterEach(func(ctx context.Context) {
				if uploadID != nil {
					lo.Must(s3Client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
						Bucket:   &bucketName,
						Key:      testKey,
						UploadId: uploadID,
					}))
				}
			})

			It("returns an error", func(ctx context.Context) {
				_, err := s3Client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
					Bucket:   &bucketName,
					Key:      testKey,
					UploadId: uploadID,
					MultipartUpload: &types.CompletedMultipartUpload{
						Parts: []types.CompletedPart{
							{PartNumber: lo.ToPtr(int32(1)), ETag: partETags[1]},
							{PartNumber: lo.ToPtr(int32(2)), ETag: partETags[3]}, // Using part 3's etag for part 2
							{PartNumber: lo.ToPtr(int32(3)), ETag: partETags[3]},
						},
					},
				})
				Expect(err).To(HaveOccurred())

				var httpErr interface{ HTTPStatusCode() int }
				Expect(errors.As(err, &httpErr)).To(BeTrue())
				Expect(httpErr.HTTPStatusCode()).To(Equal(400))
			})
		})

		When("part etag is empty string", func() {
			var (
				uploadID  *string
				partETags []*string
			)

			BeforeEach(func(ctx context.Context) {
				createMultipartUploadOutput, err := s3Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
					Bucket: &bucketName,
					Key:    testKey,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(createMultipartUploadOutput.UploadId).NotTo(BeNil())
				uploadID = createMultipartUploadOutput.UploadId
				partETags = make([]*string, 4)

				for i := 1; i <= 3; i++ {
					output, err := s3Client.UploadPart(ctx, &s3.UploadPartInput{
						Bucket:     &bucketName,
						Key:        testKey,
						PartNumber: lo.ToPtr(int32(i)),
						UploadId:   uploadID,
						Body:       strings.NewReader(fmt.Sprintf("part %d data", i)),
					})
					Expect(err).NotTo(HaveOccurred())

					partETags[i] = output.ETag
				}
			})

			AfterEach(func(ctx context.Context) {
				if uploadID != nil {
					lo.Must(s3Client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
						Bucket:   &bucketName,
						Key:      testKey,
						UploadId: uploadID,
					}))
				}
			})

			It("returns an error", func(ctx context.Context) {
				emptyETag := lo.ToPtr("")
				_, err := s3Client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
					Bucket:   &bucketName,
					Key:      testKey,
					UploadId: uploadID,
					MultipartUpload: &types.CompletedMultipartUpload{
						Parts: []types.CompletedPart{
							{PartNumber: lo.ToPtr(int32(1)), ETag: partETags[1]},
							{PartNumber: lo.ToPtr(int32(2)), ETag: emptyETag},
							{PartNumber: lo.ToPtr(int32(3)), ETag: partETags[3]},
						},
					},
				})
				Expect(err).To(HaveOccurred())

				var httpErr interface{ HTTPStatusCode() int }
				Expect(errors.As(err, &httpErr)).To(BeTrue())
				Expect(httpErr.HTTPStatusCode()).To(Equal(400))
			})
		})

		When("first part has invalid etag", func() {
			var (
				uploadID  *string
				partETags []*string
			)

			BeforeEach(func(ctx context.Context) {
				createMultipartUploadOutput, err := s3Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
					Bucket: &bucketName,
					Key:    testKey,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(createMultipartUploadOutput.UploadId).NotTo(BeNil())
				uploadID = createMultipartUploadOutput.UploadId
				partETags = make([]*string, 4)

				for i := 1; i <= 3; i++ {
					output, err := s3Client.UploadPart(ctx, &s3.UploadPartInput{
						Bucket:     &bucketName,
						Key:        testKey,
						PartNumber: lo.ToPtr(int32(i)),
						UploadId:   uploadID,
						Body:       strings.NewReader(fmt.Sprintf("part %d data", i)),
					})
					Expect(err).NotTo(HaveOccurred())

					partETags[i] = output.ETag
				}
			})

			AfterEach(func(ctx context.Context) {
				if uploadID != nil {
					lo.Must(s3Client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
						Bucket:   &bucketName,
						Key:      testKey,
						UploadId: uploadID,
					}))
				}
			})

			It("returns an error", func(ctx context.Context) {
				wrongETag := lo.ToPtr("\"wrong-etag\"")
				_, err := s3Client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
					Bucket:   &bucketName,
					Key:      testKey,
					UploadId: uploadID,
					MultipartUpload: &types.CompletedMultipartUpload{
						Parts: []types.CompletedPart{
							{PartNumber: lo.ToPtr(int32(1)), ETag: wrongETag},
							{PartNumber: lo.ToPtr(int32(2)), ETag: partETags[2]},
							{PartNumber: lo.ToPtr(int32(3)), ETag: partETags[3]},
						},
					},
				})
				Expect(err).To(HaveOccurred())

				var httpErr interface{ HTTPStatusCode() int }
				Expect(errors.As(err, &httpErr)).To(BeTrue())
				Expect(httpErr.HTTPStatusCode()).To(Equal(400))
			})
		})

		When("last part has invalid etag", func() {
			var (
				uploadID  *string
				partETags []*string
			)

			BeforeEach(func(ctx context.Context) {
				createMultipartUploadOutput, err := s3Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
					Bucket: &bucketName,
					Key:    testKey,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(createMultipartUploadOutput.UploadId).NotTo(BeNil())
				uploadID = createMultipartUploadOutput.UploadId
				partETags = make([]*string, 4)

				for i := 1; i <= 3; i++ {
					output, err := s3Client.UploadPart(ctx, &s3.UploadPartInput{
						Bucket:     &bucketName,
						Key:        testKey,
						PartNumber: lo.ToPtr(int32(i)),
						UploadId:   uploadID,
						Body:       strings.NewReader(fmt.Sprintf("part %d data", i)),
					})
					Expect(err).NotTo(HaveOccurred())

					partETags[i] = output.ETag
				}
			})

			AfterEach(func(ctx context.Context) {
				if uploadID != nil {
					lo.Must(s3Client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
						Bucket:   &bucketName,
						Key:      testKey,
						UploadId: uploadID,
					}))
				}
			})

			It("returns an error", func(ctx context.Context) {
				wrongETag := lo.ToPtr("\"wrong-etag\"")
				_, err := s3Client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
					Bucket:   &bucketName,
					Key:      testKey,
					UploadId: uploadID,
					MultipartUpload: &types.CompletedMultipartUpload{
						Parts: []types.CompletedPart{
							{PartNumber: lo.ToPtr(int32(1)), ETag: partETags[1]},
							{PartNumber: lo.ToPtr(int32(2)), ETag: partETags[2]},
							{PartNumber: lo.ToPtr(int32(3)), ETag: wrongETag},
						},
					},
				})
				Expect(err).To(HaveOccurred())

				var httpErr interface{ HTTPStatusCode() int }
				Expect(errors.As(err, &httpErr)).To(BeTrue())
				Expect(httpErr.HTTPStatusCode()).To(Equal(400))
			})
		})
	})

	Describe("Multipart Upload key validation", func() {
		When("UploadPart is called with mismatched key", func() {
			var uploadID *string

			BeforeEach(func(ctx context.Context) {
				createMultipartUploadOutput, err := s3Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
					Bucket: &bucketName,
					Key:    lo.ToPtr("file1.txt"),
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(createMultipartUploadOutput.UploadId).NotTo(BeNil())
				uploadID = createMultipartUploadOutput.UploadId
			})

			AfterEach(func(ctx context.Context) {
				if uploadID != nil {
					lo.Must(s3Client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
						Bucket:   &bucketName,
						Key:      lo.ToPtr("file1.txt"),
						UploadId: uploadID,
					}))
				}
			})

			It("returns an error", func(ctx context.Context) {
				_, err := s3Client.UploadPart(ctx, &s3.UploadPartInput{
					Bucket:     &bucketName,
					Key:        lo.ToPtr("file2.txt"), // Different key
					PartNumber: lo.ToPtr(int32(1)),
					UploadId:   uploadID,
					Body:       strings.NewReader("part 1 data"),
				})
				Expect(err).To(HaveOccurred())

				var httpErr interface{ HTTPStatusCode() int }
				Expect(errors.As(err, &httpErr)).To(BeTrue())
				Expect(httpErr.HTTPStatusCode()).To(Equal(400))
			})
		})

		When("CompleteMultipartUpload is called with mismatched key", func() {
			var (
				uploadID  *string
				partETags []*string
			)

			BeforeEach(func(ctx context.Context) {
				createMultipartUploadOutput, err := s3Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
					Bucket: &bucketName,
					Key:    lo.ToPtr("file1.txt"),
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(createMultipartUploadOutput.UploadId).NotTo(BeNil())
				uploadID = createMultipartUploadOutput.UploadId
				partETags = make([]*string, 2)

				// Upload parts with correct key
				output, err := s3Client.UploadPart(ctx, &s3.UploadPartInput{
					Bucket:     &bucketName,
					Key:        lo.ToPtr("file1.txt"),
					PartNumber: lo.ToPtr(int32(1)),
					UploadId:   uploadID,
					Body:       strings.NewReader("part 1 data"),
				})
				Expect(err).NotTo(HaveOccurred())

				partETags[1] = output.ETag
			})

			AfterEach(func(ctx context.Context) {
				if uploadID != nil {
					lo.Must(s3Client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
						Bucket:   &bucketName,
						Key:      lo.ToPtr("file1.txt"),
						UploadId: uploadID,
					}))
				}
			})

			It("returns an error", func(ctx context.Context) {
				_, err := s3Client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
					Bucket:   &bucketName,
					Key:      lo.ToPtr("file2.txt"), // Different key
					UploadId: uploadID,
					MultipartUpload: &types.CompletedMultipartUpload{
						Parts: []types.CompletedPart{
							{PartNumber: lo.ToPtr(int32(1)), ETag: partETags[1]},
						},
					},
				})
				Expect(err).To(HaveOccurred())

				var httpErr interface{ HTTPStatusCode() int }
				Expect(errors.As(err, &httpErr)).To(BeTrue())
				Expect(httpErr.HTTPStatusCode()).To(Equal(400))
			})
		})

		When("AbortMultipartUpload is called with mismatched key", func() {
			var uploadID *string

			BeforeEach(func(ctx context.Context) {
				createMultipartUploadOutput, err := s3Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
					Bucket: &bucketName,
					Key:    lo.ToPtr("file1.txt"),
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(createMultipartUploadOutput.UploadId).NotTo(BeNil())
				uploadID = createMultipartUploadOutput.UploadId
			})

			It("returns an error", func(ctx context.Context) {
				_, err := s3Client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
					Bucket:   &bucketName,
					Key:      lo.ToPtr("file2.txt"), // Different key
					UploadId: uploadID,
				})
				Expect(err).To(HaveOccurred())

				var httpErr interface{ HTTPStatusCode() int }
				Expect(errors.As(err, &httpErr)).To(BeTrue())
				Expect(httpErr.HTTPStatusCode()).To(Equal(400))

				// Clean up with correct key
				lo.Must(s3Client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
					Bucket:   &bucketName,
					Key:      lo.ToPtr("file1.txt"),
					UploadId: uploadID,
				}))
			})
		})

		When("all operations use matching keys", func() {
			var (
				uploadID  *string
				partETags []*string
			)

			It("completes successfully", func(ctx context.Context) {
				// Create multipart upload
				createMultipartUploadOutput, err := s3Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
					Bucket: &bucketName,
					Key:    lo.ToPtr("file1.txt"),
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(createMultipartUploadOutput.UploadId).NotTo(BeNil())
				uploadID = createMultipartUploadOutput.UploadId
				partETags = make([]*string, 3)

				// Upload parts with matching key
				for i := 1; i <= 2; i++ {
					output, err := s3Client.UploadPart(ctx, &s3.UploadPartInput{
						Bucket:     &bucketName,
						Key:        lo.ToPtr("file1.txt"),
						PartNumber: lo.ToPtr(int32(i)),
						UploadId:   uploadID,
						Body:       strings.NewReader(fmt.Sprintf("part %d data", i)),
					})
					Expect(err).NotTo(HaveOccurred())

					partETags[i] = output.ETag
				}

				// Complete with matching key
				completeOutput, err := s3Client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
					Bucket:   &bucketName,
					Key:      lo.ToPtr("file1.txt"),
					UploadId: uploadID,
					MultipartUpload: &types.CompletedMultipartUpload{
						Parts: []types.CompletedPart{
							{PartNumber: lo.ToPtr(int32(1)), ETag: partETags[1]},
							{PartNumber: lo.ToPtr(int32(2)), ETag: partETags[2]},
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(completeOutput.ETag).NotTo(BeNil())

				// Verify object exists at correct location
				headOutput, err := s3Client.HeadObject(ctx, &s3.HeadObjectInput{
					Bucket: &bucketName,
					Key:    lo.ToPtr("file1.txt"),
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(headOutput.ETag).To(Equal(completeOutput.ETag))
			})
		})
	})
})
