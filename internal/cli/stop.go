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

func newStopCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name:  "stop",
		Usage: "stop running distrobox containers",
		UsageText: `distrobox stop [options] [container-name...]

Examples:
    distrobox stop container-name
    distrobox stop container1 container2
    distrobox stop --all
    distrobox stop --yes container-name`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
				Usage:   "stop all distroboxes",
			},
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"Y"},
				Value:   cfg.NonInteractive,
				Usage:   "non-interactive, stop without asking",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return stopAction(ctx, cmd, cfg)
		},
	}
}

func stopAction(ctx context.Context, cmd *cli.Command, cfg *config.Values) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	all := cmd.Bool("all")
	nonInteractive := cmd.Bool("yes")
	containerNames := cmd.Args().Slice()

	// Mirror shell distrobox-stop:90,200-202: when no positional and not --all,
	// fall back to the env-set container name. The env value comes from
	// cfg.ContainerName (resolved by pkg/config from DBX_CONTAINER_NAME); we
	// only consult it here, never read the env directly. An unset env leaves
	// containerNames empty so StopCommand applies its default-name fallback.
	if !all && len(containerNames) == 0 && cfg.ContainerName != "" {
		containerNames = []string{cfg.ContainerName}
	}

	options := &commands.StopOptions{
		ContainerNames: containerNames,
		NonInteractive: nonInteractive,
		All:            all,
	}

	printer := ui.NewPrinter(os.Stdout, true)
	errPrinter := ui.NewPrinter(os.Stderr, true)
	prompter := ui.NewPrompter(*bufio.NewReader(os.Stdin), os.Stdout)

	stopCmd := commands.NewStopCommand(cfg, containerManager, prompter)

	err := stopCmd.Execute(ctx, options)

	if errors.Is(err, commands.ErrStopAbortedByUserError) {
		printer.Println("Aborted.")
		return nil
	}

	if errors.Is(err, commands.ErrEmptyContainerList) {
		errPrinter.Println("No containers found.")
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to stop containers: %w", err)
	}

	return nil
}
