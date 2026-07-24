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
	printer := ui.NewPrinter(io.Discard, false)
	return commands.NewRmCommand(&config.Values{}, mock, prompter, printer)
}

func newTestRmCommandWithInput(mock *testutil.MockContainerManager, input string) *commands.RmCommand {
	prompter := ui.NewPrompter(*bufio.NewReader(strings.NewReader(input)), io.Discard)
	printer := ui.NewPrinter(io.Discard, false)
	return commands.NewRmCommand(&config.Values{}, mock, prompter, printer)
}

// lastRemoveOptions returns the RmOptions passed to the most recent
// containerManager.Remove call recorded by the mock.
func lastRemoveOptions(t *testing.T, mock *testutil.MockContainerManager) containermanager.RmOptions {
	t.Helper()
	require.NotEmpty(t, mock.Spy.Remove, "expected containerManager.Remove to be called")
	last := mock.Spy.Remove[len(mock.Spy.Remove)-1]
	opts, ok := last[1].(containermanager.RmOptions)
	require.True(t, ok, "second Remove arg should be containermanager.RmOptions")
	return opts
}

// `rm` without --rm-home must never remove the custom home, even
// interactively with a custom home and a "yes" waiting on stdin (the shell only
// ever prompts/removes when --rm-home is set, distrobox-rm:364-371).
func TestRmCommand_RmHome_NotRequested_NeverPrompts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	containerName := "test-rm-no-flag"
	mock := &testutil.MockContainerManager{
		ListContainersResult: []containermanager.Container{
			{Name: containerName, Status: "Exited", Labels: map[string]string{"manager": "distrobox"}},
		},
		InspectContainerResult: &containermanager.InspectResult{ContainerHome: "/custom/home/dir"},
	}
	// Seed a "y" — if the home prompt were (wrongly) shown, it would be consumed
	// and the home removed. Force bypasses the top-level deletion confirmation so
	// the only prompt that could fire is the home one.
	cmd := newTestRmCommandWithInput(mock, "y\n")

	_, err := cmd.Execute(context.Background(), commands.RmOptions{
		ContainerNames: []string{containerName},
		RemoveHome:     false,
		Force:          true,
	})
	require.NoError(t, err)
	assert.False(t, lastRemoveOptions(t, mock).RemoveHome,
		"home must not be removed without --rm-home")
}

// `rm --rm-home` interactively prompts; an explicit "yes" removes the home.
func TestRmCommand_RmHome_Requested_ConfirmedRemoves(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	containerName := "test-rm-flag-yes"
	mock := &testutil.MockContainerManager{
		ListContainersResult: []containermanager.Container{
			{Name: containerName, Status: "Exited", Labels: map[string]string{"manager": "distrobox"}},
		},
		InspectContainerResult: &containermanager.InspectResult{ContainerHome: "/custom/home/dir"},
	}
	// Force bypasses the top-level deletion confirmation; the seeded "y" answers
	// the home-removal prompt.
	cmd := newTestRmCommandWithInput(mock, "y\n")

	_, err := cmd.Execute(context.Background(), commands.RmOptions{
		ContainerNames: []string{containerName},
		RemoveHome:     true,
		Force:          true,
	})
	require.NoError(t, err)
	assert.True(t, lastRemoveOptions(t, mock).RemoveHome,
		"home should be removed with --rm-home and a confirmed prompt")
}

// `rm --rm-home --yes` (non-interactive) never removes the home, matching
// the shell which skips the prompt and defaults to "no" (distrobox-rm:365-366).
func TestRmCommand_RmHome_Requested_NonInteractiveKeepsHome(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	containerName := "test-rm-flag-yes-noninteractive"
	mock := &testutil.MockContainerManager{
		ListContainersResult: []containermanager.Container{
			{Name: containerName, Status: "Exited", Labels: map[string]string{"manager": "distrobox"}},
		},
		InspectContainerResult: &containermanager.InspectResult{ContainerHome: "/custom/home/dir"},
	}
	cmd := newTestRmCommand(mock)

	_, err := cmd.Execute(context.Background(), commands.RmOptions{
		ContainerNames: []string{containerName},
		RemoveHome:     true,
		NoTTY:          true,
	})
	require.NoError(t, err)
	assert.False(t, lastRemoveOptions(t, mock).RemoveHome,
		"non-interactive rm must not remove the home even with --rm-home")
}

func TestRmCommand_TopLevelConfirmation_DeclineAborts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	mock := &testutil.MockContainerManager{
		ListContainersResult: []containermanager.Container{
			{Name: "boxa", Status: "Exited", Labels: map[string]string{"manager": "distrobox"}},
		},
		InspectContainerResult: &containermanager.InspectResult{ContainerHome: "/home/x"},
	}
	cmd := newTestRmCommandWithInput(mock, "n\n")

	_, err := cmd.Execute(context.Background(), commands.RmOptions{ContainerNames: []string{"boxa"}})
	require.ErrorIs(t, err, commands.ErrRmAbortedByUser)
	assert.Empty(t, mock.Spy.Remove, "declining the confirmation must remove nothing")
}

func TestRmCommand_TopLevelConfirmation_DefaultYesProceeds(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	mock := &testutil.MockContainerManager{
		ListContainersResult: []containermanager.Container{
			{Name: "boxa", Status: "Exited", Labels: map[string]string{"manager": "distrobox"}},
		},
		InspectContainerResult: &containermanager.InspectResult{ContainerHome: "/home/x"},
	}
	cmd := newTestRmCommandWithInput(mock, "\n")

	_, err := cmd.Execute(context.Background(), commands.RmOptions{ContainerNames: []string{"boxa"}})
	require.NoError(t, err)
	assert.Len(t, mock.Spy.Remove, 1, "a bare Enter (default yes) must proceed with deletion")
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
