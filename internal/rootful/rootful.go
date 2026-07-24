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

// Package rootful provides utilities for running operations that require
// root privileges via a configurable sudo command.
package rootful

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
)

//nolint:gochecknoglobals // singleton: process-wide memoization is the intent
var (
	validateOnce      sync.Once
	errValidate       error
	cachedSudoCommand string
)

// Validate ensures that sudoCommand is available and the user can elevate.
// It runs `<sudoCommand> -v` at most once per process: the first call performs
// the check and caches the result; subsequent calls return the cached result
// without re-running the command. Passing a different sudoCommand after the
// first call is a programming error and returns an explicit error.
func Validate(ctx context.Context, sudoCommand string) error {
	if cachedSudoCommand != "" && cachedSudoCommand != sudoCommand {
		return fmt.Errorf("sudoCommand mismatch: already validated with %q, got %q", cachedSudoCommand, sudoCommand)
	}
	validateOnce.Do(func() {
		cachedSudoCommand = sudoCommand
		cmd := exec.CommandContext(ctx, sudoCommand, "-v")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			errValidate = fmt.Errorf("failed to validate %q: %w", sudoCommand, err)
		}
	})
	return errValidate
}
