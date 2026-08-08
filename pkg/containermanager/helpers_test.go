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

package containermanager_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/89luca89/distrobox/pkg/containermanager"
)

// TestPathExistsAndIsSymlink covers the symlink edge cases that drive mount
// decisions in makeCreateCommand (e.g. the /dev/shm symlink branch, selinux,
// /var/home). PathExists uses Stat (follows links); IsSymlink uses Lstat.
func TestPathExistsAndIsSymlink(t *testing.T) {
	assert.False(t, containermanager.PathExists("/does/not/exist"))

	dir := t.TempDir()
	assert.True(t, containermanager.PathExists(dir))
	assert.False(t, containermanager.IsSymlink(dir))

	// Broken symlink: Stat fails (false), Lstat sees the link (true).
	broken := filepath.Join(dir, "broken")
	require.NoError(t, os.Symlink("/nope/nope", broken))
	assert.False(t, containermanager.PathExists(broken))
	assert.True(t, containermanager.IsSymlink(broken))

	// Symlink to an existing target: both true.
	good := filepath.Join(dir, "good")
	require.NoError(t, os.Symlink(dir, good))
	assert.True(t, containermanager.PathExists(good))
	assert.True(t, containermanager.IsSymlink(good))
}

// TestFilterEnvVars pins the enter env allow/deny logic: excluded prefixes, the
// XDG_*_DIRS pattern, and values containing shell-special characters are dropped.
func TestFilterEnvVars(t *testing.T) {
	for _, k := range []string{"HOME", "PATH", "SHELL", "HOSTNAME", "XDG_SEAT", "XDG_CONFIG_DIRS", "_UNDERSCORE"} {
		t.Setenv(k, "x")
	}
	t.Setenv("DBX_TEST_KEEP", "keep")
	t.Setenv("DBX_TEST_BAD", "has$dollar") // '$' -> excluded

	has := func(prefix string) bool {
		for _, e := range containermanager.FilterEnvVars() {
			if strings.HasPrefix(e, prefix) {
				return true
			}
		}
		return false
	}

	assert.True(t, has("DBX_TEST_KEEP="))
	for _, dropped := range []string{
		"HOME=", "PATH=", "SHELL=", "HOSTNAME=", "XDG_SEAT=", "XDG_CONFIG_DIRS=", "_UNDERSCORE=", "DBX_TEST_BAD=",
	} {
		assert.False(t, has(dropped), dropped)
	}
}

// TestGetWorkDir covers the /run/host prefixing and the noWorkDir branch.
func TestGetWorkDir(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	got, err := containermanager.GetWorkDir("/home/tester", true) // noWorkDir -> container home
	require.NoError(t, err)
	assert.Equal(t, "/home/tester", got)

	got, err = containermanager.GetWorkDir(wd, false) // cwd under home -> as-is
	require.NoError(t, err)
	assert.Equal(t, wd, got)

	got, err = containermanager.GetWorkDir("/definitely/not/a/prefix/xyzzy", false) // outside -> /run/host
	require.NoError(t, err)
	assert.Equal(t, "/run/host"+wd, got)
}

// TestBuildXDGPaths covers dedup + append of standard paths onto the env value.
func TestBuildXDGPaths(t *testing.T) {
	t.Setenv("DBX_TEST_XDG", "/a:/b")
	assert.Equal(t, "/a:/b:/c", containermanager.BuildXDGPaths("DBX_TEST_XDG", []string{"/b", "/c"}))

	assert.Equal(t, "/usr/local/share:/usr/share",
		containermanager.BuildXDGPaths("DBX_TEST_XDG_UNSET_UNIQUE_9421", []string{"/usr/local/share", "/usr/share"}))
}
