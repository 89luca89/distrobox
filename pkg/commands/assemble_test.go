package commands_test

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/89luca89/distrobox/pkg/commands"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/internal/testutil"
	"github.com/89luca89/distrobox/pkg/manifest"
	"github.com/89luca89/distrobox/pkg/ui"
)

func newTestAssembleCommand(mock *testutil.MockContainerManager) *commands.AssembleCommand {
	progress := ui.NewDevNullProgress()
	prompter := ui.NewPrompter(*bufio.NewReader(strings.NewReader("")), io.Discard)
	printer := ui.NewPrinter(io.Discard, false)
	return commands.NewAssembleCommand(mock, prompter, progress, printer)
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.Spy.Enter) == 0 {
		t.Fatal("expected Enter to be called when StartNow is true, but it was not")
	}

	opts := getEnterOptions(mock.Spy, 0)
	if opts.ContainerName != "test-box" {
		t.Errorf("expected ContainerName %q, got %q", "test-box", opts.ContainerName)
	}
	if opts.CustomCommand != "true" {
		t.Errorf("expected CustomCommand %q, got %q", "true", opts.CustomCommand)
	}
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
			if err != nil {
				t.Fatalf("expected valid app name %q to succeed, got error: %v", app, err)
			}
			if len(mock.Spy.Enter) != len(apps) {
				t.Fatalf("expected Enter to be called %d times, got %d", len(apps), len(mock.Spy.Enter))
			}
			for i, a := range apps {
				opts := getEnterOptions(mock.Spy, i)
				expectedCmd := fmt.Sprintf("distrobox-export --app %s", a)
				if opts.ContainerName != "test-box" {
					t.Errorf("call %d: expected ContainerName %q, got %q", i, "test-box", opts.ContainerName)
				}
				if opts.CustomCommand != expectedCmd {
					t.Errorf("call %d: expected CustomCommand %q, got %q", i, expectedCmd, opts.CustomCommand)
				}
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
			if err == nil {
				t.Fatalf("expected invalid app name %q to be rejected, but got no error", app)
			}
			if len(mock.Spy.Enter) != 0 {
				t.Errorf("expected Enter to not be called for invalid app name %q, but it was called %d times", app, len(mock.Spy.Enter))
			}
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
			if err != nil {
				t.Fatalf("expected valid bin path %q to succeed, got error: %v", bin, err)
			}
			if len(mock.Spy.Enter) != len(bins) {
				t.Fatalf("expected Enter to be called %d times, got %d", len(bins), len(mock.Spy.Enter))
			}
			for i, b := range bins {
				opts := getEnterOptions(mock.Spy, i)
				expectedCmd := fmt.Sprintf("distrobox-export --bin %s --export-path /home/user/.local/bin", b)
				if opts.ContainerName != "test-box" {
					t.Errorf("call %d: expected ContainerName %q, got %q", i, "test-box", opts.ContainerName)
				}
				if opts.CustomCommand != expectedCmd {
					t.Errorf("call %d: expected CustomCommand %q, got %q", i, expectedCmd, opts.CustomCommand)
				}
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
			if err == nil {
				t.Fatalf("expected invalid bin path %q to be rejected, but got no error", bin)
			}
			if len(mock.Spy.Enter) != 0 {
				t.Errorf("expected Enter to not be called for invalid bin path %q, but it was called %d times", bin, len(mock.Spy.Enter))
			}
		})
	}
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
			if err == nil {
				t.Fatalf("expected invalid export path %q to be rejected, but got no error", path)
			}
			if len(mock.Spy.Enter) != 0 {
				t.Errorf("expected Enter to not be called for invalid export path %q, but it was called %d times", path, len(mock.Spy.Enter))
			}
		})
	}
}
