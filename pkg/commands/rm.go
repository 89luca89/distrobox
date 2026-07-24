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
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/89luca89/distrobox/internal/userenv"
	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

type RmResult struct {
	Containers []containermanager.Container
}

// ErrRmAbortedByUser is returned when the user declines the top-level deletion
// confirmation, so the CLI can print "Aborted." and exit cleanly.
var ErrRmAbortedByUser = errors.New("rm operation aborted by user")

type RmCommand struct {
	cfg              *config.Values
	containerManager containermanager.ContainerManager
	listCmd          *ListCommand
	generateEntryCmd *GenerateEntryCommand
	prompter         *ui.Prompter
	printer          *ui.Printer
}

type RmOptions struct {
	NoTTY          bool
	Force          bool
	All            bool
	RemoveHome     bool
	Verbose        bool
	ContainerNames []string
}

func NewRmCommand(
	cfg *config.Values,
	cm containermanager.ContainerManager,
	prompter *ui.Prompter,
	printer *ui.Printer,
) *RmCommand {
	listCmd := NewListCommand(cfg, cm)
	generateEntryCmd := NewGenerateEntryCommand(cfg, listCmd)
	return &RmCommand{
		cfg:              cfg,
		containerManager: cm,
		listCmd:          listCmd,
		generateEntryCmd: generateEntryCmd,
		prompter:         prompter,
		printer:          printer,
	}
}

func (c *RmCommand) Execute(ctx context.Context, options RmOptions) (*RmResult, error) {
	if !options.NoTTY && c.prompter == nil {
		return nil, errors.New("prompter is required for interactive mode")
	}

	listResult, err := c.listCmd.Execute(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed while listing containers: %w", err)
	}

	distroboxesToRemove := getContainersToRemove(listResult.Containers, options.ContainerNames, options.All)

	// Mirror shell distrobox-rm:353-356: surface typos in explicit names
	// instead of silently no-op'ing.
	if !options.All && len(options.ContainerNames) > 0 {
		c.warnUnknownContainers(options.ContainerNames, distroboxesToRemove)
	}

	// Single top-level confirmation, matching the shell (distrobox-rm:413-419):
	// default "yes", skipped under --force/--yes; a bare Enter proceeds.
	if !options.Force && !options.NoTTY && len(distroboxesToRemove) > 0 {
		names := make([]string, 0, len(distroboxesToRemove))
		for _, d := range distroboxesToRemove {
			names = append(names, d.Name)
		}
		if !c.prompter.Prompt(
			fmt.Sprintf("Do you really want to delete containers: %s?", strings.Join(names, " ")),
			true,
		) {
			return nil, ErrRmAbortedByUser
		}
	}

	userEnv := userenv.LoadUserEnvironment(ctx)
	userHome := userEnv.Home

	var removedDistroboxes []containermanager.Container
	for _, currentDistrobox := range distroboxesToRemove {
		err := c.removeContainer(ctx, currentDistrobox, options.Force, options.NoTTY, options.RemoveHome, options.Verbose, userHome)
		if err != nil {
			c.printer.PrintErrorln("error deleting %s: %s", currentDistrobox.Name, err)
		}
		removedDistroboxes = append(removedDistroboxes, currentDistrobox)
	}

	return &RmResult{Containers: removedDistroboxes}, nil
}

// warnUnknownContainers prints "Cannot find container X." for each explicitly
// requested name that didn't resolve to a distrobox, mirroring the shell
// (distrobox-rm:353-356) so typos surface instead of silently no-op'ing.
func (c *RmCommand) warnUnknownContainers(requested []string, resolved []containermanager.Container) {
	found := make(map[string]bool, len(resolved))
	for _, d := range resolved {
		found[d.Name] = true
	}
	for _, name := range requested {
		if !found[name] {
			c.printer.PrintWarningln("Cannot find container %s.", name)
		}
	}
}

func (c *RmCommand) removeContainer(
	ctx context.Context,
	container containermanager.Container,
	force bool,
	noTTY bool,
	removeHomeRequested bool,
	verbose bool,
	userHome string,
) error {
	forceRemove := force
	if !forceRemove && !noTTY && container.IsRunning() {
		// Shell defaults this prompt to "yes" (distrobox-rm:424-426).
		if c.prompter.Prompt("Container is running, do you want to force delete it?", true) {
			forceRemove = true
		} else {
			return nil
		}
	}

	inspectOutput, err := c.containerManager.InspectContainer(ctx, container.Name)
	if err != nil {
		return fmt.Errorf("error inspecting the container: %w", err)
	}

	removeHome := false
	if removeHomeRequested && !noTTY && inspectOutput.ContainerHome != userHome {
		question := fmt.Sprintf(
			"Do you really want to remove custom home of container %s (%s)?",
			container.Name,
			inspectOutput.ContainerHome,
		)
		removeHome = c.prompter.Prompt(question, false)
	}

	cmOptions := containermanager.RmOptions{
		Force:         forceRemove,
		RemoveHome:    removeHome,
		ContainerHome: inspectOutput.ContainerHome,
	}
	err = c.containerManager.Remove(ctx, container.Name, cmOptions)
	if err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	c.cleanup(ctx, userHome, container.Name, verbose)

	return nil
}

func (c *RmCommand) cleanup(ctx context.Context, userHome, containerName string, verbose bool) {
	bins := findExportedBinaries(userHome, containerName)
	desktopApps := c.findExportedDesktopApps(userHome, containerName)

	toDelete := slices.Concat(bins, desktopApps)

	for _, path := range toDelete {
		if err := os.Remove(path); err != nil {
			c.printer.PrintWarningln("warning: failed to remove file '%s': %s", path, err)
		}
	}

	err := c.generateEntryCmd.Execute(
		ctx,
		&GenerateEntryOptions{
			ContainerName: containerName,
			Delete:        true,
			Verbose:       verbose,
		},
	)
	if err != nil {
		c.printer.PrintWarningln("warning: failed to remove desktop entry for container '%s': %s", containerName, err)
	}
}

func getContainersToRemove(
	containers []containermanager.Container,
	names []string,
	all bool,
) []containermanager.Container {
	if all {
		return containers
	}

	var filtered []containermanager.Container
	for _, container := range containers {
		if slices.ContainsFunc(names, func(name string) bool {
			return container.Name == name
		}) {
			filtered = append(filtered, container)
		}
	}

	return filtered
}

func findExportedBinaries(userHome, containerName string) []string {
	binDir := filepath.Join(userHome, ".local", "bin")

	entries, err := os.ReadDir(binDir)
	if err != nil {
		return nil
	}

	var files []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := filepath.Join(binDir, entry.Name())

		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		content := string(data)
		if strings.Contains(content, "# distrobox_binary") &&
			strings.Contains(content, "# name: "+containerName+"\n") {
			absPath, err := filepath.Abs(path)
			if err != nil {
				continue
			}

			files = append(files, absPath)
		}
	}

	return files
}

func (c *RmCommand) findExportedDesktopApps(userHome, containerName string) []string {
	appsPattern := filepath.Join(userHome, ".local", "share", "applications", containerName+"*")

	matches, err := filepath.Glob(appsPattern)
	if err != nil {
		c.printer.PrintWarningln("warning: failed to glob desktop apps: %s", err)
		return []string{}
	}

	var files []string

	for _, desktopFile := range matches {
		iconValue, ok := parseDesktopExport(desktopFile, containerName)
		if !ok {
			continue
		}

		absDesktop, err := filepath.Abs(desktopFile)
		if err != nil {
			continue
		}

		files = append(files, absDesktop)

		if iconValue != "" {
			files = append(files, findIconFiles(userHome, iconValue)...)
		}
	}

	return files
}

func parseDesktopExport(desktopFile, containerName string) (string, bool) {
	data, err := os.ReadFile(desktopFile)
	if err != nil {
		return "", false
	}

	hasExecMatch := false
	var iconValue string

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "Exec=") && strings.Contains(line, containerName+" ") {
			hasExecMatch = true
		}

		if strings.HasPrefix(line, "Icon=") {
			iconValue = strings.TrimPrefix(line, "Icon=")
		}
	}

	return iconValue, hasExecMatch
}

func findIconFiles(userHome, iconName string) []string {
	iconsDir := filepath.Join(userHome, ".local", "share", "icons")
	iconPrefix := iconName + "."

	var files []string

	_ = filepath.WalkDir(iconsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // skip unreadable directories
		}

		if d.IsDir() {
			return nil
		}

		if strings.HasPrefix(d.Name(), iconPrefix) {
			absIcon, err := filepath.Abs(path)
			if err == nil {
				files = append(files, absIcon)
			}
		}

		return nil
	})

	return files
}
