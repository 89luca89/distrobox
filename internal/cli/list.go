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

func newListCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name:      "list",
		Aliases:   []string{"ls"},
		Usage:     "List distroboxes",
		ArgsUsage: "[container-name]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "no-color",
				Usage: "Disable color output",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return listAction(ctx, cmd, cfg)
		},
	}
}

func listAction(ctx context.Context, cmd *cli.Command, cfg *config.Values) error {
	args := cmd.Args().Slice()
	if len(args) > 1 {
		return fmt.Errorf("expected at most 1 container name, got %d", len(args))
	}

	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	listCmd := commands.NewListCommand(cfg, containerManager)
	result, err := listCmd.Execute(ctx, &commands.ListOptions{ContainerName: cmd.Args().First()})
	if err != nil {
		return fmt.Errorf("failed to execute list command: %w", err)
	}

	noColor := cmd.Bool("no-color") || !isTerminal()
	printResult(result, noColor)

	return nil
}

func printResult(result *commands.ListResult, noColor bool) {
	rowFormat := "%-12s | %-20s | %-18s | %-30s\n"

	//nolint:forbidigo // Using fmt.Printf is acceptable here for CLI output
	fmt.Printf(rowFormat, "ID", "NAME", "STATUS", "IMAGE")

	for _, c := range result.Containers {
		var line string
		switch {
		case noColor:
			line = rowFormat
		case c.IsRunning():
			line = ui.Green(rowFormat)
		default:
			line = ui.Yellow(rowFormat)
		}

		//nolint:forbidigo // Using fmt.Printf is acceptable here for CLI output
		fmt.Printf(line, c.ID, c.Name, c.Status, c.Image)
	}
}

func isTerminal() bool {
	stat, _ := os.Stdout.Stat()
	return (stat.Mode() & os.ModeCharDevice) != 0
}
