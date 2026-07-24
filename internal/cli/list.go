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

func newListCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List distroboxes",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "no-color",
				Usage: "Disable color output",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return listAction(ctx, cmd, cfg)
		},
	}
}

func listAction(ctx context.Context, cmd *cli.Command, cfg *config.Values) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	listCmd := commands.NewListCommand(cfg, containerManager)
	result, err := listCmd.Execute(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute list command: %w", err)
	}

	noColor := cmd.Bool("no-color") || !isTerminal()
	printResult(result, noColor)

	return nil
}

func printResult(result *commands.ListResult, noColor bool) {
	rowFormat := "%-12s | %-20s | %-18s | %-30s\n"

	//nolint:forbidigo // Using fmt.Printf is acceptable here for CLI output
	fmt.Printf(rowFormat, "ID", "NAME", "STATUS", "IMAGE")

	for _, c := range result.Containers {
		var line string
		switch {
		case noColor:
			line = rowFormat
		case c.IsRunning():
			line = ui.Green(rowFormat)
		default:
			line = ui.Yellow(rowFormat)
		}

		//nolint:forbidigo // Using fmt.Printf is acceptable here for CLI output
		fmt.Printf(line, c.ID, c.Name, c.Status, c.Image)
	}
}

func isTerminal() bool {
	stat, _ := os.Stdout.Stat()
	return (stat.Mode() & os.ModeCharDevice) != 0
}
