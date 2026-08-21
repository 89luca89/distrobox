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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"

	"github.com/89luca89/distrobox/pkg/containermanager"
)

// metadataSchemaVersion is the version of the machine.json layout, bumped
// on incompatible changes. Independent from containermanager.SchemaVersion,
// which versions the distrobox container contract itself.
const metadataSchemaVersion = 1

// machineMount is a recorded bind mount, in InspectResult.Mounts shape.
type machineMount struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Options     string `json:"options,omitempty"`
}

// machineUnshare records which host namespaces the machine does not share.
type machineUnshare struct {
	IPC     bool `json:"ipc"`
	NetNS   bool `json:"netns"`
	Process bool `json:"process"`
	Devsys  bool `json:"devsys"`
}

// machine is the on-disk metadata for one nspawn-backed distrobox
// container, stored as machine.json inside the machine directory.
type machine struct {
	SchemaVersion int    `json:"schemaVersion"`
	Name          string `json:"name"`
	// MachineName is the sanitized, machined-facing name; systemd/machinectl
	// commands use it, distrobox-facing APIs use Name.
	MachineName string            `json:"machineName"`
	UnitName    string            `json:"unitName"`
	Image       string            `json:"image"`
	ImageDigest string            `json:"imageDigest,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
	Labels      map[string]string `json:"labels"`
	Home        string            `json:"home"`
	CustomHome  string            `json:"customHome,omitempty"`
	Hostname    string            `json:"hostname"`
	Init        bool              `json:"init"`
	Nvidia      bool              `json:"nvidia"`
	// ScriptsDir is the host directory the helper scripts were installed
	// from; used to refresh the in-rootfs copies.
	ScriptsDir string         `json:"scriptsDir,omitempty"`
	Unshare    machineUnshare `json:"unshare"`
	Env        []string       `json:"env"`
	// InitArgs are the arguments passed to /usr/bin/entrypoint
	// (distrobox-init); exposed as InspectResult.Cmd.
	InitArgs []string       `json:"initArgs"`
	Mounts   []machineMount `json:"mounts"`
	// StartCommand is the full argv (systemd-run ... -- systemd-nspawn ...)
	// that starts the machine, regenerated at create time.
	StartCommand []string `json:"startCommand"`
}

// dirs resolves where machines and the image store live for the given
// privilege mode. Rootful state must never land in the invoking user's XDG
// directories.
type dirs struct {
	// Data holds the machines (rootfs + metadata).
	Data string
	// Cache holds the OCI image store.
	Cache string
}

func resolveDirs(root bool) dirs {
	if root {
		return dirs{
			Data:  "/var/lib/distrobox",
			Cache: "/var/cache/distrobox",
		}
	}
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(os.Getenv("HOME"), ".local", "share")
	}
	cacheHome := os.Getenv("XDG_CACHE_HOME")
	if cacheHome == "" {
		cacheHome = filepath.Join(os.Getenv("HOME"), ".cache")
	}
	return dirs{
		Data:  filepath.Join(dataHome, "distrobox"),
		Cache: filepath.Join(cacheHome, "distrobox"),
	}
}

func (d dirs) machinesDir() string {
	return filepath.Join(d.Data, "machines")
}

func (d dirs) machineDir(name string) string {
	return filepath.Join(d.machinesDir(), name)
}

func (d dirs) rootfsDir(name string) string {
	return filepath.Join(d.machineDir(name), "rootfs")
}

func (d dirs) metadataPath(name string) string {
	return filepath.Join(d.machineDir(name), "machine.json")
}

func (d dirs) initLogPath(name string) string {
	return filepath.Join(d.machineDir(name), "init.log")
}

// errMachineNotFound is returned when a machine directory or its metadata
// does not exist.
var errMachineNotFound = errors.New("container not found")

// mkdirMachine creates the directory for a new machine (parents included).
func mkdirMachine(d dirs, name string) error {
	//nolint:gosec // 0755 so unprivileged list/inspect can read rootful machine metadata
	if err := os.MkdirAll(d.machineDir(name), 0o755); err != nil {
		return fmt.Errorf("cannot create machine directory for %s: %w", name, err)
	}
	return nil
}

// loadMachine reads and validates the metadata of one machine.
func loadMachine(d dirs, name string) (*machine, error) {
	raw, err := os.ReadFile(d.metadataPath(name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", errMachineNotFound, name)
		}
		return nil, fmt.Errorf("cannot read metadata for %s: %w", name, err)
	}
	var m machine
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("cannot parse metadata for %s: %w", name, err)
	}
	if m.SchemaVersion > metadataSchemaVersion {
		return nil, fmt.Errorf("metadata for %s has schema version %d, this distrobox supports up to %d",
			name, m.SchemaVersion, metadataSchemaVersion)
	}
	return &m, nil
}

// saveMachine atomically writes the metadata inside the machine directory,
// which must already exist.
func saveMachine(d dirs, m *machine) error {
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot encode metadata for %s: %w", m.Name, err)
	}
	path := d.metadataPath(m.Name)
	tmp := path + ".tmp"
	//nolint:gosec // metadata is public information; rootful metadata must stay readable by unprivileged list/inspect
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return fmt.Errorf("cannot write metadata for %s: %w", m.Name, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("cannot write metadata for %s: %w", m.Name, err)
	}
	return nil
}

// listMachines returns the metadata of every machine whose machine.json
// parses; unparsable entries are skipped and reported through warn.
func listMachines(d dirs, warn func(format string, a ...any)) ([]*machine, error) {
	entries, err := os.ReadDir(d.machinesDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("cannot list containers: %w", err)
	}

	machines := make([]*machine, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		m, err := loadMachine(d, entry.Name())
		if err != nil {
			if !errors.Is(err, errMachineNotFound) && warn != nil {
				warn("skipping %s: %v", entry.Name(), err)
			}
			continue
		}
		machines = append(machines, m)
	}
	return machines, nil
}

// withLock runs fn while holding an exclusive flock on path (created if
// missing). Used to serialize create/remove/start across concurrent
// distrobox processes.
func withLock(path string, fn func() error) error {
	//nolint:gosec // lock directory mirrors the machines dir permissions
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("cannot create lock directory: %w", err)
	}
	//nolint:gosec // the lock file carries no data
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("cannot open lock file: %w", err)
	}
	defer file.Close()

	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX); err != nil {
		return fmt.Errorf("cannot acquire lock: %w", err)
	}
	defer func() { _ = unix.Flock(int(file.Fd()), unix.LOCK_UN) }()

	return fn()
}

// asContainer converts machine metadata plus a live status string into the
// list-facing Container shape.
func (m *machine) asContainer(status string) containermanager.Container {
	return containermanager.Container{
		ID:     containerID(m.Name),
		Image:  m.Image,
		Name:   m.Name,
		Status: status,
		Labels: m.Labels,
	}
}
