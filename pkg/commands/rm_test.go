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
// findExportedDesktopApps will match for the given containerName (i.e.
// the per-app desktop entries created by distrobox-export, not the
// one created by `distrobox generate-entry`).
func writeExportedDesktopApp(t *testing.T, userHome, containerName string) string {
	t.Helper()
	appsDir := filepath.Join(userHome, ".local", "share", "applications")
	require.NoError(t, os.MkdirAll(appsDir, 0o755))
	desktopFile := filepath.Join(appsDir, containerName+"-app.desktop")
	content := "[Desktop Entry]\nExec=/usr/bin/distrobox enter " + containerName + " -- some-app\n"
	require.NoError(t, os.WriteFile(desktopFile, []byte(content), 0o644))
	return desktopFile
}

// writeGenerateEntryDesktop writes the desktop file produced by
// `distrobox generate-entry <name>` so the generate-entry delete
// branch (invoked from RmCommand.cleanup) can be observed.
func writeGenerateEntryDesktop(t *testing.T, userHome, containerName string) string {
	t.Helper()
	appsDir := filepath.Join(userHome, ".local", "share", "applications")
	require.NoError(t, os.MkdirAll(appsDir, 0o755))
	entryFile := filepath.Join(appsDir, containerName+".desktop")
	require.NoError(t, os.WriteFile(entryFile, []byte("[Desktop Entry]\nName="+containerName+"\n"), 0o644))
	return entryFile
}

func TestRmCommand_Execute_CleanupRemovesExportedDesktopApp(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("XDG_DATA_HOME", filepath.Join(tempHome, ".local", "share"))

	containerName := "test-rm-cleanup"
	exportedDesktopFile := writeExportedDesktopApp(t, tempHome, containerName)

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
	assert.NoFileExists(t, exportedDesktopFile, "cleanup should remove the per-app exported desktop file")
}

func TestRmCommand_Execute_CleanupRemovesGenerateEntryDesktop(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("XDG_DATA_HOME", filepath.Join(tempHome, ".local", "share"))

	containerName := "test-rm-genentry"
	generateEntryFile := writeGenerateEntryDesktop(t, tempHome, containerName)

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
	assert.NoFileExists(t, generateEntryFile, "cleanup should invoke GenerateEntry delete to remove the <name>.desktop file")
}
