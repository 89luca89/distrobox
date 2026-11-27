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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInitSkelOwnsIntermediateDirectories runs the skel block of
// distrobox-init through dash against a fake /etc/skel and checks that every
// directory created for a nested skel file is chowned to the container user,
// not only the leaf directory.
func TestInitSkelOwnsIntermediateDirectories(t *testing.T) {
	dashPath, err := exec.LookPath("dash")
	require.NoError(t, err)

	initScript, err := os.ReadFile("assets/distrobox-init")
	require.NoError(t, err)

	// Extract the skel block: from its "if" line up to the first top-level "fi".
	const startMarker = `if [ -n "${DISTROBOX_HOST_HOME-}" ] && [ -d "/etc/skel" ]; then`
	start := strings.Index(string(initScript), startMarker)
	require.NotEqual(t, -1, start, "skel block not found in distrobox-init")
	endRel := strings.Index(string(initScript)[start:], "\nfi\n")
	require.NotEqual(t, -1, endRel, "end of skel block not found in distrobox-init")
	block := string(initScript)[start : start+endRel+len("\nfi\n")]

	tmpDir := t.TempDir()
	skelDir := filepath.Join(tmpDir, "skel")
	home := filepath.Join(tmpDir, "home")
	fakeBin := filepath.Join(tmpDir, "bin")
	chownLog := filepath.Join(tmpDir, "chown.log")
	require.NoError(t, os.MkdirAll(filepath.Join(skelDir, ".config", "foobar"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skelDir, ".config", "foobar", "settings"), []byte("x\n"), 0644))
	require.NoError(t, os.MkdirAll(home, 0755))
	require.NoError(t, os.MkdirAll(fakeBin, 0755))

	// Fake chown records its arguments instead of changing ownership, so the
	// test does not need root.
	fakeChown := "#!/bin/sh\necho \"$@\" >> \"" + chownLog + "\"\n"
	require.NoError(t, os.WriteFile(filepath.Join(fakeBin, "chown"), []byte(fakeChown), 0755))

	block = strings.ReplaceAll(block, "/etc/skel", skelDir)
	script := "container_user_home=\"" + home + "\"\n" +
		"container_user_uid=1000\n" +
		"container_user_gid=1000\n" +
		block
	scriptPath := filepath.Join(tmpDir, "skel-block.sh")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0755))

	cmd := exec.Command(dashPath, scriptPath)
	cmd.Env = append(os.Environ(),
		"DISTROBOX_HOST_HOME="+home,
		"PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))

	// The skel file itself must have been copied.
	assert.FileExists(t, filepath.Join(home, ".config", "foobar", "settings"))

	log, err := os.ReadFile(chownLog)
	require.NoError(t, err)
	// Every created directory level must be chowned, including the parent ".config".
	assert.Contains(t, string(log), " "+filepath.Join(home, ".config")+"\n",
		"parent directory .config was not chowned to the container user")
	assert.Contains(t, string(log), " "+filepath.Join(home, ".config", "foobar")+"\n",
		"leaf directory foobar was not chowned to the container user")
}
