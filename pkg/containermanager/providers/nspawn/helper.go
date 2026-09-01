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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/89luca89/distrobox/pkg/ocistore"
	"golang.org/x/sys/unix"
)

// HelperCommand is the hidden CLI subcommand rootful mode re-execs itself
// through (via the sudo program). Image pulling and rootfs extraction run
// in-process, so unlike podman/docker commands they cannot be prefixed with
// sudo — instead the privileged half of the work re-enters this same
// binary as root.
const HelperCommand = "nspawn-helper"

// runHelper re-executes ourselves as root running `nspawn-helper <action>
// [args...]`, optionally feeding the machine definition on stdin.
func (n *Nspawn) runHelper(ctx context.Context, m *machine, action string, args ...string) error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine distrobox executable path: %w", err)
	}

	argv := append([]string{self, HelperCommand, action}, args...)
	command := argv[0]
	commandArgs := argv[1:]
	// A shell that is already root does not need (and may not have) sudo.
	if os.Geteuid() != 0 {
		command = n.sudoCommand
		commandArgs = argv
	}

	cmd := exec.CommandContext(ctx, command, commandArgs...)
	if m != nil {
		raw, err := json.Marshal(m)
		if err != nil {
			return fmt.Errorf("cannot encode machine definition: %w", err)
		}
		cmd.Stdin = bytes.NewReader(raw)
	}

	var stderr bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(&stderr, os.Stderr)

	if err := cmd.Run(); err != nil {
		captured := strings.TrimSpace(stderr.String())
		if captured != "" {
			return fmt.Errorf("%s", captured)
		}
		return fmt.Errorf("privileged helper failed: %w", err)
	}
	return nil
}

// The Helper* functions below are the root-side implementations, called by
// the hidden CLI subcommand. They must only run as real root.

// ErrHelperNotRoot is returned when the helper entry points run without
// root privileges.
var ErrHelperNotRoot = errors.New("nspawn-helper must run as root")

func requireRoot() error {
	if os.Geteuid() != 0 {
		return ErrHelperNotRoot
	}
	return nil
}

// HelperCreateMachine reads a machine definition from stdin and
// materializes it under the rootful storage directories.
func HelperCreateMachine(ctx context.Context, stdin io.Reader) error {
	if err := requireRoot(); err != nil {
		return err
	}
	raw, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("cannot read machine definition: %w", err)
	}
	var m machine
	if err := json.Unmarshal(raw, &m); err != nil {
		return fmt.Errorf("cannot parse machine definition: %w", err)
	}
	if m.Name == "" || m.Name != strings.TrimSpace(m.Name) || strings.ContainsAny(m.Name, "/\x00") {
		return fmt.Errorf("invalid machine name %q", m.Name)
	}
	return createMachine(ctx, resolveDirs(true), &m)
}

// HelperRemoveMachine removes a rootful machine directory by name. The
// name-based API keeps the privileged path deletion constrained to
// /var/lib/distrobox/machines.
func HelperRemoveMachine(_ context.Context, name string) error {
	if err := requireRoot(); err != nil {
		return err
	}
	d := resolveDirs(true)
	if name == "" || strings.ContainsAny(name, "/\x00") || name == "." || name == ".." {
		return fmt.Errorf("invalid machine name %q", name)
	}
	return withLock(d.machinesDir()+"/.lock", func() error {
		return os.RemoveAll(d.machineDir(name))
	})
}

// HelperPullImage pulls an image into the rootful image store.
func HelperPullImage(ctx context.Context, image, platform string) error {
	if err := requireRoot(); err != nil {
		return err
	}
	if _, err := ocistore.New(resolveDirs(true).Cache).Pull(ctx, image, platform); err != nil {
		return fmt.Errorf("cannot pull image: %w", err)
	}
	return nil
}

// HelperImageExists reports through its error whether the image exists in
// the rootful image store (nil = exists).
func HelperImageExists(_ context.Context, image string) error {
	if err := requireRoot(); err != nil {
		return err
	}
	if !ocistore.New(resolveDirs(true).Cache).Exists(image) {
		return fmt.Errorf("image %s not found", image)
	}
	return nil
}

// HelperCleanRuntime removes stale per-machine nspawn runtime state left
// behind by an unclean shutdown (nspawn refuses to start while
// /run/systemd/nspawn/unix-export/<machine> exists). Only called when the
// machine's unit is inactive.
func HelperCleanRuntime(_ context.Context, machineName string) error {
	if err := requireRoot(); err != nil {
		return err
	}
	if machineName == "" || sanitizeMachineName(machineName) != machineName {
		return fmt.Errorf("invalid machine name %q", machineName)
	}
	cleanUnixExport("/run/systemd/nspawn/unix-export", machineName)
	return nil
}

// cleanUnixExport unmounts and removes one machine's unix-export
// directory, best-effort.
func cleanUnixExport(base, machineName string) {
	dir := filepath.Join(base, machineName)
	if _, err := os.Lstat(dir); err != nil {
		return
	}
	_ = unix.Unmount(dir, unix.MNT_DETACH)
	_ = os.RemoveAll(dir)
}
