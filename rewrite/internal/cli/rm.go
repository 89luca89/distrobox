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

func newRmCommand() *cli.Command {
	return &cli.Command{
		Name:  "rm",
		Usage: "Remove distroboxes",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
				Usage:   "delete all distroboxes",
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "force deletion",
			},
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"Y"},
				Usage:   "non-interactive mode",
			},
			&cli.BoolFlag{
				Name:  "rm-home",
				Usage: "Remove container's home directory",
			},
		},

		Action: rmAction,
	}
}

func rmAction(ctx context.Context, cmd *cli.Command) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	options := commands.RmOptions{
		NoTTY:          cmd.Bool("yes"),
		Force:          cmd.Bool("force"),
		All:            cmd.Bool("all"),
		RemoveHome:     cmd.Bool("rm-home"),
		ContainerNames: cmd.Args().Slice(),
	}

	prompter := ui.NewPrompter(*bufio.NewReader(os.Stdin), os.Stdout)

	rmCmd := commands.NewRmCommand(containerManager, prompter)
	_, err := rmCmd.Execute(ctx, options)
	if err != nil {
		return fmt.Errorf("failed to execute rm command: %w", err)
	}

	return nil
}
