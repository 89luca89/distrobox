package config

import (
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
