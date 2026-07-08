package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/89luca89/distrobox/internal/rootful"
	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/containermanager/providers"
	"github.com/89luca89/distrobox/pkg/ui"
	"github.com/89luca89/distrobox/pkg/version"
)

type contextKey string

const containerManagerKey contextKey = "containerManager"

// ResolveArgs supports v1-style subcommand invocation by basename. v1 shipped
// separate scripts (distrobox-create, distrobox-enter, …); v2 ships one binary
// with subcommands.
//
//	distrobox                       → no change
//	distrobox-enter foo             → distrobox enter foo
//	distrobox-;s                    → distrobox ls
//
// argv[0] is normalised to "distrobox" so help/usage output reads naturally
// regardless of which symlink was invoked.
func ResolveArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}
	sub, ok := strings.CutPrefix(filepath.Base(args[0]), "distrobox-")
	if !ok || sub == "" {
		return args
	}

	out := make([]string, 0, len(args)+1)
	out = append(out, "distrobox", sub)
	out = append(out, args[1:]...)
	return out
}

func NewRootCommand(cfg *config.Values) *cli.Command {
	subs := subcommands(cfg)
	// Install flag-aware completion on every command, including nested
	// `assemble create`/`assemble rm`. The default would only list
	// subcommand names; this also emits flag names.
	for _, sub := range subs {
		installShellCompleteRecursively(sub)
	}

	//nolint:reassign // urfave/cli/v3 explicitly supports reassigning VersionFlag
	cli.VersionFlag = &cli.BoolFlag{
		Name:        "version",
		Aliases:     []string{"V"},
		Usage:       "print the version",
		HideDefault: true,
		Local:       true,
	}

	return &cli.Command{
		Name:                  "distrobox",
		Usage:                 "Use any Linux distribution inside your terminal",
		Version:               version.Version,
		EnableShellCompletion: true,
		ShellComplete:         flagCompleter,
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
		Commands: subs,
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
		withSudoGuard,
		withRoot,
		withContainerManager,
	)

	generateEntry := cc.apply(
		newGenerateEntryCommand,
		withSudoGuard,
		withRoot,
		withContainerManager,
	)

	create := cc.apply(
		newCreateCommand,
		withSudoGuard,
		withRoot,
		withContainerManager,
	)

	enter := cc.apply(
		newEnterCommand,
		withSudoGuard,
		withRoot,
		withContainerManager,
	)

	assemble := cc.apply(
		newAssembleCommand,
		withAssembleSudoGuard,
		withContainerManager,
	)

	rm := cc.apply(
		newRmCommand,
		withSudoGuard,
		withRoot,
		withContainerManager,
	)

	stop := cc.apply(
		newStopCommand,
		withSudoGuard,
		withRoot,
		withContainerManager,
	)

	ephemeral := cc.apply(
		newEphemeralCommand,
		withSudoGuard,
		withRoot,
		withContainerManager,
	)

	upgrade := cc.apply(
		newUpgradeCommand,
		withSudoGuard,
		withRoot,
		withContainerManager,
	)

	return []*cli.Command{
		assemble,
		create,
		enter,
		ephemeral,
		generateEntry,
		list,
		rm,
		stop,
		upgrade,
	}
}

// withSudoGuard refuses to run a command invoked through sudo/doas as root,
// mirroring the reference shell (e.g. distrobox-create:45-51). A genuine root
// login shell (uid 0 without SUDO_USER/DOAS_USER) is still allowed and becomes
// rootful via withContainerManager.
func withSudoGuard(_ *config.Values, cmd *cli.Command) *cli.Command {
	return guardSudo(cmd, false)
}

// withAssembleSudoGuard is the assemble variant: it refuses whenever sudo/doas
// is detected, regardless of uid, and points at the manifest's root=true key
// (distrobox-assemble:77-81).
func withAssembleSudoGuard(_ *config.Values, cmd *cli.Command) *cli.Command {
	return guardSudo(cmd, true)
}

func guardSudo(cmd *cli.Command, assembleMode bool) *cli.Command {
	prev := cmd.Before
	cmd.Before = func(ctx context.Context, c *cli.Command) (context.Context, error) {
		if err := refuseSudo(c.Name, assembleMode); err != nil {
			return nil, err
		}
		if prev != nil {
			return prev(ctx, c)
		}
		return ctx, nil
	}
	return cmd
}

// refuseSudo implements the SUDO/DOAS guard. Standard commands refuse only when
// the real uid is 0 (sudo/doas elevation); a real root login shell is allowed.
// assemble refuses on any sudo/doas invocation.
func refuseSudo(commandName string, assembleMode bool) error {
	viaSudo := os.Getenv("SUDO_USER") != "" || os.Getenv("DOAS_USER") != ""
	if !viaSudo {
		return nil
	}
	if !assembleMode && os.Getuid() != 0 {
		return nil
	}

	p := ui.NewPrinter(os.Stderr, false)
	if assembleMode {
		p.Println("Running distrobox %s via SUDO/DOAS is not supported.", commandName)
		p.Println("Instead, please try using root=true property in the distrobox.ini file.")
	} else {
		p.Println("Running distrobox %s via SUDO/DOAS is not supported. Instead, please try running:", commandName)
		p.Println("  %s", sudoRootSuggestion(commandName))
	}

	return errors.New("running via SUDO/DOAS is not supported")
}

// sudoRootSuggestion rebuilds the invocation with --root inserted after the
// sub-command, mirroring the shell's `<name> --root <args>` hint.
func sudoRootSuggestion(commandName string) string {
	parts := []string{"distrobox"}
	inserted := false
	for _, arg := range os.Args[1:] {
		parts = append(parts, arg)
		if !inserted && arg == commandName {
			parts = append(parts, "--root")
			inserted = true
		}
	}
	if !inserted {
		parts = append(parts, "--root")
	}

	return strings.Join(parts, " ")
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
		// Running as uid 0 (a logged-in root shell) implies rootful without
		// --root and needs no sudo elevation, so skip validation in that case.
		if c.Bool("root") && os.Getuid() != 0 {
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
			c.Bool("root") || os.Getuid() == 0,
			cfg.UsernsNoLimit,
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
	usernsNoLimit bool,
) (containermanager.ContainerManager, error) {
	errPrinter := ui.NewPrinter(os.Stderr, true)

	switch containerManagerType {
	case "docker":
		return providers.NewDocker(root, sudoCommand, verbose), nil
	case "podman":
		return providers.NewPodman(root, sudoCommand, verbose, usernsNoLimit), nil
	case "podman-launcher":
		return providers.NewPodmanLauncher(root, sudoCommand, verbose, usernsNoLimit), nil
	case "autodetect", "":
		cm, err := providers.NewAutoDetect(root, sudoCommand, verbose, usernsNoLimit)
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
