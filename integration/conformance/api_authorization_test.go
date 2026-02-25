package conformance_test

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/samber/lo"
	"github.com/zhulik/d3/integration/testhelpers"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/iampol"
	"github.com/zhulik/d3/pkg/s3actions"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Authorization", Label("conformance"), Label("authorization"), Ordered, func() {
	var (
		app                      *testhelpers.App
		tempBucketForAdminDelete string
		mgmtDeleteBucket         string
	)

	BeforeAll(func(ctx context.Context) {
		app = testhelpers.NewApp() //nolint:contextcheck
		adminS3Client := app.S3Client(ctx, "admin")

		mgmtBackend := app.ManagementBackend(ctx)

		testObjectKeys := []string{
			"public/file1.txt", "public/file2.txt", "private/file2.txt", "shared/file3.txt",
			"a/object.txt", "b/object.txt", "a/other.txt", "x/y/object.txt",
		}
		for _, key := range testObjectKeys {
			lo.Must(adminS3Client.PutObject(ctx, &s3.PutObjectInput{
				Bucket: lo.ToPtr(app.BucketName()),
				Key:    lo.ToPtr(key),
				Body:   strings.NewReader("data"),
			}))
		}

		arnPrefix := "arn:aws:s3:::"

		readOnlyPolicy := &iampol.IAMPolicy{
			ID: "read-only-policy",
			Statement: []iampol.Statement{{
				Effect:   iampol.EffectAllow,
				Action:   []s3actions.Action{s3actions.GetObject, s3actions.HeadObject, s3actions.ListObjectsV2, s3actions.GetObjectTagging},
				Resource: []string{arnPrefix + app.BucketName(), arnPrefix + app.BucketName() + "/*"},
			}},
		}
		lo.Must0(mgmtBackend.CreatePolicy(ctx, readOnlyPolicy))

		writeOnlyPolicy := &iampol.IAMPolicy{
			ID: "write-only-policy",
			Statement: []iampol.Statement{{
				Effect:   iampol.EffectAllow,
				Action:   []s3actions.Action{s3actions.PutObject, s3actions.DeleteObject, s3actions.DeleteObjects},
				Resource: []string{arnPrefix + app.BucketName(), arnPrefix + app.BucketName() + "/*"},
			}},
		}
		lo.Must0(mgmtBackend.CreatePolicy(ctx, writeOnlyPolicy))

		publicReadPolicy := &iampol.IAMPolicy{
			ID: "public-read-policy",
			Statement: []iampol.Statement{{
				Effect:   iampol.EffectAllow,
				Action:   []s3actions.Action{s3actions.GetObject},
				Resource: []string{arnPrefix + app.BucketName() + "/public/*"},
			}},
		}
		lo.Must0(mgmtBackend.CreatePolicy(ctx, publicReadPolicy))

		denyDeletePolicy := &iampol.IAMPolicy{
			ID: "deny-delete-policy",
			Statement: []iampol.Statement{{
				Effect:   iampol.EffectDeny,
				Action:   []s3actions.Action{s3actions.DeleteObject},
				Resource: []string{arnPrefix + app.BucketName() + "/*"},
			}},
		}
		lo.Must0(mgmtBackend.CreatePolicy(ctx, denyDeletePolicy))

		fullAccessPolicy := &iampol.IAMPolicy{
			ID: "full-access-policy",
			Statement: []iampol.Statement{{
				Effect:   iampol.EffectAllow,
				Action:   []s3actions.Action{s3actions.All},
				Resource: []string{arnPrefix + app.BucketName(), arnPrefix + app.BucketName() + "/*"},
			}},
		}
		lo.Must0(mgmtBackend.CreatePolicy(ctx, fullAccessPolicy))

		specificObjectPolicy := &iampol.IAMPolicy{
			ID: "specific-object-policy",
			Statement: []iampol.Statement{{
				Effect:   iampol.EffectAllow,
				Action:   []s3actions.Action{s3actions.GetObject, s3actions.HeadObject},
				Resource: []string{arnPrefix + app.BucketName() + "/shared/file3.txt"},
			}},
		}
		lo.Must0(mgmtBackend.CreatePolicy(ctx, specificObjectPolicy))

		wildcardMiddlePolicy := &iampol.IAMPolicy{
			ID: "wildcard-middle-policy",
			Statement: []iampol.Statement{{
				Effect:   iampol.EffectAllow,
				Action:   []s3actions.Action{s3actions.GetObject, s3actions.HeadObject},
				Resource: []string{arnPrefix + app.BucketName() + "/*/object.txt"},
			}},
		}
		lo.Must0(mgmtBackend.CreatePolicy(ctx, wildcardMiddlePolicy))

		listBucketsPolicy := &iampol.IAMPolicy{
			ID: "list-buckets-policy",
			Statement: []iampol.Statement{{
				Effect:   iampol.EffectAllow,
				Action:   []s3actions.Action{s3actions.ListBuckets},
				Resource: []string{arnPrefix + "*"},
			}},
		}
		lo.Must0(mgmtBackend.CreatePolicy(ctx, listBucketsPolicy))

		bucketMgmtPolicy := &iampol.IAMPolicy{
			ID: "bucket-mgmt-policy",
			Statement: []iampol.Statement{{
				Effect:   iampol.EffectAllow,
				Action:   []s3actions.Action{s3actions.CreateBucket, s3actions.DeleteBucket},
				Resource: []string{arnPrefix + "*"},
			}},
		}
		lo.Must0(mgmtBackend.CreatePolicy(ctx, bucketMgmtPolicy))

		lo.Must(mgmtBackend.CreateUser(ctx, "reader"))
		lo.Must0(mgmtBackend.CreateBinding(ctx, &core.PolicyBinding{UserName: "reader", PolicyID: "read-only-policy"}))

		lo.Must(mgmtBackend.CreateUser(ctx, "writer"))
		lo.Must0(mgmtBackend.CreateBinding(ctx, &core.PolicyBinding{UserName: "writer", PolicyID: "write-only-policy"}))

		lo.Must(mgmtBackend.CreateUser(ctx, "public-reader"))
		lo.Must0(mgmtBackend.CreateBinding(ctx, &core.PolicyBinding{UserName: "public-reader", PolicyID: "public-read-policy"}))

		lo.Must(mgmtBackend.CreateUser(ctx, "restricted-writer"))
		lo.Must0(mgmtBackend.CreateBinding(ctx, &core.PolicyBinding{UserName: "restricted-writer", PolicyID: "write-only-policy"}))
		lo.Must0(mgmtBackend.CreateBinding(ctx, &core.PolicyBinding{UserName: "restricted-writer", PolicyID: "deny-delete-policy"}))

		lo.Must(mgmtBackend.CreateUser(ctx, "no-permissions-user"))

		lo.Must(mgmtBackend.CreateUser(ctx, "specific-object-user"))
		lo.Must0(mgmtBackend.CreateBinding(ctx, &core.PolicyBinding{UserName: "specific-object-user", PolicyID: "specific-object-policy"}))

		lo.Must(mgmtBackend.CreateUser(ctx, "wildcard-middle-user"))
		lo.Must0(mgmtBackend.CreateBinding(ctx, &core.PolicyBinding{UserName: "wildcard-middle-user", PolicyID: "wildcard-middle-policy"}))

		lo.Must(mgmtBackend.CreateUser(ctx, "list-buckets-user"))
		lo.Must0(mgmtBackend.CreateBinding(ctx, &core.PolicyBinding{UserName: "list-buckets-user", PolicyID: "list-buckets-policy"}))

		lo.Must(mgmtBackend.CreateUser(ctx, "bucket-mgmt-user"))
		lo.Must0(mgmtBackend.CreateBinding(ctx, &core.PolicyBinding{UserName: "bucket-mgmt-user", PolicyID: "bucket-mgmt-policy"}))

		// Create buckets for DeleteBucket tests.
		tempBucketForAdminDelete = app.BucketName() + "-temp-admin-delete"
		lo.Must(adminS3Client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: lo.ToPtr(tempBucketForAdminDelete)}))

		mgmtDeleteBucket = app.BucketName() + "-mgmt-delete"
		lo.Must(adminS3Client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: lo.ToPtr(mgmtDeleteBucket)}))
	})

	AfterAll(func(ctx context.Context) {
		app.Stop(ctx)
	})

	DescribeTable("Authorization checks",
		func(ctx context.Context, userName string, action s3actions.Action, objectKeyOrEmpty string, shouldSucceed bool) {
			// Choose S3 client for the user.
			var s3Client = app.S3Client(ctx, userName)

			var err error

			switch action { //nolint:exhaustive
			case s3actions.PutObject:
				key := objectKeyOrEmpty
				_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
					Bucket: lo.ToPtr(app.BucketName()),
					Key:    lo.ToPtr(key),
					Body:   strings.NewReader("data"),
				})
			case s3actions.GetObject:
				key := objectKeyOrEmpty
				_, err = s3Client.GetObject(ctx, &s3.GetObjectInput{
					Bucket: lo.ToPtr(app.BucketName()),
					Key:    lo.ToPtr(key),
				})
			case s3actions.HeadObject:
				key := objectKeyOrEmpty
				_, err = s3Client.HeadObject(ctx, &s3.HeadObjectInput{
					Bucket: lo.ToPtr(app.BucketName()),
					Key:    lo.ToPtr(key),
				})
			case s3actions.ListObjectsV2:
				_, err = s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
					Bucket: lo.ToPtr(app.BucketName()),
				})
			case s3actions.DeleteObject:
				key := objectKeyOrEmpty
				_, err = s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
					Bucket: lo.ToPtr(app.BucketName()),
					Key:    lo.ToPtr(key),
				})
			case s3actions.DeleteObjects:
				key := objectKeyOrEmpty
				_, err = s3Client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
					Bucket: lo.ToPtr(app.BucketName()),
					Delete: &types.Delete{
						Objects: []types.ObjectIdentifier{{Key: lo.ToPtr(key)}},
					},
				})
			case s3actions.GetObjectTagging:
				key := objectKeyOrEmpty
				_, err = s3Client.GetObjectTagging(ctx, &s3.GetObjectTaggingInput{
					Bucket: lo.ToPtr(app.BucketName()),
					Key:    lo.ToPtr(key),
				})
			case s3actions.ListBuckets:
				_, err = s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
			case s3actions.CreateBucket:
				bucketName := objectKeyOrEmpty
				_, err = s3Client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: lo.ToPtr(bucketName)})
			case s3actions.DeleteBucket:
				bucketName := objectKeyOrEmpty
				switch bucketName {
				case "temp-admin-delete":
					bucketName = tempBucketForAdminDelete
				case "mgmt-delete":
					bucketName = mgmtDeleteBucket
				}

				_, err = s3Client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: lo.ToPtr(bucketName)})
			default:
				Fail("unsupported action in authorization table test")
			}

			if shouldSucceed {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(HaveOccurred())
			}
		},
		Entry("admin can get object", "admin", s3actions.GetObject, "public/file1.txt", true),
		Entry("admin can put object", "admin", s3actions.PutObject, "admin-new.txt", true),
		Entry("admin can list objects", "admin", s3actions.ListObjectsV2, "", true),
		Entry("reader can get object", "reader", s3actions.GetObject, "public/file1.txt", true),
		Entry("reader can head object", "reader", s3actions.HeadObject, "public/file1.txt", true),
		Entry("reader can list objects", "reader", s3actions.ListObjectsV2, "", true),
		Entry("reader cannot put object", "reader", s3actions.PutObject, "reader-new.txt", false),
		Entry("reader cannot delete object", "reader", s3actions.DeleteObject, "public/file1.txt", false),
		Entry("writer can put object", "writer", s3actions.PutObject, "writer-new.txt", true),
		Entry("writer can delete object", "writer", s3actions.DeleteObject, "public/file1.txt", true),
		Entry("writer cannot get object", "writer", s3actions.GetObject, "public/file1.txt", false),
		Entry("public-reader can get public file", "public-reader", s3actions.GetObject, "public/file2.txt", true),
		Entry("public-reader cannot get private file", "public-reader", s3actions.GetObject, "private/file2.txt", false),
		Entry("restricted-writer can put object", "restricted-writer", s3actions.PutObject, "restricted-new.txt", true),
		Entry("restricted-writer cannot delete object", "restricted-writer", s3actions.DeleteObject, "public/file1.txt", false),
		Entry("no-permissions-user cannot get object", "no-permissions-user", s3actions.GetObject, "public/file1.txt", false),
		Entry("no-permissions-user cannot put object", "no-permissions-user", s3actions.PutObject, "noperm-new.txt", false),
		Entry("specific-object-user can get shared file3", "specific-object-user", s3actions.GetObject, "shared/file3.txt", true),
		Entry("specific-object-user can head shared file3", "specific-object-user", s3actions.HeadObject, "shared/file3.txt", true),
		Entry("specific-object-user cannot get public file", "specific-object-user", s3actions.GetObject, "public/file1.txt", false),
		Entry("specific-object-user cannot get private file", "specific-object-user", s3actions.GetObject, "private/file2.txt", false),
		Entry("specific-object-user cannot put object", "specific-object-user", s3actions.PutObject, "shared/new.txt", false),
		Entry("wildcard-middle-user can get a/object.txt", "wildcard-middle-user", s3actions.GetObject, "a/object.txt", true),
		Entry("wildcard-middle-user can get b/object.txt", "wildcard-middle-user", s3actions.GetObject, "b/object.txt", true),
		Entry("wildcard-middle-user can get x/y/object.txt", "wildcard-middle-user", s3actions.GetObject, "x/y/object.txt", true),
		Entry("wildcard-middle-user can head a/object.txt", "wildcard-middle-user", s3actions.HeadObject, "a/object.txt", true),
		Entry("wildcard-middle-user cannot get a/other.txt", "wildcard-middle-user", s3actions.GetObject, "a/other.txt", false),
		Entry("wildcard-middle-user cannot get public/file1.txt", "wildcard-middle-user", s3actions.GetObject, "public/file1.txt", false),
		Entry("writer can delete objects batch", "writer", s3actions.DeleteObjects, "public/file2.txt", true),
		Entry("reader cannot delete objects batch", "reader", s3actions.DeleteObjects, "public/file1.txt", false),
		Entry("reader can get object tagging", "reader", s3actions.GetObjectTagging, "shared/file3.txt", true),
		Entry("writer cannot get object tagging", "writer", s3actions.GetObjectTagging, "shared/file3.txt", false),
		Entry("admin can list buckets", "admin", s3actions.ListBuckets, "", true),
		Entry("list-buckets-user can list buckets", "list-buckets-user", s3actions.ListBuckets, "", true),
		Entry("no-permissions-user cannot list buckets", "no-permissions-user", s3actions.ListBuckets, "", false),
		Entry("reader cannot list buckets", "reader", s3actions.ListBuckets, "", false),
		Entry("admin can create bucket", "admin", s3actions.CreateBucket, "admin-auth-new-bucket", true),
		Entry("bucket-mgmt-user can create bucket", "bucket-mgmt-user", s3actions.CreateBucket, "mgmt-auth-new-bucket", true),
		Entry("reader cannot create bucket", "reader", s3actions.CreateBucket, "reader-auth-bucket", false),
		Entry("no-permissions-user cannot create bucket", "no-permissions-user", s3actions.CreateBucket, "noperm-auth-bucket", false),
		Entry("admin can delete bucket", "admin", s3actions.DeleteBucket, "temp-admin-delete", true),
		Entry("admin delete non-existent bucket returns 404", "admin", s3actions.DeleteBucket, "non-existent-bucket", false),
		Entry("bucket-mgmt-user can delete bucket", "bucket-mgmt-user", s3actions.DeleteBucket, "mgmt-delete", true),
		Entry("reader cannot delete bucket", "reader", s3actions.DeleteBucket, "any-bucket", false),
		Entry("no-permissions-user cannot delete bucket", "no-permissions-user", s3actions.DeleteBucket, "any-bucket", false),
	)
})
