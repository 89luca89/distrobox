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

// ProvisionScripts ensures that all necessary scripts are created in the given directory.
// It returns scriptsDir unchanged on success.
func ProvisionScripts(scriptsDir string) (string, error) {
	dir := scriptsDir
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
		if exists(script.name) {
			continue
		}

		destFilePath := filepath.Join(dir, script.name)
		//nolint:gosec // 0755 is the same as from distrobox v1, let's keep it for compatibility
		if err := os.WriteFile(destFilePath, []byte(script.content), 0755); err != nil {
			return "", fmt.Errorf("failed to write script %s: %w", script.name, err)
		}
	}

	return dir, nil
}

// exists reports whether a script with the given name is already
// available on the host.
func exists(name string) bool {
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}

	return false
}
