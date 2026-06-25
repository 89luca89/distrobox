package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	insidedistrobox "github.com/89luca89/distrobox/internal/inside-distrobox"
	"github.com/89luca89/distrobox/internal/userenv"
	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

// ErrMigrateNoContainerSpecified is returned when no container name is given
// and --all is not set.
var ErrMigrateNoContainerSpecified = errors.New("please specify the name of the container to migrate")

// ErrMigrateAlreadyMigrated is returned (per-container) when a container
// already has v2 mountpoints and --force was not requested.
var ErrMigrateAlreadyMigrated = errors.New("container is already migrated to v2")

// MigrateOptions holds the options for the migrate command.
type MigrateOptions struct {
	// ContainerNames is the explicit list of containers to migrate.
	ContainerNames []string
	// All migrates every distrobox container found.
	All bool
	// Force skips the "already migrated" check and recreates anyway.
	Force bool
	// DryRun prints the commands that would be executed without running them.
	DryRun bool
	// NonInteractive skips confirmation prompts.
	NonInteractive bool
}

// MigrateCommand orchestrates the migration of v1 distrobox containers to v2.
type MigrateCommand struct {
	cfg              *config.Values
	containerManager containermanager.ContainerManager
	listCmd          *ListCommand
	printer          *ui.Printer
	prompter         *ui.Prompter
}

// NewMigrateCommand creates a new MigrateCommand.
func NewMigrateCommand(
	cfg *config.Values,
	cm containermanager.ContainerManager,
	printer *ui.Printer,
	prompter *ui.Prompter,
) *MigrateCommand {
	return &MigrateCommand{
		cfg:              cfg,
		containerManager: cm,
		listCmd:          NewListCommand(cfg, cm),
		printer:          printer,
		prompter:         prompter,
	}
}

// Execute runs the migration for the specified containers.
func (c *MigrateCommand) Execute(ctx context.Context, opts MigrateOptions) error {
	var containerNames []string

	switch {
	case opts.All:
		containers, err := c.listCmd.Execute(ctx)
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}
		if len(containers.Containers) == 0 {
			return ErrEmptyContainerList
		}
		containerNames = make([]string, 0, len(containers.Containers))
		for _, container := range containers.Containers {
			containerNames = append(containerNames, container.Name)
		}
	case len(opts.ContainerNames) > 0:
		containerNames = opts.ContainerNames
	default:
		return ErrMigrateNoContainerSpecified
	}

	var lastErr error
	for _, name := range containerNames {
		if err := c.migrateContainer(ctx, name, opts); err != nil {
			if errors.Is(err, ErrMigrateAlreadyMigrated) {
				c.printer.Println("Container '%s' is already migrated to v2, skipping.", name)
				continue
			}
			c.printer.PrintErrorln("error migrating %s: %s", name, err)
			lastErr = err
			continue
		}
	}

	return lastErr
}

// migrateContainer performs the migration for a single container.
func (c *MigrateCommand) migrateContainer(ctx context.Context, name string, opts MigrateOptions) error {
	inspectResult, err := c.containerManager.InspectContainer(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to inspect container %s: %w", name, err)
	}

	// Check if migration is needed
	if !opts.Force {
		if c.isAlreadyMigrated(inspectResult) {
			return ErrMigrateAlreadyMigrated
		}
	}

	// Confirm with the user unless non-interactive
	if !opts.NonInteractive && !opts.DryRun && c.prompter != nil {
		msg := fmt.Sprintf(
			"Migrate container '%s'? This will stop, commit, remove and recreate it.",
			name,
		)
		if !c.prompter.Prompt(msg, true) {
			c.printer.Println("Skipping '%s'.", name)
			return nil
		}
	}

	c.printer.Println("Migrating '%s'...", name)

	// Recover the original creation options from the inspect data
	createOpts := c.recoverCreateOptions(ctx, name, inspectResult)

	if opts.DryRun {
		c.printer.Println("[dry-run] Would stop, commit, remove and recreate container '%s' with image '%s'", name, createOpts.ContainerImage)
		return nil
	}

	// Step 1: Stop the container if running
	if inspectResult.ContainerStatus == containermanager.RunningStatus {
		c.printer.Println("Stopping '%s'...", name)
		if err := c.containerManager.Stop(ctx, []string{name}); err != nil {
			return fmt.Errorf("failed to stop container %s: %w", name, err)
		}
	}

	// Step 2: Commit the container's filesystem to a temporary image
	commitTag := fmt.Sprintf("%s:migrate-%s", strings.ToLower(name), time.Now().Format("2006-01-02"))
	c.printer.Println("Committing container '%s' to image '%s'...", name, commitTag)
	if err := c.containerManager.Commit(ctx, inspectResult.ContainerID, commitTag); err != nil {
		return fmt.Errorf("failed to commit container %s: %w", name, err)
	}

	// Step 3: Remove the old container
	c.printer.Println("Removing old container '%s'...", name)
	if err := c.containerManager.Remove(ctx, name, containermanager.RmOptions{Force: true}); err != nil {
		return fmt.Errorf("failed to remove old container %s: %w", name, err)
	}

	// Step 4: Provision v2 scripts (ensure they exist before creating)
	if _, err := insidedistrobox.ProvisionScripts(); err != nil {
		return fmt.Errorf("failed to provision v2 scripts: %w", err)
	}

	// Step 5: Recreate the container with the committed image and recovered options
	createOpts.ContainerImage = commitTag
	c.printer.Println("Recreating container '%s'...", name)
	if err := c.containerManager.Create(ctx, createOpts); err != nil {
		return fmt.Errorf("failed to recreate container %s: %w", name, err)
	}

	c.printer.Println("Container '%s' migrated successfully.", name)
	return nil
}

// isAlreadyMigrated checks whether the container's entrypoint mount source
// already points to the v2 scripts directory.
func (c *MigrateCommand) isAlreadyMigrated(inspect *containermanager.InspectResult) bool {
	v2Dir := insidedistrobox.ScriptsDir()
	for _, mount := range inspect.Mounts {
		if mount.Destination == "/usr/bin/entrypoint" {
			// The source should be <v2Dir>/distrobox-init
			expected := filepath.Join(v2Dir, "distrobox-init")
			if mount.Source == expected {
				return true
			}
			// Also accept if the source is within the v2 directory (e.g., symlink resolution)
			if strings.HasPrefix(mount.Source, v2Dir) {
				return true
			}
			return false
		}
	}
	// If we can't find the entrypoint mount at all, conservatively say not migrated
	return false
}

// recoverCreateOptions reconstructs the CreateOptions from the container's
// inspect data, so the recreated container matches the original as closely as
// possible.
//
//nolint:gocognit,funlen // imperative option reconstruction is inherently linear
func (c *MigrateCommand) recoverCreateOptions(
	ctx context.Context,
	name string,
	inspect *containermanager.InspectResult,
) containermanager.CreateOptions {
	opts := containermanager.CreateOptions{
		ContainerName: name,
	}

	// Use the committed image as the container image (set later by caller)
	opts.ContainerImage = inspect.ContainerImage

	// Parse Cmd/Args to recover distrobox-init arguments
	cmd := inspect.Cmd

	// Recover --init, --nvidia, --pre-init-hooks, --additional-packages,
	// --home, and the init hook (after --)
	for i := 0; i < len(cmd); i++ {
		arg := cmd[i]
		switch arg {
		case "--init":
			if i+1 < len(cmd) {
				opts.Init = cmd[i+1] == "1"
				i++
			}
		case "--nvidia":
			if i+1 < len(cmd) {
				opts.Nvidia = cmd[i+1] == "1"
				i++
			}
		case "--pre-init-hooks":
			if i+1 < len(cmd) {
				opts.ContainerPreInitHook = cmd[i+1]
				i++
			}
		case "--additional-packages":
			if i+1 < len(cmd) {
				opts.AdditionalPackages = strings.Fields(cmd[i+1])
				i++
			}
		case "--home":
			if i+1 < len(cmd) {
				home := cmd[i+1]
				userEnv := userenv.LoadUserEnvironment(ctx)
				// If the home differs from the user's real home, it's a custom home
				if home != userEnv.Home {
					opts.ContainerUserCustomHome = home
				}
				i++
			}
		case "--":
			// Everything after -- is the init hook
			if i+1 < len(cmd) {
				opts.ContainerInitHook = strings.Join(cmd[i+1:], " ")
			}
		}
	}

	// Recover unshare flags from HostConfig modes. A mode of "host"
	// means the namespace is shared; anything else means it is unshared.
	const hostMode = "host"
	opts.UnshareNetNS = inspect.NetworkMode != "" && inspect.NetworkMode != hostMode
	opts.UnshareIPC = inspect.IpcMode != "" && inspect.IpcMode != hostMode
	opts.UnshareProcess = inspect.PidMode != "" && inspect.PidMode != hostMode

	// Recover unshare_groups from inspect (already parsed label)
	opts.UnshareGroups = inspect.UnshareGroups

	// Recover UnshareDevsys: check if /dev:/dev mount is present
	const devPath = "/dev"
	opts.UnshareDevsys = true
	for _, mount := range inspect.Mounts {
		if mount.Source == devPath && mount.Destination == devPath {
			opts.UnshareDevsys = false
			break
		}
	}

	// Recover container hostname from env
	for _, env := range inspect.Env {
		if strings.HasPrefix(env, "HOSTNAME=") {
			hostname := strings.TrimPrefix(env, "HOSTNAME=")
			// Only set if it's not the default (host hostname)
			if h, err := os.Hostname(); err == nil && hostname != h {
				opts.ContainerHostname = hostname
			}
			break
		}
	}

	// Recover nopasswd from presence of /run/.nopasswd mount
	for _, mount := range inspect.Mounts {
		if mount.Destination == "/run/.nopasswd" {
			opts.Nopasswd = true
			break
		}
	}

	// Recover additional user volumes: mounts that are not standard distrobox
	// mounts and are bind-type (have a source that's an absolute path we can
	// reconstruct as src:dst[:opts])
	opts.AdditionalVolumes = c.recoverAdditionalVolumes(inspect.Mounts)

	return opts
}

// recoverAdditionalVolumes extracts user-specified additional volumes from
// the mount list, filtering out standard distrobox mounts.
func (c *MigrateCommand) recoverAdditionalVolumes(mounts []containermanager.MountInfo) []string {
	// Standard distrobox mount destinations that are managed by the create
	// command and should not be treated as user volumes.
	standardDestinations := map[string]bool{
		"/usr/bin/entrypoint":          true,
		"/usr/bin/distrobox-export":    true,
		"/usr/bin/distrobox-host-exec": true,
		"/tmp":                         true,
		"/dev":                         true,
		"/dev/pts":                     true,
		"/dev/ptmx":                    true,
		"/sys":                         true,
		"/sys/fs/selinux":              true,
		"/var/log/journal":             true,
		"/run/.nopasswd":               true,
		"/run/.distrobox.rootless":     true,
		"/etc/hosts":                   true,
		"/etc/resolv.conf":             true,
		"/etc/hostname":                true,
	}

	// Standard source paths that are managed by distrobox
	standardSources := map[string]bool{
		"/dev/null": true,
		"/dev":      true,
		"/sys":      true,
		"/tmp":      true,
		"/":         true, // root mount for /run/host
	}

	var volumes []string
	for _, mount := range mounts {
		// Skip mounts with standard destinations
		if standardDestinations[mount.Destination] {
			continue
		}

		// Skip /run/host/* mounts (host root filesystem mounts)
		if strings.HasPrefix(mount.Destination, "/run/host") {
			continue
		}

		// Skip mounts where source is a standard system path and destination
		// is also a system path
		if standardSources[mount.Source] {
			continue
		}

		// Skip user home mount (source == destination == $HOME)
		if mount.Source == mount.Destination {
			// This is likely the home directory mount or XDG_RUNTIME_DIR mount
			// These are handled by create from the user environment
			continue
		}

		// Skip mounts from the v2 scripts directory
		v2Dir := insidedistrobox.ScriptsDir()
		if strings.HasPrefix(mount.Source, v2Dir) {
			continue
		}

		// Skip anonymous volumes (empty source = container-managed volume)
		if mount.Source == "" {
			continue
		}

		// This is a user-specified additional volume: reconstruct src:dst[:opts]
		vol := mount.Source + ":" + mount.Destination
		if mount.Options != "" {
			// Filter out container-manager internal options, keep user-facing ones
			vol += ":" + mount.Options
		}
		volumes = append(volumes, vol)
	}

	return volumes
}
