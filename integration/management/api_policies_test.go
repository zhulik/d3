package management_test

import (
	"context"
	"errors"

	"github.com/samber/lo"
	"github.com/zhulik/d3/internal/client/apiclient"
	"github.com/zhulik/d3/pkg/iampol"
	"github.com/zhulik/d3/pkg/s3actions"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Policies API", Label("management"), Label("api-policies"), Ordered, func() {
	var client *apiclient.Client
	var cancelApp context.CancelFunc
	var tempDir string

	BeforeAll(func(ctx context.Context) {
		client, cancelApp, tempDir = prepareManagementTests(ctx)
	})

	AfterAll(func(ctx context.Context) {
		cleanupManagementTests(ctx, cancelApp, tempDir)
	})

	Describe("ListPolicies", func() {
		When("no policies exist", func() {
			It("returns empty list", func(ctx context.Context) {
				policies, err := client.ListPolicies(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(policies).To(BeEmpty())
			})
		})

		When("policies exist", func() {
			BeforeAll(func(ctx context.Context) {
				policy := &iampol.IAMPolicy{
					ID: "test-policy-1",
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

			It("returns list of policy IDs", func(ctx context.Context) {
				policies, err := client.ListPolicies(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(policies).To(ContainElement("test-policy-1"))
			})
		})
	})

	Describe("GetPolicy", func() {
		When("policy exists", func() {
			var createdPolicy *iampol.IAMPolicy

			BeforeAll(func(ctx context.Context) {
				createdPolicy = &iampol.IAMPolicy{
					ID: "get-policy-test",
					Statement: []iampol.Statement{
						{
							Effect:   iampol.EffectAllow,
							Action:   []s3actions.Action{"s3:GetObject", "s3:PutObject"},
							Resource: []string{"arn:aws:s3:::test-bucket/*"},
						},
					},
				}
				lo.Must0(client.CreatePolicy(ctx, createdPolicy))
			})

			It("returns policy with correct structure", func(ctx context.Context) {
				policy, err := client.GetPolicy(ctx, "get-policy-test")
				Expect(err).NotTo(HaveOccurred())
				Expect(policy).NotTo(BeNil())
				Expect(policy.ID).To(Equal("get-policy-test"))
				Expect(policy.Statement).To(HaveLen(1))
				Expect(policy.Statement[0].Effect).To(Equal(iampol.EffectAllow))
				Expect(policy.Statement[0].Action).To(ContainElement(s3actions.Action("s3:GetObject")))
				Expect(policy.Statement[0].Action).To(ContainElement(s3actions.Action("s3:PutObject")))
				Expect(policy.Statement[0].Resource).To(ContainElement("arn:aws:s3:::test-bucket/*"))
			})
		})

		When("policy does not exist", func() {
			It("returns error", func(ctx context.Context) {
				_, err := client.GetPolicy(ctx, "non-existent-policy")
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, apiclient.ErrUnexpectedStatus)).To(BeTrue())
			})
		})
	})

	Describe("CreatePolicy", func() {
		When("policy does not exist", func() {
			When("policy is valid with single statement", func() {
				It("creates policy successfully", func(ctx context.Context) {
					policy := &iampol.IAMPolicy{
						ID: "create-policy-single",
						Statement: []iampol.Statement{
							{
								Effect:   iampol.EffectAllow,
								Action:   []s3actions.Action{"s3:GetObject"},
								Resource: []string{"arn:aws:s3:::my-bucket/*"},
							},
						},
					}

					err := client.CreatePolicy(ctx, policy)
					Expect(err).NotTo(HaveOccurred())

					createdPolicy, err := client.GetPolicy(ctx, "create-policy-single")
					Expect(err).NotTo(HaveOccurred())
					Expect(createdPolicy).NotTo(BeNil())
					Expect(createdPolicy.ID).To(Equal("create-policy-single"))
					Expect(createdPolicy.Statement).To(HaveLen(1))

					policies, err := client.ListPolicies(ctx)
					Expect(err).NotTo(HaveOccurred())
					Expect(policies).To(ContainElement("create-policy-single"))
				})
			})

			When("policy is valid with multiple statements", func() {
				It("creates policy successfully", func(ctx context.Context) {
					policy := &iampol.IAMPolicy{
						ID: "create-policy-multiple",
						Statement: []iampol.Statement{
							{
								Effect:   iampol.EffectAllow,
								Action:   []s3actions.Action{"s3:GetObject"},
								Resource: []string{"arn:aws:s3:::bucket1/*"},
							},
							{
								Effect:   iampol.EffectDeny,
								Action:   []s3actions.Action{"s3:DeleteObject"},
								Resource: []string{"arn:aws:s3:::bucket2/*"},
							},
						},
					}

					err := client.CreatePolicy(ctx, policy)
					Expect(err).NotTo(HaveOccurred())

					createdPolicy, err := client.GetPolicy(ctx, "create-policy-multiple")
					Expect(err).NotTo(HaveOccurred())
					Expect(createdPolicy).NotTo(BeNil())
					Expect(createdPolicy.ID).To(Equal("create-policy-multiple"))
					Expect(createdPolicy.Statement).To(HaveLen(2))
					Expect(createdPolicy.Statement[0].Effect).To(Equal(iampol.EffectAllow))
					Expect(createdPolicy.Statement[1].Effect).To(Equal(iampol.EffectDeny))
				})
			})

			When("policy is valid with Deny effect", func() {
				It("creates policy successfully", func(ctx context.Context) {
					policy := &iampol.IAMPolicy{
						ID: "create-policy-deny",
						Statement: []iampol.Statement{
							{
								Effect:   iampol.EffectDeny,
								Action:   []s3actions.Action{"s3:DeleteObject"},
								Resource: []string{"arn:aws:s3:::protected-bucket/*"},
							},
						},
					}

					err := client.CreatePolicy(ctx, policy)
					Expect(err).NotTo(HaveOccurred())

					createdPolicy, err := client.GetPolicy(ctx, "create-policy-deny")
					Expect(err).NotTo(HaveOccurred())
					Expect(createdPolicy).NotTo(BeNil())
					Expect(createdPolicy.ID).To(Equal("create-policy-deny"))
					Expect(createdPolicy.Statement[0].Effect).To(Equal(iampol.EffectDeny))
				})
			})
		})

		When("policy already exists", func() {
			BeforeAll(func(ctx context.Context) {
				policy := &iampol.IAMPolicy{
					ID: "existing-policy",
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

			It("returns error", func(ctx context.Context) {
				policy := &iampol.IAMPolicy{
					ID: "existing-policy",
					Statement: []iampol.Statement{
						{
							Effect:   iampol.EffectAllow,
							Action:   []s3actions.Action{"s3:GetObject"},
							Resource: []string{"arn:aws:s3:::my-bucket/*"},
						},
					},
				}

				err := client.CreatePolicy(ctx, policy)
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, apiclient.ErrUnexpectedStatus)).To(BeTrue())
			})
		})

		When("policy has missing ID", func() {
			It("returns error", func(ctx context.Context) {
				policy := &iampol.IAMPolicy{
					ID: "",
					Statement: []iampol.Statement{
						{
							Effect:   iampol.EffectAllow,
							Action:   []s3actions.Action{"s3:GetObject"},
							Resource: []string{"arn:aws:s3:::my-bucket/*"},
						},
					},
				}

				err := client.CreatePolicy(ctx, policy)
				Expect(err).To(HaveOccurred())
			})
		})

		When("policy has missing Statement", func() {
			It("returns error", func(ctx context.Context) {
				policy := &iampol.IAMPolicy{
					ID:        "no-statement-policy",
					Statement: []iampol.Statement{},
				}

				err := client.CreatePolicy(ctx, policy)
				Expect(err).To(HaveOccurred())
			})
		})

		When("policy has invalid Effect", func() {
			It("returns error", func(ctx context.Context) {
				policy := &iampol.IAMPolicy{
					ID: "bad-effect-policy",
					Statement: []iampol.Statement{
						{
							Effect:   iampol.Effect("allow"), // lowercase, invalid
							Action:   []s3actions.Action{"s3:GetObject"},
							Resource: []string{"arn:aws:s3:::my-bucket/*"},
						},
					},
				}

				err := client.CreatePolicy(ctx, policy)
				Expect(err).To(HaveOccurred())
			})
		})

		When("policy has missing Action", func() {
			It("returns error", func(ctx context.Context) {
				policy := &iampol.IAMPolicy{
					ID: "missing-action-policy",
					Statement: []iampol.Statement{
						{
							Effect:   iampol.EffectAllow,
							Action:   []s3actions.Action{},
							Resource: []string{"arn:aws:s3:::my-bucket/*"},
						},
					},
				}

				err := client.CreatePolicy(ctx, policy)
				Expect(err).To(HaveOccurred())
			})
		})

		When("policy has invalid Action", func() {
			It("returns error", func(ctx context.Context) {
				policy := &iampol.IAMPolicy{
					ID: "invalid-action-policy",
					Statement: []iampol.Statement{
						{
							Effect:   iampol.EffectAllow,
							Action:   []s3actions.Action{"NotAnAction"},
							Resource: []string{"arn:aws:s3:::my-bucket/*"},
						},
					},
				}

				err := client.CreatePolicy(ctx, policy)
				Expect(err).To(HaveOccurred())
			})
		})

		When("policy has missing Resource", func() {
			It("returns error", func(ctx context.Context) {
				policy := &iampol.IAMPolicy{
					ID: "missing-resource-policy",
					Statement: []iampol.Statement{
						{
							Effect:   iampol.EffectAllow,
							Action:   []s3actions.Action{"s3:GetObject"},
							Resource: []string{},
						},
					},
				}

				err := client.CreatePolicy(ctx, policy)
				Expect(err).To(HaveOccurred())
			})
		})

		When("policy has invalid Resource", func() {
			It("returns error", func(ctx context.Context) {
				policy := &iampol.IAMPolicy{
					ID: "invalid-resource-policy",
					Statement: []iampol.Statement{
						{
							Effect:   iampol.EffectAllow,
							Action:   []s3actions.Action{"s3:GetObject"},
							Resource: []string{"arn:aws:sqs:::queue"},
						},
					},
				}

				err := client.CreatePolicy(ctx, policy)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("UpdatePolicy", func() {
		When("policy exists", func() {
			BeforeAll(func(ctx context.Context) {
				policy := &iampol.IAMPolicy{
					ID: "update-policy-test",
					Statement: []iampol.Statement{
						{
							Effect:   iampol.EffectAllow,
							Action:   []s3actions.Action{"s3:GetObject"},
							Resource: []string{"arn:aws:s3:::old-bucket/*"},
						},
					},
				}
				lo.Must0(client.CreatePolicy(ctx, policy))
			})

			It("updates policy successfully", func(ctx context.Context) {
				updatedPolicy := &iampol.IAMPolicy{
					ID: "update-policy-test",
					Statement: []iampol.Statement{
						{
							Effect:   iampol.EffectAllow,
							Action:   []s3actions.Action{"s3:GetObject", "s3:PutObject"},
							Resource: []string{"arn:aws:s3:::new-bucket/*"},
						},
					},
				}

				err := client.UpdatePolicy(ctx, "update-policy-test", updatedPolicy)
				Expect(err).NotTo(HaveOccurred())

				// Verify the update persisted
				retrieved, err := client.GetPolicy(ctx, "update-policy-test")
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.ID).To(Equal("update-policy-test"))
				Expect(retrieved.Statement[0].Action).To(ContainElement(s3actions.Action("s3:PutObject")))
				Expect(retrieved.Statement[0].Resource).To(ContainElement("arn:aws:s3:::new-bucket/*"))
			})

			It("updates policy with different statements", func(ctx context.Context) {
				updatedPolicy := &iampol.IAMPolicy{
					ID: "update-policy-test",
					Statement: []iampol.Statement{
						{
							Effect:   iampol.EffectDeny,
							Action:   []s3actions.Action{"s3:DeleteObject"},
							Resource: []string{"arn:aws:s3:::protected-bucket/*"},
						},
						{
							Effect:   iampol.EffectAllow,
							Action:   []s3actions.Action{"s3:GetObject"},
							Resource: []string{"arn:aws:s3:::protected-bucket/*"},
						},
					},
				}

				err := client.UpdatePolicy(ctx, "update-policy-test", updatedPolicy)
				Expect(err).NotTo(HaveOccurred())

				retrieved, err := client.GetPolicy(ctx, "update-policy-test")
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.Statement).To(HaveLen(2))
				Expect(retrieved.Statement[0].Effect).To(Equal(iampol.EffectDeny))
			})
		})

		When("policy does not exist", func() {
			It("returns error", func(ctx context.Context) {
				policy := &iampol.IAMPolicy{
					ID: "non-existent-policy",
					Statement: []iampol.Statement{
						{
							Effect:   iampol.EffectAllow,
							Action:   []s3actions.Action{"s3:GetObject"},
							Resource: []string{"arn:aws:s3:::my-bucket/*"},
						},
					},
				}

				err := client.UpdatePolicy(ctx, "non-existent-policy", policy)
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, apiclient.ErrUnexpectedStatus)).To(BeTrue())
			})
		})

		When("policy has invalid structure", func() {
			BeforeAll(func(ctx context.Context) {
				policy := &iampol.IAMPolicy{
					ID: "update-invalid-test",
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

			It("returns error for missing Action", func(ctx context.Context) {
				invalidPolicy := &iampol.IAMPolicy{
					ID: "update-invalid-test",
					Statement: []iampol.Statement{
						{
							Effect:   iampol.EffectAllow,
							Action:   []s3actions.Action{},
							Resource: []string{"arn:aws:s3:::my-bucket/*"},
						},
					},
				}

				err := client.UpdatePolicy(ctx, "update-invalid-test", invalidPolicy)
				Expect(err).To(HaveOccurred())
			})

			It("returns error for invalid Resource", func(ctx context.Context) {
				invalidPolicy := &iampol.IAMPolicy{
					ID: "update-invalid-test",
					Statement: []iampol.Statement{
						{
							Effect:   iampol.EffectAllow,
							Action:   []s3actions.Action{"s3:GetObject"},
							Resource: []string{"invalid-resource"},
						},
					},
				}

				err := client.UpdatePolicy(ctx, "update-invalid-test", invalidPolicy)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("DeletePolicy", func() {
		When("policy exists", func() {
			BeforeAll(func(ctx context.Context) {
				policy := &iampol.IAMPolicy{
					ID: "delete-policy-test",
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

			It("deletes policy successfully", func(ctx context.Context) {
				err := client.DeletePolicy(ctx, "delete-policy-test")
				Expect(err).NotTo(HaveOccurred())

				policies, err := client.ListPolicies(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(policies).NotTo(ContainElement("delete-policy-test"))
			})
		})

		When("policy does not exist", func() {
			It("returns error", func(ctx context.Context) {
				err := client.DeletePolicy(ctx, "does-not-exist")
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, apiclient.ErrUnexpectedStatus)).To(BeTrue())
			})
		})
	})
})
