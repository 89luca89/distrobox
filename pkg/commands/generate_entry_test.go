package commands_test

import (
	"context"
	"fmt"
	"os"
	"testing"

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
	if err != nil {
		t.Errorf("GenerateEntryCommand.Execute() error = %v", err)
	}

	expectedEntryPath := fmt.Sprintf("%s/.local/share/applications/test-container.desktop", tempDir)
	if _, err := os.Stat(expectedEntryPath); os.IsNotExist(err) {
		t.Errorf("Expected desktop entry file %s does not exist, %s", expectedEntryPath, tempDir)
	}

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
	if err != nil {
		t.Errorf("Failed to read desktop entry file: %v", err)
	}

	if string(content) != expectedContent {
		t.Errorf(
			"Desktop entry content does not match expected.\nGot:\n'%s'\nExpected:\n'%s'",
			string(content),
			expectedContent,
		)
	}

	// Delete the entry
	opts.Delete = true
	err = generateEntryCmd.Execute(ctx, opts)
	if err != nil {
		t.Errorf("GenerateEntryCommand.Execute() error on delete = %v", err)
	}

	if _, err := os.Stat(expectedEntryPath); !os.IsNotExist(err) {
		t.Errorf("Expected desktop entry file %s to be deleted, %s", expectedEntryPath, tempDir)
	}

	// Try deleting a non-existing entry
	err = generateEntryCmd.Execute(ctx, opts)
	if err != nil {
		t.Errorf("GenerateEntryCommand.Execute() error on delete non-existing = %v", err)
	}
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
	if err != nil {
		t.Errorf("GenerateEntryCommand.Execute() error = %v", err)
	}

	expectedEntryPath := fmt.Sprintf("%s/.local/share/applications/test-container.desktop", tempDir)
	if _, err := os.Stat(expectedEntryPath); os.IsNotExist(err) {
		t.Errorf("Expected desktop entry file %s does not exist, %s", expectedEntryPath, tempDir)
	}

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
	if err != nil {
		t.Errorf("Failed to read desktop entry file: %v", err)
	}

	if string(content) != expectedContent {
		t.Errorf(
			"Desktop entry content does not match expected.\nGot:\n'%s'\nExpected:\n'%s'",
			string(content),
			expectedContent,
		)
	}
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

	if err != nil {
		t.Errorf("GenerateAllEntriesCommand.Execute() error = %v", err)
	}

	// retrieve the list of containers to verify entries were created
	listResult, err := listCmd.Execute(ctx)
	if err != nil {
		t.Errorf("ListCommand.Execute() error = %v", err)
	}

	// verify that each container has a corresponding desktop entry
	for _, container := range listResult.Containers {
		expectedEntryPath := fmt.Sprintf("%s/.local/share/applications/%s.desktop", tempDir, container.Name)
		if _, err := os.Stat(expectedEntryPath); os.IsNotExist(err) {
			t.Errorf("Expected desktop entry file %s does not exist", expectedEntryPath)
		}
	}

	//
	// Delete the entries
	//

	opts.Delete = true
	err = genAllEntriesCmd.Execute(ctx, opts)

	if err != nil {
		t.Errorf("GenerateAllEntriesCommand.Execute() error = %v", err)
	}

	// verify that each container's desktop entry has been deleted
	for _, container := range listResult.Containers {
		expectedEntryPath := fmt.Sprintf("%s/.local/share/applications/%s.desktop", tempDir, container.Name)
		if _, err := os.Stat(expectedEntryPath); !os.IsNotExist(err) {
			t.Errorf("Expected desktop entry file %s to be deleted", expectedEntryPath)
		}
	}
}
