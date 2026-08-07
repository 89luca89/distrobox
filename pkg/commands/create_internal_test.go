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

package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/89luca89/distrobox/pkg/config"
)

// TestCreateCommand_makeContainerName_Slug pins the image->name slug logic with
// the exact examples from create.go's doc comment.
//
// It also documents a doc/code mismatch: the comment claims
// "ubuntu:20.04 -> ubuntu-20.04", but the code also does ReplaceAll(".", "-"),
// so the real output is "ubuntu-20-04". This asserts the ACTUAL behavior; see
// plans/tests/tests-to-adopt.md §2 — reconcile the doc against the code (or vice
// versa) and update this case with the decision.
func TestCreateCommand_makeContainerName_Slug(t *testing.T) {
	c := &CreateCommand{cfg: &config.Values{
		DefaultContainerImage: "registry.fedoraproject.org/fedora-toolbox:latest",
		DefaultContainerName:  "my-distrobox",
	}}

	for _, tc := range []struct {
		image string
		want  string
	}{
		{"alpine", "alpine"},
		{"registry.fedoraproject.org/fedora-toolbox:39", "fedora-toolbox-39"},
		{"ghcr.io/void-linux/void-linux:latest-full-x86_64", "void-linux-latest-full-x86_64"},
		{"ubuntu:20.04", "ubuntu-20-04"}, // actual behavior; doc says "ubuntu-20.04"
	} {
		assert.Equal(t, tc.want, c.makeContainerName(&CreateOptions{}, tc.image), tc.image)
	}
}
