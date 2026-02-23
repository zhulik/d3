package management_test

import (
	"context"

	"github.com/samber/lo"
	"github.com/zhulik/d3/integration/testhelpers"
	"github.com/zhulik/d3/internal/client/apiclient"
	"github.com/zhulik/d3/internal/core"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Users API", Label("management"), Label("api-users"), Ordered, func() {
	var (
		client *apiclient.Client
		app    *testhelpers.App
	)

	BeforeAll(func(ctx context.Context) {
		app = testhelpers.NewApp() //nolint:contextcheck
		client = app.ManagementClient(ctx)
	})

	AfterAll(func(ctx context.Context) {
		app.Stop(ctx)
	})

	Describe("ListUsers", func() {
		When("no users exist", func() {
			It("returns list with admin", func(ctx context.Context) {
				users, err := client.ListUsers(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(users).To(ContainElement("admin"))
			})
		})

		When("users exist", func() {
			BeforeAll(func(ctx context.Context) {
				lo.Must(client.CreateUser(ctx, "test-user-1"))
			})

			It("returns list of user names including admin and created users", func(ctx context.Context) {
				users, err := client.ListUsers(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(users).To(ContainElement("admin"))
				Expect(users).To(ContainElement("test-user-1"))
			})
		})
	})

	Describe("CreateUser", func() {
		When("user does not exist", func() {
			It("creates user successfully", func(ctx context.Context) {
				user, err := client.CreateUser(ctx, "new-user")
				Expect(err).NotTo(HaveOccurred())
				Expect(user.Name).To(Equal("new-user"))
				Expect(user.AccessKeyID).NotTo(BeEmpty())
				Expect(user.SecretAccessKey).NotTo(BeEmpty())

				users, err := client.ListUsers(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(users).To(ContainElement("new-user"))
			})
		})

		When("user already exists", func() {
			BeforeAll(func(ctx context.Context) {
				lo.Must(client.CreateUser(ctx, "existing-user"))
			})

			It("returns error", func(ctx context.Context) {
				_, err := client.CreateUser(ctx, "existing-user")
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("UpdateUser", func() {
		var originalUser core.User

		BeforeAll(func(ctx context.Context) {
			originalUser = lo.Must(client.CreateUser(ctx, "update-user"))
		})

		When("user exists", func() {
			It("updates user credentials successfully", func(ctx context.Context) {
				updatedUser, err := client.UpdateUser(ctx, "update-user")
				Expect(err).NotTo(HaveOccurred())
				Expect(updatedUser.Name).To(Equal("update-user"))
				Expect(updatedUser.AccessKeyID).NotTo(BeEmpty())
				Expect(updatedUser.SecretAccessKey).NotTo(BeEmpty())
				Expect(updatedUser.AccessKeyID).NotTo(Equal(originalUser.AccessKeyID))
				Expect(updatedUser.SecretAccessKey).NotTo(Equal(originalUser.SecretAccessKey))
			})
		})

		When("user does not exist", func() {
			It("returns error", func(ctx context.Context) {
				_, err := client.UpdateUser(ctx, "does-not-exist")
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("DeleteUser", func() {
		When("user exists", func() {
			BeforeAll(func(ctx context.Context) {
				lo.Must(client.CreateUser(ctx, "delete-user"))
			})

			It("deletes user successfully", func(ctx context.Context) {
				err := client.DeleteUser(ctx, "delete-user")
				Expect(err).NotTo(HaveOccurred())

				users, err := client.ListUsers(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(users).NotTo(ContainElement("delete-user"))
				Expect(users).To(ContainElement("admin"))
			})
		})

		When("user does not exist", func() {
			It("returns error", func(ctx context.Context) {
				err := client.DeleteUser(ctx, "does-not-exist")
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
