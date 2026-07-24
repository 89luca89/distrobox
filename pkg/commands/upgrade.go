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
	"errors"
	"fmt"

	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

//nolint:lll // upgrade command mirrors the shell version
const upgradeScript = `command -v su-exec 2>/dev/null && su-exec root /usr/bin/entrypoint --upgrade || command -v doas 2>/dev/null && doas /usr/bin/entrypoint --upgrade || sudo -S /usr/bin/entrypoint --upgrade`

type UpgradeOptions struct {
	ContainerNames []string
	All            bool
	Running        bool
}

type UpgradeCommand struct {
	cfg              *config.Values
	containerManager containermanager.ContainerManager
	listCmd          *ListCommand
	enterCmd         *EnterCommand
	printer          *ui.Printer
}

var ErrUpgradeNoContainerSpecified = errors.New("please specify the name of the container")

func NewUpgradeCommand(
	cfg *config.Values,
	cm containermanager.ContainerManager,
	progress *ui.Progress,
	printer *ui.Printer,
) *UpgradeCommand {
	return &UpgradeCommand{
		cfg:              cfg,
		containerManager: cm,
		listCmd:          NewListCommand(cfg, cm),
		enterCmd:         NewEnterCommand(cfg, cm, progress, printer),
		printer:          printer,
	}
}

func (c *UpgradeCommand) Execute(ctx context.Context, opts *UpgradeOptions) error {
	var containerNames []string

	switch {
	case opts.All, opts.Running:
		containers, err := c.listCmd.Execute(ctx)
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}

		if len(containers.Containers) == 0 {
			return ErrEmptyContainerList
		}

		containerNames = make([]string, 0, len(containers.Containers))
		for _, container := range containers.Containers {
			if opts.Running && !container.IsRunning() {
				continue
			}

			containerNames = append(containerNames, container.Name)
		}

		if len(containerNames) == 0 {
			return ErrEmptyContainerList
		}
	case len(opts.ContainerNames) > 0:
		containerNames = opts.ContainerNames
	default:
		return ErrUpgradeNoContainerSpecified
	}

	// The reference shell upgrades immediately with no confirmation
	// (distrobox-upgrade:266-272), so scripted/non-interactive upgrades never
	// block. We match that: no prompt.
	var lastErr error

	for _, name := range containerNames {
		// Per-container banner, matching the shell (distrobox-upgrade:267).
		c.printer.Println("Upgrading %s...", name)
		if err := c.upgradeContainer(ctx, name); err != nil {
			c.printer.PrintErrorln("error upgrading %s: %s", name, err)

			lastErr = err

			continue
		}
	}

	return lastErr
}

func (c *UpgradeCommand) upgradeContainer(ctx context.Context, name string) error {
	enterOpts := EnterOptions{
		ContainerName: name,
		CustomCommand: []string{"sh", "-c", upgradeScript},
	}

	if _, err := c.enterCmd.Execute(ctx, enterOpts); err != nil {
		return fmt.Errorf("failed to upgrade container %s: %w", name, err)
	}

	return nil
}
