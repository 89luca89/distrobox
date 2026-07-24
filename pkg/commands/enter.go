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

	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

type EnterResult struct{}

type EnterOptions struct {
	ContainerName   string
	AdditionalFlags string
	CustomCommand   []string
	DryRun          bool
	NoTTY           bool
	Verbose         bool
	CleanPath       bool
	NoWorkDir       bool
}

type EnterCommand struct {
	cfg              *config.Values
	containerManager containermanager.ContainerManager
	progress         *ui.Progress
	printer          *ui.Printer
}

func NewEnterCommand(
	cfg *config.Values,
	cm containermanager.ContainerManager,
	progress *ui.Progress,
	printer *ui.Printer,
) *EnterCommand {
	return &EnterCommand{
		cfg:              cfg,
		containerManager: cm,
		progress:         progress,
		printer:          printer,
	}
}

func (c *EnterCommand) Execute(ctx context.Context, opts EnterOptions) (*EnterResult, error) {
	cmdOpts := containermanager.EnterOptions{
		ContainerName:   opts.ContainerName,
		AdditionalFlags: opts.AdditionalFlags,
		CustomCommand:   opts.CustomCommand,
		DryRun:          opts.DryRun,
		NoTTY:           opts.NoTTY,
		Verbose:         opts.Verbose,
		CleanPath:       opts.CleanPath,
		NoWorkDir:       opts.NoWorkDir,
	}

	err := c.containerManager.Enter(ctx, cmdOpts, c.progress, c.printer)
	if err != nil {
		return nil, fmt.Errorf("failed to enter the container: %w", err)
	}

	return &EnterResult{}, nil
}
