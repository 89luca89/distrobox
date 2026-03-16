package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// LoadConfig loads configuration files in order, with later files taking priority.
// Environment variables are NOT overwritten by config files.
func LoadConfig() error {
	configFilePaths, err := getConfigFilePaths()
	if err != nil {
		return fmt.Errorf("failed to get config file paths: %w", err)
	}

	if err := godotenv.Load(configFilePaths...); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	return nil
}

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

// getConfigFilePaths returns a list of configuration file paths in order of priority.
func getConfigFilePaths() ([]string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate symlinks for executable path: %w", err)
	}

	selfDir := filepath.Dir(execPath)

	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfigHome == "" {
		xdgConfigHome = filepath.Join(os.Getenv("HOME"), ".config")
	}

	home := os.Getenv("HOME")

	// Highest priority first
	return []string{
		filepath.Join(home, ".distroboxrc"),
		filepath.Join(xdgConfigHome, "distrobox", "distrobox.conf"),
		"/etc/distrobox/distrobox.conf",
		"/usr/local/share/distrobox/distrobox.conf",
		"/usr/etc/distrobox/distrobox.conf",
		"/usr/share/defaults/distrobox/distrobox.conf",
		"/usr/share/distrobox/distrobox.conf",
		filepath.Join(selfDir, "..", "share", "distrobox", "distrobox.conf"),
	}, nil
}
