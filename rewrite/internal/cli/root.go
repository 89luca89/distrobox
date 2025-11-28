package cli

import (
	"context"

	"github.com/urfave/cli/v3"

	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/containermanager/providers"
)

type contextKey string

const containerManagerKey contextKey = "containerManager"

func NewRootCommand() *cli.Command {
	return &cli.Command{
		Name:    "distrobox",
		Version: "1.0.0",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "container-manager",
				Usage:   "",
				Sources: cli.EnvVars("DBX_CONTAINER_MANAGER", "container_manager"),
				Hidden:  true,
			},
			&cli.StringFlag{
				Name:    "sudo-command",
				Usage:   "",
				Sources: cli.EnvVars("DBX_SUDO_COMMAND", "sudo_command"),
				Hidden:  true,
				Value:   "sudo",
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "show more verbosity",
				Sources: cli.EnvVars("DBX_VERBOSE", "verbose"),
			},
		},
		Before: beforeAction,
		Commands: []*cli.Command{
			newListCommand(),
		},
	}
}

func beforeAction(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	containerManagerType := cmd.String("container-manager")
	verbose := cmd.Bool("verbose")

	var containerManager containermanager.ContainerManager
	switch containerManagerType {
	case "docker":
		containerManager = providers.NewDocker(verbose)
	default:
		containerManager = providers.NewDocker(verbose)
	}

	return context.WithValue(ctx, contextKey("containerManager"), containerManager), nil
}
