package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"

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
				Name: "root",
				Usage: "launch podman/docker/lilipod with root privileges. Note that if you need root this is the preferred\n" +
					"way over \"sudo distrobox\" (note: if using a program other than 'sudo' for root privileges is necessary,\n" +
					"specify it through the DBX_SUDO_PROGRAM env variable, or 'distrobox_sudo_program' config variable)",
				Aliases: []string{"r"},
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
			newGenerateEntryCommand(),
			newCreateCommand(),
			newEnterCommand(),
			newAssembleCommand(),
			newRmCommand(),
			newStopCommand(),
			newEphemeralCommand(),
			newUpgradeCommand(),
		},
	}
}

func beforeAction(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	root := cmd.Bool("root")
	if root {
		if err := validateSudo(ctx); err != nil {
			return nil, fmt.Errorf("cannot run in root mode: %w", err)
		}
	}
	sudoCommand := cmd.String("sudo-command")
	containerManagerType := cmd.String("container-manager")
	verbose := cmd.Bool("verbose")

	var containerManager containermanager.ContainerManager
	switch containerManagerType {
	case "docker":
		containerManager = providers.NewDocker(root, sudoCommand, verbose)
	case "podman", "podman-static", "":
		containerManager = providers.NewPodman(root, sudoCommand, verbose)
	default:
		return nil, fmt.Errorf("unsupported container manager: %s", containerManagerType)
	}

	return context.WithValue(ctx, contextKey("containerManager"), containerManager), nil
}

func validateSudo(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "sudo", "-v")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to validate sudo: %w", err)
	}

	return nil
}
