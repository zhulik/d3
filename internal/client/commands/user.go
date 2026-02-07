package commands

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
	"github.com/zhulik/d3/internal/client/apiclient"
	"github.com/zhulik/pal"
)

var (
	UserCommand = &cli.Command{ //nolint:gochecknoglobals
		Name:    "user",
		Aliases: []string{"u"},
		Usage:   "manage users",
		Commands: []*cli.Command{
			userAdd,
			userUpdate,
			userDelete,
		},
	}

	userAdd = &cli.Command{ //nolint:gochecknoglobals
		Name:      "add",
		Aliases:   []string{"a"},
		Usage:     "Add user",
		Arguments: usernameArg,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return validateUsernameAndInvokeClient(ctx, cmd, func(username string, client *apiclient.Client) error {
				user, err := client.CreateUser(ctx, username)
				if err != nil {
					return err
				}

				fmt.Println("User created successfully")                 //nolint:forbidigo
				fmt.Printf("Access key: \"%s\"\n", user.AccessKeyID)     //nolint:forbidigo
				fmt.Printf("Secret key: \"%s\"\n", user.SecretAccessKey) //nolint:forbidigo

				return nil
			})
		},
	}

	userUpdate = &cli.Command{ //nolint:gochecknoglobals
		Name:      "update",
		Aliases:   []string{"u"},
		Usage:     "Update user credentials",
		Arguments: usernameArg,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return validateUsernameAndInvokeClient(ctx, cmd, func(username string, client *apiclient.Client) error {
				user, err := client.UpdateUser(ctx, username)
				if err != nil {
					return err
				}

				fmt.Println("User updated successfully")                 //nolint:forbidigo
				fmt.Printf("Access key: \"%s\"\n", user.AccessKeyID)     //nolint:forbidigo
				fmt.Printf("Secret key: \"%s\"\n", user.SecretAccessKey) //nolint:forbidigo

				return nil
			})
		},
	}

	userDelete = &cli.Command{ //nolint:gochecknoglobals
		Name:      "delete",
		Aliases:   []string{"d"},
		Usage:     "Delete user",
		Arguments: usernameArg,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return validateUsernameAndInvokeClient(ctx, cmd, func(username string, client *apiclient.Client) error {
				err := client.DeleteUser(ctx, username)
				if err != nil {
					return err
				}

				fmt.Println("User deleted successfully") //nolint:forbidigo

				return nil
			})
		},
	}

	usernameArg = []cli.Argument{ //nolint:gochecknoglobals
		&cli.StringArg{
			Name:   "username",
			Config: cli.StringConfig{},
		},
	}
)

type clientFn func(string, *apiclient.Client) error

func validateUsernameAndInvokeClient(ctx context.Context, cmd *cli.Command, f clientFn) error {
	username := cmd.StringArg("username")
	if username == "" {
		return fmt.Errorf("%w: username", ErrMissingArgument)
	}

	client := pal.MustInvoke[*apiclient.Client](ctx, nil)

	return f(username, client)
}
