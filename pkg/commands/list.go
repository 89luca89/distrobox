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
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager"
)

type ListResult struct {
	Containers []containermanager.Container
}

type ListCommand struct {
	cfg              *config.Values
	containerManager containermanager.ContainerManager
}

func NewListCommand(cfg *config.Values, cm containermanager.ContainerManager) *ListCommand {
	return &ListCommand{
		cfg:              cfg,
		containerManager: cm,
	}
}

func (c *ListCommand) Execute(ctx context.Context) (*ListResult, error) {
	containers, err := c.containerManager.ListContainers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed while listing containers: %w", err)
	}

	var distroboxes []containermanager.Container
	for _, container := range containers {
		if container.IsDistrobox() {
			distroboxes = append(distroboxes, container)
		}
	}

	// Sort by container name to keep `distrobox list` output stable for downstream UIs; see https://github.com/89luca89/distrobox/issues/2071.
	slices.SortFunc(distroboxes, func(a, b containermanager.Container) int {
		return strings.Compare(a.Name, b.Name)
	})

	return &ListResult{Containers: distroboxes}, nil
}
