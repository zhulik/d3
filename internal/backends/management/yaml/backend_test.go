package yaml_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/zhulik/d3/internal/backends/management/yaml"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/internal/core/mocks"
	"github.com/zhulik/d3/pkg/iampol"
	yamlPkg "github.com/zhulik/d3/pkg/yaml"
)

var _ = Describe("YAML Backend", func() {
	var (
		tempDir    string
		configPath string
		tmpPath    string
		cfg        *core.Config
		backend    *yaml.Backend
		logger     *slog.Logger
	)

	BeforeEach(func() {
		tempDir = lo.Must(os.MkdirTemp("", "yaml-backend-test-"))

		configPath = filepath.Join(tempDir, "management.yaml")
		tmpPath = filepath.Join(tempDir, "tmp")

		cfg = &core.Config{
			ManagementBackendYAMLPath: configPath,
			ManagementBackendTmpPath:  tmpPath,
		}

		lo.Must0(os.MkdirAll(tmpPath, 0755))

		lo.Must0(os.MkdirAll(filepath.Join(tmpPath, "tmp"), 0755))

		logger = slog.New(slog.DiscardHandler)
		mockLocker := mocks.NewMockLocker(GinkgoT())

		mockLocker.EXPECT().Lock(mock.Anything, mock.Anything).Maybe().Return(
			func(ctx context.Context, _ string) (context.Context, context.CancelFunc, error) {
				return ctx, func() {}, nil
			},
		)

		backend = &yaml.Backend{
			Config: cfg,
			Locker: mockLocker,
			Logger: logger,
		}
	})

	AfterEach(func() {
		lo.Must0(os.RemoveAll(tempDir))
	})

	Describe("Init", func() {
		Context("when config file does not exist", func() {
			It("should create the config file with admin user", func(ctx context.Context) {
				err := backend.Init(ctx)
				Expect(err).NotTo(HaveOccurred())

				// Check file exists
				_, err = os.Stat(configPath)
				Expect(err).NotTo(HaveOccurred())

				// Load and verify
				config, err := yamlPkg.UnmarshalFromFile[yaml.ManagementConfig](configPath)
				Expect(err).NotTo(HaveOccurred())
				Expect(config.Version).To(Equal(1))
				Expect(config.AdminUser.AccessKeyID).NotTo(BeEmpty())
				Expect(config.AdminUser.SecretAccessKey).NotTo(BeEmpty())
				Expect(config.Users).To(BeEmpty())
				Expect(config.Policies).To(BeEmpty())
			})
		})

		Context("when config file exists", func() {
			BeforeEach(func() {
				// Create initial config as YAML string
				yamlContent := `version: 1
admin_user:
  access_key_id: admin-key
  secret_access_key: admin-secret
users:
  testuser:
    access_key_id: test-key
    secret_access_key: test-secret
policies:
  testpolicy:
    id: testpolicy
`
				lo.Must0(os.WriteFile(configPath, []byte(yamlContent), 0644))
			})

			It("should load the existing config", func(ctx context.Context) {
				err := backend.Init(ctx)
				Expect(err).NotTo(HaveOccurred())

				users, err := backend.GetUsers(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(users).To(ContainElement("admin"))
				Expect(users).To(ContainElement("testuser"))

				user, err := backend.GetUserByName(ctx, "testuser")
				Expect(err).NotTo(HaveOccurred())
				Expect(user.AccessKeyID).To(Equal("test-key"))

				policies, err := backend.GetPolicies(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(policies).To(ContainElement("testpolicy"))
			})
		})

		Context("when config version mismatches", func() {
			BeforeEach(func() {
				invalidConfig := yaml.ManagementConfig{
					Version: 999,
				}
				lo.Must0(yamlPkg.MarshalToFile(invalidConfig, configPath))
			})

			It("should return version mismatch error", func(ctx context.Context) {
				err := backend.Init(ctx)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(core.ErrConfigVersionMismatch))
			})
		})
	})

	Describe("User Management", func() {
		BeforeEach(func(ctx context.Context) {
			lo.Must0(backend.Init(ctx))
		})

		Describe("GetUsers", func() {
			It("should return admin and created users", func(ctx context.Context) {
				users, err := backend.GetUsers(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(users).To(ContainElement("admin"))
			})
		})

		Describe("GetUserByName", func() {
			Context("when user exists", func() {
				BeforeEach(func(ctx context.Context) {
					lo.Must(backend.CreateUser(ctx, "testuser"))
				})

				It("should return the user", func(ctx context.Context) {
					user, err := backend.GetUserByName(ctx, "testuser")
					Expect(err).NotTo(HaveOccurred())
					Expect(user.Name).To(Equal("testuser"))
					Expect(user.AccessKeyID).To(HaveLen(20))
					Expect(user.SecretAccessKey).To(HaveLen(40))
				})
			})

			Context("when user does not exist", func() {
				It("should return user not found error", func(ctx context.Context) {
					_, err := backend.GetUserByName(ctx, "nonexistent")
					Expect(err).To(MatchError(core.ErrUserNotFound))
				})
			})

			Context("when requesting admin", func() {
				It("should return admin user", func(ctx context.Context) {
					user, err := backend.GetUserByName(ctx, "admin")
					Expect(err).NotTo(HaveOccurred())
					Expect(user.Name).To(Equal("admin"))
				})
			})
		})

		Describe("GetUserByAccessKeyID", func() {
			Context("when access key exists", func() {
				var testKey string

				BeforeEach(func(ctx context.Context) {
					user := lo.Must(backend.CreateUser(ctx, "testuser"))
					testKey = user.AccessKeyID
				})

				It("should return the user", func(ctx context.Context) {
					user, err := backend.GetUserByAccessKeyID(ctx, testKey)
					Expect(err).NotTo(HaveOccurred())
					Expect(user.Name).To(Equal("testuser"))
				})
			})

			Context("when access key does not exist", func() {
				It("should return user not found error", func(ctx context.Context) {
					_, err := backend.GetUserByAccessKeyID(ctx, "nonexistent")
					Expect(err).To(MatchError(core.ErrUserNotFound))
				})
			})
		})

		Describe("CreateUser", func() {
			Context("with valid user", func() {
				It("should create the user", func(ctx context.Context) {
					_, err := backend.CreateUser(ctx, "newuser")
					Expect(err).NotTo(HaveOccurred())

					retrieved, err := backend.GetUserByName(ctx, "newuser")
					Expect(err).NotTo(HaveOccurred())
					Expect(retrieved.Name).To(Equal("newuser"))
					Expect(retrieved.AccessKeyID).To(HaveLen(20))
					Expect(retrieved.SecretAccessKey).To(HaveLen(40))
				})
			})

			Context("with empty name", func() {
				It("should return invalid user error", func(ctx context.Context) {
					_, err := backend.CreateUser(ctx, "")
					Expect(err).To(MatchError(core.ErrUserInvalid))
				})
			})

			Context("when user already exists", func() {
				BeforeEach(func(ctx context.Context) {
					lo.Must(backend.CreateUser(ctx, "existing"))
				})

				It("should return user already exists error", func(ctx context.Context) {
					_, err := backend.CreateUser(ctx, "existing")
					Expect(err).To(MatchError(core.ErrUserAlreadyExists))
				})
			})
		})

		Describe("UpdateUser", func() {
			BeforeEach(func(ctx context.Context) {
				lo.Must(backend.CreateUser(ctx, "updateuser"))
			})

			Context("with valid update", func() {
				It("should update the user", func(ctx context.Context) {
					updated := &core.User{
						Name:            "updateuser",
						AccessKeyID:     "new-key",
						SecretAccessKey: "new-secret",
					}
					err := backend.UpdateUser(ctx, updated)
					Expect(err).NotTo(HaveOccurred())

					user, err := backend.GetUserByName(ctx, "updateuser")
					Expect(err).NotTo(HaveOccurred())
					Expect(user.AccessKeyID).To(Equal("new-key"))
				})
			})

			Context("when user does not exist", func() {
				It("should return user not found error", func(ctx context.Context) {
					updated := &core.User{
						Name:            "nonexistent",
						AccessKeyID:     "key",
						SecretAccessKey: "secret",
					}
					err := backend.UpdateUser(ctx, updated)
					Expect(err).To(MatchError(core.ErrUserNotFound))
				})
			})

			Context("with invalid user", func() {
				It("should return invalid user error", func(ctx context.Context) {
					updated := &core.User{
						Name: "",
					}
					err := backend.UpdateUser(ctx, updated)
					Expect(err).To(MatchError(core.ErrUserInvalid))
				})
			})
		})

		Describe("DeleteUser", func() {
			BeforeEach(func(ctx context.Context) {
				lo.Must(backend.CreateUser(ctx, "deleteuser"))
			})

			Context("when user exists", func() {
				It("should delete the user", func(ctx context.Context) {
					err := backend.DeleteUser(ctx, "deleteuser")
					Expect(err).NotTo(HaveOccurred())

					_, err = backend.GetUserByName(ctx, "deleteuser")
					Expect(err).To(MatchError(core.ErrUserNotFound))
				})
			})

			Context("when user does not exist", func() {
				It("should return user not found error", func(ctx context.Context) {
					err := backend.DeleteUser(ctx, "nonexistent")
					Expect(err).To(MatchError(core.ErrUserNotFound))
				})
			})
		})
	})

	Describe("Policy Management", func() {
		BeforeEach(func(ctx context.Context) {
			lo.Must0(backend.Init(ctx))
		})

		Describe("GetPolicies", func() {
			It("should return empty list initially", func(ctx context.Context) {
				policies, err := backend.GetPolicies(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(policies).To(BeEmpty())
			})
		})

		Describe("GetPolicyByID", func() {
			Context("when policy exists", func() {
				BeforeEach(func(ctx context.Context) {
					policy := &iampol.IAMPolicy{
						ID: "testpolicy",
					}
					lo.Must0(backend.CreatePolicy(ctx, policy))
				})

				It("should return the policy", func(ctx context.Context) {
					policy, err := backend.GetPolicyByID(ctx, "testpolicy")
					Expect(err).NotTo(HaveOccurred())
					Expect(policy.ID).To(Equal("testpolicy"))
				})
			})

			Context("when policy does not exist", func() {
				It("should return policy not found error", func(ctx context.Context) {
					_, err := backend.GetPolicyByID(ctx, "nonexistent")
					Expect(err).To(MatchError(core.ErrPolicyNotFound))
				})
			})
		})

		Describe("CreatePolicy", func() {
			Context("with valid policy", func() {
				It("should create the policy", func(ctx context.Context) {
					policy := &iampol.IAMPolicy{
						ID: "newpolicy",
					}
					err := backend.CreatePolicy(ctx, policy)
					Expect(err).NotTo(HaveOccurred())

					retrieved, err := backend.GetPolicyByID(ctx, "newpolicy")
					Expect(err).NotTo(HaveOccurred())
					Expect(retrieved.ID).To(Equal("newpolicy"))
				})
			})

			Context("when policy already exists", func() {
				BeforeEach(func(ctx context.Context) {
					policy := &iampol.IAMPolicy{
						ID: "existing",
					}
					lo.Must0(backend.CreatePolicy(ctx, policy))
				})

				It("should return policy already exists error", func(ctx context.Context) {
					policy := &iampol.IAMPolicy{
						ID: "existing",
					}
					err := backend.CreatePolicy(ctx, policy)
					Expect(err).To(MatchError(core.ErrPolicyAlreadyExists))
				})
			})
		})

		Describe("UpdatePolicy", func() {
			BeforeEach(func(ctx context.Context) {
				policy := &iampol.IAMPolicy{
					ID: "updatepolicy",
				}
				lo.Must0(backend.CreatePolicy(ctx, policy))
			})

			Context("with valid update", func() {
				It("should update the policy", func(ctx context.Context) {
					updated := &iampol.IAMPolicy{
						ID: "updatepolicy",
					}
					err := backend.UpdatePolicy(ctx, updated)
					Expect(err).NotTo(HaveOccurred())

					policy, err := backend.GetPolicyByID(ctx, "updatepolicy")
					Expect(err).NotTo(HaveOccurred())
					Expect(policy.ID).To(Equal("updatepolicy"))
				})
			})

			Context("when policy does not exist", func() {
				It("should return policy not found error", func(ctx context.Context) {
					updated := &iampol.IAMPolicy{
						ID: "nonexistent",
					}
					err := backend.UpdatePolicy(ctx, updated)
					Expect(err).To(MatchError(core.ErrPolicyNotFound))
				})
			})
		})

		Describe("DeletePolicy", func() {
			BeforeEach(func(ctx context.Context) {
				policy := &iampol.IAMPolicy{
					ID: "deletepolicy",
				}
				lo.Must0(backend.CreatePolicy(ctx, policy))
			})

			Context("when policy exists", func() {
				It("should delete the policy", func(ctx context.Context) {
					err := backend.DeletePolicy(ctx, "deletepolicy")
					Expect(err).NotTo(HaveOccurred())

					_, err = backend.GetPolicyByID(ctx, "deletepolicy")
					Expect(err).To(MatchError(core.ErrPolicyNotFound))
				})
			})

			Context("when policy does not exist", func() {
				It("should return policy not found error", func(ctx context.Context) {
					err := backend.DeletePolicy(ctx, "nonexistent")
					Expect(err).To(MatchError(core.ErrPolicyNotFound))
				})
			})
		})
	})

	Describe("Binding Management", func() {
		BeforeEach(func(ctx context.Context) {
			lo.Must0(backend.Init(ctx))
		})

		Describe("GetBindings", func() {
			When("no bindings exist", func() {
				It("returns empty list", func(ctx context.Context) {
					bindings, err := backend.GetBindings(ctx)
					Expect(err).NotTo(HaveOccurred())
					Expect(bindings).To(BeEmpty())
				})
			})

			When("bindings exist", func() {
				BeforeEach(func(ctx context.Context) {
					lo.Must(backend.CreateUser(ctx, "bindinguser"))

					policy := &iampol.IAMPolicy{
						ID: "bindingpolicy",
					}
					lo.Must0(backend.CreatePolicy(ctx, policy))

					binding := &core.PolicyBinding{
						UserName: "bindinguser",
						PolicyID: "bindingpolicy",
					}
					lo.Must0(backend.CreateBinding(ctx, binding))
				})

				It("returns created bindings", func(ctx context.Context) {
					bindings, err := backend.GetBindings(ctx)
					Expect(err).NotTo(HaveOccurred())
					Expect(bindings).To(HaveLen(1))
					Expect(bindings[0].UserName).To(Equal("bindinguser"))
					Expect(bindings[0].PolicyID).To(Equal("bindingpolicy"))
				})
			})
		})

		Describe("GetBindingsByUser", func() {
			When("user exists", func() {
				BeforeEach(func(ctx context.Context) {
					lo.Must(backend.CreateUser(ctx, "bindinguser"))

					policy1 := &iampol.IAMPolicy{
						ID: "policy1",
					}
					lo.Must0(backend.CreatePolicy(ctx, policy1))

					policy2 := &iampol.IAMPolicy{
						ID: "policy2",
					}
					lo.Must0(backend.CreatePolicy(ctx, policy2))

					binding1 := &core.PolicyBinding{
						UserName: "bindinguser",
						PolicyID: "policy1",
					}
					lo.Must0(backend.CreateBinding(ctx, binding1))

					binding2 := &core.PolicyBinding{
						UserName: "bindinguser",
						PolicyID: "policy2",
					}
					lo.Must0(backend.CreateBinding(ctx, binding2))
				})

				It("returns bindings for user", func(ctx context.Context) {
					bindings, err := backend.GetBindingsByUser(ctx, "bindinguser")
					Expect(err).NotTo(HaveOccurred())
					Expect(bindings).To(HaveLen(2))
					policyIDs := lo.Map(bindings, func(binding *core.PolicyBinding, _ int) string {
						return binding.PolicyID
					})
					Expect(policyIDs).To(ContainElement("policy1"))
					Expect(policyIDs).To(ContainElement("policy2"))
				})
			})

			Context("when user does not exist", func() {
				It("returns empty list", func(ctx context.Context) {
					bindings, err := backend.GetBindingsByUser(ctx, "nonexistent")
					Expect(err).NotTo(HaveOccurred())
					Expect(bindings).To(BeEmpty())
				})
			})

			When("user has no bindings", func() {
				BeforeEach(func(ctx context.Context) {
					lo.Must(backend.CreateUser(ctx, "nobindingsuser"))
				})

				It("returns empty list", func(ctx context.Context) {
					bindings, err := backend.GetBindingsByUser(ctx, "nobindingsuser")
					Expect(err).NotTo(HaveOccurred())
					Expect(bindings).To(BeEmpty())
				})
			})
		})

		Describe("GetBindingsByPolicy", func() {
			When("policy exists", func() {
				BeforeEach(func(ctx context.Context) {
					lo.Must(backend.CreateUser(ctx, "user1"))
					lo.Must(backend.CreateUser(ctx, "user2"))

					policy := &iampol.IAMPolicy{
						ID: "sharedpolicy",
					}
					lo.Must0(backend.CreatePolicy(ctx, policy))

					binding1 := &core.PolicyBinding{
						UserName: "user1",
						PolicyID: "sharedpolicy",
					}
					lo.Must0(backend.CreateBinding(ctx, binding1))

					binding2 := &core.PolicyBinding{
						UserName: "user2",
						PolicyID: "sharedpolicy",
					}
					lo.Must0(backend.CreateBinding(ctx, binding2))
				})

				It("returns bindings for policy", func(ctx context.Context) {
					bindings, err := backend.GetBindingsByPolicy(ctx, "sharedpolicy")
					Expect(err).NotTo(HaveOccurred())
					Expect(bindings).To(HaveLen(2))
					userNames := lo.Map(bindings, func(binding *core.PolicyBinding, _ int) string {
						return binding.UserName
					})
					Expect(userNames).To(ContainElement("user1"))
					Expect(userNames).To(ContainElement("user2"))
				})
			})

			Context("when policy does not exist", func() {
				It("returns empty list", func(ctx context.Context) {
					bindings, err := backend.GetBindingsByPolicy(ctx, "nonexistent")
					Expect(err).NotTo(HaveOccurred())
					Expect(bindings).To(BeEmpty())
				})
			})

			When("policy has no bindings", func() {
				BeforeEach(func(ctx context.Context) {
					policy := &iampol.IAMPolicy{
						ID: "nobindingspolicy",
					}
					lo.Must0(backend.CreatePolicy(ctx, policy))
				})

				It("returns empty list", func(ctx context.Context) {
					bindings, err := backend.GetBindingsByPolicy(ctx, "nobindingspolicy")
					Expect(err).NotTo(HaveOccurred())
					Expect(bindings).To(BeEmpty())
				})
			})
		})

		Describe("CreateBinding", func() {
			BeforeEach(func(ctx context.Context) {
				lo.Must(backend.CreateUser(ctx, "bindinguser"))

				policy := &iampol.IAMPolicy{
					ID: "bindingpolicy",
				}
				lo.Must0(backend.CreatePolicy(ctx, policy))
			})

			When("binding is valid", func() {
				It("creates the binding", func(ctx context.Context) {
					binding := &core.PolicyBinding{
						UserName: "bindinguser",
						PolicyID: "bindingpolicy",
					}
					err := backend.CreateBinding(ctx, binding)
					Expect(err).NotTo(HaveOccurred())

					retrieved, err := backend.GetBindings(ctx)
					Expect(err).NotTo(HaveOccurred())
					Expect(retrieved).To(HaveLen(1))
					Expect(retrieved[0].UserName).To(Equal("bindinguser"))
					Expect(retrieved[0].PolicyID).To(Equal("bindingpolicy"))
				})
			})

			When("binding already exists", func() {
				BeforeEach(func(ctx context.Context) {
					binding := &core.PolicyBinding{
						UserName: "bindinguser",
						PolicyID: "bindingpolicy",
					}
					lo.Must0(backend.CreateBinding(ctx, binding))
				})

				It("returns binding already exists error", func(ctx context.Context) {
					binding := &core.PolicyBinding{
						UserName: "bindinguser",
						PolicyID: "bindingpolicy",
					}
					err := backend.CreateBinding(ctx, binding)
					Expect(err).To(MatchError(core.ErrBindingAlreadyExists))
				})
			})

			Context("when user does not exist", func() {
				It("returns user not found error", func(ctx context.Context) {
					binding := &core.PolicyBinding{
						UserName: "nonexistent",
						PolicyID: "bindingpolicy",
					}
					err := backend.CreateBinding(ctx, binding)
					Expect(err).To(MatchError(core.ErrUserNotFound))
				})
			})

			Context("when policy does not exist", func() {
				It("returns policy not found error", func(ctx context.Context) {
					binding := &core.PolicyBinding{
						UserName: "bindinguser",
						PolicyID: "nonexistent",
					}
					err := backend.CreateBinding(ctx, binding)
					Expect(err).To(MatchError(core.ErrPolicyNotFound))
				})
			})

			When("user name is empty", func() {
				It("returns invalid binding error", func(ctx context.Context) {
					binding := &core.PolicyBinding{
						UserName: "",
						PolicyID: "bindingpolicy",
					}
					err := backend.CreateBinding(ctx, binding)
					Expect(err).To(MatchError(core.ErrBindingInvalid))
				})
			})

			When("policy ID is empty", func() {
				It("returns invalid binding error", func(ctx context.Context) {
					binding := &core.PolicyBinding{
						UserName: "bindinguser",
						PolicyID: "",
					}
					err := backend.CreateBinding(ctx, binding)
					Expect(err).To(MatchError(core.ErrBindingInvalid))
				})
			})
		})

		Describe("DeleteBinding", func() {
			BeforeEach(func(ctx context.Context) {
				lo.Must(backend.CreateUser(ctx, "bindinguser"))

				policy := &iampol.IAMPolicy{
					ID: "bindingpolicy",
				}
				lo.Must0(backend.CreatePolicy(ctx, policy))

				binding := &core.PolicyBinding{
					UserName: "bindinguser",
					PolicyID: "bindingpolicy",
				}
				lo.Must0(backend.CreateBinding(ctx, binding))
			})

			When("binding exists", func() {
				It("deletes the binding", func(ctx context.Context) {
					err := backend.DeleteBinding(ctx, &core.PolicyBinding{
						UserName: "bindinguser",
						PolicyID: "bindingpolicy",
					})
					Expect(err).NotTo(HaveOccurred())

					retrieved, err := backend.GetBindings(ctx)
					Expect(err).NotTo(HaveOccurred())
					Expect(retrieved).To(BeEmpty())
				})
			})

			Context("when binding does not exist", func() {
				It("returns binding not found error", func(ctx context.Context) {
					err := backend.DeleteBinding(ctx, &core.PolicyBinding{
						UserName: "nonexistent",
						PolicyID: "nonexistent",
					})
					Expect(err).To(MatchError(core.ErrBindingNotFound))
				})
			})
		})
	})
})
