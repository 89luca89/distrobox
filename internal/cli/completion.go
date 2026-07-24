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
	"fmt"

	"github.com/urfave/cli/v3"
)

// flagCompleter prints the command's subcommand names (with aliases) and
// long-form flag names to stdout, one per line. urfave/cli v3's default
// shell completion only emits subcommand names; this fills the gap so
// `distrobox <cmd> --<TAB>` completes flags too.
//
// The bash/zsh wrapper scripts filter candidates by the user's current
// prefix, so emitting the full set unconditionally is the simplest correct
// shape — bash compgen drops anything that doesn't start with `--` when the
// user typed `--`, and drops anything that doesn't match the subcommand
// prefix otherwise.
func flagCompleter(_ context.Context, c *cli.Command) {
	// VisibleCommands skips help and any Hidden subcommands, matching
	// what the user would see in `--help` output.
	for _, sub := range c.VisibleCommands() {
		fmt.Println(sub.Name) //nolint:forbidigo // completion output by design
		for _, alias := range sub.Aliases {
			fmt.Println(alias) //nolint:forbidigo // completion output by design
		}
	}
	// VisibleFlags skips Hidden flags (e.g. --container-manager,
	// --sudo-command, --generate-shell-completion).
	seen := make(map[string]struct{})
	for _, f := range c.VisibleFlags() {
		for _, name := range f.Names() {
			if len(name) <= 1 {
				continue
			}
			if _, dup := seen[name]; dup {
				continue
			}
			seen[name] = struct{}{}
			fmt.Println("--" + name) //nolint:forbidigo // completion output by design
		}
	}
}

// installShellCompleteRecursively walks a command tree and installs
// flagCompleter on every node, so nested subcommands (e.g.
// `distrobox assemble create`) get the same completion behaviour as
// the top-level subcommands.
func installShellCompleteRecursively(cmd *cli.Command) {
	cmd.ShellComplete = flagCompleter
	for _, sub := range cmd.Commands {
		installShellCompleteRecursively(sub)
	}
}
