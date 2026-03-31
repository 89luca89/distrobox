package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetDesktopEntryDir resolves the system path for the desktop entry file
func GetDesktopEntryDir() string {
	xdgDataHome := os.Getenv("XDG_DATA_HOME")
	if xdgDataHome == "" {
		home := os.Getenv("HOME")
		return filepath.Join(home, ".local", "share")
	}
	return xdgDataHome
}

// GetDistroboxPath returns the path to the current distrobox executable
func GetDistroboxPath() (string, error) {
	// The current executable is used as distrobox path
	distroboxPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get distrobox executable path: %w", err)
	}
	return distroboxPath, nil
}
