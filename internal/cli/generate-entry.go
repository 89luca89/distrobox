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
)

func newGenerateEntryCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "generate-entry",
		Usage:   "Generate or delete distrobox entries",
		Version: "1.0.0",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "delete",
				Aliases: []string{"d"},
				Usage:   "delete the entry",
			},
			&cli.StringFlag{
				Name:    "icon",
				Aliases: []string{"i"},
				Usage:   "specify a custom icon (default auto)",
				Value:   "auto",
			},
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
				Usage:   "perform for all distroboxes",
			},
		},
		ArgsUsage: "container-name",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return generateEntryAction(ctx, cmd, cfg)
		},
	}
}

func generateEntryAction(ctx context.Context, cmd *cli.Command, cfg *config.Values) error {
	// The current executable is used as distrobox path
	distroboxPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get distrobox executable path: %w", err)
	}

	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	listCmd := commands.NewListCommand(cfg, containerManager)

	opts := &commands.GenerateEntryOptions{
		Verbose:       cmd.Bool("verbose"),
		Delete:        cmd.Bool("delete"),
		Root:          cmd.Bool("root"),
		DistroboxPath: distroboxPath,
	}
	if cmd.Bool("all") {
		opts.All = true
	} else {
		opts.ContainerName = cmd.Args().First()
		opts.Icon = cmd.String("icon")
	}

	genEntryCmd := commands.NewGenerateEntryCommand(cfg, listCmd)

	err = genEntryCmd.Execute(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to execute generate entry command: %w", err)
	}

	return nil
}
