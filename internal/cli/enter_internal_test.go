package cli

import (
	"context"
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
	full := append([]string{"distrobox"}, argv...)

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
			argv: []string{"enter", "suse", "-e", "echo", "hello"},
		},
		{
			name: "long --exec flag",
			argv: []string{"enter", "suse", "--exec", "echo", "hello"},
		},
		{
			name: "bare -- separator",
			argv: []string{"enter", "suse", "--", "echo", "hello"},
		},
		{
			name: "implicit (no -e/--exec/--)",
			argv: []string{"enter", "suse", "echo", "hello"},
		},
		{
			name: "explicit --name with -e",
			argv: []string{"enter", "--name", "suse", "-e", "echo", "hello"},
		},
		{
			name: "explicit --name with --",
			argv: []string{"enter", "--name", "suse", "--", "echo", "hello"},
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

			want := []string{"echo", "hello"}
			if strings.HasPrefix(tc.name, "custom command with short flag") ||
				strings.HasPrefix(tc.name, "-e before the container name") ||
				strings.HasPrefix(tc.name, "short flag combo with --no-tty") {
				want = []string{"bash", "-c", "echo"}
			}
			assert.Equal(t, want, opts.CustomCommand)
		})
	}
}

func TestFindExecMarkerIndex(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want int
	}{
		{
			name: "empty args",
			args: nil,
			want: -1,
		},
		{
			name: "no marker",
			args: []string{"suse", "echo", "hello"},
			want: -1,
		},
		{
			name: "short -e is the only arg",
			args: []string{"-e"},
			want: 0,
		},
		{
			name: "long --exec is the only arg",
			args: []string{"--exec"},
			want: 0,
		},
		{
			name: "-e at the start",
			args: []string{"-e", "bash", "-c", "echo"},
			want: 0,
		},
		{
			name: "--exec in the middle",
			args: []string{"suse", "--exec", "bash", "-c", "echo"},
			want: 1,
		},
		{
			name: "-e at the end with no command after it",
			args: []string{"suse", "-e"},
			want: 1,
		},
		{
			name: "first match wins when both forms are present",
			args: []string{"-e", "suse", "--exec", "echo"},
			want: 0,
		},
		{
			name: "marker-looking arg inside the custom command is not a match",
			// `bash` happens to start with `b`, not `-`, so it must
			// never be picked up. This guards against a future
			// regression that would substring-match.
			args: []string{"suse", "bash", "-e", "echo"},
			want: 2,
		},
		{
			name: "looks similar but is not the marker",
			// `--exec-foo` shares a prefix with `--exec` but is a
			// distinct token; it must not be treated as the marker.
			args: []string{"suse", "--exec-foo", "echo"},
			want: -1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, findExecMarkerIndex(tc.args))
		})
	}
}

func TestExtractAdditionalFlagsFromPositionals(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		wantArgs []string
		wantFlag string
	}{
		{
			name:     "empty args",
			args:     nil,
			wantArgs: nil,
			wantFlag: "",
		},
		{
			name:     "no additional-flags",
			args:     []string{"suse", "echo", "hello"},
			wantArgs: []string{"suse", "echo", "hello"},
			wantFlag: "",
		},
		{
			name:     "long --additional-flags with value",
			args:     []string{"suse", "--additional-flags", "--tty", "echo", "hello"},
			wantArgs: []string{"suse", "echo", "hello"},
			wantFlag: "--tty",
		},
		{
			name:     "short -a with value",
			args:     []string{"suse", "-a", "--tty", "echo", "hello"},
			wantArgs: []string{"suse", "echo", "hello"},
			wantFlag: "--tty",
		},
		{
			name:     "--additional-flags with = syntax",
			args:     []string{"suse", "--additional-flags=--tty", "echo", "hello"},
			wantArgs: []string{"suse", "echo", "hello"},
			wantFlag: "--tty",
		},
		{
			name:     "-a with = syntax",
			args:     []string{"suse", "-a=--tty", "echo", "hello"},
			wantArgs: []string{"suse", "echo", "hello"},
			wantFlag: "--tty",
		},
		{
			name:     "--additional-flags at end with no value",
			args:     []string{"suse", "--additional-flags"},
			wantArgs: []string{"suse"},
			wantFlag: "",
		},
		{
			name:     "--additional-flags before -e marker",
			args:     []string{"suse", "--additional-flags", "--tty", "-e", "echo", "hello"},
			wantArgs: []string{"suse", "-e", "echo", "hello"},
			wantFlag: "--tty",
		},
		{
			name:     "--additional-flags at the very start",
			args:     []string{"--additional-flags", "--tty", "suse", "echo", "hello"},
			wantArgs: []string{"suse", "echo", "hello"},
			wantFlag: "--tty",
		},
		{
			name:     "-a at end with no value",
			args:     []string{"suse", "echo", "hello", "-a"},
			wantArgs: []string{"suse", "echo", "hello"},
			wantFlag: "",
		},
		{
			name:     "only flag with = value",
			args:     []string{"--additional-flags=--preserve-fds"},
			wantArgs: []string{},
			wantFlag: "--preserve-fds",
		},
		{
			name:     "value is a single hyphen flag",
			args:     []string{"test", "--additional-flags", "--tty"},
			wantArgs: []string{"test"},
			wantFlag: "--tty",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotArgs, gotFlag := extractAdditionalFlagsFromPositionals(tc.args)
			assert.Equal(t, tc.wantArgs, gotArgs)
			assert.Equal(t, tc.wantFlag, gotFlag)
		})
	}
}

// TestEnterCommand_AdditionalFlagsFromTail verifies that --additional-flags
// placed after the container name (i.e. in the positional tail) is correctly
// extracted and passed as AdditionalFlags to the container manager, rather
// than ending up in the custom command. This is the regression introduced
// in the Go rewrite (v2.0.0-rc.3).
func TestEnterCommand_AdditionalFlagsFromTail(t *testing.T) {
	cases := []struct {
		name string
		argv []string
		want containermanager.EnterOptions
	}{
		{
			// Primary regression: Ptyxis uses this invocation pattern.
			name: "--additional-flags after container name (Ptyxis pattern)",
			argv: []string{"enter", "--no-tty", "test", "--additional-flags", "--tty"},
			want: containermanager.EnterOptions{
				ContainerName:   "test",
				AdditionalFlags: "--tty",
				NoTTY:           true,
				// No custom command → Go returns empty slice, not nil
				CustomCommand: []string{},
			},
		},
		{
			name: "-a after container name with custom command",
			argv: []string{"enter", "test", "-a", "--env", "FOO=bar", "echo", "hello"},
			want: containermanager.EnterOptions{
				ContainerName:   "test",
				AdditionalFlags: "--env",
				CustomCommand:   []string{"FOO=bar", "echo", "hello"},
			},
		},
		{
			name: "--additional-flags before container name (normal parsing)",
			argv: []string{"enter", "--additional-flags", "--tty", "test", "echo", "hello"},
			want: containermanager.EnterOptions{
				ContainerName:   "test",
				AdditionalFlags: "--tty",
				CustomCommand:   []string{"echo", "hello"},
			},
		},
		{
			name: "--additional-flags after container name with --exec",
			argv: []string{"enter", "--no-tty", "test", "--additional-flags", "--tty", "-e", "echo", "hello"},
			want: containermanager.EnterOptions{
				ContainerName:   "test",
				AdditionalFlags: "--tty",
				NoTTY:           true,
				CustomCommand:   []string{"echo", "hello"},
			},
		},
		{
			name: "--additional-flags with = syntax after container name",
			argv: []string{"enter", "test", "--additional-flags=--preserve-fds", "--", "bash", "-l"},
			want: containermanager.EnterOptions{
				ContainerName:   "test",
				AdditionalFlags: "--preserve-fds",
				// Note: when StopOnNthArg triggers on a non-`--` arg,
				// urfave/cli returns before the `--` separator check,
				// so `--` remains in the positional tail.
				CustomCommand: []string{"--", "bash", "-l"},
			},
		},
		{
			name: "no additional-flags should leave field empty",
			argv: []string{"enter", "test", "echo", "hello"},
			want: containermanager.EnterOptions{
				ContainerName:   "test",
				AdditionalFlags: "",
				CustomCommand:   []string{"echo", "hello"},
			},
		},
		{
			name: "--name flag with --additional-flags before positional (normal parsing)",
			argv: []string{"enter", "--name", "mybox", "--additional-flags", "--tty", "echo", "hello"},
			want: containermanager.EnterOptions{
				ContainerName:   "mybox",
				AdditionalFlags: "--tty",
				CustomCommand:   []string{"echo", "hello"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := runEnter(t, tc.argv...)
			assert.Equal(t, tc.want.ContainerName, opts.ContainerName)
			assert.Equal(t, tc.want.AdditionalFlags, opts.AdditionalFlags)
			assert.Equal(t, tc.want.CustomCommand, opts.CustomCommand)
			assert.Equal(t, tc.want.NoTTY, opts.NoTTY)
		})
	}
}
