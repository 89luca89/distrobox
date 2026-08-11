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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

func enterTestMachine() *machine {
	return &machine{
		SchemaVersion: metadataSchemaVersion,
		Name:          "my-box",
		MachineName:   "my-box",
		UnitName:      "distrobox-my-box.service",
		Home:          "/home/user",
		Labels:        map[string]string{"manager": "distrobox"},
	}
}

func TestExecArgvInit(t *testing.T) {
	n := New(false, "sudo", false)
	m := enterTestMachine()
	m.Init = true

	cmd := n.execArgvInit(m, testUserEnv(), "/home/user/src", []string{"PWD=/home/user/src", "FOO=bar"},
		[]string{"/bin/sh", "-c", "exec bash"}, containermanager.EnterOptions{NoTTY: true})
	cmdStr := strings.Join(cmd, " ")

	assert.Equal(t, "systemd-run", cmd[0])
	assert.Contains(t, cmdStr, "--machine=my-box")
	assert.Contains(t, cmdStr, "--wait")
	assert.Contains(t, cmdStr, "--collect")
	assert.Contains(t, cmdStr, "--pipe")
	assert.NotContains(t, cmdStr, "--pty")
	assert.Contains(t, cmdStr, "--uid=1000")
	assert.Contains(t, cmdStr, "--working-directory=/home/user/src")
	assert.Contains(t, cmdStr, "--setenv=PWD=/home/user/src")
	assert.Contains(t, cmdStr, "--setenv=FOO=bar")
	assert.Contains(t, cmdStr, "-- /bin/sh -c exec bash")
	// Inside-machine sessions talk to the machine's own system manager.
	assert.NotContains(t, cmd[:len(cmd)-3], "--user")
}

func TestExecArgvInitUnshareGroupsRunsAsRoot(t *testing.T) {
	n := New(false, "sudo", false)
	m := enterTestMachine()
	m.Init = true
	m.Labels["distrobox.unshare_groups"] = "1"

	cmd := n.execArgvInit(m, testUserEnv(), "/home/user", nil, []string{"sh"}, containermanager.EnterOptions{NoTTY: true})
	assert.Contains(t, strings.Join(cmd, " "), "--uid=0")
}

func TestExecArgvNsenterRootless(t *testing.T) {
	n := New(false, "sudo", false)
	m := enterTestMachine()

	cmd := n.execArgvNsenter(m, testUserEnv(), "12345", "/home/user",
		[]string{"PWD=/home/user"}, []string{"/bin/sh", "-l"}, containermanager.EnterOptions{})
	cmdStr := strings.Join(cmd, " ")

	assert.Equal(t, "nsenter", cmd[0])
	assert.Contains(t, cmdStr, "--target 12345")
	assert.Contains(t, cmdStr, "--mount --uts --ipc --pid")
	// Rootless machines always have a private, pasta-backed network
	// namespace; the session must join it.
	assert.Contains(t, cmdStr, "--net")
	// Unprivileged machines: join the caller-owned user namespace first.
	assert.Contains(t, cmdStr, "--user")
	assert.Contains(t, cmdStr, "--setuid 1000")
	assert.Contains(t, cmdStr, "--setgid 1000")
	assert.Contains(t, cmdStr, "--wdns=/home/user")
	assert.Contains(t, cmdStr, "env -i PWD=/home/user /bin/sh -l")
}

func TestExecArgvNsenterRootfulPrivateNetwork(t *testing.T) {
	n := New(true, "sudo", false)
	m := enterTestMachine()
	m.Unshare.NetNS = true

	cmd := n.execArgvNsenter(m, testUserEnv(), "12345", "/home/user", nil, []string{"sh"}, containermanager.EnterOptions{})
	cmdStr := strings.Join(cmd, " ")

	assert.Contains(t, cmdStr, "--net")
	assert.NotContains(t, cmdStr, "--user")
}

func TestExecArgvNsenterUnshareGroupsStaysRoot(t *testing.T) {
	n := New(true, "sudo", false)
	m := enterTestMachine()
	m.Labels["distrobox.unshare_groups"] = "1"

	cmd := n.execArgvNsenter(m, testUserEnv(), "12345", "/", nil, []string{"sh"}, containermanager.EnterOptions{})
	cmdStr := strings.Join(cmd, " ")

	assert.NotContains(t, cmdStr, "--setuid")
	assert.NotContains(t, cmdStr, "--setgid")
	// Rootful machine sharing the host network: nothing to join.
	assert.NotContains(t, cmdStr, "--net")
}

func TestBuildEnterEnv(t *testing.T) {
	m := enterTestMachine()
	m.CustomHome = "/home/user/boxes/my-box"

	env, workdir, err := buildEnterEnv(m, containermanager.EnterOptions{NoWorkDir: true})
	require.NoError(t, err)
	assert.Equal(t, "/home/user/boxes/my-box", workdir)

	envStr := strings.Join(env, "\n")
	assert.Contains(t, envStr, "CONTAINER_ID=my-box")
	assert.Contains(t, envStr, "PWD=/home/user/boxes/my-box")
	assert.Contains(t, envStr, "XDG_CONFIG_HOME=/home/user/boxes/my-box/.config")
	assert.Contains(t, envStr, "HOME=/home/user/boxes/my-box")
	assert.Contains(t, envStr, "DISTROBOX_PATH=")
	assert.Contains(t, envStr, "PATH=")
}

func waitTestSetup(t *testing.T, logContent string) error {
	t.Helper()
	n := New(false, "sudo", false)
	m := enterTestMachine()

	logPath := filepath.Join(t.TempDir(), "init.log")
	require.NoError(t, os.WriteFile(logPath, []byte(logContent), 0o644))

	var out bytes.Buffer
	progress := ui.NewDevNullProgress()
	printer := ui.NewPrinter(&out, false)
	return n.waitForSetup(t.Context(), m, logPath, 0, progress, printer)
}

func TestWaitForSetupDone(t *testing.T) {
	err := waitTestSetup(t, strings.Join([]string{
		"+ setting up user",
		"distrobox: Installing packages",
		"Warning: something minor",
		"container_setup_done",
		"",
	}, "\n"))
	assert.NoError(t, err)
}

func TestWaitForSetupError(t *testing.T) {
	err := waitTestSetup(t, "Error: could not install packages\n")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not install packages")
}

func TestWaitForSetupRespectsOffset(t *testing.T) {
	n := New(false, "sudo", false)
	m := enterTestMachine()

	logPath := filepath.Join(t.TempDir(), "init.log")
	previousBoot := "Error: old failure from a previous boot\n"
	require.NoError(t, os.WriteFile(logPath, []byte(previousBoot+"container_setup_done\n"), 0o644))

	err := n.waitForSetup(t.Context(), m, logPath, int64(len(previousBoot)),
		ui.NewDevNullProgress(), ui.NewPrinter(&bytes.Buffer{}, false))
	assert.NoError(t, err)
}

func TestReadLogChunk(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "init.log")
	require.NoError(t, os.WriteFile(logPath, []byte("hello\nworld\n"), 0o644))

	chunk, offset, err := readLogChunk(logPath, 6)
	require.NoError(t, err)
	assert.Equal(t, "world\n", chunk)
	assert.Equal(t, int64(12), offset)

	// Truncated log restarts from the beginning.
	require.NoError(t, os.WriteFile(logPath, []byte("new\n"), 0o644))
	chunk, offset, err = readLogChunk(logPath, 12)
	require.NoError(t, err)
	assert.Equal(t, "new\n", chunk)
	assert.Equal(t, int64(4), offset)
}

func TestTailOf(t *testing.T) {
	long := strings.Repeat("line\n", 30)
	tail := tailOf(long)
	assert.Len(t, strings.Split(tail, "\n"), 15)
	assert.Equal(t, "short", tailOf("short\n"))
}
