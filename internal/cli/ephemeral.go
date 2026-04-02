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
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

func newEphemeralCommand() *cli.Command {
	createCmd := newCreateCommand()

	ignoredFlags := []string{
		"compatibility",
		"no-entry",
	}
	flags := make([]cli.Flag, 0, len(createCmd.Flags))
	for _, f := range createCmd.Flags {
		if slices.Contains(ignoredFlags, f.Names()[0]) {
			continue
		}
		flags = append(flags, f)
	}

	return &cli.Command{
		Name:  "ephemeral",
		Usage: "create a temporary distrobox container that is automatically removed on exit",
		UsageText: `distrobox ephemeral [options] [-- command]

Examples:
    distrobox ephemeral
    distrobox ephemeral --image alpine:latest -- cat /etc/os-release
    distrobox ephemeral --root --image fedora:39
    distrobox ephemeral -- bash -c "echo hello"`,
		Flags:  flags,
		Action: ephemeralAction,
	}
}

func ephemeralAction(ctx context.Context, cmd *cli.Command) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
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
		DryRun: cmd.Bool("dry-run"),
	}

	progress := ui.NewProgress(os.Stderr)
	printer := ui.NewPrinter(os.Stderr, true)
	prompter := ui.NewPrompter(*bufio.NewReader(os.Stdin), os.Stdout)

	ephemeralCmd := commands.NewEphemeralCommand(containerManager, progress, printer, prompter)

	err := ephemeralCmd.Execute(ctx, opts)
	if err != nil {
		return fmt.Errorf("ephemeral command failed: %w", err)
	}

	return nil
}
