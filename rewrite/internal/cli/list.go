package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/89luca89/distrobox/pkg/commands"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/urfave/cli/v3"
)

const (
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorReset  = "\033[0m"
)

func newListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List distroboxes",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "no-color",
				Usage: "Disable color output",
			},
		},
		Action: listAction,
	}
}

func listAction(ctx context.Context, cmd *cli.Command) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return fmt.Errorf("container manager not found in context")
	}

	listCmd := commands.NewListCommand(containerManager)
	result, err := listCmd.Execute(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute list command: %w", err)
	}

	noColor := cmd.Bool("no-color") || !isTerminal()
	printResult(result, noColor)

	return nil
}

func printResult(result *commands.ListResult, noColor bool) {
	fmt.Printf("%-12s | %-20s | %-18s | %-30s\n",
		"ID", "NAME", "STATUS", "IMAGE")

	for _, c := range result.Containers {
		if noColor {
			fmt.Printf("%-12s | %-20s | %-18s | %-30s\n",
				c.ID, c.Name, c.Status, c.Image)
		} else {
			color := colorYellow
			if c.IsRunning() {
				color = colorGreen
			}
			fmt.Printf("%s%-12s | %-20s | %-18s | %-30s%s\n",
				color, c.ID, c.Name, c.Status, c.Image, colorReset)
		}
	}
}

func isTerminal() bool {
	stat, _ := os.Stdout.Stat()
	return (stat.Mode() & os.ModeCharDevice) != 0
}

