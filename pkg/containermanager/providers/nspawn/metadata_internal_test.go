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
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeMachineNameValidNamesUnchanged(t *testing.T) {
	for _, name := range []string{"my-distrobox", "ubuntu24", "a", "Fedora-Toolbox-41"} {
		assert.Equal(t, name, sanitizeMachineName(name), "name %q should be unchanged", name)
	}
}

func TestSanitizeMachineNameModifiedNamesGetHashSuffix(t *testing.T) {
	tests := []struct {
		in         string
		wantPrefix string
	}{
		{"my_box", "my-box-"},
		{"ubuntu:24.04", "ubuntu24-04-"},
		{"box with spaces", "boxwithspaces-"},
		{"-leading-dash", "leading-dash-"},
	}
	for _, tt := range tests {
		got := sanitizeMachineName(tt.in)
		assert.True(t, strings.HasPrefix(got, tt.wantPrefix),
			"sanitize(%q) = %q, want prefix %q", tt.in, got, tt.wantPrefix)
		assert.LessOrEqual(t, len(got), maxMachineNameLength)
	}
}

func TestSanitizeMachineNameLongNamesTruncatedStably(t *testing.T) {
	long := strings.Repeat("a", 100)
	got := sanitizeMachineName(long)
	assert.LessOrEqual(t, len(got), maxMachineNameLength)
	assert.Equal(t, got, sanitizeMachineName(long), "sanitization must be deterministic")

	// Two long names sharing a 64-char prefix must not collide.
	other := strings.Repeat("a", 99) + "b"
	assert.NotEqual(t, got, sanitizeMachineName(other))
}

func TestSanitizeMachineNameAllInvalidRunes(t *testing.T) {
	got := sanitizeMachineName("日本語")
	assert.True(t, strings.HasPrefix(got, "distrobox-"), "got %q", got)
	assert.LessOrEqual(t, len(got), maxMachineNameLength)
}

func TestSanitizeMachineNameDistinctInputsDistinctOutputs(t *testing.T) {
	assert.NotEqual(t, sanitizeMachineName("my_box"), sanitizeMachineName("my-box"))
	assert.NotEqual(t, sanitizeMachineName("my.box"), sanitizeMachineName("my_box"))
}

func TestUnitName(t *testing.T) {
	assert.Equal(t, "distrobox-my-box.service", unitName("my-box"))
}

func TestContainerIDStable(t *testing.T) {
	id := containerID("my-distrobox")
	assert.Len(t, id, 12)
	assert.Equal(t, id, containerID("my-distrobox"))
	assert.NotEqual(t, id, containerID("other"))
}

func TestResolveDirsRootless(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/xdg-data")
	t.Setenv("XDG_CACHE_HOME", "/tmp/xdg-cache")
	d := resolveDirs(false)
	assert.Equal(t, "/tmp/xdg-data/distrobox", d.Data)
	assert.Equal(t, "/tmp/xdg-cache/distrobox", d.Cache)
}

func TestResolveDirsRootlessFallbacks(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("HOME", "/home/test")
	d := resolveDirs(false)
	assert.Equal(t, "/home/test/.local/share/distrobox", d.Data)
	assert.Equal(t, "/home/test/.cache/distrobox", d.Cache)
}

func TestResolveDirsRootful(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/xdg-data")
	d := resolveDirs(true)
	assert.Equal(t, "/var/lib/distrobox", d.Data)
	assert.Equal(t, "/var/cache/distrobox", d.Cache)
}

func testMachine(name string) *machine {
	return &machine{
		SchemaVersion: metadataSchemaVersion,
		Name:          name,
		MachineName:   sanitizeMachineName(name),
		UnitName:      unitName(sanitizeMachineName(name)),
		Image:         "registry.fedoraproject.org/fedora-toolbox:latest",
		Labels: map[string]string{
			"manager":           "distrobox",
			"distrobox.version": "2",
		},
		Home:         "/home/test",
		Hostname:     "host",
		Env:          []string{"SHELL=/bin/bash"},
		InitArgs:     []string{"--name", "test"},
		StartCommand: []string{"systemd-run", "--user"},
	}
}

func testDirs(t *testing.T) dirs {
	t.Helper()
	base := t.TempDir()
	return dirs{Data: base + "/data", Cache: base + "/cache"}
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func TestMachineMetadataRoundtrip(t *testing.T) {
	d := testDirs(t)
	m := testMachine("my-box")
	require.NoError(t, mkdirMachine(d, m.Name))
	require.NoError(t, saveMachine(d, m))

	loaded, err := loadMachine(d, "my-box")
	require.NoError(t, err)
	assert.Equal(t, m, loaded)
}

func TestLoadMachineNotFound(t *testing.T) {
	d := testDirs(t)
	_, err := loadMachine(d, "missing")
	assert.ErrorIs(t, err, errMachineNotFound)
}

func TestLoadMachineNewerSchemaRejected(t *testing.T) {
	d := testDirs(t)
	m := testMachine("future")
	m.SchemaVersion = metadataSchemaVersion + 1
	require.NoError(t, mkdirMachine(d, m.Name))
	require.NoError(t, saveMachine(d, m))

	_, err := loadMachine(d, "future")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "schema version")
}

func TestListMachinesSkipsUnparsable(t *testing.T) {
	d := testDirs(t)
	good := testMachine("good")
	require.NoError(t, mkdirMachine(d, good.Name))
	require.NoError(t, saveMachine(d, good))

	require.NoError(t, mkdirMachine(d, "broken"))
	require.NoError(t, writeFile(d.metadataPath("broken"), "{not json"))

	// A directory without machine.json (e.g. mid-create) is skipped quietly.
	require.NoError(t, mkdirMachine(d, "half-created"))

	var warnings []string
	machines, err := listMachines(d, func(format string, _ ...any) {
		warnings = append(warnings, format)
	})
	require.NoError(t, err)
	require.Len(t, machines, 1)
	assert.Equal(t, "good", machines[0].Name)
	assert.Len(t, warnings, 1)
}

func TestListMachinesEmptyOnMissingDir(t *testing.T) {
	d := testDirs(t)
	machines, err := listMachines(d, nil)
	require.NoError(t, err)
	assert.Empty(t, machines)
}

func TestWithLockSerializes(t *testing.T) {
	d := testDirs(t)
	lockPath := d.machinesDir() + "/.lock"

	inCritical := false
	done := make(chan error, 2)
	for range 2 {
		go func() {
			done <- withLock(lockPath, func() error {
				if inCritical {
					return assert.AnError
				}
				inCritical = true
				defer func() { inCritical = false }()
				return nil
			})
		}()
	}
	require.NoError(t, <-done)
	require.NoError(t, <-done)
}

func TestAsContainer(t *testing.T) {
	m := testMachine("my-box")
	c := m.asContainer("running")
	assert.Equal(t, "my-box", c.Name)
	assert.Equal(t, "running", c.Status)
	assert.Equal(t, containerID("my-box"), c.ID)
	assert.True(t, c.IsDistrobox())
	assert.True(t, c.IsRunning())
}
