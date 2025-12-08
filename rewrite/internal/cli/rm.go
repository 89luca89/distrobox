package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/89luca89/distrobox/internal/prompt"
	"github.com/89luca89/distrobox/pkg/commands"
	"github.com/89luca89/distrobox/pkg/containermanager"
)

func newRmCommand() *cli.Command {
	return &cli.Command{
		Name:  "rm",
		Usage: "Remove distroboxes",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "all",
				Usage: "delete all distroboxes",
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "force deletion",
			},
			&cli.BoolFlag{
				Name:  "yes",
				Usage: "non-interactive mode",
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

	options := containermanager.RmOptions{
		NoTTY:      cmd.Bool("yes"),
		Force:      cmd.Bool("force"),
		All:        cmd.Bool("all"),
		RemoveHome: cmd.Bool("rm-home"),
	}

	removedContainers := cmd.Args().Slice()

	prompter := prompt.NewPrompter(*bufio.NewReader(os.Stdin), os.Stdout)

	rmCmd := commands.NewRmCommand(containerManager, options, prompter)
	_, err := rmCmd.Execute(ctx, removedContainers)
	if err != nil {
		return fmt.Errorf("failed to execute rm command: %w", err)
	}

	return nil
}
