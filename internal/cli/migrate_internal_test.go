package cli

import (
	"context"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

// migrateSpyContainerManager is a minimal spy that records calls to the
// methods used by the migrate command. All other methods are no-ops.
type migrateSpyContainerManager struct {
	stops   [][]string
	creates []containermanager.CreateOptions
	removes []string
	commits []string

	inspectResult *containermanager.InspectResult

	// needsMigrationResult, when non-nil, overrides the default return
	// of NeedsMigration. The default is true (migrate), matching a v1
	// container with no version label.
	needsMigrationResult *bool
}

func (s *migrateSpyContainerManager) Name() string                                   { return "spy" }
func (s *migrateSpyContainerManager) CloneAsRoot() containermanager.ContainerManager { return s }
func (s *migrateSpyContainerManager) Enter(_ context.Context, _ containermanager.EnterOptions, _ *ui.Progress, _ *ui.Printer) error {
	return nil
}
func (s *migrateSpyContainerManager) ListContainers(_ context.Context) ([]containermanager.Container, error) {
	return nil, nil
}
func (s *migrateSpyContainerManager) Create(_ context.Context, opts containermanager.CreateOptions) error {
	s.creates = append(s.creates, opts)
	return nil
}
func (s *migrateSpyContainerManager) Remove(_ context.Context, name string, _ containermanager.RmOptions) error {
	s.removes = append(s.removes, name)
	return nil
}
func (s *migrateSpyContainerManager) Exists(_ context.Context, _ string) bool { return true }
func (s *migrateSpyContainerManager) Stop(_ context.Context, names []string) error {
	s.stops = append(s.stops, names)
	return nil
}
func (s *migrateSpyContainerManager) InspectContainer(_ context.Context, _ string) (*containermanager.InspectResult, error) {
	if s.inspectResult != nil {
		return s.inspectResult, nil
	}
	return &containermanager.InspectResult{}, nil
}
func (s *migrateSpyContainerManager) Commit(_ context.Context, containerID string, _ string) error {
	s.commits = append(s.commits, containerID)
	return nil
}
func (s *migrateSpyContainerManager) ImageExists(_ context.Context, _ string) bool { return true }
func (s *migrateSpyContainerManager) PullImage(_ context.Context, _ string, _ string, _ bool) error {
	return nil
}
func (s *migrateSpyContainerManager) NeedsMigration(_ context.Context, _ string) (bool, error) {
	if s.needsMigrationResult != nil {
		return *s.needsMigrationResult, nil
	}
	return true, nil
}

// runMigrate runs the migrate subcommand with the given argv (starting from
// "migrate") against a spy container manager. It returns the spy so the
// caller can assert on recorded calls.
func runMigrate(t *testing.T, spy *migrateSpyContainerManager, argv ...string) {
	t.Helper()

	cfg := config.DefaultValues()
	cfg.NonInteractive = true

	cmd := newMigrateCommand(cfg)
	// Override the Before hook to inject our spy instead of detecting a
	// real container manager, matching the pattern in enter_internal_test.go.
	cmd.Before = func(ctx context.Context, _ *cli.Command) (context.Context, error) {
		return context.WithValue(ctx, containerManagerKey, spy), nil
	}

	root := &cli.Command{Commands: []*cli.Command{cmd}}
	full := append([]string{"distrobox"}, argv...)

	if err := root.Run(context.Background(), full); err != nil {
		t.Fatalf("unexpected error running migrate: %v", err)
	}
}

func TestNewMigrateCommand_HasFlags(t *testing.T) {
	cfg := config.DefaultValues()
	cmd := newMigrateCommand(cfg)

	if cmd.Name != "migrate" {
		t.Errorf("expected name 'migrate', got %q", cmd.Name)
	}

	flagNames := map[string]bool{
		"all":     false,
		"force":   false,
		"dry-run": false,
		"yes":     false,
	}
	for _, flag := range cmd.Flags {
		for name := range flagNames {
			if flag.Names()[0] == name {
				flagNames[name] = true
			}
		}
	}
	for name, found := range flagNames {
		if !found {
			t.Errorf("expected flag --%s to be defined", name)
		}
	}
}

func TestMigrateAction_NoContainerSpecified_ReturnsError(t *testing.T) {
	t.Setenv("DBX_SCRIPTS_DIR", t.TempDir())

	cfg := config.DefaultValues()
	cfg.ContainerName = ""
	cfg.NonInteractive = true

	spy := &migrateSpyContainerManager{}
	cmd := newMigrateCommand(cfg)
	cmd.Before = func(ctx context.Context, _ *cli.Command) (context.Context, error) {
		return context.WithValue(ctx, containerManagerKey, spy), nil
	}

	root := &cli.Command{Commands: []*cli.Command{cmd}}
	args := []string{"distrobox", "migrate"}

	err := root.Run(context.Background(), args)
	if err == nil {
		t.Fatal("expected error when no container specified and no --all")
	}
}

func TestMigrateAction_DryRun(t *testing.T) {
	t.Setenv("DBX_SCRIPTS_DIR", t.TempDir())

	spy := &migrateSpyContainerManager{
		inspectResult: &containermanager.InspectResult{
			ContainerID:     "abc123",
			ContainerStatus: "exited",
			ContainerImage:  "alpine:latest",
			NetworkMode:     "host",
			IpcMode:         "host",
			PidMode:         "host",
			Env:             []string{"HOME=/home/testuser"},
			Cmd: []string{
				"--verbose", "--name", "testuser", "--user", "1000",
				"--group", "1000", "--home", "/home/testuser",
				"--init", "0", "--nvidia", "0",
				"--pre-init-hooks", "", "--additional-packages", "",
				"--", "",
			},
			Mounts: []containermanager.MountInfo{
				{Source: "/usr/lib/distrobox/distrobox-init", Destination: "/usr/bin/entrypoint"},
			},
		},
	}

	runMigrate(t, spy, "migrate", "--dry-run", "my-box")

	// Dry run: no side effects
	if len(spy.stops) != 0 {
		t.Errorf("expected 0 Stop calls, got %d", len(spy.stops))
	}
	if len(spy.commits) != 0 {
		t.Errorf("expected 0 Commit calls, got %d", len(spy.commits))
	}
	if len(spy.removes) != 0 {
		t.Errorf("expected 0 Remove calls, got %d", len(spy.removes))
	}
	if len(spy.creates) != 0 {
		t.Errorf("expected 0 Create calls, got %d", len(spy.creates))
	}
}

func TestMigrateAction_V2Container_Skipped(t *testing.T) {
	v2ScriptDir := t.TempDir()
	t.Setenv("DBX_SCRIPTS_DIR", v2ScriptDir)

	notNeeded := false
	spy := &migrateSpyContainerManager{
		inspectResult: &containermanager.InspectResult{
			ContainerID:     "abc123",
			ContainerStatus: "exited",
			ContainerImage:  "alpine:latest",
			NetworkMode:     "host",
			IpcMode:         "host",
			PidMode:         "host",
			Env:             []string{"HOME=/home/testuser"},
			Labels: map[string]string{
				containermanager.VersionLabelKey: "2",
			},
		},
		needsMigrationResult: &notNeeded,
	}

	runMigrate(t, spy, "migrate", "my-box")

	// Already migrated: no side effects
	if len(spy.stops) != 0 {
		t.Errorf("expected 0 Stop calls, got %d", len(spy.stops))
	}
	if len(spy.commits) != 0 {
		t.Errorf("expected 0 Commit calls, got %d", len(spy.commits))
	}
	if len(spy.removes) != 0 {
		t.Errorf("expected 0 Remove calls, got %d", len(spy.removes))
	}
	if len(spy.creates) != 0 {
		t.Errorf("expected 0 Create calls, got %d", len(spy.creates))
	}
}

func TestMigrateAction_ForceRecreates(t *testing.T) {
	t.Setenv("USER", "testuser")
	t.Setenv("HOME", "/home/testuser")
	t.Setenv("SHELL", "/bin/sh")

	v2ScriptDir := t.TempDir()
	t.Setenv("DBX_SCRIPTS_DIR", v2ScriptDir)

	spy := &migrateSpyContainerManager{
		inspectResult: &containermanager.InspectResult{
			ContainerID:     "abc123",
			ContainerStatus: "exited",
			ContainerImage:  "alpine:latest",
			NetworkMode:     "host",
			IpcMode:         "host",
			PidMode:         "host",
			Env:             []string{"HOME=/home/testuser"},
			Cmd: []string{
				"--verbose", "--name", "testuser", "--user", "1000",
				"--group", "1000", "--home", "/home/testuser",
				"--init", "0", "--nvidia", "0",
				"--pre-init-hooks", "", "--additional-packages", "",
				"--", "",
			},
			Mounts: []containermanager.MountInfo{
				{Source: v2ScriptDir + "/distrobox-init", Destination: "/usr/bin/entrypoint"},
			},
		},
	}

	runMigrate(t, spy, "migrate", "--force", "my-box")

	// Force: should recreate even if already v2
	if len(spy.commits) != 1 {
		t.Errorf("expected 1 Commit call, got %d", len(spy.commits))
	}
	if len(spy.removes) != 1 {
		t.Errorf("expected 1 Remove call, got %d", len(spy.removes))
	}
	if len(spy.creates) != 1 {
		t.Errorf("expected 1 Create call, got %d", len(spy.creates))
	}
}
