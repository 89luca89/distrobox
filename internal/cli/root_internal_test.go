package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "empty",
			in:   nil,
			want: nil,
		},
		{
			name: "plain distrobox is untouched",
			in:   []string{"distrobox"},
			want: []string{"distrobox"},
		},
		{
			name: "plain distrobox with subcommand is untouched",
			in:   []string{"distrobox", "enter", "foo"},
			want: []string{"distrobox", "enter", "foo"},
		},
		{
			name: "distrobox-enter dispatches to enter",
			in:   []string{"distrobox-enter", "foo"},
			want: []string{"distrobox", "enter", "foo"},
		},
		{
			name: "absolute path is stripped to basename for dispatch",
			in:   []string{"/usr/local/bin/distrobox-enter", "--name", "foo"},
			want: []string{"distrobox", "enter", "--name", "foo"},
		},
		{
			name: "two-word subcommand survives intact",
			in:   []string{"distrobox-generate-entry", "foo"},
			want: []string{"distrobox", "generate-entry", "foo"},
		},
		{
			name: "distrobox- with empty suffix is left alone",
			in:   []string{"distrobox-"},
			want: []string{"distrobox-"},
		},
		{
			name: "unrelated names are left alone",
			in:   []string{"something-else", "arg"},
			want: []string{"something-else", "arg"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, ResolveArgs(tc.in))
		})
	}
}
