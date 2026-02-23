package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/89luca89/distrobox/pkg/commands"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

func newUpgradeCommand() *cli.Command {
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
				Usage:   "non-interactive, upgrade without asking",
			},
		},
		Action: upgradeAction,
	}
}

func upgradeAction(ctx context.Context, cmd *cli.Command) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	options := &commands.UpgradeOptions{
		ContainerNames: cmd.Args().Slice(),
		All:            cmd.Bool("all"),
		Running:        cmd.Bool("running"),
		NonInteractive: cmd.Bool("yes"),
	}

	printer := ui.NewPrinter(os.Stdout, true)
	errPrinter := ui.NewPrinter(os.Stderr, true)
	progress := ui.NewProgress(os.Stderr)
	prompter := ui.NewPrompter(*bufio.NewReader(os.Stdin), os.Stdout)

	upgradeCmd := commands.NewUpgradeCommand(containerManager, progress, printer, prompter)

	err := upgradeCmd.Execute(ctx, options)

	if errors.Is(err, commands.ErrUpgradeAbortedByUser) {
		printer.Println("Aborted.")
		return nil
	}

	if errors.Is(err, commands.ErrEmptyContainerList) {
		errPrinter.Println("No containers found.")
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to upgrade containers: %w", err)
	}

	return nil
}
