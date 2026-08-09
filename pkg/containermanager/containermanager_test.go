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

package containermanager_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/89luca89/distrobox/pkg/containermanager"
)

func TestContainer_IsDistrobox_StandardManagerLabel(t *testing.T) {
	c := containermanager.Container{
		Labels: map[string]string{"manager": "distrobox", "distrobox.unshare_groups": "0"},
	}
	assert.True(t, c.IsDistrobox())
}

// Regression: when the user overrides the manager label via
// `--additional-flags --label=manager=apx`, the container is still a
// distrobox container — the `distrobox.unshare_groups` label is always set on
// creation and is enough to identify it.
func TestContainer_IsDistrobox_ManagerLabelOverridden(t *testing.T) {
	c := containermanager.Container{
		Labels: map[string]string{"manager": "apx", "distrobox.unshare_groups": "0"},
	}
	assert.True(t, c.IsDistrobox())
}

func TestContainer_IsDistrobox_NoDistroboxLabels(t *testing.T) {
	c := containermanager.Container{
		Labels: map[string]string{"manager": "toolbox"},
	}
	assert.False(t, c.IsDistrobox())
}

func TestContainer_IsDistrobox_NilLabels(t *testing.T) {
	c := containermanager.Container{Labels: nil}
	assert.False(t, c.IsDistrobox())
}

// Regression: a label *value* that happens to contain the substring "distrobox"
// (e.g. another tool tagging the container with a workdir under a directory
// named "distrobox") must NOT make us claim the container as ours. Only label
// keys with the distrobox. prefix, or manager=distrobox, count.
func TestContainer_IsDistrobox_LabelValueSubstringIgnored(t *testing.T) {
	c := containermanager.Container{
		Labels: map[string]string{
			"example.dir": "/home/luca/distrobox",
		},
	}
	assert.False(t, c.IsDistrobox())
}

// A foreign manager label combined with a distrobox-suffixed value must not
// trigger detection on the value side either. The container has to actually
// carry a distrobox.* key (or manager=distrobox) to be ours.
func TestContainer_IsDistrobox_ForeignManagerWithDistroboxValue(t *testing.T) {
	c := containermanager.Container{
		Labels: map[string]string{
			"manager":           "compose",
			"com.example.image": "registry.opensuse.org/opensuse/distrobox",
		},
	}
	assert.False(t, c.IsDistrobox())
}

// Keys that merely contain the substring "distrobox" but don't use the
// reserved distrobox. namespace (e.g. another project's labels) must not
// be treated as ours. Matches the docs at pkg/containermanager/providers
// where create always sets distrobox.<something>=… keys.
func TestContainer_IsDistrobox_UnrelatedKeyContainingSubstring(t *testing.T) {
	c := containermanager.Container{
		Labels: map[string]string{
			"my-distrobox-thing": "1",
		},
	}
	assert.False(t, c.IsDistrobox())
}

// The manager-label override case (24b31ed8): the user supplied
// --additional-flags --label=manager=apx, so manager!=distrobox, but the
// distrobox.unshare_groups label is still set by the create path and is
// enough to identify the container.
func TestContainer_IsDistrobox_DistroboxKeyPrefix(t *testing.T) {
	c := containermanager.Container{
		Labels: map[string]string{
			"distrobox.unshare_groups": "0",
		},
	}
	assert.True(t, c.IsDistrobox())
}

// TestIsTTY_DevNullNotATerminal guards against the char-device heuristic:
// /dev/null is a character device but NOT a terminal, and treating it as one
// made headless invocations (`enter ... > /dev/null 2>&1`) allocate a --tty
// that docker exec rejects with "cannot attach stdin to a TTY-enabled
// container because stdin is not a terminal".
func TestIsTTY_DevNullNotATerminal(t *testing.T) {
	// Swap both stdio fds to /dev/null for the duration of the test.
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("open %s: %v", os.DevNull, err)
	}
	defer devNull.Close()

	origStdin := os.Stdin
	origStdout := os.Stdout
	defer func() {
		os.Stdin = origStdin
		os.Stdout = origStdout
	}()

	os.Stdin = devNull
	os.Stdout = devNull

	assert.False(t, containermanager.IsTTY(),
		"stdin/stdout pointing at /dev/null must not be considered a tty")
}
