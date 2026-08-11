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
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/89luca89/distrobox/internal/userenv"
	"github.com/89luca89/distrobox/pkg/containermanager"
)

func testUserEnv() *userenv.UserEnvironment {
	return &userenv.UserEnvironment{
		User:    "user",
		UserID:  "1000",
		GroupID: "1000",
		Home:    "/home/user",
		Shell:   "/bin/bash",
	}
}

func buildTestMachine(t *testing.T, n *Nspawn, opts containermanager.CreateOptions) *machine {
	t.Helper()
	if opts.ContainerName == "" {
		opts.ContainerName = "my-box"
	}
	if opts.ContainerImage == "" {
		opts.ContainerImage = "example.com/test/image:v1"
	}
	if opts.ContainerHostname == "" {
		opts.ContainerHostname = "my-box"
	}
	m, err := n.buildMachine(opts, testUserEnv(), resolveDirs(n.root), "/scripts")
	require.NoError(t, err)
	return m
}

func TestBuildMachineRootlessStartCommand(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/data")
	n := New(false, "sudo", false)
	m := buildTestMachine(t, n, containermanager.CreateOptions{})
	cmd := strings.Join(m.StartCommand, " ")

	assert.Equal(t, "systemd-run", m.StartCommand[0])
	assert.Contains(t, cmd, "systemd-run --user")
	assert.Contains(t, cmd, "--unit=distrobox-my-box.service")
	assert.Contains(t, cmd, "--property=StandardOutput=append:/data/distrobox/machines/my-box/init.log")
	assert.Contains(t, cmd, "-- systemd-nspawn")
	assert.Contains(t, cmd, "--directory=/data/distrobox/machines/my-box/rootfs")
	assert.Contains(t, cmd, "--machine=my-box")
	assert.Contains(t, cmd, "--private-users=managed")
	assert.Contains(t, cmd, "--bind=/etc:/run/host/etc")
	assert.Contains(t, cmd, "--bind=/usr:/run/host/usr")
	assert.NotContains(t, cmd, "--bind=/:/run/host")
	assert.Contains(t, cmd, "--bind=/tmp:/tmp")
	assert.Contains(t, cmd, "--bind=/home/user:/home/user")

	assert.Contains(t, cmd, "--bind-ro=/dev/null:/run/.distrobox.rootless")
	assert.Contains(t, cmd, "--tmpfs=/var/log/journal")
	assert.Contains(t, cmd, "--resolv-conf=replace-host")
	assert.Contains(t, cmd, "--setenv=container=systemd-nspawn")
	assert.Contains(t, cmd, "--setenv=CONTAINER_ID=my-box")
	assert.Contains(t, cmd, "--setenv=SHELL=bash")
	assert.Contains(t, cmd, "/usr/bin/entrypoint --verbose --name user --user 1000 --group 1000 --home /home/user --init 0 --nvidia 0")
	assert.NotContains(t, cmd, "--private-network")
	assert.NotContains(t, cmd, "--kill-signal")
	assert.NotContains(t, cmd, "--boot")
}

func TestBuildMachineRootfulStartCommand(t *testing.T) {
	n := New(true, "sudo", false)
	m := buildTestMachine(t, n, containermanager.CreateOptions{})
	cmd := strings.Join(m.StartCommand, " ")

	// The systemd-run half (before the "--" separator) must target the
	// system manager: no --user. distrobox-init's own --user flag appears
	// later in the entrypoint args and is unrelated.
	systemdRunArgs := m.StartCommand[:slices.Index(m.StartCommand, "--")]
	assert.NotContains(t, systemdRunArgs, "--user")
	assert.Contains(t, cmd, "--private-users=no")
	assert.Contains(t, cmd, "--directory=/var/lib/distrobox/machines/my-box/rootfs")
	assert.NotContains(t, cmd, "idmap")
	assert.NotContains(t, cmd, "/run/.distrobox.rootless")
}

func TestBuildMachineUnshareNetNS(t *testing.T) {
	n := New(false, "sudo", false)
	m := buildTestMachine(t, n, containermanager.CreateOptions{UnshareNetNS: true})
	cmd := strings.Join(m.StartCommand, " ")

	assert.Contains(t, cmd, "--private-network")
	assert.NotContains(t, cmd, "--resolv-conf=replace-host")
	assert.NotContains(t, cmd, "/etc/hosts")
}

func TestBuildMachineInit(t *testing.T) {
	n := New(false, "sudo", false)
	m := buildTestMachine(t, n, containermanager.CreateOptions{Init: true})
	cmd := strings.Join(m.StartCommand, " ")

	assert.Contains(t, cmd, "--property=KillSignal=SIGRTMIN+3")
	assert.Contains(t, cmd, "--kill-signal=SIGRTMIN+3")
	assert.Contains(t, cmd, "--init 1")
	// Initful machines get a dedicated systemd user session instead of the
	// host's XDG_RUNTIME_DIR.
	assert.NotContains(t, cmd, "/run/user/1000")
	// distrobox-init exec's systemd itself; nspawn must not boot the image.
	assert.NotContains(t, cmd, "--boot")
}

func TestBuildMachineCustomHome(t *testing.T) {
	n := New(false, "sudo", false)
	m := buildTestMachine(t, n, containermanager.CreateOptions{
		ContainerUserCustomHome: "/home/user/boxes/my-box",
	})
	cmd := strings.Join(m.StartCommand, " ")

	assert.Contains(t, cmd, "--setenv=HOME=/home/user/boxes/my-box")
	assert.Contains(t, cmd, "--setenv=DISTROBOX_HOST_HOME=/home/user")
	assert.Contains(t, cmd, "--bind=/home/user/boxes/my-box:/home/user/boxes/my-box")
	assert.Contains(t, cmd, "--home /home/user/boxes/my-box")
	assert.Equal(t, "/home/user/boxes/my-box", m.CustomHome)
}

func TestBuildMachineNopasswd(t *testing.T) {
	n := New(false, "sudo", false)
	m := buildTestMachine(t, n, containermanager.CreateOptions{Nopasswd: true})
	assert.Contains(t, strings.Join(m.StartCommand, " "), "--bind-ro=/dev/null:/run/.nopasswd")
}

func TestBuildMachineAdditionalVolumesAndFlags(t *testing.T) {
	n := New(false, "sudo", false)
	m := buildTestMachine(t, n, containermanager.CreateOptions{
		AdditionalVolumes: []string{"/src:/dst", "/roshare:/roshare:ro"},
		AdditionalFlags:   []string{"--capability=CAP_NET_ADMIN"},
	})
	cmd := strings.Join(m.StartCommand, " ")

	assert.Contains(t, cmd, "--bind=/src:/dst")
	assert.Contains(t, cmd, "--bind-ro=/roshare:/roshare")
	assert.Contains(t, cmd, "--capability=CAP_NET_ADMIN")
}

func TestBuildMachineRejectsUnknownVolumeOption(t *testing.T) {
	n := New(false, "sudo", false)
	_, err := n.buildMachine(containermanager.CreateOptions{
		ContainerName:     "my-box",
		ContainerImage:    "img",
		AdditionalVolumes: []string{"/src:/dst:nosuchopt"},
	}, testUserEnv(), resolveDirs(false), "/scripts")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nosuchopt")
}

func TestBuildMachineLabels(t *testing.T) {
	n := New(false, "sudo", false)
	m := buildTestMachine(t, n, containermanager.CreateOptions{
		UnshareGroups: true,
		Labels: map[string]string{
			"custom.label": "value",
			// User labels must not override provider-managed ones.
			"manager": "someone-else",
		},
	})

	assert.Equal(t, "distrobox", m.Labels["manager"])
	assert.Equal(t, "1", m.Labels["distrobox.unshare_groups"])
	assert.Equal(t, "2", m.Labels[containermanager.VersionLabelKey])
	assert.Equal(t, "value", m.Labels["custom.label"])
	assert.True(t, m.asContainer("").IsDistrobox())
}

func TestBuildMachineSanitizedName(t *testing.T) {
	n := New(false, "sudo", false)
	m := buildTestMachine(t, n, containermanager.CreateOptions{ContainerName: "my_box.dev"})
	cmd := strings.Join(m.StartCommand, " ")

	assert.Equal(t, "my_box.dev", m.Name)
	assert.NotEqual(t, m.Name, m.MachineName)
	assert.True(t, strings.HasPrefix(m.MachineName, "my-box-dev-"))
	assert.Contains(t, cmd, "--machine="+m.MachineName)
	assert.Contains(t, cmd, "--unit=distrobox-"+m.MachineName+".service")
	// The distrobox-facing identity keeps the original name.
	assert.Contains(t, cmd, "--setenv=CONTAINER_ID=my_box.dev")
}

func TestTranslateVolumes(t *testing.T) {
	mounts, err := translateVolumes([]string{
		"/a:/b",
		"/c:/d:ro",
		"/e:/f:rw",
		"/g:/h:ro,z,rslave",
	}, false)
	require.NoError(t, err)

	assert.Equal(t, []machineMount{
		{Source: "/a", Destination: "/b"},
		{Source: "/c", Destination: "/d", Options: "ro"},
		{Source: "/e", Destination: "/f"},
		{Source: "/g", Destination: "/h", Options: "ro"},
	}, mounts)

	_, err = translateVolumes([]string{"/only-source"}, false)
	require.Error(t, err)
	_, err = translateVolumes([]string{":/dst"}, false)
	require.Error(t, err)
}

func TestBindFlag(t *testing.T) {
	assert.Equal(t, "--bind=/a:/b", bindFlag(machineMount{Source: "/a", Destination: "/b"}))
	assert.Equal(t, "--bind-ro=/a:/b", bindFlag(machineMount{Source: "/a", Destination: "/b", Options: "ro"}))
	assert.Equal(t, "--bind=/a:/b:idmap", bindFlag(machineMount{Source: "/a", Destination: "/b", Options: "idmap"}))
}
