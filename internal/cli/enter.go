package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/89luca89/distrobox/pkg/commands"
	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

func newEnterCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name:  "enter",
		Usage: "Enter a distrobox",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "name",
				Aliases: []string{"n"},
				Sources: cli.EnvVars("DBX_CONTAINER_NAME"),
				Usage:   "name for the distrobox",
			},
			&cli.BoolFlag{
				Name:    "exec",
				Aliases: []string{"e"},
				Usage:   "end arguments: execute the rest as command to execute at login\n" +
					"(equivalent to the bare `--` separator; the custom command is\n" +
					"always taken from the positional args after the container name)",
			},
			&cli.BoolFlag{
				Name:    "dry-run",
				Aliases: []string{"d"},
				Usage:   "only print the container manager command generated",
			},
			&cli.BoolFlag{
				Name:    "clean-path",
				Aliases: []string{"c"},
				Sources: cli.EnvVars("DBX_CONTAINER_CLEAN_PATH"),
				Usage:   "reset PATH inside the container to FHS standard",
			},
			&cli.StringFlag{
				Name:    "additional-flags",
				Aliases: []string{"a"},
				Usage:   "additional flags to pass to the container manager command",
			},
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"y"},
				Sources: cli.EnvVars("DBX_NON_INTERACTIVE"),
				Usage:   "non-interactive, do not ask questions",
			},
			&cli.BoolFlag{
				Name:    "no-tty",
				Aliases: []string{"T", "H"},
				Usage:   "do not instantiate a tty",
			},
			&cli.BoolFlag{
				Name:    "no-workdir",
				Aliases: []string{"nw"},
				Usage:   "always start the container from container's home directory",
			},
		},
		UseShortOptionHandling: false,
		StopOnNthArg:           ptr(1),
		SkipFlagParsing:        false,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return enterAction(ctx, cmd, cfg)
		},
	}
}

func enterAction(ctx context.Context, cmd *cli.Command, cfg *config.Values) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	// Container name: --name flag takes priority, otherwise first positional arg.
	// Everything after the container name (or after --) is the custom command.
	//
	// The CLI is configured with StopOnNthArg: 1, so urfave/cli stops flag
	// parsing as soon as the first positional arg is seen. The trailing
	// positional args (which include the custom command and any -e/--exec
	// marker that came after the container name) are returned verbatim.
	containerName := cmd.String("name")

	args := cmd.Args().Slice()

	// If the user placed -e/--exec AFTER the container name, it lands in
	// the positional tail. In that case the first positional arg is still
	// the container name and the custom command starts right after the
	// marker. When the marker is consumed as a flag (i.e. it appeared
	// before the container name) the tail is just the custom command.
	markerIndex := findExecMarkerIndex(args)

	var customCommand []string
	switch {
	case markerIndex >= 0:
		// -e/--exec was placed after the container name.
		if containerName == "" {
			containerName = args[0]
		}
		customCommand = args[markerIndex+1:]
	case containerName == "" && len(args) > 0:
		containerName = args[0]
		customCommand = args[1:]
	default:
		customCommand = args
	}

	if containerName == "" {
		containerName = cfg.DefaultContainerName
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

	enterCmd := commands.NewEnterCommand(cfg, containerManager, progress, printer)
	_, err := enterCmd.Execute(ctx, options)
	if err != nil {
		return fmt.Errorf("failed to execute enter command: %w", err)
	}

	return nil
}
