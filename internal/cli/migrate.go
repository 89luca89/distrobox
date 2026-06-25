package cli

import (
	"bufio"
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

func newMigrateCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name:  "migrate",
		Usage: "migrate an old (v1) distrobox container to work with v2",
		UsageText: `distrobox migrate [container-name | --all] [options]

Examples:
    distrobox migrate my-box
    distrobox migrate --all
    distrobox migrate --dry-run my-box
    distrobox migrate --force --yes my-box

Migrate containers created with distrobox v1 to work with v2.
This recreates the container with updated mount points for the v2 script locations.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
				Usage:   "migrate all distrobox containers that need it",
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "skip the already-migrated check and recreate anyway",
			},
			&cli.BoolFlag{
				Name:    "dry-run",
				Aliases: []string{"d"},
				Usage:   "print the commands that would be executed, do nothing",
			},
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"Y"},
				Value:   cfg.NonInteractive,
				Usage:   "non-interactive, do not ask for confirmation",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return migrateAction(ctx, cmd, cfg)
		},
	}
}

func migrateAction(ctx context.Context, cmd *cli.Command, cfg *config.Values) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	all := cmd.Bool("all")
	containerNames := cmd.Args().Slice()

	// Fall back to default container name when neither --all nor positional args
	// are given, matching stop/rm behavior.
	if !all && len(containerNames) == 0 && cfg.ContainerName != "" {
		containerNames = []string{cfg.ContainerName}
	}

	options := commands.MigrateOptions{
		ContainerNames: containerNames,
		All:            all,
		Force:          cmd.Bool("force"),
		DryRun:         cmd.Bool("dry-run"),
		NonInteractive: cmd.Bool("yes"),
	}

	printer := ui.NewPrinter(os.Stderr, true)
	prompter := ui.NewPrompter(*bufio.NewReader(os.Stdin), os.Stdout)

	migrateCmd := commands.NewMigrateCommand(cfg, containerManager, printer, prompter)
	err := migrateCmd.Execute(ctx, options)

	if errors.Is(err, commands.ErrEmptyContainerList) {
		errPrinter := ui.NewPrinter(os.Stderr, true)
		errPrinter.Println("No containers found.")
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to execute migrate command: %w", err)
	}

	return nil
}
