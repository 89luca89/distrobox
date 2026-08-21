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
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/89luca89/distrobox/pkg/containermanager"
)

// Stop stops the machines' transient units. systemctl stop both shuts the
// machine down (nspawn forwards the unit's kill signal, SIGRTMIN+3 for
// initful machines) and reaps the unit.
func (n *Nspawn) Stop(ctx context.Context, containerNames []string) error {
	d := n.dirs()
	units := make([]string, 0, len(containerNames))
	for _, name := range containerNames {
		m, err := loadMachine(d, name)
		if err != nil {
			return fmt.Errorf("error stopping containers: %w", err)
		}
		units = append(units, m.UnitName)
	}

	argv := []string{"systemctl"}
	if !n.root {
		argv = append(argv, "--user")
	}
	argv = append(append(argv, "stop"), units...)

	if _, err := n.run(ctx, argv, runOptions{Privileged: true}); err != nil {
		return fmt.Errorf("error stopping containers: %w", err)
	}
	return nil
}

// Remove deletes the machine: its transient unit if running (only with
// Force), then the machine directory (rootfs + metadata) and optionally the
// custom home.
func (n *Nspawn) Remove(
	ctx context.Context,
	containerName string,
	options containermanager.RmOptions,
) error {
	d := n.dirs()
	m, err := loadMachine(d, containerName)
	if err != nil {
		return fmt.Errorf("error removing the container: %w", err)
	}

	if n.machineStatus(ctx, m) == containermanager.RunningStatus {
		if !options.Force {
			return fmt.Errorf("cannot remove running container %s, use --force to stop and remove it", containerName)
		}
		if err := n.Stop(ctx, []string{containerName}); err != nil {
			return err
		}
	}

	if err := n.removeMachineDir(ctx, d, containerName); err != nil {
		return err
	}

	if options.RemoveHome {
		if err := os.RemoveAll(options.ContainerHome); err != nil {
			return fmt.Errorf("error removing home directory %s: %w", options.ContainerHome, err)
		}
	}
	return nil
}

// removeMachineDir deletes the machine directory, delegating to the
// privileged helper for the root-owned rootful store.
func (n *Nspawn) removeMachineDir(ctx context.Context, d dirs, containerName string) error {
	if n.root {
		if err := n.runHelper(ctx, nil, "remove-machine", "--name", containerName); err != nil {
			return fmt.Errorf("error removing the container: %w", err)
		}
		return nil
	}

	err := withLock(filepath.Join(d.machinesDir(), ".lock"), func() error {
		return removeAllForce(d.machineDir(containerName))
	})
	if err == nil {
		return nil
	}
	if errors.Is(err, os.ErrPermission) {
		return fmt.Errorf("error removing the container: %w\n"+
			"some files are not owned by you; remove them with: sudo rm -rf %s",
			err, d.machineDir(containerName))
	}
	return fmt.Errorf("error removing the container: %w", err)
}

// removeAllForce removes a directory tree that may contain user-owned but
// non-writable directories (e.g. Fedora ships /root as r-x): unlinking a
// child requires write permission on its parent, so on the first failure
// every directory is chmodded owner-writable and the removal retried.
func removeAllForce(path string) error {
	if err := os.RemoveAll(path); err == nil || !errors.Is(err, os.ErrPermission) {
		//nolint:wrapcheck // wrapped by the caller
		return err
	}

	_ = filepath.WalkDir(path, func(entry string, d os.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			_ = os.Chmod(entry, 0o700)
		}
		return nil
	})
	//nolint:wrapcheck // wrapped by the caller
	return os.RemoveAll(path)
}
