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
