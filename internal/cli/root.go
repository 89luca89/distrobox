package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/89luca89/distrobox/internal/rootful"
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
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "show more verbosity",
				Value:   cfg.Verbose,
			},
			&cli.StringFlag{
				Name:   "sudo-command",
				Usage:  "",
				Hidden: true,
				Value:  cfg.SudoProgram,
			},
		},
		Commands: subcommands(cfg),
	}
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

func subcommands(cfg *config.Values) []*cli.Command {
	cc := &CommandComposer[config.Values]{cfg: cfg}

	list := cc.apply(
		newListCommand,
		withRoot,
		withContainerManager,
	)

	generateEntry := cc.apply(
		newGenerateEntryCommand,
		withRoot,
		withContainerManager,
	)

	create := cc.apply(
		newCreateCommand,
		withRoot,
		withContainerManager,
	)

	enter := cc.apply(
		newEnterCommand,
		withRoot,
		withContainerManager,
	)

	assemble := cc.apply(
		newAssembleCommand,
		withContainerManager,
	)

	rm := cc.apply(
		newRmCommand,
		withRoot,
		withContainerManager,
	)

	stop := cc.apply(
		newStopCommand,
		withRoot,
		withContainerManager,
	)

	ephemeral := cc.apply(
		newEphemeralCommand,
		withRoot,
		withContainerManager,
	)

	upgrade := cc.apply(
		newUpgradeCommand,
		withRoot,
		withContainerManager,
	)

	return []*cli.Command{
		list,
		generateEntry,
		create,
		enter,
		assemble,
		rm,
		stop,
		ephemeral,
		upgrade,
	}
}

// withRoot declares the --root flag on a command and, when it is set,
// validates that sudo is usable. The actual root-mode container manager is
// built by withContainerManager, which reads the same flag.
func withRoot(_ *config.Values, cmd *cli.Command) *cli.Command {
	cmd.Flags = append(cmd.Flags, &cli.BoolFlag{
		Name:    "root",
		Aliases: []string{"r"},
		Usage: "launch podman/docker/lilipod with root privileges. Note that if you need root this is the preferred\n" +
			"way over \"sudo distrobox\" (note: if using a program other than 'sudo' for root privileges is necessary,\n" +
			"specify it through the DBX_SUDO_PROGRAM env variable, or 'distrobox_sudo_program' config variable)",
	})

	prev := cmd.Before
	cmd.Before = func(ctx context.Context, c *cli.Command) (context.Context, error) {
		if prev != nil {
			var err error
			ctx, err = prev(ctx, c)
			if err != nil {
				return nil, err
			}
		}
		if c.Bool("root") {
			if err := rootful.Validate(ctx, c.String("sudo-command")); err != nil {
				return nil, fmt.Errorf("cannot run in root mode: %w", err)
			}
		}
		return ctx, nil
	}
	return cmd
}

// withContainerManager builds the container manager for a command and stores
// it in the context. It reads --root from the command (zero value if the flag
// is not declared), so commands without withRootSupport always get a rootless
// manager.
func withContainerManager(cfg *config.Values, cmd *cli.Command) *cli.Command {
	cmd.Flags = append(cmd.Flags, &cli.StringFlag{
		Name:   "container-manager",
		Usage:  "",
		Hidden: true,
		Value:  cfg.ContainerManagerType,
	})

	prev := cmd.Before
	cmd.Before = func(ctx context.Context, c *cli.Command) (context.Context, error) {
		if prev != nil {
			var err error
			ctx, err = prev(ctx, c)
			if err != nil {
				return nil, err
			}
		}
		cm, err := buildContainerManager(
			ctx,
			c.String("container-manager"),
			c.String("sudo-command"),
			c.Bool("verbose"),
			c.Bool("root"),
		)
		if err != nil {
			return nil, err
		}
		return context.WithValue(ctx, containerManagerKey, cm), nil
	}
	return cmd
}

func buildContainerManager(
	_ context.Context,
	containerManagerType string,
	sudoCommand string,
	verbose bool,
	root bool,
) (containermanager.ContainerManager, error) {
	errPrinter := ui.NewPrinter(os.Stderr, true)

	switch containerManagerType {
	case "docker":
		return providers.NewDocker(root, sudoCommand, verbose), nil
	case "podman":
		return providers.NewPodman(root, sudoCommand, verbose), nil
	case "podman-launcher":
		return providers.NewPodmanLauncher(root, sudoCommand, verbose), nil
	case "autodetect", "":
		cm, err := providers.NewAutoDetect(root, sudoCommand, verbose)
		if err != nil {
			if errors.Is(err, providers.ErrNoContainerManager) {
				printMissingContainerManager(errPrinter)
			}
			return nil, fmt.Errorf("failed to auto-detect container manager: %w", err)
		}
		return cm, nil
	default:
		printInvalidContainerManager(errPrinter, containerManagerType)
		return nil, fmt.Errorf("invalid input %s", containerManagerType)
	}
}

// CommandComposer is a helper for building commands with options.
// It holds the config, so that options don't need to receive it as an argument.
type CommandComposer[CFG any] struct {
	cfg *CFG
}

// apply builds a command factory by applying options left-to-right.
// Order matches Before execution: the first option's work runs first.
func (cc *CommandComposer[CFG]) apply(factory func(*CFG) *cli.Command, options ...func(*CFG, *cli.Command) *cli.Command) *cli.Command {
	cmd := factory(cc.cfg)
	for _, option := range options {
		cmd = option(cc.cfg, cmd)
	}
	return cmd
}
