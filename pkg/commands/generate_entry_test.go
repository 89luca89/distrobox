package commands_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/89luca89/distrobox/pkg/commands"
	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager/providers"
)

func TestGenerateEntryCommand_Execute(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	defer os.RemoveAll(tempDir)

	// create the list command
	containerManager := providers.NewDocker(false, "sudo", false)
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

	containerManager := providers.NewDocker(false, "sudo", false)
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
