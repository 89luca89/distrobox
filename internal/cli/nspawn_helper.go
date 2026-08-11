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
	"os"

	cli "github.com/urfave/cli/v3"

	"github.com/89luca89/distrobox/pkg/containermanager/providers/nspawn"
)

// newNspawnHelperCommand is the hidden, root-only half of the nspawn
// backend. Rootful mode re-executes distrobox through the sudo program with
// this subcommand for the operations that run in-process and therefore
// cannot be sudo-prefixed like podman/docker invocations: image pulls into
// /var/cache/distrobox and machine materialization/removal under
// /var/lib/distrobox.
func newNspawnHelperCommand() *cli.Command {
	return &cli.Command{
		Name:   nspawn.HelperCommand,
		Hidden: true,
		Usage:  "internal helper for the rootful nspawn container manager",
		Commands: []*cli.Command{
			{
				Name:  "create-machine",
				Usage: "materialize a machine from a JSON definition on stdin",
				Action: func(ctx context.Context, _ *cli.Command) error {
					return nspawn.HelperCreateMachine(ctx, os.Stdin)
				},
			},
			{
				Name:  "remove-machine",
				Usage: "remove a machine directory by name",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "name", Required: true},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					return nspawn.HelperRemoveMachine(ctx, c.String("name"))
				},
			},
			{
				Name:  "pull",
				Usage: "pull an image into the rootful image store",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "image", Required: true},
					&cli.StringFlag{Name: "platform"},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					return nspawn.HelperPullImage(ctx, c.String("image"), c.String("platform"))
				},
			},
			{
				Name:  "image-exists",
				Usage: "exit 0 when the image exists in the rootful image store",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "image", Required: true},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					return nspawn.HelperImageExists(ctx, c.String("image"))
				},
			},
			{
				Name:  "clean-runtime",
				Usage: "remove stale nspawn runtime state for an inactive machine",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "name", Required: true},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					return nspawn.HelperCleanRuntime(ctx, c.String("name"))
				},
			},
		},
	}
}
