package insidedistrobox

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed assets/distrobox-host-exec
var hostExecScript string

//go:embed assets/distrobox-init
var initScript string

//go:embed assets/distrobox-export
var exportScripts string

// ProvisionScripts ensures that all necessary scripts are created in the host directory.
// It returns the path to the directory where the scripts are stored.
func ProvisionScripts() (string, error) {
	dir := hostDir()
	//nolint:gosec // 0755 is the same as from distrobox v1, let's keep it for compatibility
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create scripts directory: %w", err)
	}

	scripts := []struct {
		name    string
		content string
	}{
		{"distrobox-host-exec", hostExecScript},
		{"distrobox-init", initScript},
		{"distrobox-export", exportScripts},
	}

	for _, script := range scripts {
		destFilePath := filepath.Join(dir, script.name)
		//nolint:gosec // 0755 is the same as from distrobox v1, let's keep it for compatibility
		if err := os.WriteFile(destFilePath, []byte(script.content), 0755); err != nil {
			return "", fmt.Errorf("failed to write script %s: %w", script.name, err)
		}
	}

	return dir, nil
}

// hostDir returns the directory path where the scripts should be stored.
// Evaluates DBX_SCRIPTS_DIR env var first, then HOME env var, and falls back to default path.
func hostDir() string {
	// First check DBX_SCRIPTS_DIR env var
	if dir := os.Getenv("DBX_SCRIPTS_DIR"); dir != "" {
		return dir
	}

	// Then, check HOME env var
	// v2 is added to avoid collisions with v1 installations
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".local", "share", "distrobox", "v2")
	}

	// Fallback to default path
	return "/var/lib/distrobox/v2"
}
