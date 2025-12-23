package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/89luca89/distrobox/pkg/commands"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

func newEnterCommand() *cli.Command {
	return &cli.Command{
		Name:  "enter",
		Usage: "Enter a distrobox",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "name",
				Aliases: []string{"n"},
				Usage:   "name for the distrobox",
			},
			&cli.BoolFlag{
				Name:    "dry-run",
				Aliases: []string{"d"},
				Usage:   "only print the container manager command generated",
			},
			&cli.BoolFlag{
				Name:    "clean-path",
				Aliases: []string{"c"},
				Usage:   "only print the container manager command generated",
			},
			&cli.StringFlag{
				Name:    "additional-flags",
				Aliases: []string{"a"},
				Usage:   "additional flags to pass to the container manager command",
			},
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"y"},
				Usage:   "only print the container manager command generated",
			},
			&cli.BoolFlag{
				Name:    "no-tty",
				Aliases: []string{"T", "H"},
				Usage:   "show more verbosity",
			},
			&cli.BoolFlag{
				Name:    "no-workdir",
				Aliases: []string{"nw"},
				Usage:   "always start the container from container's home directory",
			},
		},
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name: "name",
			},
		},
		Action: enterAction,
	}
}

func enterAction(ctx context.Context, cmd *cli.Command) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	containerName := getContainerName(cmd)

	options := commands.EnterOptions{
		ContainerName:   containerName,
		AdditionalFlags: cmd.String("additional-flags"),
		NoTTY:           cmd.Bool("no-tty"),
		CleanPath:       cmd.Bool("clean-path"),
		Verbose:         cmd.Bool("verbose"),
	}

	progress := ui.NewProgress(os.Stderr)
	printer := ui.NewPrinter(os.Stderr, true)

	enterCmd := commands.NewEnterCommand(containerManager, progress, printer)
	_, err := enterCmd.Execute(ctx, options)
	if err != nil {
		return fmt.Errorf("failed to execute create command: %w", err)
	}

	return nil
}

func getContainerName(cmd *cli.Command) string {
	argName := cmd.StringArg("name")
	if len(argName) == 0 {
		return cmd.String("name")
	}
	return argName
}
