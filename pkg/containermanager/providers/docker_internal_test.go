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

package providers

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/89luca89/distrobox/internal/userenv"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

func TestDocker_makeCreateCommand(t *testing.T) {
	docker := NewDocker(false, "sudo", false)

	userEnv := &userenv.UserEnvironment{
		User:    "user",
		UserID:  "1000",
		GroupID: "1000",
		Home:    "/home/user",
		Shell:   "/bin/sh",
	}

	containerAdditionalFlags := []string{}
	containerAdditionalPackages := []string{}
	containerAdditionalVolumes := []string{
		"/path/to/my-volume:/var/local/my-volume:ro",
	}

	cmd := docker.makeCreateCommand(
		"my-container",                // containerName
		"my-image",                    // containerImage
		containerAdditionalFlags,      // containerAdditionalFlags
		"my-hostname",                 // containerHostname
		containerAdditionalPackages,   // containerAdditionalPackages
		containerAdditionalVolumes,    // containerAdditionalVolumes
		"",                            // containerUserCustomHome
		"",                            // containerPlatform
		false,                         // nopasswd
		true,                          // init
		"echo 'pre-init-hook'",        // containerPreInitHook
		"echo 'init-hook'",            // containerInitHook
		false,                         // nvidia
		false,                         // unshareDevsys
		false,                         // unshareGroups
		false,                         // unshareIPC
		false,                         // unshareNetNS
		false,                         // unshareProcess
		userEnv,                       // userEnv
		"/path/to/distrobox-init",     // distroboxInitPath
		"/path/to/distrobox-export",   // distroboxExportPath
		"/path/to/distrobox-hostexec", // distroboxHostexecPath
	)

	// Build expected string dynamically for paths that depend on host filesystem
	selinuxVolume := ""
	if containermanager.PathExists("/sys/fs/selinux") {
		selinuxVolume = " --volume /sys/fs/selinux"
	}

	// Group forwarding mirrors os.Getgroups(), which varies by host.
	groupAddFlags := ""
	if groups, err := os.Getgroups(); err == nil {
		for _, gid := range groups {
			groupAddFlags += " --group-add " + strconv.Itoa(gid)
		}
	}

	expected := oneline(`
 create
 --hostname my-hostname
 --name my-container
 --privileged
 --security-opt label=disable
 --security-opt apparmor=unconfined
 --pids-limit=-1
 --user root:root
 --ipc host
 --network host
 --pid host
 --label manager=distrobox
 --label distrobox.unshare_groups=0
 --label distrobox.version=2
 --env SHELL=sh
 --env HOME=/home/user
 --env container=docker
 --env TERMINFO_DIRS=/usr/share/terminfo:/run/host/usr/share/terminfo
 --env CONTAINER_ID=my-container
 --volume /tmp:/tmp:rslave
 --volume /path/to/distrobox-export:/usr/bin/distrobox-export:ro
 --volume /path/to/distrobox-hostexec:/usr/bin/distrobox-host-exec:ro
 --volume /home/user:/home/user:rslave
 --volume /:/run/host/:rslave
 --volume /dev:/dev:rslave
 --volume /sys:/sys:rslave
 --cgroupns host
 --stop-signal SIGRTMIN+3
 --mount type=tmpfs,destination=/run
 --mount type=tmpfs,destination=/run/lock
 --mount type=tmpfs,destination=/var/lib/journal
 --volume /dev/pts
 --volume /dev/null:/dev/ptmx
 ` + selinuxVolume + `
 --volume /var/log/journal
 --volume /etc/hosts:/etc/hosts:ro
 --volume /etc/resolv.conf:/etc/resolv.conf:ro
 --volume /dev/null:/run/.distrobox.rootless:ro
 ` + groupAddFlags + `
 --volume /path/to/my-volume:/var/local/my-volume:ro
 --volume /path/to/distrobox-init:/usr/bin/entrypoint:ro
 --entrypoint /usr/bin/entrypoint
 my-image
 --verbose
 --name user
 --user 1000
 --group 1000
 --home /home/user
 --init 1
 --nvidia 0
 --pre-init-hooks echo 'pre-init-hook'
 --additional-packages
 -- echo 'init-hook'
`)

	got := oneline(strings.Join(cmd, " "))

	assert.Equal(t, expected, got)
}

func TestBuildContainerPath(t *testing.T) {
	tests := []struct {
		name          string
		cleanPath     bool
		hostPath      string
		containerPath string
		want          string
	}{
		{
			name:          "cleanPath returns only standard paths",
			cleanPath:     true,
			hostPath:      "/custom/path:/other/path",
			containerPath: "/container/path",
			want:          "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		},
		{
			name:      "cleanPath ignores hostPath and containerPath",
			cleanPath: true,
			hostPath:  "",
			want:      "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		},
		{
			name:          "hostPath has all standard paths - returns hostPath only",
			cleanPath:     false,
			hostPath:      "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			containerPath: "/container/path",
			want:          "/usr/local/sbin:/usr/sbin:/usr/local/bin:/usr/bin:/sbin:/bin",
		},
		{
			name:          "hostPath missing some standard paths - adds them",
			cleanPath:     false,
			hostPath:      "/usr/bin:/bin",
			containerPath: "/container/path",
			want:          "/usr/local/bin:/usr/bin:/bin:/usr/local/sbin:/usr/sbin:/sbin",
		},
		{
			name:          "hostPath with custom paths and missing standard paths",
			cleanPath:     false,
			hostPath:      "/custom/bin:/usr/bin:/another/path",
			containerPath: "/container/path",
			want:          "/custom/bin:/usr/local/bin:/usr/bin:/another/path:/usr/local/sbin:/usr/sbin:/sbin:/bin",
		},
		{
			name:          "empty hostPath - returns containerPath",
			cleanPath:     false,
			hostPath:      "",
			containerPath: "/container/path",
			want:          "/container/path",
		},
		{
			name:      "empty hostPath and containerPath - returns standard paths",
			cleanPath: false,
			hostPath:  "",
			want:      "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		},
		{
			name:      "hostPath with standard paths at beginning",
			cleanPath: false,
			hostPath:  "/bin:/usr/bin:/custom/path",
			want:      "/bin:/usr/local/bin:/usr/bin:/custom/path:/usr/local/sbin:/usr/sbin:/sbin",
		},
		{
			name:      "hostPath with standard paths at end",
			cleanPath: false,
			hostPath:  "/custom/path:/bin:/usr/bin",
			want:      "/custom/path:/bin:/usr/local/bin:/usr/bin:/usr/local/sbin:/usr/sbin:/sbin",
		},
		{
			name:      "hostPath with similar but not exact standard paths",
			cleanPath: false,
			hostPath:  "/usr/binary:/binfo",
			want:      "/usr/binary:/binfo:/usr/local/sbin:/usr/sbin:/usr/local/bin:/usr/bin:/sbin:/bin",
		},
		{
			name:          "single custom path in hostPath",
			cleanPath:     false,
			hostPath:      "/opt/custom/bin",
			containerPath: "/container/path",
			want:          "/opt/custom/bin:/usr/local/sbin:/usr/sbin:/usr/local/bin:/usr/bin:/sbin:/bin",
		},
		{
			name:      "empty containerPath with non-empty hostPath",
			cleanPath: false,
			hostPath:  "/custom/path",
			want:      "/custom/path:/usr/local/sbin:/usr/sbin:/usr/local/bin:/usr/bin:/sbin:/bin",
		},
		{
			name: "empty containerPath with empty hostPath",
			want: "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containermanager.BuildContainerPath(tt.cleanPath, tt.hostPath, tt.containerPath)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestBuildContainerPathEdgeCases tests edge cases and boundary conditions
func TestBuildContainerPathEdgeCases(t *testing.T) {
	t.Run("hostPath with colon at start", func(t *testing.T) {
		got := containermanager.BuildContainerPath(false, ":/usr/bin", "")
		// Should treat this as hostPath not containing standard paths initially
		assert.True(t, contains(got, "/usr/local/sbin"), "Expected standard paths to be added")
	})

	t.Run("hostPath with colon at end", func(t *testing.T) {
		got := containermanager.BuildContainerPath(false, "/usr/bin:", "")
		assert.True(t, contains(got, "/usr/local/sbin"), "Expected standard paths to be added")
	})

	t.Run("hostPath with multiple colons", func(t *testing.T) {
		got := containermanager.BuildContainerPath(false, "/usr/bin::/sbin", "")
		// Should still add missing standard paths
		assert.True(t, contains(got, "/usr/local/sbin"), "Expected standard paths to be added")
	})
}

func TestBuildCommandArgs(t *testing.T) {
	tests := []struct {
		name          string
		customCommand []string
		user          string
		noTTY         bool
		unshareGroups bool
		want          string
	}{
		{
			name:          "custom command without unshare",
			customCommand: []string{"/bin/bash"},
			user:          "testuser",
			noTTY:         false,
			unshareGroups: false,
			want:          "/bin/bash",
		},
		{
			name:          "custom command with unshare, no TTY",
			customCommand: []string{"/usr/bin/python3"},
			user:          "testuser",
			noTTY:         true,
			unshareGroups: true,
			want:          `su|-m|-s|/bin/sh|-c|"$0" "$@"|--|testuser|/usr/bin/python3`,
		},
		{
			name:          "custom command with unshare and TTY",
			customCommand: []string{"/bin/zsh"},
			user:          "testuser",
			noTTY:         false,
			unshareGroups: true,
			want:          `su|--pty|-m|-s|/bin/sh|-c|"$0" "$@"|--|testuser|/bin/zsh`,
		},
		{
			name:          "default shell without unshare",
			customCommand: nil,
			user:          "alice",
			noTTY:         false,
			unshareGroups: false,
			want:          "/bin/sh|-c|$(getent passwd 'alice' | cut -f 7 -d :) -l",
		},
		{
			name:          "default shell with unshare, no TTY",
			customCommand: nil,
			user:          "bob",
			noTTY:         true,
			unshareGroups: true,
			want:          `su|-m|-s|/bin/sh|-c|"$0" "$@"|--|bob|/bin/sh|-c|$(getent passwd 'bob' | cut -f 7 -d :) -l`,
		},
		{
			name:          "default shell with unshare and TTY",
			customCommand: nil,
			user:          "charlie",
			noTTY:         false,
			unshareGroups: true,
			want:          `su|--pty|-m|-s|/bin/sh|-c|"$0" "$@"|--|charlie|/bin/sh|-c|$(getent passwd 'charlie' | cut -f 7 -d :) -l`,
		},
		{
			name:          "empty custom command treated as default",
			customCommand: []string{},
			user:          "testuser",
			noTTY:         false,
			unshareGroups: false,
			want:          "/bin/sh|-c|$(getent passwd 'testuser' | cut -f 7 -d :) -l",
		},
		{
			name:          "multi-arg custom command preserves argv boundaries",
			customCommand: []string{"/bin/bash", "-c", "echo hello"},
			user:          "testuser",
			noTTY:         false,
			unshareGroups: false,
			want:          "/bin/bash|-c|echo hello",
		},
		{
			name:          "multi-arg custom command preserves argv boundaries through unshare",
			customCommand: []string{"/bin/bash", "-c", "echo hello"},
			user:          "testuser",
			noTTY:         true,
			unshareGroups: true,
			want:          `su|-m|-s|/bin/sh|-c|"$0" "$@"|--|testuser|/bin/bash|-c|echo hello`,
		},
		{
			name:          "user with special characters in default shell",
			customCommand: nil,
			user:          "user-name.test",
			noTTY:         false,
			unshareGroups: false,
			want:          "/bin/sh|-c|$(getent passwd 'user-name.test' | cut -f 7 -d :) -l",
		},
		{
			name:          "root user without unshare",
			customCommand: nil,
			user:          "root",
			noTTY:         false,
			unshareGroups: false,
			want:          "/bin/sh|-c|$(getent passwd 'root' | cut -f 7 -d :) -l",
		},
		{
			name:          "root user with unshare",
			customCommand: nil,
			user:          "root",
			noTTY:         true,
			unshareGroups: true,
			want:          `su|-m|-s|/bin/sh|-c|"$0" "$@"|--|root|/bin/sh|-c|$(getent passwd 'root' | cut -f 7 -d :) -l`,
		},
		{
			name:          "empty user with default shell",
			customCommand: nil,
			user:          "",
			noTTY:         false,
			unshareGroups: false,
			want:          "/bin/sh|-c|$(getent passwd '' | cut -f 7 -d :) -l",
		},
		{
			name:          "complex sh -c script preserved as single arg through unshare",
			customCommand: []string{"/bin/bash", "-c", "for i in {1..100}; do echo $i; done"},
			user:          "user",
			noTTY:         true,
			unshareGroups: true,
			want:          `su|-m|-s|/bin/sh|-c|"$0" "$@"|--|user|/bin/bash|-c|for i in {1..100}; do echo $i; done`,
		},
		{
			name:          "single-arg shell-quoted command preserved verbatim",
			customCommand: []string{"echo a || echo b"},
			user:          "user",
			noTTY:         true,
			unshareGroups: false,
			want:          "echo a || echo b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containermanager.BuildCommandArgs(tt.customCommand, tt.user, tt.noTTY, tt.unshareGroups)
			gotStr := strings.Join(got, "|")

			assert.Equal(t, tt.want, gotStr)
		})
	}
}

// TestBuildCommandArgsMatrix tests all combinations of boolean flags
func TestBuildCommandArgsMatrix(t *testing.T) {
	tests := []struct {
		name            string
		customCommand   []string
		noTTY           bool
		unshareGroups   bool
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:            "no flags with custom command",
			customCommand:   []string{"/custom/cmd"},
			noTTY:           false,
			unshareGroups:   false,
			wantContains:    []string{"/custom/cmd"},
			wantNotContains: []string{"su", "--pty"},
		},
		{
			name:            "unshare only with custom command",
			customCommand:   []string{"/custom/cmd"},
			noTTY:           true,
			unshareGroups:   true,
			wantContains:    []string{"su", "/custom/cmd"},
			wantNotContains: []string{"--pty"},
		},
		{
			name:            "noTTY and unshare with custom command",
			customCommand:   []string{"/custom/cmd"},
			noTTY:           false,
			unshareGroups:   true,
			wantContains:    []string{"su", "--pty", "/custom/cmd"},
			wantNotContains: []string{},
		},
		{
			name:            "noTTY without unshare",
			customCommand:   []string{"/custom/cmd"},
			noTTY:           true,
			unshareGroups:   false,
			wantContains:    []string{"/custom/cmd"},
			wantNotContains: []string{"su", "--pty"},
		},
		{
			name:            "default shell without flags",
			customCommand:   nil,
			noTTY:           false,
			unshareGroups:   false,
			wantContains:    []string{"/bin/sh", "-c", "getent passwd"},
			wantNotContains: []string{"su", "--pty"},
		},
		{
			name:            "default shell with unshare",
			customCommand:   nil,
			noTTY:           true,
			unshareGroups:   true,
			wantContains:    []string{"su", "/bin/sh", "-c", "getent passwd"},
			wantNotContains: []string{"--pty"},
		},
		{
			name:            "default shell with both flags",
			customCommand:   nil,
			noTTY:           false,
			unshareGroups:   true,
			wantContains:    []string{"su", "--pty", "/bin/sh", "-c", "getent passwd"},
			wantNotContains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containermanager.BuildCommandArgs(tt.customCommand, "user", tt.noTTY, tt.unshareGroups)
			resultStr := strings.Join(result, "|")

			for _, want := range tt.wantContains {
				assert.Contains(t, resultStr, want)
			}

			for _, notWant := range tt.wantNotContains {
				assert.NotContains(t, resultStr, notWant)
			}
		})
	}
}

// oneline is a helper function that removes newlines and trims spaces from a string.
func oneline(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(s, "\n", ""), "  ", " "))
}

// Helper function for edge case tests
func contains(path, substring string) bool {
	return len(path) >= len(substring) &&
		(path[:len(substring)] == substring ||
			len(path) > len(substring) && path[len(path)-len(substring):] == substring ||
			len(path) > len(substring)+1 && hasSubstring(path, ":"+substring+":") ||
			hasSubstring(path, ":"+substring) ||
			hasSubstring(path, substring+":"))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestDockerEnterPropagatesStartError(t *testing.T) {
	installFakeDockerRuntime(t)
	t.Setenv("FAKE_INSPECT_STDOUT", dockerFakeInspectJSON("exited"))
	t.Setenv("FAKE_START_EXIT", "9")
	t.Setenv("FAKE_START_STDERR", "start failed")

	err := NewDocker(false, "sudo", false).Enter(
		t.Context(),
		containermanager.EnterOptions{
			ContainerName: "box",
			NoTTY:         true,
			NoWorkDir:     true,
		},
		ui.NewDevNullProgress(),
		ui.NewPrinter(io.Discard, false),
	)

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to start container")
}

func TestDockerEnterPropagatesExecError(t *testing.T) {
	installFakeDockerRuntime(t)
	t.Setenv("FAKE_INSPECT_STDOUT", dockerFakeInspectJSON("running"))
	t.Setenv("FAKE_EXEC_EXIT", "7")
	t.Setenv("FAKE_EXEC_STDERR", "exec failed")

	err := NewDocker(false, "sudo", false).Enter(
		t.Context(),
		containermanager.EnterOptions{
			ContainerName: "box",
			NoTTY:         true,
			NoWorkDir:     true,
		},
		ui.NewDevNullProgress(),
		ui.NewPrinter(io.Discard, false),
	)

	require.Error(t, err)
	assert.ErrorContains(t, err, "exit status 7")
}

func TestDockerInspectContainer(t *testing.T) {
	installFakeDockerRuntime(t)

	tests := []struct {
		name    string
		inspect string
		want    containermanager.InspectResult
	}{
		{
			name:    "docker output",
			inspect: dockerFakeInspectFullJSON(),
			want: containermanager.InspectResult{
				ContainerID:     "abc123def456ghi789",
				ContainerStatus: "running",
				ContainerImage:  "registry.fedoraproject.org/fedora:40",
				ContainerHome:   "/home/user",
				ContainerPath:   "/usr/local/bin:/usr/bin:/bin",
				UnshareGroups:   true,
				NetworkMode:     "host",
				IpcMode:         "host",
				PidMode:         "host",
				// Docker exposes the entrypoint args as Config.Cmd, so the
				// top-level Args field must be ignored.
				Cmd: []string{"/usr/bin/entrypoint", "my-box", "init", "--nvidia"},
				Env: []string{
					"HOME=/home/user",
					"PATH=/usr/local/bin:/usr/bin:/bin",
					"HOSTNAME=my-box",
				},
				Labels: map[string]string{
					"distrobox":                "1",
					"distrobox.unshare_groups": "1",
					"distrobox.version":        "2.0.0",
				},
				Mounts: []containermanager.MountInfo{
					{Source: "/home/user/.local/share/distrobox/v2/distrobox-init", Destination: "/usr/bin/entrypoint", Options: "ro,z"},
					{Source: "/home/user", Destination: "/home/user", Options: "rw"},
				},
			},
		},
		{
			// Same parser is used for podman inspect output (ImageName and
			// mount Options are present there, and must take precedence).
			name:    "podman-shaped output",
			inspect: `[{"Id":"pod123","State":{"Status":"running"},"ImageName":"docker.io/library/ubuntu:24.04","Args":["/usr/bin/entrypoint","box","init"],"Config":{"Image":"ignored","Labels":{"distrobox":"1"},"Env":["HOME=/root","PATH=/usr/sbin:/usr/bin"],"Cmd":["/usr/bin/entrypoint","box","init"]},"Mounts":[{"Type":"bind","Source":"/root","Destination":"/root","Options":["rbind","rw"]}],"HostConfig":{"NetworkMode":"private","IpcMode":"shareable","PidMode":""}}]`,
			want: containermanager.InspectResult{
				ContainerID:     "pod123",
				ContainerStatus: "running",
				ContainerImage:  "docker.io/library/ubuntu:24.04",
				ContainerHome:   "/root",
				ContainerPath:   "/usr/sbin:/usr/bin",
				NetworkMode:     "private",
				IpcMode:         "shareable",
				Cmd:             []string{"/usr/bin/entrypoint", "box", "init"},
				Env:             []string{"HOME=/root", "PATH=/usr/sbin:/usr/bin"},
				Labels:          map[string]string{"distrobox": "1"},
				Mounts: []containermanager.MountInfo{
					{Source: "/root", Destination: "/root", Options: "rbind,rw"},
				},
			},
		},
		{
			name:    "minimal container",
			inspect: `[{"Id":"min","State":{"Status":"exited"},"Config":{"Labels":{"distrobox.unshare_groups":"0"}}}]`,
			want: containermanager.InspectResult{
				ContainerID:     "min",
				ContainerStatus: "exited",
				Labels:          map[string]string{"distrobox.unshare_groups": "0"},
				Mounts:          []containermanager.MountInfo{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("FAKE_INSPECT_STDOUT", tt.inspect)
			got, err := NewDocker(false, "sudo", false).InspectContainer(t.Context(), "my-box")
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.want, *got)
		})
	}
}

func TestDockerInspectContainerNotFound(t *testing.T) {
	installFakeDockerRuntime(t)
	t.Setenv("FAKE_INSPECT_STDOUT", "[]")

	got, err := NewDocker(false, "sudo", false).InspectContainer(t.Context(), "missing-box")
	require.Nil(t, got)
	require.Error(t, err)
	assert.ErrorContains(t, err, "container not found")
}

func installFakeDockerRuntime(t *testing.T) {
	t.Helper()

	tmpDir := t.TempDir()
	runtimePath := filepath.Join(tmpDir, "docker")

	const script = `#!/bin/sh
cmd="$1"
shift
case "$cmd" in
  inspect)
    if [ -n "$FAKE_INSPECT_STDOUT" ]; then
      printf "%s" "$FAKE_INSPECT_STDOUT"
    fi
    exit "${FAKE_INSPECT_EXIT:-0}"
    ;;
  start)
    if [ -n "$FAKE_START_STDERR" ]; then
      printf "%s" "$FAKE_START_STDERR" >&2
    fi
    exit "${FAKE_START_EXIT:-0}"
    ;;
  logs)
    if [ -n "$FAKE_LOGS_STDOUT" ]; then
      printf "%s" "$FAKE_LOGS_STDOUT"
    fi
    exit "${FAKE_LOGS_EXIT:-0}"
    ;;
  exec)
    if [ -n "$FAKE_EXEC_STDERR" ]; then
      printf "%s" "$FAKE_EXEC_STDERR" >&2
    fi
    exit "${FAKE_EXEC_EXIT:-0}"
    ;;
  *)
    exit 0
    ;;
esac
`

	err := os.WriteFile(runtimePath, []byte(script), 0o755)
	require.NoError(t, err)

	t.Setenv("PATH", tmpDir+":"+os.Getenv("PATH"))
	t.Setenv("USER", "testuser")
	t.Setenv("HOME", "/home/testuser")
	t.Setenv("SHELL", "/bin/sh")
}

func dockerFakeInspectJSON(status string) string {
	return `[{"Id":"container-id","State":{"Status":"` + status + `"},"Config":{"Labels":{"distrobox.unshare_groups":"0"},"Env":["HOME=/home/testuser","PATH=/usr/bin:/bin"]}}]`
}

// dockerFakeInspectFullJSON mirrors the shape of real `docker inspect` output:
// the image name lives in Config.Image (no top-level ImageName field) and
// bind-mount options live in Mode (there is no Options field).
func dockerFakeInspectFullJSON() string {
	return `[{
  "Id": "abc123def456ghi789",
  "Created": "2026-06-25T18:25:46Z",
  "Path": "/usr/bin/entrypoint",
  "Args": ["my-box"],
  "State": {"Status": "running"},
  "Image": "sha256:2a5415177518478c9cacb7becb2c27cffc5f1d5e147667024a9b7bb4dddec284",
  "HostConfig": {
    "NetworkMode": "host",
    "IpcMode": "host",
    "PidMode": "host"
  },
  "Mounts": [
    {
      "Type": "bind",
      "Source": "/home/user/.local/share/distrobox/v2/distrobox-init",
      "Destination": "/usr/bin/entrypoint",
      "Mode": "ro,z",
      "RW": false,
      "Propagation": "rprivate"
    },
    {
      "Type": "bind",
      "Source": "/home/user",
      "Destination": "/home/user",
      "Mode": "rw",
      "RW": true,
      "Propagation": "rprivate"
    }
  ],
  "Config": {
    "Image": "registry.fedoraproject.org/fedora:40",
    "Labels": {
      "distrobox": "1",
      "distrobox.unshare_groups": "1",
      "distrobox.version": "2.0.0"
    },
    "Env": [
      "HOME=/home/user",
      "PATH=/usr/local/bin:/usr/bin:/bin",
      "HOSTNAME=my-box"
    ],
    "Cmd": ["/usr/bin/entrypoint", "my-box", "init", "--nvidia"]
  }
}]`
}
