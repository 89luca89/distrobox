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

package insidedistrobox_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	insidedistrobox "github.com/89luca89/distrobox/internal/inside-distrobox"
)

// expectedScripts is the canonical triad of in-container helper scripts.
// ProvisionScripts must always end with all three present in the returned
// directory, regardless of which resolution branch produced it.
//
//nolint:gochecknoglobals // shared fixture across the suite, behaves like a constant
var expectedScripts = []string{
	"distrobox-host-exec",
	"distrobox-init",
	"distrobox-export",
}

func assertAllScripts(t *testing.T, dir string) {
	t.Helper()
	for _, name := range expectedScripts {
		assert.FileExists(t, filepath.Join(dir, name), "expected %s in %s", name, dir)
	}
}

// isolatePath wipes PATH for the duration of the test so the PATH branch
// of exists() never finds a system-installed distrobox-init while we are
// trying to exercise other resolution branches.
func isolatePath(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", "")
}

// TestProvisionScripts_CustomDir checks that ProvisionScripts writes all three
// scripts into the given directory and returns it unchanged.
func TestProvisionScripts_CustomDir(t *testing.T) {
	tmpDir := t.TempDir()
	isolatePath(t)

	dir, err := insidedistrobox.ProvisionScripts(tmpDir)
	require.NoError(t, err)
	require.Equal(t, tmpDir, dir)
	assertAllScripts(t, dir)
}

// TestProvisionScripts_DetectOnPath confirms the skip-write shortcut via
// the PATH branch of exists(): when the helper scripts already exist
// somewhere on PATH, ProvisionScripts leaves them byte-for-byte
// untouched rather than overwriting from the embedded copies.
func TestProvisionScripts_DetectOnPath(t *testing.T) {
	writeDir := t.TempDir()
	scriptsDir := t.TempDir()
	marker := "#!/bin/sh\n# pre-existing-marker\n"
	for _, name := range expectedScripts {
		require.NoError(t, os.WriteFile(filepath.Join(scriptsDir, name), []byte(marker), 0755))
	}

	t.Setenv("PATH", scriptsDir)

	_, err := insidedistrobox.ProvisionScripts(writeDir)
	require.NoError(t, err)

	for _, name := range expectedScripts {
		got, err := os.ReadFile(filepath.Join(scriptsDir, name))
		require.NoError(t, err)
		require.Equal(t, marker, string(got), "%s was overwritten despite existing on PATH", name)
	}
}

// TestExportBinaryProtectsExistingFiles runs the POSIX distrobox-export script
// through dash to ensure an existing host file is never replaced unless it is
// an export created by distrobox itself.
func TestExportBinaryProtectsExistingFiles(t *testing.T) {
	dashPath, err := exec.LookPath("dash")
	require.NoError(t, err)

	originalPath := os.Getenv("PATH")
	isolatePath(t)
	scriptsDir := t.TempDir()
	_, err = insidedistrobox.ProvisionScripts(scriptsDir)
	require.NoError(t, err)
	t.Setenv("PATH", originalPath)

	tmpDir := t.TempDir()
	guestBin := filepath.Join(tmpDir, "guest", "demo-tool")
	existingPath := filepath.Join(tmpDir, "existing")
	wrapperPath := filepath.Join(tmpDir, "wrapper")
	require.NoError(t, os.MkdirAll(filepath.Dir(guestBin), 0755))
	require.NoError(t, os.WriteFile(guestBin, []byte("#!/bin/sh\necho guest binary\n"), 0755))
	require.NoError(t, os.MkdirAll(existingPath, 0755))
	existingFile := filepath.Join(existingPath, "demo-tool")
	original := []byte("#!/bin/sh\necho host original\n")
	require.NoError(t, os.WriteFile(existingFile, original, 0755))

	t.Setenv("CONTAINER_ID", "first")
	t.Setenv("container", "1")
	t.Setenv("HOME", tmpDir)
	t.Setenv("DISTROBOX_HOST_HOME", tmpDir)
	t.Setenv("DISTROBOX_PATH", "distrobox")

	runExport := func(containerName, exportPath string, force bool) ([]byte, error) {
		t.Setenv("CONTAINER_ID", containerName)
		t.Setenv("DISTROBOX_EXPORT_PATH", exportPath)
		args := []string{"--bin", guestBin}
		if force {
			args = append(args, "--force")
		}
		cmd := exec.Command(dashPath, append([]string{filepath.Join(scriptsDir, "distrobox-export")}, args...)...)
		return cmd.CombinedOutput()
	}

	output, err := runExport("first", existingPath, false)
	require.Error(t, err)
	assert.Contains(t, string(output), "already exists")
	got, err := os.ReadFile(existingFile)
	require.NoError(t, err)
	assert.Equal(t, original, got)

	output, err = runExport("forced", existingPath, true)
	require.NoError(t, err, string(output))
	forced, err := os.ReadFile(existingFile)
	require.NoError(t, err)
	assert.Contains(t, string(forced), "# distrobox_binary")
	assert.Contains(t, string(forced), "# name: forced")

	output, err = runExport("first", wrapperPath, false)
	require.NoError(t, err, string(output))
	wrapperFile := filepath.Join(wrapperPath, "demo-tool")
	wrapper, err := os.ReadFile(wrapperFile)
	require.NoError(t, err)
	assert.Contains(t, string(wrapper), "# distrobox_binary")

	output, err = runExport("updated", wrapperPath, false)
	require.NoError(t, err, string(output))
	updated, err := os.ReadFile(wrapperFile)
	require.NoError(t, err)
	assert.Contains(t, string(updated), "# distrobox_binary")
	assert.Contains(t, string(updated), "# name: updated")
}

// TestProvisionScripts_ExtractsAdjacentToBinary verifies that ProvisionScripts
// writes to the given directory.
func TestProvisionScripts_ExtractsAdjacentToBinary(t *testing.T) {
	isolatePath(t)

	exe, err := os.Executable()
	require.NoError(t, err)
	exeDir := filepath.Dir(exe)
	if !isDirWritable(exeDir) {
		t.Skipf("binary-adjacent dir %s is not writable; cannot exercise this branch here", exeDir)
	}
	t.Cleanup(func() {
		for _, name := range expectedScripts {
			_ = os.Remove(filepath.Join(exeDir, name))
		}
	})

	dir, err := insidedistrobox.ProvisionScripts(exeDir)
	require.NoError(t, err)
	require.Equal(t, exeDir, dir)
	assertAllScripts(t, dir)
}

// isDirWritable does a probing-write into dir to determine whether a
// non-root user can create files there. Used by the extraction-adjacent
// test as a defensive skip when the binary's directory isn't writable
// (e.g. read-only test harness, distros with hardened tmpfs).
func isDirWritable(dir string) bool {
	probe, err := os.CreateTemp(dir, ".dbx-write-probe-")
	if err != nil {
		return false
	}
	defer os.Remove(probe.Name())
	defer probe.Close()
	return true
}
