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
)

func newGenerateEntryCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name:  "generate-entry",
		Usage: "Generate or delete distrobox entries",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "delete",
				Aliases: []string{"d"},
				Usage:   "delete the entry",
			},
			&cli.StringFlag{
				Name:    "icon",
				Aliases: []string{"i"},
				Usage:   "specify a custom icon (default auto)",
				Value:   "auto",
			},
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
				Usage:   "perform for all distroboxes",
			},
		},
		ArgsUsage: "container-name",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return generateEntryAction(ctx, cmd, cfg)
		},
	}
}

func generateEntryAction(ctx context.Context, cmd *cli.Command, cfg *config.Values) error {
	// The current executable is used as distrobox path
	distroboxPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get distrobox executable path: %w", err)
	}

	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	listCmd := commands.NewListCommand(cfg, containerManager)

	opts := &commands.GenerateEntryOptions{
		Verbose:       cmd.Bool("verbose"),
		Delete:        cmd.Bool("delete"),
		Root:          cmd.Bool("root"),
		DistroboxPath: distroboxPath,
	}
	if cmd.Bool("all") {
		opts.All = true
	} else {
		opts.ContainerName = cmd.Args().First()
		opts.Icon = cmd.String("icon")
	}

	genEntryCmd := commands.NewGenerateEntryCommand(cfg, listCmd)

	err = genEntryCmd.Execute(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to execute generate entry command: %w", err)
	}

	return nil
}
