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

package nspawn

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// runOptions controls how an external command is executed.
type runOptions struct {
	// DryRun prints the command instead of executing it.
	DryRun bool
	// Interactive attaches the command to the caller's stdio.
	Interactive bool
	// TailLogs streams output to stdout/stderr while also capturing it.
	TailLogs bool
	// Privileged prefixes the command with the sudo program when the
	// manager is in root mode. Unlike podman/docker — where every engine
	// call goes through the engine binary — nspawn talks to systemd, and
	// reading unit or machine state needs no privileges even for system
	// units, so only mutating commands set this.
	Privileged bool
}

// run executes argv (argv[0] is the executable). In root mode privileged
// commands are re-written to `<sudoCommand> argv...`, mirroring the
// podman/docker providers.
func (n *Nspawn) run(ctx context.Context, argv []string, opts runOptions) (string, error) {
	command := argv[0]
	args := argv[1:]
	if n.root && opts.Privileged {
		args = argv
		command = n.sudoCommand
	}

	if opts.DryRun {
		//nolint:forbidigo // Print command in dry-run mode
		fmt.Println(command, strings.Join(args, " "))
		return "", nil
	}

	cmd := exec.CommandContext(ctx, command, args...)

	if opts.Interactive {
		cmd.Stdout = os.Stdout
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("error running the interactive command :%w", err)
		}
		return "", nil
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if opts.TailLogs {
		cmd.Stdout = io.MultiWriter(&stdout, os.Stdout)
		cmd.Stderr = io.MultiWriter(&stderr, os.Stderr)
	}

	err := cmd.Run()
	if err != nil {
		captured := strings.TrimSpace(stderr.String())
		if captured != "" {
			return "", fmt.Errorf("command execution failed: %s", captured)
		}
		return "", fmt.Errorf("command execution failed: %w", err)
	}
	return stdout.String(), nil
}
