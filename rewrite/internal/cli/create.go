package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/89luca89/distrobox/pkg/commands"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

//nolint:funlen // function length is acceptable for CLI command definition
func newCreateCommand() *cli.Command {
	return &cli.Command{

		Name:  "create",
		Usage: "create a new distrobox container",
		UsageText: `distrobox create [options] [container_name]

Examples:
    distrobox create --image alpine:latest --name test --init-hooks "touch /var/tmp/test1 && touch /var/tmp/test2"
    distrobox create --image fedora:39 --name test --additional-flags "--env MY_VAR=value"
    distrobox create --image fedora:39 --name test --volume /opt/my-dir:/usr/local/my-dir:rw --additional-flags "--pids-limit 100"
    distrobox create -i docker.io/almalinux/8-init --init --name test --pre-init-hooks "dnf config-manager --enable powertools && dnf -y install epel-release"
    distrobox create --clone fedora-39 --name fedora-39-copy
    distrobox create --image alpine my-alpine-container
    distrobox create --image registry.fedoraproject.org/fedora-toolbox:latest --name fedora-toolbox-latest
    distrobox create --pull --image centos:stream9 --home ~/distrobox/centos9
    distrobox create --image alpine:latest --name test2 --additional-packages "git tmux vim"
    distrobox create --image ubuntu:22.04 --name ubuntu-nvidia --nvidia

    DBX_NON_INTERACTIVE=1 DBX_CONTAINER_NAME=test-alpine DBX_CONTAINER_IMAGE=alpine distrobox-create`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "image",
				Aliases: []string{"i"},
				Usage: fmt.Sprintf(
					"image to use for the container (default: %s)",
					commands.DefaultCreateContainerImage,
				),
				Value: commands.DefaultCreateContainerImage,
			},
			&cli.StringFlag{
				Name:    "name",
				Aliases: []string{"n"},
				Usage:   fmt.Sprintf("name for the distrobox (default: %s)", commands.DefaultCreateContainerName),
			},
			&cli.StringFlag{
				Name:  "hostname",
				Usage: "hostname for the distrobox",
			},
			&cli.BoolFlag{
				Name:    "pull",
				Aliases: []string{"p"},
				Usage:   "pull the image even if it exists locally (implies --yes)",
			},
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"Y"},
				Usage:   "non-interactive, pull images without asking",
			},
			&cli.BoolFlag{
				Name:    "root",
				Aliases: []string{"r"},
				Usage: `launch podman/docker/lilipod with root privileges. This is the only supported way to run with root
privileges. Do not use "sudo distrobox". If you need to specify a different program (e.g. 'doas') for root privileges,
use the DBX_SUDO_PROGRAM environment variable or the 'distrobox_sudo_program' config variable.`,
			},
			&cli.StringFlag{
				Name:    "clone",
				Aliases: []string{"c"},
				Usage: `name of the distrobox container to use as base for a new container
this will be useful to either rename an existing distrobox or have multiple copies
of the same environment.`,
			},
			&cli.StringFlag{
				Name:    "home",
				Aliases: []string{"H"},
				Usage:   "select a custom HOME directory for the container. Useful to avoid host's home littering with temp files.",
			},
			&cli.StringSliceFlag{
				Name:  "volume",
				Usage: "additional volumes to add to the container",
			},
			&cli.StringSliceFlag{
				Name:    "additional-flags",
				Aliases: []string{"a"},
				Usage:   "additional flags to pass to the container manager command",
			},
			&cli.StringSliceFlag{
				Name:    "additional-packages",
				Aliases: []string{"ap"},
				Usage:   "additional packages to install during initial container setup",
			},
			&cli.StringFlag{
				Name:  "init-hooks",
				Usage: "additional commands to execute at the end of container initialization",
			},
			&cli.StringFlag{
				Name:  "pre-init-hooks",
				Usage: "additional commands to execute at the start of container initialization",
			},
			&cli.BoolFlag{
				Name:    "init",
				Aliases: []string{"I"},
				Usage: `use init system (like systemd) inside the container.
this will make host's processes not visible from within the container. (assumes --unshare-process)
may require additional packages depending on the container image: https://github.com/89luca89/distrobox/blob/main/docs/useful_tips.md#using-init-system-inside-a-distrobox`,
			},
			&cli.BoolFlag{
				Name:  "nvidia",
				Usage: "try to integrate host's nVidia drivers in the guest",
			},
			&cli.StringFlag{
				Name:  "platform",
				Usage: "specify which platform to use, eg: linux/arm64",
			},
			&cli.BoolFlag{
				Name:  "unshare-devsys",
				Usage: "do not share host devices and sysfs dirs from host",
			},
			&cli.BoolFlag{
				Name:  "unshare-groups",
				Usage: "do not forward user's additional groups into the container",
			},
			&cli.BoolFlag{
				Name:  "unshare-ipc",
				Usage: "do not share ipc namespace with host",
			},
			&cli.BoolFlag{
				Name:  "unshare-netns",
				Usage: "do not share the net namespace with host",
			},
			&cli.BoolFlag{
				Name:  "unshare-process",
				Usage: "do not share process namespace with host",
			},
			&cli.BoolFlag{
				Name:  "unshare-all",
				Usage: "activate all the unshare flags below",
			},
			&cli.BoolFlag{
				Name:  "no-entry",
				Usage: "do not generate a container entry in the application list",
			},
			&cli.BoolFlag{
				Name:    "dry-run",
				Aliases: []string{"d"},
				Usage:   "only print the container manager command generated",
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "show more verbosity",
			},
			&cli.BoolFlag{
				Name:  "absolutely-disable-root-password-i-am-really-positively-sure",
				Usage: `⚠️ ⚠️ when setting up a rootful distrobox, this will skip user password setup, leaving it blank. ⚠️ ⚠️`,
			},
			&cli.BoolFlag{
				Name:    "compatibility",
				Aliases: []string{"C"},
				Usage:   "show compatibility information and exit",
			},
		},
		Action: createAction,
	}
}

func createAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.Bool("compatibility") {
		err := showCompatibility()
		if err != nil {
			return fmt.Errorf("compatibility check failed: %w", err)
		}
		return nil
	}

	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	opts := commands.CreateOptions{
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
		DryRun:                  cmd.Bool("dry-run"),
		GenerateEntry:           !cmd.Bool("no-entry"),
		Rootful:                 cmd.Bool("root"),
	}

	progress := ui.NewProgress(os.Stderr)

	createCmd := commands.NewCreateCommand(containerManager, progress)
	err := createCmd.Execute(ctx, opts)
	if err != nil {
		return fmt.Errorf("create command failed: %w", err)
	}

	if !opts.DryRun {
		printCreateCompleted(progress, opts.ContainerName, opts.Rootful)
	}

	return nil
}

func showCompatibility() error {
	// TODO: fetch compatibility
	// https://github.com/89luca89/distrobox/blob/main/distrobox-create#L254
	return nil
}

func printCreateCompleted(progress *ui.Progress, containerName string, rootful bool) {
	rootFlag := ""
	if rootful {
		rootFlag = "--root "
	}

	msg := "Distrobox '%s' successfully created.\nTo enter, run:\n\ndistrobox enter %s%s\n\n"

	progress.Finalize(msg, containerName, rootFlag, containerName)
}
