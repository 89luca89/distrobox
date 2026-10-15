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

// ProvisionScripts ensures the helper scripts exist on disk and returns their
// directory: scriptsDir when they are already there or it is writable, else a
// per-user data dir (so a system-wide install still works when run rootless).
func ProvisionScripts(scriptsDir string) (string, error) {
	scripts := []struct {
		name    string
		content string
	}{
		{"distrobox-host-exec", hostExecScript},
		{"distrobox-init", initScript},
		{"distrobox-export", exportScripts},
	}

	// Reuse scripts already present (e.g. packaged), even if read-only.
	allPresent := true
	for _, script := range scripts {
		if _, err := os.Stat(filepath.Join(scriptsDir, script.name)); err != nil {
			allPresent = false
			break
		}
	}
	if allPresent {
		return scriptsDir, nil
	}

	// Extract into scriptsDir, or a per-user dir when it is not writable.
	dir := scriptsDir
	//nolint:gosec // 0755 is the same as from distrobox v1, let's keep it for compatibility
	if err := os.MkdirAll(dir, 0755); err != nil || !dirWritable(dir) {
		dir = userScriptsDir()
	}
	//nolint:gosec // 0755 is the same as from distrobox v1, let's keep it for compatibility
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create scripts directory %s: %w", dir, err)
	}

	for _, script := range scripts {
		destFilePath := filepath.Join(dir, script.name)
		if _, err := os.Stat(destFilePath); err == nil {
			continue
		}
		//nolint:gosec // 0755 is the same as from distrobox v1, let's keep it for compatibility
		if err := os.WriteFile(destFilePath, []byte(script.content), 0755); err != nil {
			return "", fmt.Errorf("failed to write script %s: %w", script.name, err)
		}
	}

	return dir, nil
}

// dirWritable reports whether a file can be created in dir (probes with a temp file).
func dirWritable(dir string) bool {
	probe, err := os.CreateTemp(dir, ".dbx-write-probe-")
	if err != nil {
		return false
	}
	defer func() {
		_ = probe.Close()
		_ = os.Remove(probe.Name())
	}()

	return true
}

// userScriptsDir is the per-user fallback for extracted scripts.
func userScriptsDir() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "distrobox")
	}
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".local", "share", "distrobox")
	}

	return filepath.Join(os.TempDir(), "distrobox")
}
