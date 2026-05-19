package commands_test

import (
	"bufio"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/89luca89/distrobox/pkg/commands"
	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/internal/testutil"
	"github.com/89luca89/distrobox/pkg/ui"
)

func newTestRmCommand(mock *testutil.MockContainerManager) *commands.RmCommand {
	prompter := ui.NewPrompter(*bufio.NewReader(strings.NewReader("")), io.Discard)
	return commands.NewRmCommand(&config.Values{}, mock, prompter)
}

// writeExportedDesktopApp writes a minimal desktop file that
// findExportedDesktopApps will match for the given containerName.
func writeExportedDesktopApp(t *testing.T, userHome, containerName string) string {
	t.Helper()
	appsDir := filepath.Join(userHome, ".local", "share", "applications")
	require.NoError(t, os.MkdirAll(appsDir, 0o755))
	desktopFile := filepath.Join(appsDir, containerName+"-app.desktop")
	content := "[Desktop Entry]\nExec=/usr/bin/distrobox enter " + containerName + " -- some-app\n"
	require.NoError(t, os.WriteFile(desktopFile, []byte(content), 0o644))
	return desktopFile
}

func TestRmCommand_Execute_CleanupRemovesExportedDesktopApp(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	containerName := "test-rm-cleanup"
	desktopFile := writeExportedDesktopApp(t, tempHome, containerName)

	mock := &testutil.MockContainerManager{
		ListContainersResult: []containermanager.Container{
			{
				Name:   containerName,
				Status: "Exited",
				Labels: map[string]string{"manager": "distrobox"},
			},
		},
		InspectContainerResult: &containermanager.InspectResult{
			ContainerHome: tempHome,
		},
	}
	cmd := newTestRmCommand(mock)

	_, err := cmd.Execute(context.Background(), commands.RmOptions{
		ContainerNames: []string{containerName},
		Force:          true,
		NoTTY:          true,
	})
	require.NoError(t, err)
	assert.NoFileExists(t, desktopFile, "cleanup should remove the exported desktop file")
}

func TestRmCommand_Execute_VerbosePropagatesThroughCleanup(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	containerName := "test-rm-verbose"
	desktopFile := writeExportedDesktopApp(t, tempHome, containerName)

	mock := &testutil.MockContainerManager{
		ListContainersResult: []containermanager.Container{
			{
				Name:   containerName,
				Status: "Exited",
				Labels: map[string]string{"manager": "distrobox"},
			},
		},
		InspectContainerResult: &containermanager.InspectResult{
			ContainerHome: tempHome,
		},
	}
	cmd := newTestRmCommand(mock)

	// With Verbose: true the cleanup path (including GenerateEntry delete)
	// should still complete without error and remove exported artifacts.
	_, err := cmd.Execute(context.Background(), commands.RmOptions{
		ContainerNames: []string{containerName},
		Force:          true,
		NoTTY:          true,
		Verbose:        true,
	})
	require.NoError(t, err)
	assert.NoFileExists(t, desktopFile, "cleanup should remove the exported desktop file when verbose is true")
}
