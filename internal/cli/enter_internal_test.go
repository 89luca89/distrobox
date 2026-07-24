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

package cli

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"

	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

// spyContainerManager records every Enter call so the test can assert on
// the EnterOptions the enter subcommand built from its argv. All other
// methods are no-ops — enterAction only ever calls Enter.
//
// existsResult controls what Exists returns. The enter command takes a
// detour into the create flow when Exists==false, so tests that only want
// to exercise argv parsing set it to true; tests that rely on a unique
// generated name (ephemeral) leave it false.
type spyContainerManager struct {
	mu           sync.Mutex
	calls        []containermanager.EnterOptions
	existsResult bool
}

func (s *spyContainerManager) Enter(_ context.Context, opts containermanager.EnterOptions, _ *ui.Progress, _ *ui.Printer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, opts)
	return nil
}

// The remaining methods are stubs to satisfy containermanager.ContainerManager.
// enterAction never reaches them.
func (s *spyContainerManager) Name() string                                   { return "spy" }
func (s *spyContainerManager) CloneAsRoot() containermanager.ContainerManager { return s }
func (s *spyContainerManager) Create(_ context.Context, _ containermanager.CreateOptions) error {
	return nil
}
func (s *spyContainerManager) ListContainers(_ context.Context) ([]containermanager.Container, error) {
	return nil, nil
}
func (s *spyContainerManager) Remove(_ context.Context, _ string, _ containermanager.RmOptions) error {
	return nil
}
func (s *spyContainerManager) Exists(_ context.Context, _ string) bool  { return s.existsResult }
func (s *spyContainerManager) Stop(_ context.Context, _ []string) error { return nil }
func (s *spyContainerManager) InspectContainer(_ context.Context, _ string) (*containermanager.InspectResult, error) {
	return &containermanager.InspectResult{}, nil
}
func (s *spyContainerManager) Commit(_ context.Context, _, _ string) error { return nil }
func (s *spyContainerManager) ImageExists(_ context.Context, _ string) bool {
	return false
}
func (s *spyContainerManager) PullImage(_ context.Context, _, _ string, _ bool) error { return nil }

// runEnter runs the enter subcommand with the given argv (starting from
// "enter") against a spy container manager and returns the EnterOptions
// the spy was invoked with.
func runEnter(t *testing.T, argv ...string) containermanager.EnterOptions {
	t.Helper()

	// enterAction offers to create the container when Exists==false, which
	// would drag this argv-parsing test into the create flow. Pretend the
	// container is already there so we go straight to Enter.
	spy := &spyContainerManager{existsResult: true}
	cmd := newEnterCommand(config.DefaultValues())

	// The production Before hook (installed by withContainerManager in
	// root.go) tries to detect a real container manager and validate
	// sudo. We don't need any of that for parsing tests — we just want
	// to plant the spy where enterAction looks for it.
	cmd.Before = func(ctx context.Context, _ *cli.Command) (context.Context, error) {
		return context.WithValue(ctx, containerManagerKey, spy), nil
	}

	root := &cli.Command{Commands: []*cli.Command{cmd}}
	// The real entrypoint (cmd/distrobox/main.go) always runs argv through
	// PrepareArgs before handing it to urfave; mirror that here.
	full := PrepareArgs(root, append([]string{"distrobox"}, argv...))

	require.NoError(t, root.Run(context.Background(), full))
	require.NotEmpty(t, spy.calls, "expected Enter to be called for argv %v", argv)

	return spy.calls[0]
}

// TestEnterCommand_CustomCommandVariants locks down the three equivalent
// ways of supplying a custom command to `distrobox enter` — `-e`,
// `--exec`, and the bare `--` separator — as well as the implicit form
// where positional args after the container name are taken as the
// custom command. All four must produce the same (container,
// customCommand) pair, matching the original bash distrobox-enter.
func TestEnterCommand_CustomCommandVariants(t *testing.T) {
	cases := []struct {
		name string
		argv []string
	}{
		{
			name: "short -e flag",
			argv: []string{"enter", "suse", "-e", "echo", "ciao"},
		},
		{
			name: "long --exec flag",
			argv: []string{"enter", "suse", "--exec", "echo", "ciao"},
		},
		{
			name: "bare -- separator",
			argv: []string{"enter", "suse", "--", "echo", "ciao"},
		},
		{
			name: "implicit (no -e/--exec/--)",
			argv: []string{"enter", "suse", "echo", "ciao"},
		},
		{
			name: "explicit --name with -e",
			argv: []string{"enter", "--name", "suse", "-e", "echo", "ciao"},
		},
		{
			name: "explicit --name with --",
			argv: []string{"enter", "--name", "suse", "--", "echo", "ciao"},
		},
		{
			// Regression: the custom command itself contains a short
			// flag (e.g. `bash -c echo`). The CLI must not eat it as
			// a distrobox flag — matching the original bash script.
			name: "custom command with short flag after -e",
			argv: []string{"enter", "suse", "-e", "bash", "-c", "echo"},
		},
		{
			name: "custom command with short flag after --",
			argv: []string{"enter", "suse", "--", "bash", "-c", "echo"},
		},
		{
			name: "-e before the container name",
			argv: []string{"enter", "-e", "suse", "bash", "-c", "echo"},
		},
		{
			name: "short flag combo with --no-tty before container name",
			argv: []string{"enter", "--no-tty", "suse", "-e", "bash", "-c", "echo"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := runEnter(t, tc.argv...)

			assert.Equal(t, "suse", opts.ContainerName)

			want := []string{"echo", "ciao"}
			if strings.HasPrefix(tc.name, "custom command with short flag") ||
				strings.HasPrefix(tc.name, "-e before the container name") ||
				strings.HasPrefix(tc.name, "short flag combo with --no-tty") {
				want = []string{"bash", "-c", "echo"}
			}
			assert.Equal(t, want, opts.CustomCommand)
		})
	}
}

// TestEnterCommand_FlagsAfterContainerName is the regression suite for the
// bug this change fixes: distrobox flags placed after the bare container name
// must be parsed as distrobox flags (not swallowed into the custom command),
// mirroring the original bash distrobox-enter.
func TestEnterCommand_FlagsAfterContainerName(t *testing.T) {
	t.Run("additional-flags after name is consumed", func(t *testing.T) {
		opts := runEnter(t, "enter", "suse", "--additional-flags", "--env FOO=bar", "--", "printenv", "FOO")
		assert.Equal(t, "suse", opts.ContainerName)
		assert.Equal(t, "--env FOO=bar", opts.AdditionalFlags)
		assert.Equal(t, []string{"printenv", "FOO"}, opts.CustomCommand)
	})

	t.Run("short -a after name, implicit command", func(t *testing.T) {
		opts := runEnter(t, "enter", "suse", "-a", "--pids-limit 100", "bash")
		assert.Equal(t, "suse", opts.ContainerName)
		assert.Equal(t, "--pids-limit 100", opts.AdditionalFlags)
		assert.Equal(t, []string{"bash"}, opts.CustomCommand)
	})

	t.Run("--no-tty after name is consumed", func(t *testing.T) {
		opts := runEnter(t, "enter", "suse", "--no-tty", "--", "bash")
		assert.Equal(t, "suse", opts.ContainerName)
		assert.True(t, opts.NoTTY)
		assert.Equal(t, []string{"bash"}, opts.CustomCommand)
	})

	// Regression for the naive StopOnNthArg:2 fix: when --name supplies the
	// name, the command's own short flag (-c) must not be parsed as
	// --clean-path.
	t.Run("--name then command with short flag", func(t *testing.T) {
		opts := runEnter(t, "enter", "--name", "suse", "bash", "-c", "echo")
		assert.Equal(t, "suse", opts.ContainerName)
		assert.False(t, opts.CleanPath)
		assert.Equal(t, []string{"bash", "-c", "echo"}, opts.CustomCommand)
	})

	// A flag that follows the command word belongs to the command.
	t.Run("flag after command word stays in the command", func(t *testing.T) {
		opts := runEnter(t, "enter", "suse", "vim", "--help")
		assert.Equal(t, "suse", opts.ContainerName)
		assert.Equal(t, []string{"vim", "--help"}, opts.CustomCommand)
	})
}

// runEnterRaw runs enter through PrepareArgs and returns the spy plus the Run
// error, without asserting either — for cases that intentionally stop before
// Enter (help, invalid flag).
func runEnterRaw(t *testing.T, argv ...string) (*spyContainerManager, error) {
	t.Helper()

	spy := &spyContainerManager{existsResult: true}
	cmd := newEnterCommand(config.DefaultValues())
	cmd.Before = func(ctx context.Context, _ *cli.Command) (context.Context, error) {
		return context.WithValue(ctx, containerManagerKey, spy), nil
	}

	root := &cli.Command{Commands: []*cli.Command{cmd}, Writer: io.Discard, ErrWriter: io.Discard}
	full := PrepareArgs(root, append([]string{"distrobox"}, argv...))

	return spy, root.Run(context.Background(), full)
}

// TestEnterCommand_NoEnterAfterName pins down that a recognized help flag
// after the name shows help, and an unrecognized flag after the name is
// rejected — in both cases without entering the container.
func TestEnterCommand_NoEnterAfterName(t *testing.T) {
	t.Run("--help after name shows help, does not enter", func(t *testing.T) {
		spy, err := runEnterRaw(t, "enter", "suse", "--help")
		require.NoError(t, err)
		assert.Empty(t, spy.calls, "expected help, not Enter")
	})

	t.Run("unknown flag after name errors, does not enter", func(t *testing.T) {
		spy, err := runEnterRaw(t, "enter", "suse", "--frobnicate")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not defined")
		assert.Empty(t, spy.calls, "expected error, not Enter")
	})
}
