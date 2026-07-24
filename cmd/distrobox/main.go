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

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/89luca89/distrobox/internal/cli"
	"github.com/89luca89/distrobox/pkg/config"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.LoadValues()
	if err != nil {
		//nolint:wrapcheck // main reports errors as-is
		return err
	}

	// SIGINT register
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cmd := cli.NewRootCommand(cfg)

	// PrepareArgs lets `enter`/`ephemeral` accept distrobox flags after the
	// container name by splicing a "--" in front of the custom command.
	args := cli.PrepareArgs(cmd, cli.ResolveArgs(os.Args))

	//nolint:wrapcheck // main reports errors as-is
	return cmd.Run(ctx, args)
}
