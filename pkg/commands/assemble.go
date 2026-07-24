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
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/manifest"
	"github.com/89luca89/distrobox/pkg/ui"
)

const assembleCleanupTimeout = 30 * time.Second

type AssembleOptions struct {
	Items []manifest.Item
	// Boxname is the name of the box to assemble
	// If specified, the Assemble command will only assemble the given box
	// If empty, the command will assemble all boxes defined in the manifest
	Boxname string
	// Delete indicates whether to delete the existing box before assembling
	// true=delete, false=create or update
	Delete  bool
	Replace bool
	Verbose bool
	DryRun  bool
}

type AssembleCommand struct {
	cfg           *config.Values
	createCmd     *CreateCommand
	rmCmd         *RmCommand
	enterCmd      *EnterCommand
	createCmdRoot *CreateCommand
	rmCmdRoot     *RmCommand
	enterCmdRoot  *EnterCommand
	progress      *ui.Progress
	printer       *ui.Printer
}

func NewAssembleCommand(
	cfg *config.Values,
	cm containermanager.ContainerManager,
	prompter *ui.Prompter,
	progress *ui.Progress,
	printer *ui.Printer,
) *AssembleCommand {
	cmRoot := cm.CloneAsRoot()
	return &AssembleCommand{
		cfg:           cfg,
		createCmd:     NewCreateCommand(cfg, cm, ui.NewDevNullProgress(), prompter),
		rmCmd:         NewRmCommand(cfg, cm, prompter, printer),
		enterCmd:      NewEnterCommand(cfg, cm, progress, printer),
		createCmdRoot: NewCreateCommand(cfg, cmRoot, ui.NewDevNullProgress(), prompter),
		rmCmdRoot:     NewRmCommand(cfg, cmRoot, prompter, printer),
		enterCmdRoot:  NewEnterCommand(cfg, cmRoot, progress, printer),
		progress:      progress,
		printer:       printer,
	}
}

func (ac *AssembleCommand) Execute(ctx context.Context, opts AssembleOptions) error {
	var items []manifest.Item
	if opts.Boxname != "" {
		idx := slices.IndexFunc(opts.Items, func(i manifest.Item) bool {
			return i.Name == opts.Boxname
		})
		if idx == -1 {
			return fmt.Errorf("box '%s' not found in manifest", opts.Boxname)
		}
		items = []manifest.Item{opts.Items[idx]}
	} else {
		items = opts.Items
	}

	for _, item := range items {
		switch {
		case opts.Delete:
			if err := ac.deleteItem(ctx, item, opts.DryRun); err != nil {
				return fmt.Errorf("failed to delete item '%s': %w", item.Name, err)
			}
		case opts.Replace:
			if err := ac.replaceItem(ctx, item, opts.DryRun); err != nil {
				return fmt.Errorf("failed to replace item '%s': %w", item.Name, err)
			}
		default:
			if err := ac.createItem(ctx, item, opts.DryRun); err != nil {
				return fmt.Errorf("failed to create item '%s': %w", item.Name, err)
			}
		}
	}

	return nil
}

func (ac *AssembleCommand) deleteItem(ctx context.Context, item manifest.Item, dryRun bool) error {
	ac.progress.Next("Deleting %s...", item.Name)
	// Shell skips the rm step entirely in dry-run (distrobox-assemble:316-319):
	// `assemble rm --dry-run` and `assemble create --replace --dry-run` must not
	// actually delete the container.
	if dryRun {
		ac.progress.Done()
		return nil
	}
	opts := RmOptions{
		NoTTY:          true, // assemble is non-interactive
		Force:          true,
		All:            false,
		RemoveHome:     false,
		ContainerNames: []string{item.Name},
	}

	rmCmd := ac.rmCmd
	if item.Root {
		rmCmd = ac.rmCmdRoot
	}

	_, err := rmCmd.Execute(ctx, opts)
	if err != nil {
		ac.progress.Fail()
		return fmt.Errorf("failed to execute delete item '%s': %w", item.Name, err)
	}
	ac.progress.Done()
	return nil
}

func (ac *AssembleCommand) replaceItem(ctx context.Context, item manifest.Item, dryRun bool) error {
	err := ac.deleteItem(ctx, item, dryRun)
	if err != nil {
		return err
	}

	return ac.createItem(ctx, item, dryRun)
}

func (ac *AssembleCommand) createItem(ctx context.Context, item manifest.Item, dryRun bool) error {
	ac.progress.Next("Creating %s...", item.Name)
	opts := CreateOptions{
		ContainerClone:          item.Clone,
		ContainerName:           item.Name,
		ContainerImage:          item.Image,
		ContainerHostname:       item.Hostname,
		UnshareNetNs:            item.UnshareNetns || item.UnshareAll,
		UnshareDevsys:           item.UnshareDevsys || item.UnshareAll,
		UnshareGroups:           item.UnshareGroups || item.UnshareAll || item.Init,
		UnshareIpc:              item.UnshareIPC || item.UnshareAll,
		UnshareProcess:          item.UnshareProcess || item.UnshareAll || item.Init,
		AdditionalFlags:         item.AdditionalFlags,
		AdditionalVolumes:       item.Volumes,
		AdditionalPackages:      item.AdditionalPackages,
		ContainerPreInitHook:    ac.joinHooks(item.PreInitHooks),
		ContainerInitHook:       ac.joinHooks(item.InitHooks),
		ContainerUserCustomHome: item.Home,
		Init:                    item.Init,
		Nvidia:                  item.Nvidia,
		GenerateEntry:           item.Entry,
		Rootful:                 item.Root,
		DryRun:                  dryRun,
		NonInteractive:          true,
		ContainerAlwaysPull:     item.AlwaysPull,
	}

	createCmd := ac.createCmd
	if item.Root {
		createCmd = ac.createCmdRoot
	}
	_, err := createCmd.Execute(ctx, opts)
	if err != nil {
		var alreadyExists *ContainerAlreadyExistsError
		if errors.As(err, &alreadyExists) {
			ac.progress.Done()
			ac.printer.Println("%s already exists", item.Name)
			return nil
		}
		ac.progress.Fail()
		return err
	}

	success := false
	defer func() {
		if success || dryRun {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), assembleCleanupTimeout)
		defer cancel()
		rmCmd := ac.rmCmd
		if item.Root {
			rmCmd = ac.rmCmdRoot
		}
		if _, rmErr := rmCmd.Execute(cleanupCtx, RmOptions{
			NoTTY:          true,
			Force:          true,
			ContainerNames: []string{item.Name},
		}); rmErr != nil {
			ac.printer.PrintWarningln("warning: %s: %s", item.Name, rmErr)
		}
	}()

	if !dryRun {
		err = ac.setupBox(ctx, item)
		if err != nil {
			ac.progress.Fail()
			return err
		}
	}

	ac.progress.Done()
	success = true
	return nil
}

func (ac *AssembleCommand) joinHooks(hooks []string) string {
	// A hook that already ends in its own terminator (`;` or `&&`) keeps
	// it; otherwise insert ` && ` so consecutive hooks don't run together.
	selfTerminated := regexp.MustCompile(`(;|&&)[[:space:]]?$`)

	sb := strings.Builder{}
	for i, hook := range hooks {
		sb.WriteString(hook)
		if i == len(hooks)-1 {
			continue
		}
		if selfTerminated.MatchString(hook) {
			sb.WriteString(" ")
		} else {
			sb.WriteString(" && ")
		}
	}
	return sb.String()
}

func (ac *AssembleCommand) setupBox(ctx context.Context, item manifest.Item) error {
	enterCmd := ac.enterCmd
	if item.Root {
		enterCmd = ac.enterCmdRoot
	}
	if item.StartNow {
		_, err := enterCmd.Execute(ctx, EnterOptions{
			ContainerName: item.Name,
			NoTTY:         true,
			CustomCommand: []string{"true"}, // we just want to run the init hooks, so we can skip the shell
			DryRun:        false,
		})
		if err != nil {
			return fmt.Errorf("failed to execute init hooks for item '%s': %w", item.Name, err)
		}
	}

	// validate app name to prevent command injection, since it's used in a custom command
	var validAppName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._+\-]*$`)
	for _, app := range item.ExportedApps {
		if !validAppName.MatchString(app) {
			return fmt.Errorf("invalid app name '%s' for item '%s': must be alphanumeric (with dots, underscores, hyphens)", app, item.Name)
		}
	}
	for _, app := range item.ExportedApps {
		_, err := enterCmd.Execute(ctx, EnterOptions{
			ContainerName: item.Name,
			NoTTY:         true,
			CustomCommand: []string{"distrobox-export", "--app", app},
			DryRun:        false,
		})
		if err != nil {
			return fmt.Errorf("failed to export app '%s' for item '%s': %w", app, item.Name, err)
		}
	}

	// validate bin path to prevent command injection, since it's used in a custom command
	var validBinPath = regexp.MustCompile(`^/[a-zA-Z0-9._+\-/]+$`)
	if len(item.ExportedBins) > 0 && !validBinPath.MatchString(item.ExportedBinsPath) {
		return fmt.Errorf("invalid exported bins path '%s' for item '%s': must be an absolute path with alphanumeric characters, dots, underscores, or hyphens", item.ExportedBinsPath, item.Name)
	}
	// we allow slashes in bin paths, but we validate each path segment to prevent command injection
	for _, bin := range item.ExportedBins {
		if !validBinPath.MatchString(bin) {
			return fmt.Errorf("invalid bin path '%s' for item '%s': must be an absolute path with alphanumeric characters, dots, underscores, or hyphens", bin, item.Name)
		}
	}
	for _, bin := range item.ExportedBins {
		_, err := enterCmd.Execute(ctx, EnterOptions{
			ContainerName: item.Name,
			NoTTY:         true,
			CustomCommand: []string{"distrobox-export", "--bin", bin, "--export-path", item.ExportedBinsPath},
			DryRun:        false,
		})
		if err != nil {
			return fmt.Errorf("failed to export bin '%s' for item '%s': %w", bin, item.Name, err)
		}
	}

	return nil
}
