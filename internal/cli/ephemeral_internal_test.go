package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"

	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager"
)

// runEphemeral runs the ephemeral subcommand with the given argv against a
// spy container manager and returns the EnterOptions that the enter step
// was finally invoked with. Ephemeral always calls Enter exactly once (the
// create + enter are paired), so this is enough to assert the custom
// command shape.
func runEphemeral(t *testing.T, argv ...string) containermanager.EnterOptions {
	t.Helper()

	spy := &spyContainerManager{}
	cmd := newEphemeralCommand(config.DefaultValues())

	// Replace the production Before hook (which would auto-detect a
	// real container manager) with a stub that just installs the spy
	// onto the context.
	cmd.Before = func(ctx context.Context, _ *cli.Command) (context.Context, error) {
		return context.WithValue(ctx, containerManagerKey, spy), nil
	}

	root := &cli.Command{Commands: []*cli.Command{cmd}}
	// Mirror the real entrypoint, which runs argv through PrepareArgs.
	full := PrepareArgs(root, append([]string{"distrobox"}, argv...))

	require.NoError(t, root.Run(context.Background(), full))
	require.NotEmpty(t, spy.calls, "expected Enter to be called for argv %v", argv)

	return spy.calls[0]
}

// TestEphemeralCommand_CustomCommandVariants locks down the three
// equivalent ways of supplying a custom command to `distrobox ephemeral`:
// `-e`, `--exec`, and the bare `--` separator. All three must produce
// the same CustomCommand slice on the EnterOptions that ephemeral passes
// to the enter step, matching the original bash distrobox-ephemeral.
func TestEphemeralCommand_CustomCommandVariants(t *testing.T) {
	cases := []struct {
		name string
		argv []string
		want []string
	}{
		{
			name: "short -e flag",
			argv: []string{"ephemeral", "--image", "alpine", "-e", "echo", "ciao"},
			want: []string{"echo", "ciao"},
		},
		{
			name: "long --exec flag",
			argv: []string{"ephemeral", "--image", "alpine", "--exec", "echo", "ciao"},
			want: []string{"echo", "ciao"},
		},
		{
			name: "bare -- separator",
			argv: []string{"ephemeral", "--image", "alpine", "--", "echo", "ciao"},
			want: []string{"echo", "ciao"},
		},
		{
			name: "implicit (no -e/--exec/--)",
			argv: []string{"ephemeral", "--image", "alpine", "echo", "ciao"},
			want: []string{"echo", "ciao"},
		},
		{
			name: "explicit --name with -e",
			argv: []string{"ephemeral", "--name", "foo", "--image", "alpine", "-e", "echo", "ciao"},
			want: []string{"echo", "ciao"},
		},
		{
			// Regression: the custom command itself contains a short
			// flag (e.g. `bash -c echo`). The CLI must not eat `-c` as
			// the inherited `--clone` flag from `distrobox-create`.
			name: "custom command with short flag after -e",
			argv: []string{"ephemeral", "--image", "alpine", "-e", "bash", "-c", "echo"},
			want: []string{"bash", "-c", "echo"},
		},
		{
			name: "custom command with short flag after --",
			argv: []string{"ephemeral", "--image", "alpine", "--", "bash", "-c", "echo"},
			want: []string{"bash", "-c", "echo"},
		},
		{
			name: "-e before the image flag",
			argv: []string{"ephemeral", "-e", "echo", "ciao"},
			want: []string{"echo", "ciao"},
		},
		{
			name: "short alias for image with -- separator",
			argv: []string{"ephemeral", "-i", "alpine", "--", "bash", "-c", "echo"},
			want: []string{"bash", "-c", "echo"},
		},
		{
			name: "no custom command (only flags)",
			argv: []string{"ephemeral", "--image", "alpine"},
			want: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := runEphemeral(t, tc.argv...)

			if tc.want == nil {
				assert.Empty(t, opts.CustomCommand)
			} else {
				assert.Equal(t, tc.want, opts.CustomCommand)
			}
		})
	}
}

// TestEphemeralCommand_ContainerNameFromFlag ensures that an explicit
// --name is propagated through the create+enter pair. This is the
// ephemeral-specific path (the normal flow auto-generates a name), so
// it's worth pinning down separately.
func TestEphemeralCommand_ContainerNameFromFlag(t *testing.T) {
	opts := runEphemeral(t, "ephemeral", "--image", "alpine", "--name", "my-ephemeral", "-e", "echo", "ciao")

	assert.Equal(t, "my-ephemeral", opts.ContainerName)
	assert.Equal(t, []string{"echo", "ciao"}, opts.CustomCommand)
}
