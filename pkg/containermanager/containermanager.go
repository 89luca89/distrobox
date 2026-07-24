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

package containermanager

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/89luca89/distrobox/pkg/ui"
)

// darwinGOOS is the runtime.GOOS value reported on macOS.
const darwinGOOS = "darwin"

// bindGOOS is the OS used to decide bind-mount propagation. It is a package
// variable so tests can exercise the macOS branch without running on darwin.
//
//nolint:gochecknoglobals // deliberate test seam for the macOS bind-propagation branch
var bindGOOS = runtime.GOOS

// BindPropagation returns the propagation suffix for shared bind mounts.
//
// macOS (Docker Desktop / Colima) runs containers inside a Linux VM that mounts
// host paths as private, so rslave/rshared propagation is unsupported and must
// be dropped there — mirroring the reference shell (distrobox-create:677-685).
func BindPropagation() string {
	if bindGOOS == darwinGOOS {
		return ""
	}
	return ":rslave"
}

// ReadOnlyBindPropagation is BindPropagation for read-only mounts.
func ReadOnlyBindPropagation() string {
	if bindGOOS == darwinGOOS {
		return ":ro"
	}
	return ":ro,rslave"
}

const (
	RunningStatus = "running"
)

type Container struct {
	ID     string
	Image  string
	Name   string
	Status string
	Labels map[string]string
}

type InspectResult struct {
	ContainerID     string
	ContainerStatus string
	ContainerHome   string
	ContainerPath   string
	UnshareGroups   bool
}

type CreateOptions struct {
	ContainerName           string
	ContainerImage          string
	ContainerClone          string
	ContainerUserCustomHome string
	ContainerHostname       string
	ContainerPlatform       string
	ContainerUserHome       string
	Nopasswd                bool
	UnshareDevsys           bool
	UnshareGroups           bool
	UnshareIPC              bool
	UnshareNetNS            bool
	UnshareProcess          bool
	AdditionalFlags         []string
	AdditionalVolumes       []string
	AdditionalPackages      []string
	ContainerPreInitHook    string
	ContainerInitHook       string
	Init                    bool
	Nvidia                  bool
	DryRun                  bool
	// ScriptsDir is the directory where helper scripts (distrobox-init,
	// distrobox-export, distrobox-host-exec) are stored. It is resolved
	// by config.Values.ScriptsDir and passed through so the provider can
	// mount them into the container.
	ScriptsDir string
}

type EnterOptions struct {
	ContainerName   string
	AdditionalFlags string
	CustomCommand   []string
	DryRun          bool
	NoTTY           bool
	NoWorkDir       bool
	CleanPath       bool
	Verbose         bool
}

type RmOptions struct {
	Force         bool
	RemoveHome    bool
	ContainerHome string
}

// IsDistrobox returns true when the container was created by distrobox.
//
// Two label shapes count as distrobox-owned, mirroring what the create path
// always sets (pkg/containermanager/providers/{docker,podman}.go):
//   - manager=distrobox (the standard case)
//   - any label key prefixed with "distrobox." (e.g. distrobox.unshare_groups)
//
// The key-prefix branch keeps the 24b31ed8 fix working for containers whose
// manager label is overridden via --additional-flags --label=manager=apx,
// while staying out of label *values* — those are arbitrary tool/user
// strings (workdirs, mount paths, project names) and substring-matching
// them produces false positives on unrelated containers.
func (c Container) IsDistrobox() bool {
	if c.Labels["manager"] == "distrobox" {
		return true
	}
	for key := range c.Labels {
		if strings.HasPrefix(key, "distrobox.") {
			return true
		}
	}
	return false
}

func (c Container) IsRunning() bool {
	s := strings.ToLower(c.Status)
	return strings.Contains(s, "up") || strings.Contains(s, "running")
}

//nolint:revive // ContainerManagerType is intentionally named for clarity despite the stutter
type ContainerManagerType string

type ContainerManager interface {
	Name() string
	// CloneAsRoot returns a copy of the manager configured to run in root
	// mode. The original instance is not modified.
	CloneAsRoot() ContainerManager
	Enter(ctx context.Context, options EnterOptions, progress *ui.Progress, printer *ui.Printer) error
	ListContainers(ctx context.Context) ([]Container, error)
	Create(ctx context.Context, opts CreateOptions) error
	Remove(ctx context.Context, containerName string, opts RmOptions) error
	Exists(ctx context.Context, containerName string) bool
	ImageExists(ctx context.Context, imageName string) bool
	Stop(ctx context.Context, containerNames []string) error
	InspectContainer(ctx context.Context, containerName string) (*InspectResult, error)
	PullImage(ctx context.Context, imageName string, platform string, dryRun bool) error
	Commit(ctx context.Context, containerID string, imageTag string) error
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func IsSymlink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}

func Btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

func BuildContainerPath(cleanPath bool, hostPath string, containerPath string) string {
	standardPaths := []string{"/usr/local/sbin", "/usr/local/bin", "/usr/sbin", "/usr/bin", "/sbin", "/bin"}

	if cleanPath {
		return strings.Join(standardPaths, ":")
	}

	// If no host PATH, use the container's PATH if available
	if hostPath == "" {
		if containerPath != "" {
			return containerPath
		}
		return strings.Join(standardPaths, ":")
	}

	// Add standard paths not in host PATH
	hostSegments := strings.Split(hostPath, ":")
	var additionalPaths []string
	for _, sp := range standardPaths {
		if !slices.Contains(hostSegments, sp) {
			additionalPaths = append(additionalPaths, sp)
		}
	}

	merged := hostPath
	if len(additionalPaths) > 0 {
		merged = hostPath + ":" + strings.Join(additionalPaths, ":")
	}

	return reorderFHSPath(merged)
}

// reorderFHSPath ensures /usr/local/bin precedes /usr/bin and /usr/local/sbin
// precedes /usr/sbin, so distrobox wrappers in /usr/local/* win — mirroring the
// reference shell (distrobox-enter:478-512).
func reorderFHSPath(path string) string {
	var reordered []string
	for _, p := range strings.Split(path, ":") {
		switch p {
		case "/usr/local/bin", "/usr/local/sbin":
			// Skip here; re-inserted right before its /usr counterpart.
		case "/usr/bin":
			reordered = append(reordered, "/usr/local/bin", "/usr/bin")
		case "/usr/sbin":
			reordered = append(reordered, "/usr/local/sbin", "/usr/sbin")
		default:
			reordered = append(reordered, p)
		}
	}

	result := strings.Join(reordered, ":")

	// If /usr/bin or /usr/sbin were absent, their local counterparts were
	// skipped above; re-add any that went missing (prepended, like the shell).
	for _, lp := range []string{"/usr/local/bin", "/usr/local/sbin"} {
		if !slices.Contains(strings.Split(result, ":"), lp) {
			result = lp + ":" + result
		}
	}

	return result
}

func BuildXDGPaths(envVar string, standardPaths []string) string {
	containerPaths := os.Getenv(envVar)

	for _, sp := range standardPaths {
		if containerPaths == "" {
			containerPaths = sp
		} else if !slices.Contains(strings.Split(containerPaths, ":"), sp) {
			containerPaths = containerPaths + ":" + sp
		}
	}

	return containerPaths
}

func FilterEnvVars() []string {
	result := []string{}

	// Compile regex for XDG_.*_DIRS pattern
	xdgDirsPattern := regexp.MustCompile(`^XDG_.*_DIRS$`)

	// Excluded prefixes
	excludedPrefixes := []string{
		"CONTAINER_ID",
		"FPATH",
		"HOST",
		"HOSTNAME",
		"HOME",
		"PATH",
		"PROFILEREAD",
		"PWD", // host PWD is replaced by --env=PWD=<workdir>
		"SHELL",
		"XDG_SEAT",
		"XDG_VTNR",
		"_", // Variables starting with underscore
	}

	for _, env := range os.Environ() {
		// Must contain '='
		if !strings.Contains(env, "=") {
			continue
		}

		// Exclude if contains ", `, or $
		if strings.ContainsAny(env, "\"`$") {
			continue
		}

		// Split into key and value
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]

		// Check excluded prefixes
		excluded := false
		for _, prefix := range excludedPrefixes {
			if strings.HasPrefix(key, prefix) {
				excluded = true
				break
			}
		}

		if excluded || xdgDirsPattern.MatchString(key) {
			continue
		}

		result = append(result, env)
	}

	return result
}

// IsTTY returns true if both stdin and stdout are terminals.
// Mirrors the shell's: if [ ! -t 0 ] || [ ! -t 1 ]; then headless=1; fi
func IsTTY() bool {
	if fi, err := os.Stdin.Stat(); err != nil || fi.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	if fi, err := os.Stdout.Stat(); err != nil || fi.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	return true
}

func GetWorkDir(containerHome string, noWorkDir bool) (string, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("error getting working dir: %w", err)
	}

	if noWorkDir {
		return containerHome, nil
	}

	if workDir == "" && containerHome == "" {
		return "/", nil
	}

	if workDir == "" {
		return containerHome, nil
	}

	if !strings.Contains(workDir, containerHome) {
		return "/run/host" + workDir, nil
	}

	return workDir, nil
}

func BuildCommandArgs(customCommand []string, user string, noTTY bool, unshareGroups bool) []string {
	args := customCommand
	if len(args) == 0 {
		// Default: execute user's shell with login
		args = []string{"/bin/sh", "-c", fmt.Sprintf("$(getent passwd '%s' | cut -f 7 -d :) -l", user)}
	}

	// Handle unshare_groups mode - use su to trigger proper login
	if unshareGroups {
		unshareArgs := []string{"su"}
		if !noTTY {
			unshareArgs = append(unshareArgs, "--pty")
		}
		unshareArgs = append(unshareArgs, "-m", "-s", "/bin/sh", "-c", `"$0" "$@"`, "--", user)
		unshareArgs = append(unshareArgs, args...)
		return unshareArgs
	}

	return args
}

func TimestampNow() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
