package commands_test

import (
	"bufio"
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"

	insidedistrobox "github.com/89luca89/distrobox/internal/inside-distrobox"
	"github.com/89luca89/distrobox/pkg/commands"
	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager"
	testutil "github.com/89luca89/distrobox/pkg/internal/testutil"
	"github.com/89luca89/distrobox/pkg/ui"
)

func newMigrateTestSetup(t *testing.T) (*commands.MigrateCommand, *testutil.MockContainerManager) {
	t.Helper()
	t.Setenv("USER", "testuser")
	t.Setenv("HOME", "/home/testuser")
	t.Setenv("SHELL", "/bin/sh")

	// Use a temp directory for scripts to avoid touching the real HOME
	t.Setenv("DBX_SCRIPTS_DIR", t.TempDir())

	cfg := config.DefaultValues()
	mock := &testutil.MockContainerManager{}

	printer := ui.NewPrinter(io.Discard, false)
	prompter := ui.NewPrompter(*bufio.NewReader(strings.NewReader("")), nil)

	migrateCmd := commands.NewMigrateCommand(cfg, mock, printer, prompter)
	return migrateCmd, mock
}

func TestMigrate_V1Container_TriggersStopCommitRemoveCreate(t *testing.T) {
	migrateCmd, mock := newMigrateTestSetup(t)

	// Simulate a v1 container: entrypoint mount source does NOT point to v2 dir
	v1EntrypointPath := "/usr/lib/distrobox/distrobox-init"
	ctx := context.Background()

	mock.InspectContainerResult = &containermanager.InspectResult{
		ContainerID:     "abc123",
		ContainerStatus: "running",
		ContainerImage:  "quay.io/toolbx-images/alpine-toolbox:edge",
		ContainerHome:   "/home/testuser",
		NetworkMode:     "host",
		IpcMode:         "host",
		PidMode:         "host",
		Env:             []string{"HOME=/home/testuser", "HOSTNAME=testhost"},
		Cmd: []string{
			"--verbose",
			"--name", "testuser",
			"--user", "1000",
			"--group", "1000",
			"--home", "/home/testuser",
			"--init", "0",
			"--nvidia", "0",
			"--pre-init-hooks", "",
			"--additional-packages", " vim git",
			"--", "",
		},
		Mounts: []containermanager.MountInfo{
			{Source: v1EntrypointPath, Destination: "/usr/bin/entrypoint"},
			{Source: "/usr/lib/distrobox/distrobox-export", Destination: "/usr/bin/distrobox-export"},
			{Source: "/usr/lib/distrobox/distrobox-host-exec", Destination: "/usr/bin/distrobox-host-exec"},
			{Source: "/home/testuser", Destination: "/home/testuser"},
			{Source: "/tmp", Destination: "/tmp"},
			{Source: "/dev", Destination: "/dev"},
		},
	}

	opts := commands.MigrateOptions{
		ContainerNames: []string{"my-box"},
		NonInteractive: true,
	}

	err := migrateCmd.Execute(ctx, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the sequence: Stop, Commit, Remove, Create
	if len(mock.Spy.Stop) != 1 {
		t.Errorf("expected 1 Stop call, got %d", len(mock.Spy.Stop))
	}
	stopArgs, _ := mock.Spy.Stop[0][0].([]string)
	if len(stopArgs) != 1 || stopArgs[0] != "my-box" {
		t.Errorf("expected Stop(['my-box']), got %v", stopArgs)
	}

	if len(mock.Spy.Commit) != 1 {
		t.Errorf("expected 1 Commit call, got %d", len(mock.Spy.Commit))
	}
	commitContainerID, _ := mock.Spy.Commit[0][0].(string)
	if commitContainerID != "abc123" {
		t.Errorf("expected Commit('abc123', ...), got %s", commitContainerID)
	}

	if len(mock.Spy.Remove) != 1 {
		t.Errorf("expected 1 Remove call, got %d", len(mock.Spy.Remove))
	}
	removeName, _ := mock.Spy.Remove[0][0].(string)
	if removeName != "my-box" {
		t.Errorf("expected Remove('my-box', ...), got %s", removeName)
	}

	if len(mock.Spy.Create) != 1 {
		t.Errorf("expected 1 Create call, got %d", len(mock.Spy.Create))
	}
	createOpts, _ := mock.Spy.Create[0][0].(containermanager.CreateOptions)
	if createOpts.ContainerName != "my-box" {
		t.Errorf("expected ContainerName 'my-box', got %s", createOpts.ContainerName)
	}
	// The committed image tag should be used as the image
	if createOpts.ContainerImage == "" {
		t.Error("expected non-empty ContainerImage (committed tag)")
	}
	if createOpts.ContainerImage == "quay.io/toolbx-images/alpine-toolbox:edge" {
		t.Error("expected committed image tag, not the original image")
	}
}

func TestMigrate_V2Container_Skipped(t *testing.T) {
	migrateCmd, mock := newMigrateTestSetup(t)

	// Simulate a v2 container: entrypoint mount source points to v2 dir
	v2Dir := insidedistrobox.ScriptsDir()
	v2EntrypointPath := filepath.Join(v2Dir, "distrobox-init")

	ctx := context.Background()

	mock.InspectContainerResult = &containermanager.InspectResult{
		ContainerID:     "abc123",
		ContainerStatus: "exited",
		ContainerImage:  "alpine:latest",
		Mounts: []containermanager.MountInfo{
			{Source: v2EntrypointPath, Destination: "/usr/bin/entrypoint"},
		},
	}

	opts := commands.MigrateOptions{
		ContainerNames: []string{"my-box"},
		NonInteractive: true,
	}

	err := migrateCmd.Execute(ctx, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify no stop/commit/remove/create calls were made
	if len(mock.Spy.Stop) != 0 {
		t.Errorf("expected 0 Stop calls, got %d", len(mock.Spy.Stop))
	}
	if len(mock.Spy.Commit) != 0 {
		t.Errorf("expected 0 Commit calls, got %d", len(mock.Spy.Commit))
	}
	if len(mock.Spy.Remove) != 0 {
		t.Errorf("expected 0 Remove calls, got %d", len(mock.Spy.Remove))
	}
	if len(mock.Spy.Create) != 0 {
		t.Errorf("expected 0 Create calls, got %d", len(mock.Spy.Create))
	}
}

func TestMigrate_V2Container_ForceRecreates(t *testing.T) {
	migrateCmd, mock := newMigrateTestSetup(t)

	v2Dir := insidedistrobox.ScriptsDir()
	v2EntrypointPath := filepath.Join(v2Dir, "distrobox-init")

	ctx := context.Background()

	mock.InspectContainerResult = &containermanager.InspectResult{
		ContainerID:     "abc123",
		ContainerStatus: "exited",
		ContainerImage:  "alpine:latest",
		NetworkMode:     "host",
		IpcMode:         "host",
		PidMode:         "host",
		Env:             []string{"HOME=/home/testuser"},
		Cmd: []string{
			"--verbose",
			"--name", "testuser",
			"--user", "1000",
			"--group", "1000",
			"--home", "/home/testuser",
			"--init", "0",
			"--nvidia", "0",
			"--pre-init-hooks", "",
			"--additional-packages", "",
			"--", "",
		},
		Mounts: []containermanager.MountInfo{
			{Source: v2EntrypointPath, Destination: "/usr/bin/entrypoint"},
			{Source: filepath.Join(v2Dir, "distrobox-export"), Destination: "/usr/bin/distrobox-export"},
		},
	}

	opts := commands.MigrateOptions{
		ContainerNames: []string{"my-box"},
		Force:          true,
		NonInteractive: true,
	}

	err := migrateCmd.Execute(ctx, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With --force, the container should be recreated (Commit + Remove + Create)
	// Stop should NOT be called because the container is not running
	if len(mock.Spy.Stop) != 0 {
		t.Errorf("expected 0 Stop calls (container not running), got %d", len(mock.Spy.Stop))
	}
	if len(mock.Spy.Commit) != 1 {
		t.Errorf("expected 1 Commit call, got %d", len(mock.Spy.Commit))
	}
	if len(mock.Spy.Remove) != 1 {
		t.Errorf("expected 1 Remove call, got %d", len(mock.Spy.Remove))
	}
	if len(mock.Spy.Create) != 1 {
		t.Errorf("expected 1 Create call, got %d", len(mock.Spy.Create))
	}
}

func TestMigrate_DryRun_NoSideEffects(t *testing.T) {
	migrateCmd, mock := newMigrateTestSetup(t)

	// Use a v1 container (entrypoint not in v2 dir)
	v1EntrypointPath := "/usr/lib/distrobox/distrobox-init"

	ctx := context.Background()

	mock.InspectContainerResult = &containermanager.InspectResult{
		ContainerID:     "abc123",
		ContainerStatus: "running",
		ContainerImage:  "alpine:latest",
		NetworkMode:     "host",
		IpcMode:         "host",
		PidMode:         "host",
		Env:             []string{"HOME=/home/testuser"},
		Cmd: []string{
			"--verbose",
			"--name", "testuser",
			"--user", "1000",
			"--group", "1000",
			"--home", "/home/testuser",
			"--init", "0",
			"--nvidia", "0",
			"--pre-init-hooks", "",
			"--additional-packages", "",
			"--", "",
		},
		Mounts: []containermanager.MountInfo{
			{Source: v1EntrypointPath, Destination: "/usr/bin/entrypoint"},
		},
	}

	opts := commands.MigrateOptions{
		ContainerNames: []string{"my-box"},
		DryRun:         true,
		NonInteractive: true,
	}

	err := migrateCmd.Execute(ctx, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Dry run: no side effects
	if len(mock.Spy.Stop) != 0 {
		t.Errorf("expected 0 Stop calls, got %d", len(mock.Spy.Stop))
	}
	if len(mock.Spy.Commit) != 0 {
		t.Errorf("expected 0 Commit calls, got %d", len(mock.Spy.Commit))
	}
	if len(mock.Spy.Remove) != 0 {
		t.Errorf("expected 0 Remove calls, got %d", len(mock.Spy.Remove))
	}
	if len(mock.Spy.Create) != 0 {
		t.Errorf("expected 0 Create calls, got %d", len(mock.Spy.Create))
	}
}

func TestMigrate_OptionReconstruction(t *testing.T) {
	migrateCmd, mock := newMigrateTestSetup(t)

	v1EntrypointPath := "/usr/lib/distrobox/distrobox-init"

	ctx := context.Background()

	mock.InspectContainerResult = &containermanager.InspectResult{
		ContainerID:     "abc123",
		ContainerStatus: "exited",
		ContainerImage:  "alpine:latest",
		NetworkMode:     "host", // not unshare-netns
		IpcMode:         "host", // not unshare-ipc
		PidMode:         "host", // not unshare-process
		Env:             []string{"HOME=/home/testuser"},
		// Test Init=true, Nvidia=true, custom packages and hooks
		Cmd: []string{
			"--verbose",
			"--name", "testuser",
			"--user", "1000",
			"--group", "1000",
			"--home", "/home/testuser",
			"--init", "1",
			"--nvidia", "1",
			"--pre-init-hooks", "echo hello",
			"--additional-packages", "vim neovim",
			"--", "echo world",
		},
		Mounts: []containermanager.MountInfo{
			{Source: v1EntrypointPath, Destination: "/usr/bin/entrypoint"},
			{Source: "/dev", Destination: "/dev"},                // UnshareDevsys = false
			{Source: "/dev/null", Destination: "/run/.nopasswd"}, // Nopasswd = true
			// Additional user volume
			{Source: "/opt/myapp", Destination: "/usr/local/myapp", Options: "rbind"},
		},
	}

	opts := commands.MigrateOptions{
		ContainerNames: []string{"my-box"},
		NonInteractive: true,
	}

	err := migrateCmd.Execute(ctx, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.Spy.Create) != 1 {
		t.Fatalf("expected 1 Create call, got %d", len(mock.Spy.Create))
	}

	createOpts, _ := mock.Spy.Create[0][0].(containermanager.CreateOptions)

	// Verify Init and Nvidia
	if !createOpts.Init {
		t.Error("expected Init=true")
	}
	if !createOpts.Nvidia {
		t.Error("expected Nvidia=true")
	}

	// Verify pre-init hooks
	if createOpts.ContainerPreInitHook != "echo hello" {
		t.Errorf("expected pre-init hooks 'echo hello', got %q", createOpts.ContainerPreInitHook)
	}

	// Verify additional packages
	if len(createOpts.AdditionalPackages) != 2 {
		t.Errorf("expected 2 additional packages, got %d: %v", len(createOpts.AdditionalPackages), createOpts.AdditionalPackages)
	}
	if createOpts.AdditionalPackages[0] != "vim" || createOpts.AdditionalPackages[1] != "neovim" {
		t.Errorf("expected [vim neovim], got %v", createOpts.AdditionalPackages)
	}

	// Verify init hook
	if createOpts.ContainerInitHook != "echo world" {
		t.Errorf("expected init hook 'echo world', got %q", createOpts.ContainerInitHook)
	}

	// Verify UnshareDevsys (false because /dev:/dev mount exists)
	if createOpts.UnshareDevsys {
		t.Error("expected UnshareDevsys=false")
	}

	// Verify unshare flags (all false because modes are "host")
	if createOpts.UnshareNetNS {
		t.Error("expected UnshareNetNS=false")
	}
	if createOpts.UnshareIPC {
		t.Error("expected UnshareIPC=false")
	}
	if createOpts.UnshareProcess {
		t.Error("expected UnshareProcess=false")
	}

	// Verify Nopasswd
	if !createOpts.Nopasswd {
		t.Error("expected Nopasswd=true")
	}

	// Verify additional volumes
	foundVolume := false
	for _, vol := range createOpts.AdditionalVolumes {
		if vol == "/opt/myapp:/usr/local/myapp:rbind" {
			foundVolume = true
			break
		}
	}
	if !foundVolume {
		t.Errorf("expected additional volume '/opt/myapp:/usr/local/myapp:rbind' in %v", createOpts.AdditionalVolumes)
	}
}

func TestMigrate_OptionReconstruction_UnshareFlags(t *testing.T) {
	migrateCmd, mock := newMigrateTestSetup(t)

	v1EntrypointPath := "/usr/lib/distrobox/distrobox-init"

	ctx := context.Background()

	mock.InspectContainerResult = &containermanager.InspectResult{
		ContainerID:     "abc123",
		ContainerStatus: "exited",
		ContainerImage:  "alpine:latest",
		// Non-"host" modes indicate unshare flags
		NetworkMode: "bridge",
		IpcMode:     "private",
		PidMode:     "container:other",
		Env:         []string{"HOME=/home/testuser"},
		Cmd: []string{
			"--verbose",
			"--name", "testuser",
			"--user", "1000",
			"--group", "1000",
			"--home", "/home/testuser",
			"--init", "0",
			"--nvidia", "0",
			"--pre-init-hooks", "",
			"--additional-packages", "",
			"--", "",
		},
		Mounts: []containermanager.MountInfo{
			{Source: v1EntrypointPath, Destination: "/usr/bin/entrypoint"},
			// No /dev:/dev mount means UnshareDevsys = true
		},
	}

	opts := commands.MigrateOptions{
		ContainerNames: []string{"my-box"},
		NonInteractive: true,
	}

	err := migrateCmd.Execute(ctx, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.Spy.Create) != 1 {
		t.Fatalf("expected 1 Create call, got %d", len(mock.Spy.Create))
	}

	createOpts, _ := mock.Spy.Create[0][0].(containermanager.CreateOptions)

	if !createOpts.UnshareNetNS {
		t.Error("expected UnshareNetNS=true (NetworkMode=bridge)")
	}
	if !createOpts.UnshareIPC {
		t.Error("expected UnshareIPC=true (IpcMode=private)")
	}
	if !createOpts.UnshareProcess {
		t.Error("expected UnshareProcess=true (PidMode=container:other)")
	}
	if !createOpts.UnshareDevsys {
		t.Error("expected UnshareDevsys=true (no /dev:/dev mount)")
	}
}

func TestMigrate_OptionReconstruction_CustomHome(t *testing.T) {
	migrateCmd, mock := newMigrateTestSetup(t)

	v1EntrypointPath := "/usr/lib/distrobox/distrobox-init"
	customHome := "/custom/home/mybox"

	ctx := context.Background()

	mock.InspectContainerResult = &containermanager.InspectResult{
		ContainerID:     "abc123",
		ContainerStatus: "exited",
		ContainerImage:  "alpine:latest",
		NetworkMode:     "host",
		IpcMode:         "host",
		PidMode:         "host",
		Env:             []string{"HOME=/custom/home/mybox"},
		Cmd: []string{
			"--verbose",
			"--name", "testuser",
			"--user", "1000",
			"--group", "1000",
			"--home", customHome,
			"--init", "0",
			"--nvidia", "0",
			"--pre-init-hooks", "",
			"--additional-packages", "",
			"--", "",
		},
		Mounts: []containermanager.MountInfo{
			{Source: v1EntrypointPath, Destination: "/usr/bin/entrypoint"},
			{Source: "/dev", Destination: "/dev"},
		},
	}

	opts := commands.MigrateOptions{
		ContainerNames: []string{"my-box"},
		NonInteractive: true,
	}

	err := migrateCmd.Execute(ctx, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.Spy.Create) != 1 {
		t.Fatalf("expected 1 Create call, got %d", len(mock.Spy.Create))
	}

	createOpts, _ := mock.Spy.Create[0][0].(containermanager.CreateOptions)

	if createOpts.ContainerUserCustomHome != customHome {
		t.Errorf("expected ContainerUserCustomHome %q, got %q", customHome, createOpts.ContainerUserCustomHome)
	}
}

func TestMigrate_NoContainerSpecified(t *testing.T) {
	migrateCmd, _ := newMigrateTestSetup(t)

	ctx := context.Background()

	opts := commands.MigrateOptions{}

	err := migrateCmd.Execute(ctx, opts)
	if err == nil {
		t.Fatal("expected error when no container specified")
	}
}

func TestMigrate_All_NoContainers(t *testing.T) {
	migrateCmd, _ := newMigrateTestSetup(t)

	ctx := context.Background()

	opts := commands.MigrateOptions{
		All: true,
	}

	err := migrateCmd.Execute(ctx, opts)
	if err == nil {
		t.Fatal("expected error when --all but no containers found")
	}
}
