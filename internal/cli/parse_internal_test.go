package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"

	"github.com/89luca89/distrobox/pkg/config"
)

// TestPrepareArgs exercises the argv rewrite against the real, fully composed
// root command (so the flag sets and arities match production). It asserts
// where a bare "--" is spliced in to isolate the custom command, and that the
// distrobox flags placed after the container name are left for urfave to
// parse rather than swallowed into the command.
func TestPrepareArgs(t *testing.T) {
	root := NewRootCommand(config.DefaultValues())

	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "flag after name is consumed, bare word opens command",
			in:   []string{"distrobox", "enter", "suse", "--additional-flags", "foo", "vim", "arg"},
			want: []string{"distrobox", "enter", "suse", "--additional-flags", "foo", "--", "vim", "arg"},
		},
		{
			name: "help after name drops the name so urfave shows command help",
			in:   []string{"distrobox", "enter", "suse", "--help"},
			want: []string{"distrobox", "enter", "--help"},
		},
		{
			name: "short -h after name behaves the same",
			in:   []string{"distrobox", "enter", "suse", "-h"},
			want: []string{"distrobox", "enter", "--help"},
		},
		{
			name: "implicit command",
			in:   []string{"distrobox", "enter", "suse", "echo", "ciao"},
			want: []string{"distrobox", "enter", "suse", "--", "echo", "ciao"},
		},
		{
			name: "-e after name becomes --",
			in:   []string{"distrobox", "enter", "suse", "-e", "bash", "-c", "echo"},
			want: []string{"distrobox", "enter", "suse", "--", "bash", "-c", "echo"},
		},
		{
			name: "existing -- is left untouched",
			in:   []string{"distrobox", "enter", "suse", "--", "bash", "-c", "echo"},
			want: []string{"distrobox", "enter", "suse", "--", "bash", "-c", "echo"},
		},
		{
			name: "--name then command with short flag (regression)",
			in:   []string{"distrobox", "enter", "--name", "suse", "bash", "-c", "echo"},
			want: []string{"distrobox", "enter", "--name", "suse", "--", "bash", "-c", "echo"},
		},
		{
			// A marker before the name is a no-op: it is left in place (urfave
			// consumes -e/--exec as a bool flag) and the first bare token is
			// still the name.
			name: "-e before name is a no-op, first bare token is the name",
			in:   []string{"distrobox", "enter", "-e", "suse", "bash", "-c", "echo"},
			want: []string{"distrobox", "enter", "-e", "suse", "--", "bash", "-c", "echo"},
		},
		{
			name: "command word before its own flag",
			in:   []string{"distrobox", "enter", "suse", "vim", "--help"},
			want: []string{"distrobox", "enter", "suse", "--", "vim", "--help"},
		},
		{
			name: "unknown flag after name is left for urfave to reject",
			in:   []string{"distrobox", "enter", "suse", "--frobnicate"},
			want: []string{"distrobox", "enter", "suse", "--frobnicate"},
		},
		{
			name: "global flag before the subcommand",
			in:   []string{"distrobox", "--verbose", "enter", "suse", "bash"},
			want: []string{"distrobox", "--verbose", "enter", "suse", "--", "bash"},
		},
		{
			name: "inherited --verbose after the name is consumed",
			in:   []string{"distrobox", "enter", "suse", "--verbose", "bash"},
			want: []string{"distrobox", "enter", "suse", "--verbose", "--", "bash"},
		},
		{
			name: "shell completion flag is never touched",
			in:   []string{"distrobox", "enter", "suse", "--generate-shell-completion"},
			want: []string{"distrobox", "enter", "suse", "--generate-shell-completion"},
		},
		{
			name: "ephemeral implicit command (no positional name)",
			in:   []string{"distrobox", "ephemeral", "--image", "alpine", "cat", "/etc/os-release"},
			want: []string{"distrobox", "ephemeral", "--image", "alpine", "--", "cat", "/etc/os-release"},
		},
		{
			name: "ephemeral -e becomes --",
			in:   []string{"distrobox", "ephemeral", "-e", "bash", "-c", "echo"},
			want: []string{"distrobox", "ephemeral", "--", "bash", "-c", "echo"},
		},
		{
			name: "non-exec subcommand is left untouched",
			in:   []string{"distrobox", "list", "--verbose"},
			want: []string{"distrobox", "list", "--verbose"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, PrepareArgs(root, tc.in))
		})
	}
}

// TestPrepareArgs_EnvContainerName covers the DBX_CONTAINER_NAME path: when a
// name is already supplied via the --name flag's default, enter must not
// consume the first bare token as the name — it is the command.
func TestPrepareArgs_EnvContainerName(t *testing.T) {
	enter := &cli.Command{
		Name: "enter",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Value: "envbox"},
			&cli.BoolFlag{Name: "clean-path", Aliases: []string{"c"}},
			&cli.BoolFlag{Name: "exec", Aliases: []string{"e"}},
		},
	}
	root := &cli.Command{Name: "distrobox", Commands: []*cli.Command{enter}}

	got := PrepareArgs(root, []string{"distrobox", "enter", "bash", "-c", "echo"})
	assert.Equal(t, []string{"distrobox", "enter", "--", "bash", "-c", "echo"}, got)
}
