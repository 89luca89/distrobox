package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/urfave/cli/v3"

	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/containermanager/providers"
	"github.com/89luca89/distrobox/pkg/ui"
)

type contextKey string

const containerManagerKey contextKey = "containerManager"

func NewRootCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "distrobox",
		Version: "1.0.0",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:   "container-manager",
				Usage:  "",
				Hidden: true,
				Value:  cfg.ContainerManagerType,
			},
			&cli.StringFlag{
				Name:   "sudo-command",
				Usage:  "",
				Hidden: true,
				Value:  cfg.SudoProgram,
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
				Value:   cfg.Verbose,
			},
		},
		Before: beforeAction,
		Commands: []*cli.Command{
			newListCommand(cfg),
			newGenerateEntryCommand(cfg),
			newCreateCommand(cfg),
			newEnterCommand(cfg),
			newAssembleCommand(cfg),
			newRmCommand(cfg),
			newStopCommand(cfg),
			newEphemeralCommand(cfg),
			newUpgradeCommand(cfg),
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

	errPrinter := ui.NewPrinter(os.Stderr, true)
	var containerManager containermanager.ContainerManager
	switch containerManagerType {
	case "docker":
		containerManager = providers.NewDocker(root, sudoCommand, verbose)
	case "podman":
		containerManager = providers.NewPodman(root, sudoCommand, verbose)
	case "podman-launcher":
		containerManager = providers.NewPodmanLauncher(root, sudoCommand, verbose)
	case "autodetect", "":
		var err error
		containerManager, err = providers.NewAutoDetect(root, sudoCommand, verbose)
		if err != nil {
			if errors.Is(err, providers.ErrNoContainerManager) {
				printMissingContainerManager(errPrinter)
			}

			return nil, fmt.Errorf("failed to auto-detect container manager: %w", err)
		}
	default:
		printInvalidContainerManager(errPrinter, containerManagerType)

		return nil, fmt.Errorf("invalid input %s", containerManagerType)
	}

	return context.WithValue(ctx, contextKey("containerManager"), containerManager), nil
}

func printMissingContainerManager(p *ui.Printer) {
	p.Println("Missing dependency: we need a container manager.")
	p.Println("Please install one of podman, podman-launcher, or docker.")
	p.Println("You can follow the documentation on:")
	p.Println("\tman distrobox-compatibility")
	p.Println("or:")
	p.Println("\thttps://github.com/89luca89/distrobox/blob/main/docs/compatibility.md")
}

func printInvalidContainerManager(p *ui.Printer, containerManagerType string) {
	p.Println("Invalid input %s.", containerManagerType)
	p.Println("The available choices are: 'autodetect', 'podman', 'podman-launcher', 'docker'")
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
