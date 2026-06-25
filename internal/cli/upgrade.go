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

func newUpgradeCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name:  "upgrade",
		Usage: "upgrade packages inside distrobox containers",
		UsageText: `distrobox upgrade [options] [container-name...]

Examples:
    distrobox upgrade container-name
    distrobox upgrade container1 container2
    distrobox upgrade --all
    distrobox upgrade --all --running
    distrobox upgrade --yes container-name`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
				Usage:   "upgrade all distroboxes",
			},
			&cli.BoolFlag{
				Name:  "running",
				Usage: "upgrade only running distroboxes (requires --all)",
			},
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"Y"},
				Usage:   "accepted for compatibility; upgrades never prompt (matches the shell)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return upgradeAction(ctx, cmd, cfg)
		},
	}
}

func upgradeAction(ctx context.Context, cmd *cli.Command, cfg *config.Values) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	options := &commands.UpgradeOptions{
		ContainerNames: cmd.Args().Slice(),
		All:            cmd.Bool("all"),
		Running:        cmd.Bool("running"),
	}

	printer := ui.NewPrinter(os.Stdout, true)
	errPrinter := ui.NewPrinter(os.Stderr, true)
	progress := ui.NewProgress(os.Stderr)

	upgradeCmd := commands.NewUpgradeCommand(cfg, containerManager, progress, printer)

	err := upgradeCmd.Execute(ctx, options)

	if errors.Is(err, commands.ErrEmptyContainerList) {
		errPrinter.Println("No containers found.")
		return nil
	}

	if errors.Is(err, commands.ErrUpgradeNoContainerSpecified) {
		errPrinter.Println("Please specify the name of the container.")
		//nolint:wrapcheck // sentinel returned as-is so caller exits non-zero
		return err
	}

	if err != nil {
		return fmt.Errorf("failed to upgrade containers: %w", err)
	}

	return nil
}
