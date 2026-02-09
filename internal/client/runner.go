package client

import (
	"context"
	"os"

	"github.com/urfave/cli/v3"
	"github.com/zhulik/d3/internal/client/commands"
)

type Runner struct{}

func (c *Runner) Run(ctx context.Context) error {
	return (&cli.Command{
		Name:  "d3cli",
		Usage: "D3 management client",
		Commands: []*cli.Command{
			commands.UserCommand,
			commands.BindingCommand,
		},
	}).Run(ctx, os.Args)
}
