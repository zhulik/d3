package conformance_test

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/samber/lo"
	"github.com/zhulik/d3/integration/testhelpers"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/iampol"
	"github.com/zhulik/d3/pkg/s3actions"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Authorization", Label("conformance"), Label("authorization"), Ordered, func() {
	var app *testhelpers.App

	BeforeAll(func(ctx context.Context) {
		app = testhelpers.NewApp() //nolint:contextcheck
		adminS3Client := app.S3Client(ctx, "admin")

		mgmtBackend := app.ManagementBackend(ctx)

		testObjectKeys := []string{"public/file1.txt", "public/file2.txt", "private/file2.txt", "shared/file3.txt"}
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
				Action:   []s3actions.Action{s3actions.GetObject, s3actions.HeadObject, s3actions.ListObjectsV2},
				Resource: []string{arnPrefix + app.BucketName(), arnPrefix + app.BucketName() + "/*"},
			}},
		}
		lo.Must0(mgmtBackend.CreatePolicy(ctx, readOnlyPolicy))

		writeOnlyPolicy := &iampol.IAMPolicy{
			ID: "write-only-policy",
			Statement: []iampol.Statement{{
				Effect:   iampol.EffectAllow,
				Action:   []s3actions.Action{s3actions.PutObject, s3actions.DeleteObject},
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
	)
})
