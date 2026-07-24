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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/89luca89/distrobox/pkg/commands"
	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/internal/testutil"
)

// ListCommand must drop containers that are not distrobox-owned, even when a
// foreign tool's label value happens to contain the substring "distrobox"
// (the regression that motivated tightening Container.IsDistrobox).
func TestListCommand_FiltersNonDistroboxContainers(t *testing.T) {
	mock := &testutil.MockContainerManager{
		ListContainersResult: []containermanager.Container{
			{
				Name:   "real-box",
				Image:  "registry.fedoraproject.org/fedora-toolbox:latest",
				Status: "Up",
				Labels: map[string]string{"manager": "distrobox", "distrobox.unshare_groups": "0"},
			},
			{
				Name:   "apx-box",
				Image:  "docker.io/library/ubuntu:22.04",
				Status: "Exited",
				Labels: map[string]string{"manager": "apx", "distrobox.unshare_groups": "0"},
			},
			{
				Name:   "plain-box",
				Image:  "docker.io/library/alpine:latest",
				Status: "Up",
				Labels: map[string]string{"foo": "bar"},
			},
			{
				Name:   "dir-box",
				Image:  "docker.io/library/alpine:latest",
				Status: "Up",
				Labels: map[string]string{
					"example.dir": "/home/luca/distrobox",
				},
			},
		},
	}

	cmd := commands.NewListCommand(&config.Values{}, mock)
	result, err := cmd.Execute(context.Background())
	require.NoError(t, err)

	names := make([]string, 0, len(result.Containers))
	for _, c := range result.Containers {
		names = append(names, c.Name)
	}
	assert.Equal(t, []string{"apx-box", "real-box"}, names,
		"plain-box (no distrobox labels) and dir-box (distrobox only inside a label value) must be filtered out; result must be sorted by name")
}

// Empty container list propagates through the filter without surprise.
func TestListCommand_EmptyResult(t *testing.T) {
	mock := &testutil.MockContainerManager{
		ListContainersResult: []containermanager.Container{},
	}

	cmd := commands.NewListCommand(&config.Values{}, mock)
	result, err := cmd.Execute(context.Background())
	require.NoError(t, err)
	assert.Empty(t, result.Containers)
}
