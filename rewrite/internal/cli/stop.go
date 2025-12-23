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

func newStopCommand() *cli.Command {
	return &cli.Command{
		Name:  "stop",
		Usage: "stop running distrobox containers",
		UsageText: `distrobox stop [options] [container-name...]

Examples:
    distrobox stop container-name
    distrobox stop container1 container2
    distrobox stop --all
    distrobox stop --yes container-name`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
				Usage:   "stop all distroboxes",
			},
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"Y"},
				Usage:   "non-interactive, stop without asking",
			},
		},
		Action: stopAction,
	}
}

func stopAction(ctx context.Context, cmd *cli.Command) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	all := cmd.Bool("all")
	nonInteractive := cmd.Bool("yes")
	containerNames := cmd.Args().Slice()

	options := &commands.StopOptions{
		ContainerNames: containerNames,
		NonInteractive: nonInteractive,
		All:            all,
	}

	printer := ui.NewPrinter(os.Stdout, true)
	errPrinter := ui.NewPrinter(os.Stderr, true)
	prompter := ui.NewPrompter(*bufio.NewReader(os.Stdin), os.Stdout)

	stopCmd := commands.NewStopCommand(containerManager, prompter)

	err := stopCmd.Execute(ctx, options)

	if errors.Is(err, commands.ErrStopAbortedByUserError) {
		printer.Println("Aborted.")
		return nil
	}

	if errors.Is(err, commands.ErrEmptyContainerList) {
		errPrinter.Println("No containers found.")
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to stop containers: %w", err)
	}

	return nil
}
