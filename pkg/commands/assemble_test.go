package commands_test

import (
	"bufio"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/89luca89/distrobox/pkg/commands"
	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/internal/testutil"
	"github.com/89luca89/distrobox/pkg/manifest"
	"github.com/89luca89/distrobox/pkg/ui"
)

func newTestAssembleCommand(mock *testutil.MockContainerManager) *commands.AssembleCommand {
	progress := ui.NewDevNullProgress()
	prompter := ui.NewPrompter(*bufio.NewReader(strings.NewReader("")), io.Discard)
	printer := ui.NewPrinter(io.Discard, false)
	return commands.NewAssembleCommand(&config.Values{}, mock, prompter, progress, printer)
}

func getEnterOptions(spy testutil.ContainerManagerSpy, index int) containermanager.EnterOptions {
	return spy.Enter[index][0].(containermanager.EnterOptions)
}

func TestAssembleCommand_SetupBox_StartNowTrue(t *testing.T) {
	mock := &testutil.MockContainerManager{}
	cmd := newTestAssembleCommand(mock)

	err := cmd.Execute(context.Background(), commands.AssembleOptions{
		Items: []manifest.Item{
			{
				Name:     "test-box",
				Image:    "ubuntu:latest",
				StartNow: true,
			},
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, mock.Spy.Enter, "expected Enter to be called when StartNow is true")

	opts := getEnterOptions(mock.Spy, 0)
	assert.Equal(t, "test-box", opts.ContainerName)
	assert.Equal(t, []string{"true"}, opts.CustomCommand)
}

func TestAssembleCommand_SetupBox_ExportedApps_Valid(t *testing.T) {
	validNames := []string{
		"firefox",
		"org.mozilla.firefox",
		"gnome-terminal",
		"lib2to3",
		"g++",
		"my_app.v2",
		"A",
	}

	for _, app := range validNames {
		t.Run(app, func(t *testing.T) {
			mock := &testutil.MockContainerManager{}
			cmd := newTestAssembleCommand(mock)

			apps := []string{app, "another-app"}
			err := cmd.Execute(context.Background(), commands.AssembleOptions{
				Items: []manifest.Item{
					{
						Name:         "test-box",
						Image:        "ubuntu:latest",
						ExportedApps: apps,
					},
				},
			})
			require.NoError(t, err, "expected valid app name %q to succeed", app)
			require.Len(t, mock.Spy.Enter, len(apps))
			for i, a := range apps {
				opts := getEnterOptions(mock.Spy, i)
				expectedCmd := []string{"distrobox-export", "--app", a}
				assert.Equal(t, "test-box", opts.ContainerName, "call %d", i)
				assert.Equal(t, expectedCmd, opts.CustomCommand, "call %d", i)
			}
		})
	}
}

func TestAssembleCommand_SetupBox_ExportedApps_Invalid(t *testing.T) {
	invalidNames := []string{
		"--delete",
		"-rf",
		"",
		"app name with spaces",
		".hidden",
		"_leading",
		"app;rm -rf /",
		"app$(cmd)",
	}

	for _, app := range invalidNames {
		t.Run(app, func(t *testing.T) {
			mock := &testutil.MockContainerManager{}
			cmd := newTestAssembleCommand(mock)

			err := cmd.Execute(context.Background(), commands.AssembleOptions{
				Items: []manifest.Item{
					{
						Name:         "test-box",
						Image:        "ubuntu:latest",
						ExportedApps: []string{app},
					},
				},
			})
			require.Error(t, err, "expected invalid app name %q to be rejected", app)
			assert.Empty(t, mock.Spy.Enter, "expected Enter to not be called for invalid app name %q", app)
		})
	}
}

func TestAssembleCommand_SetupBox_ExportedBins_Valid(t *testing.T) {
	validBins := []string{
		"/usr/bin/vim",
		"/usr/local/bin/node",
		"/opt/app/bin/tool",
		"/usr/bin/g++",
		"/usr/bin/python3.11",
		"/usr/bin/my_tool",
	}

	for _, bin := range validBins {
		t.Run(bin, func(t *testing.T) {
			mock := &testutil.MockContainerManager{}
			cmd := newTestAssembleCommand(mock)

			bins := []string{bin, "/usr/bin/another"}
			err := cmd.Execute(context.Background(), commands.AssembleOptions{
				Items: []manifest.Item{
					{
						Name:             "test-box",
						Image:            "ubuntu:latest",
						ExportedBins:     bins,
						ExportedBinsPath: "/home/user/.local/bin",
					},
				},
			})
			require.NoError(t, err, "expected valid bin path %q to succeed", bin)
			require.Len(t, mock.Spy.Enter, len(bins))
			for i, b := range bins {
				opts := getEnterOptions(mock.Spy, i)
				expectedCmd := []string{"distrobox-export", "--bin", b, "--export-path", "/home/user/.local/bin"}
				assert.Equal(t, "test-box", opts.ContainerName, "call %d", i)
				assert.Equal(t, expectedCmd, opts.CustomCommand, "call %d", i)
			}
		})
	}
}

func TestAssembleCommand_SetupBox_ExportedBins_Invalid(t *testing.T) {
	invalidBins := []string{
		"--delete",
		"-rf",
		"",
		"relative/path/bin",
		"/path with spaces/bin",
		"/usr/bin/app;rm -rf /",
		"/usr/bin/app$(cmd)",
	}

	for _, bin := range invalidBins {
		t.Run(bin, func(t *testing.T) {
			mock := &testutil.MockContainerManager{}
			cmd := newTestAssembleCommand(mock)

			err := cmd.Execute(context.Background(), commands.AssembleOptions{
				Items: []manifest.Item{
					{
						Name:             "test-box",
						Image:            "ubuntu:latest",
						ExportedBins:     []string{bin},
						ExportedBinsPath: "/home/user/.local/bin",
					},
				},
			})
			require.Error(t, err, "expected invalid bin path %q to be rejected", bin)
			assert.Empty(t, mock.Spy.Enter, "expected Enter to not be called for invalid bin path %q", bin)
		})
	}
}

func TestAssembleCommand_RoutesCreateAndSetupByItemRoot(t *testing.T) {
	mock := &testutil.MockContainerManager{}
	cmd := newTestAssembleCommand(mock)

	require.NotNil(t, mock.RootClone, "AssembleCommand constructor should call CloneAsRoot")

	err := cmd.Execute(context.Background(), commands.AssembleOptions{
		Items: []manifest.Item{
			{Name: "rootless-box", Image: "ubuntu:latest", Root: false, StartNow: true},
			{Name: "rootful-box", Image: "ubuntu:latest", Root: true, StartNow: true},
		},
	})
	require.NoError(t, err)

	require.Len(t, mock.Spy.Create, 1, "rootless mock should receive exactly one Create")
	createOpts := mock.Spy.Create[0][0].(containermanager.CreateOptions)
	assert.Equal(t, "rootless-box", createOpts.ContainerName)

	require.Len(t, mock.RootClone.Spy.Create, 1, "root mock should receive exactly one Create")
	createOptsRoot := mock.RootClone.Spy.Create[0][0].(containermanager.CreateOptions)
	assert.Equal(t, "rootful-box", createOptsRoot.ContainerName)

	require.Len(t, mock.Spy.Enter, 1, "rootless mock should receive exactly one Enter (StartNow)")
	assert.Equal(t, "rootless-box", getEnterOptions(mock.Spy, 0).ContainerName)

	require.Len(t, mock.RootClone.Spy.Enter, 1, "root mock should receive exactly one Enter (StartNow)")
	assert.Equal(t, "rootful-box", getEnterOptions(mock.RootClone.Spy, 0).ContainerName)
}

func TestAssembleCommand_RootlessOnlyManifest_DoesNotInvokeRootMock(t *testing.T) {
	mock := &testutil.MockContainerManager{}
	cmd := newTestAssembleCommand(mock)

	err := cmd.Execute(context.Background(), commands.AssembleOptions{
		Items: []manifest.Item{
			{Name: "a", Image: "ubuntu:latest", Root: false},
			{Name: "b", Image: "ubuntu:latest", Root: false},
		},
	})
	require.NoError(t, err)

	assert.Len(t, mock.Spy.Create, 2)
	require.NotNil(t, mock.RootClone, "constructor still calls CloneAsRoot eagerly")
	assert.Empty(t, mock.RootClone.Spy.Create, "root mock should not receive Create for rootless items")
	assert.Empty(t, mock.RootClone.Spy.Enter, "root mock should not receive Enter for rootless items")
}

func TestAssembleCommand_SetupBox_ExportedBins_InvalidExportPath(t *testing.T) {
	invalidPaths := []string{
		"",
		"relative/path",
		"--some-flag",
		"/path with spaces",
		"/path;injection",
	}

	for _, path := range invalidPaths {
		t.Run(path, func(t *testing.T) {
			mock := &testutil.MockContainerManager{}
			cmd := newTestAssembleCommand(mock)

			err := cmd.Execute(context.Background(), commands.AssembleOptions{
				Items: []manifest.Item{
					{
						Name:             "test-box",
						Image:            "ubuntu:latest",
						ExportedBins:     []string{"/usr/bin/vim"},
						ExportedBinsPath: path,
					},
				},
			})
			require.Error(t, err, "expected invalid export path %q to be rejected", path)
			assert.Empty(t, mock.Spy.Enter, "expected Enter to not be called for invalid export path %q", path)
		})
	}
}
