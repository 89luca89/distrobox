package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/urfave/cli/v3"

	"github.com/89luca89/distrobox/pkg/commands"
	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

func newEphemeralCommand(cfg *config.Values) *cli.Command {
	createCmd := newCreateCommand(cfg)

	ignoredFlags := []string{
		"compatibility",
		"no-entry",
	}
	flags := make([]cli.Flag, 0, len(createCmd.Flags)+1)
	for _, f := range createCmd.Flags {
		if slices.Contains(ignoredFlags, f.Names()[0]) {
			continue
		}
		flags = append(flags, f)
	}
	// -e/--exec is the marker that tells distrobox to treat everything
	// after it as the custom command to run inside the ephemeral
	// container. It is a no-op marker from urfave/cli's point of view
	// because the custom command is always taken from the positional
	// args — we just need the alias to exist so the parser doesn't
	// reject the flag. The original distrobox-ephemeral bash script
	// accepts -e, --exec, and `--` interchangeably.
	flags = append(flags, &cli.BoolFlag{
		Name:    "exec",
		Aliases: []string{"e"},
		Usage: "end arguments: execute the rest as command to execute at login\n" +
			"(equivalent to the bare `--` separator; the custom command is\n" +
			"always taken from the positional args)",
	})

	return &cli.Command{
		Name:  "ephemeral",
		Usage: "create a temporary distrobox container that is automatically removed on exit",
		UsageText: `distrobox ephemeral [options] [-- command]

Examples:
    distrobox ephemeral
    distrobox ephemeral --image alpine:latest -- cat /etc/os-release
    distrobox ephemeral --root --image fedora:39
    distrobox ephemeral -- bash -c "echo hello"`,
		Flags: flags,
		// StopOnNthArg: 1 mirrors the semantics of the original bash
		// distrobox-ephemeral: every flag must appear before the first
		// positional arg, and everything from the first positional arg
		// onward is treated as the custom command. Without this, short
		// flags inherited from `distrobox-create` (e.g. -c for --clone)
		// would be eaten out of the custom command. The bare `--`
		// separator still works because urfave/cli checks for it before
		// honouring StopOnNthArg.
		UseShortOptionHandling: false,
		StopOnNthArg:           ptr(1),
		SkipFlagParsing:        false,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return ephemeralAction(ctx, cmd, cfg)
		},
	}
}

func ephemeralAction(ctx context.Context, cmd *cli.Command, cfg *config.Values) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	// The CLI is configured with StopOnNthArg: 1, so urfave/cli stops
	// flag parsing as soon as it sees the first positional arg. From
	// that point on, anything — including the -e/--exec marker, if the
	// user placed it after a positional arg — is captured verbatim into
	// the positional tail. We use findExecMarkerIndex to split the
	// custom command out of the tail.
	args := cmd.Args().Slice()
	markerIndex := findExecMarkerIndex(args)

	var customCommand []string
	if markerIndex >= 0 {
		customCommand = args[markerIndex+1:]
	} else {
		customCommand = args
	}

	opts := commands.EphemeralOptions{
		CreateOptions: commands.CreateOptions{
			ContainerImage:          cmd.String("image"),
			ContainerName:           cmd.String("name"),
			ContainerHostname:       cmd.String("hostname"),
			ContainerClone:          cmd.String("clone"),
			UnshareNetNs:            cmd.Bool("unshare-netns") || cmd.Bool("unshare-all"),
			UnshareDevsys:           cmd.Bool("unshare-devsys") || cmd.Bool("unshare-all"),
			UnshareGroups:           cmd.Bool("unshare-groups") || cmd.Bool("unshare-all") || cmd.Bool("init"),
			UnshareIpc:              cmd.Bool("unshare-ipc") || cmd.Bool("unshare-all"),
			UnshareProcess:          cmd.Bool("unshare-process") || cmd.Bool("unshare-all") || cmd.Bool("init"),
			AdditionalFlags:         cmd.StringSlice("additional-flags"),
			AdditionalVolumes:       cmd.StringSlice("volume"),
			AdditionalPackages:      cmd.StringSlice("additional-packages"),
			Nopasswd:                cmd.Bool("absolutely-disable-root-password-i-am-really-positively-sure"),
			ContainerUserCustomHome: cmd.String("home"),
			Init:                    cmd.Bool("init"),
			Nvidia:                  cmd.Bool("nvidia"),
			ContainerInitHook:       cmd.String("init-hooks"),
			ContainerPreInitHook:    cmd.String("pre-init-hooks"),
			ContainerPlatform:       cmd.String("platform"),
			GenerateEntry:           false,
			Rootful:                 cmd.Bool("root"),
		},
		CustomCommand: customCommand,
		DryRun:        cmd.Bool("dry-run"),
	}

	progress := ui.NewProgress(os.Stderr)
	printer := ui.NewPrinter(os.Stderr, true)
	prompter := ui.NewPrompter(*bufio.NewReader(os.Stdin), os.Stdout)

	ephemeralCmd := commands.NewEphemeralCommand(cfg, containerManager, progress, printer, prompter)

	err := ephemeralCmd.Execute(ctx, opts)
	if err != nil {
		return fmt.Errorf("ephemeral command failed: %w", err)
	}

	return nil
}
