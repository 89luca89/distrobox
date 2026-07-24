// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of the distrobox project:
//    https://github.com/89luca89/distrobox
//
// Copyright (C) 2021 distrobox contributors
//
// distrobox is free software; you can redistribute it and/or modify it
// under the terms of the GNU General Public License version 3
// as published by the Free Software Foundation.
//
// distrobox is distributed in the hope that it will be useful, but
// WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU
// General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with distrobox; if not, see <http://www.gnu.org/licenses/>.

package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/89luca89/distrobox/pkg/commands"
	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

func newEnterCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name:  "enter",
		Usage: "Enter a distrobox",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "name",
				Aliases: []string{"n"},
				Value:   cfg.ContainerName,
				Usage:   "name for the distrobox",
			},
			&cli.BoolFlag{
				Name:    "exec",
				Aliases: []string{"e"},
				Usage: "end arguments: execute the rest as command to execute at login\n" +
					"(equivalent to the bare `--` separator; the custom command is\n" +
					"always taken from the positional args after the container name)",
			},
			&cli.BoolFlag{
				Name:    "dry-run",
				Aliases: []string{"d"},
				Usage:   "only print the container manager command generated",
			},
			&cli.BoolFlag{
				Name:    "clean-path",
				Aliases: []string{"c"},
				Value:   cfg.CleanPath,
				Usage:   "reset PATH inside the container to FHS standard",
			},
			&cli.StringFlag{
				Name:    "additional-flags",
				Aliases: []string{"a"},
				Usage:   "additional flags to pass to the container manager command",
			},
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"y"},
				Value:   cfg.NonInteractive,
				Usage:   "non-interactive, do not ask questions",
			},
			&cli.BoolFlag{
				Name:    "no-tty",
				Aliases: []string{"T", "H"},
				Usage:   "do not instantiate a tty",
			},
			&cli.BoolFlag{
				Name:    "no-workdir",
				Aliases: []string{"nw"},
				Value:   cfg.SkipWorkDir,
				Usage:   "always start the container from container's home directory",
			},
		},
		UseShortOptionHandling: false,
		SkipFlagParsing:        false,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return enterAction(ctx, cmd, cfg)
		},
	}
}

func enterAction(ctx context.Context, cmd *cli.Command, cfg *config.Values) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	// --name (or DBX_CONTAINER_NAME) wins; otherwise the first positional is
	// the name. PrepareArgs (parse.go) already split the command off behind a
	// "--", so the tail is [name?] + command.
	containerName := cmd.String("name")

	args := cmd.Args().Slice()

	var customCommand []string
	if containerName == "" && len(args) > 0 {
		containerName = args[0]
		customCommand = args[1:]
	} else {
		customCommand = args
	}

	if containerName == "" {
		containerName = cfg.DefaultContainerName
	}

	dryRun := cmd.Bool("dry-run")

	// If the container is missing, offer to create it (auto-create when
	// non-interactive), mirroring the shell (distrobox-enter:660-699). Skipped in
	// dry-run, which builds the command from host state without the container.
	if !dryRun && !containerManager.Exists(ctx, containerName) {
		created, err := offerCreateMissing(ctx, cfg, containerManager, containerName, cmd.Bool("yes"))
		if err != nil {
			return err
		}
		if !created {
			return nil // user declined; guidance already printed
		}
	}

	options := commands.EnterOptions{
		ContainerName:   containerName,
		AdditionalFlags: cmd.String("additional-flags"),
		CustomCommand:   customCommand,
		DryRun:          dryRun,
		NoTTY:           cmd.Bool("no-tty"),
		CleanPath:       cmd.Bool("clean-path"),
		Verbose:         cmd.Bool("verbose"),
		NoWorkDir:       cmd.Bool("no-workdir"),
	}

	progress := ui.NewProgress(os.Stderr)
	printer := ui.NewPrinter(os.Stderr, true)

	enterCmd := commands.NewEnterCommand(cfg, containerManager, progress, printer)
	_, err := enterCmd.Execute(ctx, options)
	if err != nil {
		return fmt.Errorf("failed to execute enter command: %w", err)
	}

	return nil
}

// offerCreateMissing implements the shell's "Create it now, out of image …?"
// flow (distrobox-enter:660-699). It returns true when the container was
// created (so the caller proceeds to enter), false when the user declined.
func offerCreateMissing(
	ctx context.Context,
	cfg *config.Values,
	cm containermanager.ContainerManager,
	containerName string,
	nonInteractive bool,
) (bool, error) {
	image := cfg.DefaultContainerImage
	errPrinter := ui.NewPrinter(os.Stderr, true)

	if !nonInteractive {
		prompter := ui.NewPrompter(*bufio.NewReader(os.Stdin), os.Stderr)
		if !prompter.Prompt(fmt.Sprintf("Create it now, out of image %s?", image), true) {
			errPrinter.Println("Ok. For creating it, run this command:")
			errPrinter.Println("\tdistrobox create <name-of-container> --image <remote>/<docker>:<tag>")
			return false, nil
		}
	}

	errPrinter.Println("Creating the container %s", containerName)

	progress := ui.NewProgress(os.Stderr)
	prompter := ui.NewPrompter(*bufio.NewReader(os.Stdin), os.Stdout)
	createCmd := commands.NewCreateCommand(cfg, cm, progress, prompter)
	_, err := createCmd.Execute(ctx, commands.CreateOptions{
		ContainerName:  containerName,
		ContainerImage: image,
		GenerateEntry:  true,
		NonInteractive: true,
	})
	if err != nil {
		return false, fmt.Errorf("failed to create container %s: %w", containerName, err)
	}

	return true, nil
}
