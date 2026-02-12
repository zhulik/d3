package management_test

import (
	"context"
	"errors"

	"github.com/samber/lo"
	"github.com/zhulik/d3/integration/testhelpers"
	"github.com/zhulik/d3/internal/client/apiclient"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/iampol"
	"github.com/zhulik/d3/pkg/s3actions"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Bindings API", Label("management"), Label("api-bindings"), Ordered, func() {
	var client *apiclient.Client
	var app *testhelpers.App

	BeforeAll(func(ctx context.Context) {
		app = testhelpers.NewApp() //nolint:contextcheck
		client = app.ManagementClient(ctx)
	})

	AfterAll(func(ctx context.Context) {
		app.Stop(ctx)
	})

	Describe("ListBindings", func() {
		When("no bindings exist", func() {
			It("returns empty list", func(ctx context.Context) {
				bindings, err := client.ListBindings(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(bindings).To(BeEmpty())
			})
		})

		When("bindings exist", func() {
			BeforeAll(func(ctx context.Context) {
				lo.Must(client.CreateUser(ctx, "binding-user-1"))
				policy := &iampol.IAMPolicy{
					ID: "binding-policy-1",
					Statement: []iampol.Statement{
						{
							Effect:   iampol.EffectAllow,
							Action:   []s3actions.Action{"s3:GetObject"},
							Resource: []string{"arn:aws:s3:::my-bucket/*", "arn:aws:s3:::my-bucket/key"},
						},
					},
				}
				lo.Must0(client.CreatePolicy(ctx, policy))

				binding := &core.PolicyBinding{
					UserName: "binding-user-1",
					PolicyID: "binding-policy-1",
				}
				lo.Must0(client.CreateBinding(ctx, binding))
			})

			It("returns list of bindings", func(ctx context.Context) {
				bindings, err := client.ListBindings(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(bindings).NotTo(BeEmpty())
				Expect(bindings).To(ContainElement(core.PolicyBinding{
					UserName: "binding-user-1",
					PolicyID: "binding-policy-1",
				}))
			})
		})
	})

	Describe("GetBindingsByUser", func() {
		When("user exists", func() {
			BeforeAll(func(ctx context.Context) {
				lo.Must(client.CreateUser(ctx, "binding-user-2"))
				policy1 := &iampol.IAMPolicy{
					ID: "binding-policy-2a",
					Statement: []iampol.Statement{
						{
							Effect:   iampol.EffectAllow,
							Action:   []s3actions.Action{"s3:GetObject"},
							Resource: []string{"arn:aws:s3:::my-bucket/*"},
						},
					},
				}
				lo.Must0(client.CreatePolicy(ctx, policy1))

				policy2 := &iampol.IAMPolicy{
					ID: "binding-policy-2b",
					Statement: []iampol.Statement{
						{
							Effect:   iampol.EffectAllow,
							Action:   []s3actions.Action{"s3:PutObject"},
							Resource: []string{"arn:aws:s3:::my-bucket/*"},
						},
					},
				}
				lo.Must0(client.CreatePolicy(ctx, policy2))

				binding1 := &core.PolicyBinding{
					UserName: "binding-user-2",
					PolicyID: "binding-policy-2a",
				}
				lo.Must0(client.CreateBinding(ctx, binding1))

				binding2 := &core.PolicyBinding{
					UserName: "binding-user-2",
					PolicyID: "binding-policy-2b",
				}
				lo.Must0(client.CreateBinding(ctx, binding2))
			})

			It("returns bindings for user", func(ctx context.Context) {
				bindings, err := client.GetBindingsByUser(ctx, "binding-user-2")
				Expect(err).NotTo(HaveOccurred())
				Expect(bindings).To(HaveLen(2))
				policyIDs := lo.Map(bindings, func(binding core.PolicyBinding, _ int) string {
					return binding.PolicyID
				})
				Expect(policyIDs).To(ContainElement("binding-policy-2a"))
				Expect(policyIDs).To(ContainElement("binding-policy-2b"))
			})
		})

		When("user does not exist", func() {
			It("returns empty list", func(ctx context.Context) {
				bindings, err := client.GetBindingsByUser(ctx, "nonexistent-user")
				Expect(err).NotTo(HaveOccurred())
				Expect(bindings).To(BeEmpty())
			})
		})

		When("user has no bindings", func() {
			BeforeAll(func(ctx context.Context) {
				lo.Must(client.CreateUser(ctx, "binding-user-no-bindings"))
			})

			It("returns empty list", func(ctx context.Context) {
				bindings, err := client.GetBindingsByUser(ctx, "binding-user-no-bindings")
				Expect(err).NotTo(HaveOccurred())
				Expect(bindings).To(BeEmpty())
			})
		})
	})

	Describe("GetBindingsByPolicy", func() {
		When("policy exists", func() {
			BeforeAll(func(ctx context.Context) {
				lo.Must(client.CreateUser(ctx, "binding-user-3a"))
				lo.Must(client.CreateUser(ctx, "binding-user-3b"))

				policy := &iampol.IAMPolicy{
					ID: "binding-policy-3",
					Statement: []iampol.Statement{
						{
							Effect:   iampol.EffectAllow,
							Action:   []s3actions.Action{"s3:GetObject"},
							Resource: []string{"arn:aws:s3:::my-bucket/*"},
						},
					},
				}
				lo.Must0(client.CreatePolicy(ctx, policy))

				binding1 := &core.PolicyBinding{
					UserName: "binding-user-3a",
					PolicyID: "binding-policy-3",
				}
				lo.Must0(client.CreateBinding(ctx, binding1))

				binding2 := &core.PolicyBinding{
					UserName: "binding-user-3b",
					PolicyID: "binding-policy-3",
				}
				lo.Must0(client.CreateBinding(ctx, binding2))
			})

			It("returns bindings for policy", func(ctx context.Context) {
				bindings, err := client.GetBindingsByPolicy(ctx, "binding-policy-3")
				Expect(err).NotTo(HaveOccurred())
				Expect(bindings).To(HaveLen(2))
				userNames := lo.Map(bindings, func(binding core.PolicyBinding, _ int) string {
					return binding.UserName
				})
				Expect(userNames).To(ContainElement("binding-user-3a"))
				Expect(userNames).To(ContainElement("binding-user-3b"))
			})
		})

		When("policy does not exist", func() {
			It("returns empty list", func(ctx context.Context) {
				bindings, err := client.GetBindingsByPolicy(ctx, "nonexistent-policy")
				Expect(err).NotTo(HaveOccurred())
				Expect(bindings).To(BeEmpty())
			})
		})

		When("policy has no bindings", func() {
			BeforeAll(func(ctx context.Context) {
				policy := &iampol.IAMPolicy{
					ID: "binding-policy-no-bindings",
					Statement: []iampol.Statement{
						{
							Effect:   iampol.EffectAllow,
							Action:   []s3actions.Action{"s3:GetObject"},
							Resource: []string{"arn:aws:s3:::my-bucket/*"},
						},
					},
				}
				lo.Must0(client.CreatePolicy(ctx, policy))
			})

			It("returns empty list", func(ctx context.Context) {
				bindings, err := client.GetBindingsByPolicy(ctx, "binding-policy-no-bindings")
				Expect(err).NotTo(HaveOccurred())
				Expect(bindings).To(BeEmpty())
			})
		})
	})

	Describe("CreateBinding", func() {
		BeforeAll(func(ctx context.Context) {
			lo.Must(client.CreateUser(ctx, "binding-user-create"))
			policy := &iampol.IAMPolicy{
				ID: "binding-policy-create",
				Statement: []iampol.Statement{
					{
						Effect:   iampol.EffectAllow,
						Action:   []s3actions.Action{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::my-bucket/*"},
					},
				},
			}
			lo.Must0(client.CreatePolicy(ctx, policy))
		})

		When("binding does not exist", func() {
			It("creates binding successfully", func(ctx context.Context) {
				binding := &core.PolicyBinding{
					UserName: "binding-user-create",
					PolicyID: "binding-policy-create",
				}
				err := client.CreateBinding(ctx, binding)
				Expect(err).NotTo(HaveOccurred())

				retrieved, err := client.ListBindings(ctx)
				Expect(err).NotTo(HaveOccurred())

				found := lo.ContainsBy(retrieved, func(binding core.PolicyBinding) bool {
					return binding.UserName == "binding-user-create" && binding.PolicyID == "binding-policy-create"
				})
				Expect(found).To(BeTrue())
			})
		})

		When("binding already exists", func() {
			BeforeAll(func(ctx context.Context) {
				lo.Must(client.CreateUser(ctx, "binding-user-existing"))
				policy := &iampol.IAMPolicy{
					ID: "binding-policy-existing",
					Statement: []iampol.Statement{
						{
							Effect:   iampol.EffectAllow,
							Action:   []s3actions.Action{"s3:GetObject"},
							Resource: []string{"arn:aws:s3:::my-bucket/*"},
						},
					},
				}
				lo.Must0(client.CreatePolicy(ctx, policy))

				binding := &core.PolicyBinding{
					UserName: "binding-user-existing",
					PolicyID: "binding-policy-existing",
				}
				lo.Must0(client.CreateBinding(ctx, binding))
			})

			It("returns error", func(ctx context.Context) {
				binding := &core.PolicyBinding{
					UserName: "binding-user-existing",
					PolicyID: "binding-policy-existing",
				}
				err := client.CreateBinding(ctx, binding)
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, apiclient.ErrUnexpectedStatus)).To(BeTrue())
			})
		})

		When("user does not exist", func() {
			It("returns error", func(ctx context.Context) {
				binding := &core.PolicyBinding{
					UserName: "nonexistent-user",
					PolicyID: "binding-policy-create",
				}
				err := client.CreateBinding(ctx, binding)
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, apiclient.ErrUnexpectedStatus)).To(BeTrue())
			})
		})

		When("policy does not exist", func() {
			BeforeAll(func(ctx context.Context) {
				lo.Must(client.CreateUser(ctx, "binding-user-no-policy"))
			})

			It("returns error", func(ctx context.Context) {
				binding := &core.PolicyBinding{
					UserName: "binding-user-no-policy",
					PolicyID: "nonexistent-policy",
				}
				err := client.CreateBinding(ctx, binding)
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, apiclient.ErrUnexpectedStatus)).To(BeTrue())
			})
		})
	})

	Describe("DeleteBinding", func() {
		When("binding exists", func() {
			BeforeAll(func(ctx context.Context) {
				lo.Must(client.CreateUser(ctx, "binding-user-delete"))
				policy := &iampol.IAMPolicy{
					ID: "binding-policy-delete",
					Statement: []iampol.Statement{
						{
							Effect:   iampol.EffectAllow,
							Action:   []s3actions.Action{"s3:GetObject"},
							Resource: []string{"arn:aws:s3:::my-bucket/*"},
						},
					},
				}
				lo.Must0(client.CreatePolicy(ctx, policy))

				binding := &core.PolicyBinding{
					UserName: "binding-user-delete",
					PolicyID: "binding-policy-delete",
				}
				lo.Must0(client.CreateBinding(ctx, binding))
			})

			It("deletes binding successfully", func(ctx context.Context) {
				err := client.DeleteBinding(ctx, "binding-user-delete", "binding-policy-delete")
				Expect(err).NotTo(HaveOccurred())

				retrieved, err := client.ListBindings(ctx)
				Expect(err).NotTo(HaveOccurred())
				found := lo.ContainsBy(retrieved, func(binding core.PolicyBinding) bool {
					return binding.UserName == "binding-user-delete" && binding.PolicyID == "binding-policy-delete"
				})
				Expect(found).To(BeFalse())
			})
		})

		When("binding does not exist", func() {
			It("returns error", func(ctx context.Context) {
				err := client.DeleteBinding(ctx, "nonexistent", "nonexistent")
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, apiclient.ErrUnexpectedStatus)).To(BeTrue())
			})
		})
	})
})
