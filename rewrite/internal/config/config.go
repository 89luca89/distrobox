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
