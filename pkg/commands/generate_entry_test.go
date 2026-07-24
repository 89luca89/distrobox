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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/89luca89/distrobox/pkg/commands"
	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/containermanager/providers"
	"github.com/89luca89/distrobox/pkg/internal/testutil"
)

func TestGenerateEntryCommand_Execute(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	defer os.RemoveAll(tempDir)

	// A mock manager that lists the target box, so single-mode resolveTargets
	// finds it: generate-entry refuses to create an entry for a missing box.
	containerManager := &testutil.MockContainerManager{
		ListContainersResult: []containermanager.Container{{
			Name:   "test-container",
			Image:  "registry.fedoraproject.org/fedora-toolbox:latest",
			Labels: map[string]string{"manager": "distrobox"},
		}},
	}
	listCmd := commands.NewListCommand(&config.Values{}, containerManager)

	//
	// Generate the entry
	//

	generateEntryCmd := commands.NewGenerateEntryCommand(&config.Values{}, listCmd)

	opts := &commands.GenerateEntryOptions{
		ContainerName:       "test-container",
		Verbose:             true,
		Delete:              false,
		Icon:                "https://raw.githubusercontent.com/89luca89/distrobox/main/icons/terminal-distrobox-icon.svg",
		Root:                false,
		DesktopEntryBaseDir: fmt.Sprintf("%s/.local/share/", tempDir),
		DistroboxPath:       "/usr/bin/distrobox",
	}

	err := generateEntryCmd.Execute(ctx, opts)
	require.NoError(t, err, "GenerateEntryCommand.Execute()")

	expectedEntryPath := fmt.Sprintf("%s/.local/share/applications/test-container.desktop", tempDir)
	assert.FileExists(t, expectedEntryPath)

	expectedContent := `[Desktop Entry]
Name=Test-container
GenericName=Terminal entering Test-container
Comment=Terminal entering Test-container
Categories=Distrobox;System;Utility
Exec=/usr/bin/distrobox enter test-container
Icon=https://raw.githubusercontent.com/89luca89/distrobox/main/icons/terminal-distrobox-icon.svg
Keywords=distrobox;
NoDisplay=false
Terminal=true
TryExec=/usr/bin/distrobox
Type=Application
Actions=Remove;

[Desktop Action Remove]
Name=Remove Test-container from system
Exec=/usr/bin/distrobox rm test-container
`

	content, err := os.ReadFile(expectedEntryPath)
	require.NoError(t, err, "Failed to read desktop entry file")
	assert.Equal(t, expectedContent, string(content), "Desktop entry content mismatch")

	// Delete the entry
	opts.Delete = true
	err = generateEntryCmd.Execute(ctx, opts)
	require.NoError(t, err, "GenerateEntryCommand.Execute() on delete")

	assert.NoFileExists(t, expectedEntryPath)

	// Try deleting a non-existing entry
	err = generateEntryCmd.Execute(ctx, opts)
	assert.NoError(t, err, "GenerateEntryCommand.Execute() on delete non-existing")
}

func TestGenerateEntryCommand_Execute_Root(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	defer os.RemoveAll(tempDir)

	containerManager := &testutil.MockContainerManager{
		ListContainersResult: []containermanager.Container{{
			Name:   "test-container",
			Image:  "registry.fedoraproject.org/fedora-toolbox:latest",
			Labels: map[string]string{"manager": "distrobox"},
		}},
	}
	listCmd := commands.NewListCommand(&config.Values{}, containerManager)

	generateEntryCmd := commands.NewGenerateEntryCommand(&config.Values{}, listCmd)

	opts := &commands.GenerateEntryOptions{
		ContainerName:       "test-container",
		Verbose:             true,
		Delete:              false,
		Icon:                "https://raw.githubusercontent.com/89luca89/distrobox/main/icons/terminal-distrobox-icon.svg",
		Root:                true,
		DesktopEntryBaseDir: fmt.Sprintf("%s/.local/share/", tempDir),
		DistroboxPath:       "/usr/bin/distrobox",
	}

	err := generateEntryCmd.Execute(ctx, opts)
	require.NoError(t, err, "GenerateEntryCommand.Execute()")

	expectedEntryPath := fmt.Sprintf("%s/.local/share/applications/test-container.desktop", tempDir)
	assert.FileExists(t, expectedEntryPath)

	expectedContent := `[Desktop Entry]
Name=Test-container
GenericName=Terminal entering Test-container
Comment=Terminal entering Test-container
Categories=Distrobox;System;Utility
Exec=/usr/bin/distrobox enter --root test-container
Icon=https://raw.githubusercontent.com/89luca89/distrobox/main/icons/terminal-distrobox-icon.svg
Keywords=distrobox;
NoDisplay=false
Terminal=true
TryExec=/usr/bin/distrobox
Type=Application
Actions=Remove;

[Desktop Action Remove]
Name=Remove Test-container from system
Exec=/usr/bin/distrobox rm --root test-container
`

	content, err := os.ReadFile(expectedEntryPath)
	require.NoError(t, err, "Failed to read desktop entry file")
	assert.Equal(t, expectedContent, string(content), "Desktop entry content mismatch")
}

func TestGenerateAllEntriesCommand_Execute(t *testing.T) {
	ctx := context.Background()

	// tempDir is the test directory where we expect to create desktop entries
	tempDir := t.TempDir()
	defer os.RemoveAll(tempDir)

	// create the list command
	containerManager := providers.NewDocker(false, "sudo", false)
	listCmd := commands.NewListCommand(&config.Values{}, containerManager)

	// create the generate all entries command
	genAllEntriesCmd := commands.NewGenerateEntryCommand(&config.Values{}, listCmd)

	//
	// Generate the entries
	//

	opts := &commands.GenerateEntryOptions{
		All:                 true,
		Verbose:             false,
		Delete:              false,
		DesktopEntryBaseDir: fmt.Sprintf("%s/.local/share/", tempDir),
		DistroboxPath:       "/usr/bin/distrobox",
	}
	err := genAllEntriesCmd.Execute(ctx, opts)
	require.NoError(t, err, "GenerateAllEntriesCommand.Execute()")

	// retrieve the list of containers to verify entries were created
	listResult, err := listCmd.Execute(ctx)
	require.NoError(t, err, "ListCommand.Execute()")

	// verify that each container has a corresponding desktop entry
	for _, container := range listResult.Containers {
		expectedEntryPath := fmt.Sprintf("%s/.local/share/applications/%s.desktop", tempDir, container.Name)
		assert.FileExists(t, expectedEntryPath)
	}

	//
	// Delete the entries
	//

	opts.Delete = true
	err = genAllEntriesCmd.Execute(ctx, opts)
	require.NoError(t, err, "GenerateAllEntriesCommand.Execute() on delete")

	// verify that each container's desktop entry has been deleted
	for _, container := range listResult.Containers {
		expectedEntryPath := fmt.Sprintf("%s/.local/share/applications/%s.desktop", tempDir, container.Name)
		assert.NoFileExists(t, expectedEntryPath)
	}
}

// single-container generate-entry detects the distro from the container's
// image, not its name. A box named "dev" running Ubuntu gets the Ubuntu icon.
func TestGenerateEntryCommand_SingleModeUsesImageHint(t *testing.T) {
	tempDir := t.TempDir()

	mock := &testutil.MockContainerManager{
		ListContainersResult: []containermanager.Container{
			{
				Name:   "dev",
				Image:  "docker.io/library/ubuntu:22.04",
				Status: "Exited",
				Labels: map[string]string{"manager": "distrobox"},
			},
		},
	}
	listCmd := commands.NewListCommand(&config.Values{}, mock)
	cmd := commands.NewGenerateEntryCommand(&config.Values{}, listCmd)

	// Pre-seed the icon cache so resolution reuses it (no network) while still
	// proving the distro was detected from the image, not the box name.
	iconPath := filepath.Join(tempDir, "icons", "distrobox", "ubuntu-distrobox.png")
	require.NoError(t, os.MkdirAll(filepath.Dir(iconPath), 0o750))
	require.NoError(t, os.WriteFile(iconPath, []byte("png"), 0o644))

	err := cmd.Execute(context.Background(), &commands.GenerateEntryOptions{
		ContainerName:       "dev",
		Icon:                "auto",
		DesktopEntryBaseDir: tempDir,
		DistroboxPath:       "/usr/bin/distrobox",
	})
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(tempDir, "applications", "dev.desktop"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "ubuntu-distrobox.png",
		"single-mode must key auto-detection off the image (cached icon path), not the box name 'dev'")
}

// generate-entry refuses to write an entry for a container that does not
// exist (matches the shell, distrobox-generate-entry:285-288).
func TestGenerateEntryCommand_RefusesNonExistentContainer(t *testing.T) {
	tempDir := t.TempDir()
	mock := &testutil.MockContainerManager{
		ListContainersResult: []containermanager.Container{
			{Name: "other", Image: "alpine", Status: "Exited", Labels: map[string]string{"manager": "distrobox"}},
		},
	}
	listCmd := commands.NewListCommand(&config.Values{}, mock)
	cmd := commands.NewGenerateEntryCommand(&config.Values{}, listCmd)

	err := cmd.Execute(context.Background(), &commands.GenerateEntryOptions{
		ContainerName:       "ghost",
		DesktopEntryBaseDir: tempDir,
		DistroboxPath:       "/usr/bin/distrobox",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot find container ghost")
	assert.NoFileExists(t, filepath.Join(tempDir, "applications", "ghost.desktop"))
}

// --delete must NOT require the container to exist (its box may already
// be gone when rm cleans up the entry).
func TestGenerateEntryCommand_DeleteNonExistentNoError(t *testing.T) {
	tempDir := t.TempDir()
	mock := &testutil.MockContainerManager{
		ListContainersResult: []containermanager.Container{},
	}
	listCmd := commands.NewListCommand(&config.Values{}, mock)
	cmd := commands.NewGenerateEntryCommand(&config.Values{}, listCmd)

	err := cmd.Execute(context.Background(), &commands.GenerateEntryOptions{
		ContainerName:       "ghost",
		Delete:              true,
		DesktopEntryBaseDir: tempDir,
		DistroboxPath:       "/usr/bin/distrobox",
	})
	require.NoError(t, err)
}
