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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParsePodmanContainerList_Fixtures asserts the podman `ps --format json`
// parser against a captured fixture, so a podman JSON change is caught here
// rather than in the field. Refresh testdata/podman-ps.json from a real
// `podman ps -a --no-trunc --format json` (scrubbed) when adding new versions.
func TestParsePodmanContainerList_Fixtures(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "podman-ps.json"))
	require.NoError(t, err)

	got, err := parsePodmanContainerList(string(raw))
	require.NoError(t, err)
	require.Len(t, got, 2)

	assert.Equal(t, "fedora-box", got[0].Name)
	assert.Equal(t, "abc123def456", got[0].ID) // truncated to 12 chars
	assert.Equal(t, "distrobox", got[0].Labels["manager"])
	assert.Equal(t, "0", got[0].Labels["distrobox.unshare_groups"])
	assert.Equal(t, "ubuntu-box", got[1].Name)
}

// TestParseContainerList_DockerFixtures asserts the docker `ps --format '{{json .}}'`
// (JSONL) parser, including the comma-joined Labels string -> map conversion.
func TestParseContainerList_DockerFixtures(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "docker-ps.json"))
	require.NoError(t, err)

	got, err := parseContainerList(string(raw))
	require.NoError(t, err)
	require.Len(t, got, 2)

	assert.Equal(t, "fedora-box", got[0].Name)
	assert.Equal(t, "abc123def456", got[0].ID)
	assert.Equal(t, "distrobox", got[0].Labels["manager"])
	assert.Equal(t, "0", got[0].Labels["distrobox.unshare_groups"])
	assert.Equal(t, "ubuntu-box", got[1].Name)
}

// TestParseLabels pins the docker label-string parser, including the known
// limitation that a label value containing a comma is mis-split.
func TestParseLabels(t *testing.T) {
	got := parseLabels("manager=distrobox,distrobox.unshare_groups=0")
	assert.Equal(t, "distrobox", got["manager"])
	assert.Equal(t, "0", got["distrobox.unshare_groups"])

	assert.Empty(t, parseLabels(""))

	got = parseLabels("k=a,b") // value with a comma -> the ",b" tail is dropped
	assert.Equal(t, "a", got["k"])
	_, hasB := got["b"]
	assert.False(t, hasB)
}
