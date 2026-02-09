package commands

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
	"github.com/zhulik/d3/internal/client/apiclient"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/pal"
)

var (
	BindingCommand = &cli.Command{ //nolint:gochecknoglobals
		Name:    "binding",
		Aliases: []string{"b"},
		Usage:   "manage policy bindings",
		Commands: []*cli.Command{
			bindingList,
			bindingListByUser,
			bindingListByPolicy,
			bindingAdd,
			bindingDelete,
		},
	}

	bindingList = &cli.Command{ //nolint:gochecknoglobals
		Name:    "list",
		Aliases: []string{"ls", "l"},
		Usage:   "List all policy bindings",
		Action: func(ctx context.Context, _ *cli.Command) error {
			return invokeClient(ctx, func(client *apiclient.Client) error {
				bindings, err := client.ListBindings(ctx)
				if err != nil {
					return err
				}

				printBindings(bindings)

				return nil
			})
		},
	}

	bindingListByUser = &cli.Command{ //nolint:gochecknoglobals
		Name:      "list-by-user",
		Aliases:   []string{"user", "u"},
		Usage:     "List policy bindings for a user",
		Arguments: usernameArg,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return validateUsernameAndInvokeClient(ctx, cmd, func(username string, client *apiclient.Client) error {
				bindings, err := client.GetBindingsByUser(ctx, username)
				if err != nil {
					return err
				}

				printBindings(bindings)

				return nil
			})
		},
	}

	bindingListByPolicy = &cli.Command{ //nolint:gochecknoglobals
		Name:      "list-by-policy",
		Aliases:   []string{"policy", "p"},
		Usage:     "List policy bindings for a policy",
		Arguments: policyIDArg,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return validatePolicyIDAndInvokeClient(ctx, cmd, func(policyID string, client *apiclient.Client) error {
				bindings, err := client.GetBindingsByPolicy(ctx, policyID)
				if err != nil {
					return err
				}

				printBindings(bindings)

				return nil
			})
		},
	}

	bindingAdd = &cli.Command{ //nolint:gochecknoglobals
		Name:      "add",
		Aliases:   []string{"a"},
		Usage:     "Bind a policy to a user",
		Arguments: userAndPolicyArgs,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return validateUserAndPolicyAndInvoke(ctx, cmd, func(userName, policyID string, client *apiclient.Client) error {
				err := client.CreateBinding(ctx, &core.PolicyBinding{
					UserName: userName,
					PolicyID: policyID,
				})
				if err != nil {
					return err
				}

				fmt.Println("Binding created successfully") //nolint:forbidigo

				return nil
			})
		},
	}

	bindingDelete = &cli.Command{ //nolint:gochecknoglobals
		Name:      "delete",
		Aliases:   []string{"d"},
		Usage:     "Remove a policy binding from a user",
		Arguments: userAndPolicyArgs,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return validateUserAndPolicyAndInvoke(ctx, cmd, func(userName, policyID string, client *apiclient.Client) error {
				err := client.DeleteBinding(ctx, userName, policyID)
				if err != nil {
					return err
				}

				fmt.Println("Binding deleted successfully") //nolint:forbidigo

				return nil
			})
		},
	}

	policyIDArg = []cli.Argument{ //nolint:gochecknoglobals
		&cli.StringArg{
			Name:   "policy-id",
			Config: cli.StringConfig{},
		},
	}

	userAndPolicyArgs = []cli.Argument{ //nolint:gochecknoglobals
		&cli.StringArg{
			Name:   "username",
			Config: cli.StringConfig{},
		},
		&cli.StringArg{
			Name:   "policy-id",
			Config: cli.StringConfig{},
		},
	}
)

func printBindings(bindings []core.PolicyBinding) {
	for _, b := range bindings {
		fmt.Printf("%s\t%s\n", b.UserName, b.PolicyID) //nolint:forbidigo
	}
}

func invokeClient(ctx context.Context, f func(*apiclient.Client) error) error {
	client := pal.MustInvoke[*apiclient.Client](ctx, nil)

	return f(client)
}

type bindingByPolicyFn func(string, *apiclient.Client) error

func validatePolicyIDAndInvokeClient(ctx context.Context, cmd *cli.Command, f bindingByPolicyFn) error {
	policyID := cmd.StringArg("policy-id")
	if policyID == "" {
		return fmt.Errorf("%w: policy-id", ErrMissingArgument)
	}

	client := pal.MustInvoke[*apiclient.Client](ctx, nil)

	return f(policyID, client)
}

type bindingAddDeleteFn func(string, string, *apiclient.Client) error

func validateUserAndPolicyAndInvoke(ctx context.Context, cmd *cli.Command, f bindingAddDeleteFn) error {
	userName := cmd.StringArg("username")
	if userName == "" {
		return fmt.Errorf("%w: username", ErrMissingArgument)
	}

	policyID := cmd.StringArg("policy-id")
	if policyID == "" {
		return fmt.Errorf("%w: policy-id", ErrMissingArgument)
	}

	client := pal.MustInvoke[*apiclient.Client](ctx, nil)

	return f(userName, policyID, client)
}
