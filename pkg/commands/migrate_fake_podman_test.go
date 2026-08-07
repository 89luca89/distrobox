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

package commands_test

import (
	"bufio"
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/89luca89/distrobox/pkg/commands"
	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/containermanager/providers"
	"github.com/89luca89/distrobox/pkg/ui"
)

// v1InspectJSON is a realistic `podman inspect` document for a v1 distrobox
// container: helper scripts bind-mounted from the v1 system location
// (/usr/lib/distrobox), no distrobox.version label, and the v1
// distrobox-init invocation stored in the podman-specific top-level Args.
const v1InspectJSON = `[{
  "Id": "abc123def456",
  "State": {"Status": "running"},
  "ImageName": "quay.io/toolbx-images/alpine-toolbox:edge",
  "Args": [
    "--verbose", "--name", "testuser", "--user", "1000", "--group", "1000",
    "--home", "/home/testuser", "--init", "0", "--nvidia", "0",
    "--pre-init-hooks", "", "--additional-packages", " vim git", "--", ""
  ],
  "Config": {
    "Image": "quay.io/toolbx-images/alpine-toolbox:edge",
    "Labels": {"manager": "distrobox", "distrobox.unshare_groups": "0"},
    "Env": [
      "HOME=/home/testuser", "PATH=/usr/bin:/bin", "SHELL=/bin/sh",
      "container=podman", "HOSTNAME=boxhost"
    ]
  },
  "HostConfig": {"NetworkMode": "host", "IpcMode": "host", "PidMode": "host"},
  "Mounts": [
    {"Source": "/usr/lib/distrobox/distrobox-init", "Destination": "/usr/bin/entrypoint", "Type": "bind", "Options": ["ro"]},
    {"Source": "/usr/lib/distrobox/distrobox-export", "Destination": "/usr/bin/distrobox-export", "Type": "bind", "Options": ["ro"]},
    {"Source": "/usr/lib/distrobox/distrobox-host-exec", "Destination": "/usr/bin/distrobox-host-exec", "Type": "bind", "Options": ["ro"]},
    {"Source": "/home/testuser", "Destination": "/home/testuser", "Type": "bind", "Options": ["rslave"]},
    {"Source": "/tmp", "Destination": "/tmp", "Type": "bind", "Options": ["rslave"]},
    {"Source": "/dev", "Destination": "/dev", "Type": "bind", "Options": ["rslave"]},
    {"Source": "/", "Destination": "/run/host", "Type": "bind", "Options": ["rslave"]},
    {"Source": "/run/.nopasswd", "Destination": "/run/.nopasswd", "Type": "bind", "Options": ["ro"]}
  ]
}]`

// installMigrateFakePodman puts a fake `podman` binary first on PATH. It
// answers `inspect` with FAKE_INSPECT_STDOUT (the v1 document above) and
// records every other invocation, one argument per line, into the returned
// log file. This lets the test drive the real Podman provider through a full
// migration without a container runtime.
func installMigrateFakePodman(t *testing.T) string {
	t.Helper()

	logFile := filepath.Join(t.TempDir(), "podman.log")
	t.Setenv("FAKE_LOG_FILE", logFile)
	t.Setenv("FAKE_INSPECT_STDOUT", v1InspectJSON)

	tmpDir := t.TempDir()
	runtimePath := filepath.Join(tmpDir, "podman")

	const script = `#!/bin/sh
cmd="$1"
shift
if [ "$cmd" = "inspect" ]; then
  if [ -n "$FAKE_INSPECT_STDOUT" ]; then
    printf "%s" "$FAKE_INSPECT_STDOUT"
  fi
  exit "${FAKE_INSPECT_EXIT:-0}"
fi
if [ -n "$FAKE_LOG_FILE" ]; then
  printf 'cmd %s\n' "$cmd" >> "$FAKE_LOG_FILE"
  for a in "$@"; do
    printf 'arg %s\n' "$a" >> "$FAKE_LOG_FILE"
  done
fi
exit 0
`

	require.NoError(t, os.WriteFile(runtimePath, []byte(script), 0o755))
	t.Setenv("PATH", tmpDir+":"+os.Getenv("PATH"))
	t.Setenv("USER", "testuser")
	t.Setenv("HOME", "/home/testuser")
	t.Setenv("SHELL", "/bin/sh")

	return logFile
}

// fakeCall is one invocation recorded by the fake podman binary.
type fakeCall struct {
	name string
	args []string
}

// readFakeLog parses the fake podman log into ordered invocations. Each
// invocation is a "cmd <name>" line followed by one "arg <value>" line per
// argument.
func readFakeLog(t *testing.T, logFile string) []fakeCall {
	t.Helper()

	data, err := os.ReadFile(logFile)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var calls []fakeCall
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "cmd "):
			calls = append(calls, fakeCall{name: strings.TrimPrefix(line, "cmd ")})
		case strings.HasPrefix(line, "arg ") && len(calls) > 0:
			calls[len(calls)-1].args = append(calls[len(calls)-1].args, strings.TrimPrefix(line, "arg "))
		}
	}
	return calls
}

// findCall returns the first recorded invocation whose name or argument list
// contains match.
func findCall(calls []fakeCall, match string) *fakeCall {
	for i := range calls {
		if calls[i].name == match {
			return &calls[i]
		}
		for _, arg := range calls[i].args {
			if arg == match {
				return &calls[i]
			}
		}
	}
	return nil
}

// TestMigrateEndToEnd_V1ContainerBecomesV2 runs the full migration against a
// fake podman runtime: a v1 inspect document goes in, and the recreated
// container's spec (the `podman create` invocation) must come out with v2
// mount points sourced from the configured scripts directory, tagged with the
// current schema version, and with every v1 helper mount gone.
func TestMigrateEndToEnd_V1ContainerBecomesV2(t *testing.T) {
	logFile := installMigrateFakePodman(t)
	scriptsDir := t.TempDir()
	t.Setenv("DBX_SCRIPTS_DIR", scriptsDir)

	cfg := config.DefaultValues()
	require.Equal(t, scriptsDir, cfg.ScriptsDir)

	printer := ui.NewPrinter(io.Discard, false)
	prompter := ui.NewPrompter(*bufio.NewReader(strings.NewReader("")), nil)
	cm := providers.NewPodman(false, "sudo", false, false)

	migrateCmd := commands.NewMigrateCommand(cfg, cm, printer, prompter)
	err := migrateCmd.Execute(context.Background(), commands.MigrateOptions{
		ContainerNames: []string{"my-box"},
		NonInteractive: true,
	})
	require.NoError(t, err)

	// The migration must stop, commit, remove and recreate in order.
	calls := readFakeLog(t, logFile)
	stop := findCall(calls, "stop")
	commit := findCall(calls, "commit")
	rm := findCall(calls, "rm")
	create := findCall(calls, "create")
	require.NotNil(t, stop, "expected a podman stop invocation")
	require.NotNil(t, commit, "expected a podman commit invocation")
	require.NotNil(t, rm, "expected a podman rm invocation")
	require.NotNil(t, create, "expected a podman create invocation")

	// Order matters: stop -> commit -> remove -> create.
	names := []string{}
	for i := range calls {
		names = append(names, calls[i].name)
	}
	assert.Less(t, indexOf(names, "stop"), indexOf(names, "container"), "stop must precede commit")
	assert.Less(t, indexOf(names, "container"), indexOf(names, "rm"), "commit must precede remove")
	assert.Less(t, indexOf(names, "rm"), indexOf(names, "create"), "remove must precede create")

	// The temporary image is a commit of the original container.
	require.Len(t, commit.args, 3) // "commit" + container ID + tag
	require.Equal(t, "abc123def456", commit.args[1])
	assert.Regexp(t, `^my-box:migrate-\d{8}-\d{6}$`, commit.args[2])

	// The recreated container's spec is the "v2 json": compare it against the
	// known-good expectations below.
	args := create.args

	// Identity is preserved.
	assert.Contains(t, args, "--name")
	assert.Equal(t, "my-box", argValue(args, "--name"))

	// The image is the freshly committed one, not the original image.
	assert.Regexp(t, `^my-box:migrate-\d{8}-\d{6}$`, args[createImageIndex(args)])

	// Helper scripts are now mounted from the configured v2 scripts dir.
	assert.Contains(t, args, filepath.Join(scriptsDir, "distrobox-init")+":/usr/bin/entrypoint:ro")
	assert.Contains(t, args, filepath.Join(scriptsDir, "distrobox-export")+":/usr/bin/distrobox-export:ro")
	assert.Contains(t, args, filepath.Join(scriptsDir, "distrobox-host-exec")+":/usr/bin/distrobox-host-exec:ro")

	// The v1 helper mounts are gone.
	for _, arg := range args {
		assert.NotContains(t, arg, "/usr/lib/distrobox", "v1 helper mount leaked into v2 spec: %s", arg)
	}

	// The container is tagged with the current schema version, so the next
	// run of `distrobox migrate` skips it.
	versionLabel := containermanager.VersionLabelKey + "=" + strconv.Itoa(containermanager.SchemaVersion)
	assert.Contains(t, args, versionLabel)

	// The nopasswd marker mount recovered from the v1 mount list is restored.
	assert.Contains(t, args, "/dev/null:/run/.nopasswd:ro")

	// Options recovered from the v1 init args and inspect data survive.
	assert.Equal(t, "boxhost", argValue(args, "--hostname"))
	assert.Equal(t, "host", argValue(args, "--network"))
	assert.Equal(t, "host", argValue(args, "--ipc"))
	assert.Equal(t, "host", argValue(args, "--pid"))
	assert.Equal(t, "vim git", argValue(args, "--additional-packages"))
}

// argValue returns the value following the first occurrence of flag in args.
func argValue(args []string, flag string) string {
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// createImageIndex finds the image positional argument in the create args:
// it is the first argument that is not a flag, not a flag value, and matches
// the commit-tag shape.
func createImageIndex(args []string) int {
	for i, arg := range args {
		if strings.HasPrefix(arg, "my-box:migrate-") {
			return i
		}
	}
	return -1
}

func indexOf(list []string, v string) int {
	for i, item := range list {
		if item == v {
			return i
		}
	}
	return len(list) + 1
}
