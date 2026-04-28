package containermanager

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/89luca89/distrobox/pkg/ui"
)

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

func (c Container) IsDistrobox() bool {
	return c.Labels["manager"] == "distrobox"
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
	PullImage(ctx context.Context, imageName string, platform string) error
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
	var additionalPaths []string
	for _, sp := range standardPaths {
		pattern := regexp.MustCompile(`(:|^)` + regexp.QuoteMeta(sp) + `(:|$)`)
		if !pattern.MatchString(hostPath) {
			additionalPaths = append(additionalPaths, sp)
		}
	}

	if len(additionalPaths) > 0 {
		return hostPath + ":" + strings.Join(additionalPaths, ":")
	}

	return hostPath
}

func BuildXDGPaths(envVar string, standardPaths []string) string {
	containerPaths := os.Getenv(envVar)

	for _, sp := range standardPaths {
		pattern := regexp.MustCompile(`(:|^)` + regexp.QuoteMeta(sp) + `(:|$)`)
		if containerPaths == "" {
			containerPaths = sp
		} else if !pattern.MatchString(containerPaths) {
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
	return time.Now().UTC().Format("2006-01-02T15:04:05.000000000+00:00")
}
