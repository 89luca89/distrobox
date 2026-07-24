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

func newRmCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name:  "rm",
		Usage: "Remove distroboxes",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
				Usage:   "delete all distroboxes",
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "force deletion",
			},
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"Y"},
				Value:   cfg.NonInteractive,
				Usage:   "non-interactive mode",
			},
			&cli.BoolFlag{
				Name:  "rm-home",
				Value: cfg.RmCustomHome,
				Usage: "Remove container's home directory",
			},
		},

		Action: func(ctx context.Context, cmd *cli.Command) error {
			return rmAction(ctx, cmd, cfg)
		},
	}
}

func rmAction(ctx context.Context, cmd *cli.Command, cfg *config.Values) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	names := cmd.Args().Slice()
	if len(names) == 0 && !cmd.Bool("all") {
		names = []string{cfg.DefaultContainerName}
	}

	options := commands.RmOptions{
		NoTTY:          cmd.Bool("yes"),
		Force:          cmd.Bool("force"),
		All:            cmd.Bool("all"),
		RemoveHome:     cmd.Bool("rm-home"),
		Verbose:        cmd.Bool("verbose"),
		ContainerNames: names,
	}

	prompter := ui.NewPrompter(*bufio.NewReader(os.Stdin), os.Stdout)
	printer := ui.NewPrinter(os.Stdout, true)

	rmCmd := commands.NewRmCommand(cfg, containerManager, prompter, printer)
	_, err := rmCmd.Execute(ctx, options)
	if errors.Is(err, commands.ErrRmAbortedByUser) {
		ui.NewPrinter(os.Stdout, true).Println("Aborted.")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to execute rm command: %w", err)
	}

	return nil
}
