package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

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
		UseShortOptionHandling: true,
		SkipFlagParsing:        false,
		Action:                 enterAction,
	}
}

func enterAction(ctx context.Context, cmd *cli.Command) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	// Container name: --name flag takes priority, otherwise first positional arg.
	// Everything after the container name (or after --) is the custom command.
	containerName := cmd.String("name")
	var customCommand string

	args := cmd.Args().Slice()
	if containerName == "" && len(args) > 0 {
		containerName = args[0]
		args = args[1:]
	}

	if len(args) > 0 {
		customCommand = strings.Join(args, " ")
	}

	options := commands.EnterOptions{
		ContainerName:   containerName,
		AdditionalFlags: cmd.String("additional-flags"),
		CustomCommand:   customCommand,
		DryRun:          cmd.Bool("dry-run"),
		NoTTY:           cmd.Bool("no-tty"),
		CleanPath:       cmd.Bool("clean-path"),
		Verbose:         cmd.Bool("verbose"),
	}

	progress := ui.NewProgress(os.Stderr)
	printer := ui.NewPrinter(os.Stderr, true)

	enterCmd := commands.NewEnterCommand(containerManager, progress, printer)
	_, err := enterCmd.Execute(ctx, options)
	if err != nil {
		return fmt.Errorf("failed to execute enter command: %w", err)
	}

	return nil
}
