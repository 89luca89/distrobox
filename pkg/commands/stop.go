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

type StopCommand struct {
	cfg              *config.Values
	containerManager containermanager.ContainerManager
	listCmd          *ListCommand
	prompter         *ui.Prompter
}

type StopOptions struct {
	ContainerNames []string
	NonInteractive bool
	All            bool
}

var ErrEmptyContainerList = errors.New("cannot find containers to stop")
var ErrStopAbortedByUserError = errors.New("stop operation aborted by user")

func NewStopCommand(cfg *config.Values, containerManager containermanager.ContainerManager, prompter *ui.Prompter) *StopCommand {
	return &StopCommand{
		cfg:              cfg,
		containerManager: containerManager,
		listCmd:          NewListCommand(cfg, containerManager),
		prompter:         prompter,
	}
}

func (c *StopCommand) Execute(ctx context.Context, opts *StopOptions) error {
	var containerNames []string
	switch {
	case opts.All:
		containers, err := c.listCmd.Execute(ctx)
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}
		if len(containers.Containers) == 0 {
			return ErrEmptyContainerList
		}
		containerNames = make([]string, 0, len(containers.Containers))
		for _, container := range containers.Containers {
			containerNames = append(containerNames, container.Name)
		}
	case len(opts.ContainerNames) > 0:
		containerNames = opts.ContainerNames
	default:
		containerNames = []string{c.cfg.DefaultContainerName}
	}

	proceed := opts.NonInteractive || c.prompter.Prompt(
		fmt.Sprintf("Do you really want to stop %s?", containerNames),
		true,
	)

	if !proceed {
		return ErrStopAbortedByUserError
	}

	err := c.containerManager.Stop(ctx, containerNames)
	if err != nil {
		return fmt.Errorf("failed to stop containers: %w", err)
	}

	return nil
}
