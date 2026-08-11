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
	"strings"
	"sync"

	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ocistore"
)

// ErrUnsupported marks operations the nspawn backend does not implement.
var ErrUnsupported = errors.New("not supported with the nspawn container manager")

// Nspawn is a ContainerManager backed by systemd-nspawn machines running as
// systemd-run transient units. Container metadata lives in per-machine JSON
// files (see metadata.go); images come from a local OCI store (pkg/ocistore).
type Nspawn struct {
	root        bool
	sudoCommand string
	verbose     bool

	probeOnce   sync.Once
	probeResult error
}

var _ containermanager.ContainerManager = &Nspawn{}

// New returns an nspawn-backed container manager.
func New(root bool, sudoCommand string, verbose bool) *Nspawn {
	return &Nspawn{
		root:        root,
		sudoCommand: sudoCommand,
		verbose:     verbose,
	}
}

func (n *Nspawn) Name() string {
	return "nspawn"
}

func (n *Nspawn) CloneAsRoot() containermanager.ContainerManager {
	return &Nspawn{
		root:        true,
		sudoCommand: n.sudoCommand,
		verbose:     n.verbose,
	}
}

// dirs resolves storage locations for the current privilege mode.
func (n *Nspawn) dirs() dirs {
	return resolveDirs(n.root)
}

func (n *Nspawn) store() *ocistore.Store {
	return ocistore.New(n.dirs().Cache)
}

// Exists reports whether a distrobox-owned machine with this name exists.
// Metadata-only: works on hosts where nspawn itself is unusable.
func (n *Nspawn) Exists(_ context.Context, containerName string) bool {
	m, err := loadMachine(n.dirs(), containerName)
	return err == nil && m.asContainer("").IsDistrobox()
}

func (n *Nspawn) ListContainers(ctx context.Context) ([]containermanager.Container, error) {
	d := n.dirs()
	machines, err := listMachines(d, nil)
	if err != nil {
		return nil, err
	}

	containers := make([]containermanager.Container, 0, len(machines))
	for _, m := range machines {
		containers = append(containers, m.asContainer(n.machineStatus(ctx, m)))
	}
	return containers, nil
}

func (n *Nspawn) InspectContainer(ctx context.Context, containerName string) (*containermanager.InspectResult, error) {
	m, err := loadMachine(n.dirs(), containerName)
	if err != nil {
		return nil, err
	}

	home := m.Home
	if m.CustomHome != "" {
		home = m.CustomHome
	}

	networkMode := "host"
	if m.Unshare.NetNS {
		networkMode = "private"
	}

	mounts := make([]containermanager.MountInfo, 0, len(m.Mounts))
	for _, mount := range m.Mounts {
		mounts = append(mounts, containermanager.MountInfo{
			Source:      mount.Source,
			Destination: mount.Destination,
			Options:     mount.Options,
		})
	}

	return &containermanager.InspectResult{
		ContainerID:     containerID(m.Name),
		ContainerStatus: n.machineStatus(ctx, m),
		ContainerHome:   home,
		// The container PATH is unknown before entering; leaving it empty
		// makes BuildContainerPath fall back to the standard paths.
		ContainerPath:  "",
		UnshareGroups:  m.Labels["distrobox.unshare_groups"] == "1",
		ContainerImage: m.Image,
		Mounts:         mounts,
		NetworkMode:    networkMode,
		// nspawn always gives machines private IPC and PID namespaces;
		// there is no host-sharing equivalent of podman's --ipc/--pid host.
		IpcMode: "private",
		PidMode: "private",
		Cmd:     m.InitArgs,
		Env:     m.Env,
		Labels:  m.Labels,
	}, nil
}

// machineStatus maps the transient unit's ActiveState onto the podman-like
// status strings the rest of distrobox expects.
func (n *Nspawn) machineStatus(ctx context.Context, m *machine) string {
	argv := []string{"systemctl"}
	if !n.root {
		argv = append(argv, "--user")
	}
	argv = append(argv, "show", m.UnitName, "--property=ActiveState", "--value")

	// Reading unit state needs no privileges, even for system units.
	out, err := n.run(ctx, argv, runOptions{})
	if err != nil {
		return "unknown"
	}

	switch strings.TrimSpace(out) {
	case "active", "activating", "deactivating":
		return containermanager.RunningStatus
	default:
		// systemctl reports inactive both for stopped and never-started
		// units; the init log only exists once the machine ran at least
		// once.
		if containermanager.PathExists(n.dirs().initLogPath(m.Name)) {
			return "exited"
		}
		return "created"
	}
}

func (n *Nspawn) Commit(_ context.Context, _ string, _ string) error {
	return fmt.Errorf("%w: cloning/committing containers is not implemented yet, use podman or docker for --clone", ErrUnsupported)
}

// NeedsMigration reports whether the named container was created with a
// distrobox schema version older than the one this binary supports.
func (n *Nspawn) NeedsMigration(_ context.Context, containerName string) (bool, error) {
	m, err := loadMachine(n.dirs(), containerName)
	if err != nil {
		return false, fmt.Errorf("failed to inspect container %s: %w", containerName, err)
	}
	return containermanager.NeedsMigrationFromLabels(m.Labels), nil
}
