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

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToStruct(t *testing.T) {
	input := map[string]string{
		"container_manager":          "docker",
		"sudo_program":               "doas",
		"verbose":                    "true",
		"container_image":            "ubuntu:latest",
		"container_name":             "mybox",
		"container_image_env":        "alpine:3.21",
		"container_name_env":         "envbox",
		"container_hostname":         "myhost",
		"container_user_custom_home": "/tmp/home",
		"container_home_prefix":      "/data/boxes",
		"container_always_pull":      "1",
		"non_interactive":            "yes",
		"container_generate_entry":   "false",
		"container_clean_path":       "on",
		"container_skip_workdir":     "true",
		"userns_nolimit":             "1",
		"rm_home":                    "yes",
	}

	cfg := toStruct(input)

	assert.Equal(t, "docker", cfg.ContainerManagerType)
	assert.Equal(t, "doas", cfg.SudoProgram)
	assert.True(t, cfg.Verbose)
	// Default* keep distrobox.conf semantics; *_env keys are env-only and
	// stay separate so the create command can distinguish "user has set X"
	// from "X is the hardcoded default".
	assert.Equal(t, "ubuntu:latest", cfg.DefaultContainerImage)
	assert.Equal(t, "mybox", cfg.DefaultContainerName)
	assert.Equal(t, "alpine:3.21", cfg.ContainerImage)
	assert.Equal(t, "envbox", cfg.ContainerName)
	assert.Equal(t, "myhost", cfg.ContainerHostname)
	assert.Equal(t, "/tmp/home", cfg.ContainerCustomHome)
	assert.Equal(t, "/data/boxes", cfg.ContainerHomePrefix)
	assert.True(t, cfg.ContainerAlwaysPull)
	assert.True(t, cfg.NonInteractive)
	assert.False(t, cfg.GenerateEntry)
	assert.True(t, cfg.CleanPath)
	assert.True(t, cfg.SkipWorkDir)
	assert.True(t, cfg.UsernsNoLimit)
	assert.True(t, cfg.RmCustomHome)
}

func TestToStruct_MissingKeys(t *testing.T) {
	cfg := toStruct(map[string]string{})

	assert.Empty(t, cfg.ContainerManagerType)
	assert.Empty(t, cfg.SudoProgram)
	assert.False(t, cfg.Verbose)
	assert.Empty(t, cfg.DefaultContainerImage)
	assert.Empty(t, cfg.DefaultContainerName)
	assert.Empty(t, cfg.ContainerImage)
	assert.Empty(t, cfg.ContainerName)
	assert.Empty(t, cfg.ContainerHostname)
	assert.Empty(t, cfg.ContainerCustomHome)
	assert.Empty(t, cfg.ContainerHomePrefix)
	assert.False(t, cfg.ContainerAlwaysPull)
	assert.False(t, cfg.NonInteractive)
	assert.False(t, cfg.GenerateEntry)
	assert.False(t, cfg.CleanPath)
	assert.False(t, cfg.SkipWorkDir)
	assert.False(t, cfg.UsernsNoLimit)
	assert.False(t, cfg.RmCustomHome)
}

// toBool must accept every truthy form the shell does (`1`, `true`, `yes`,
// `on`) — case-insensitive, surrounding whitespace ignored — and reject
// everything else, including the empty string. Anchors the
// `DBX_VERBOSE=1`/`DBX_USERNS_NOLIMIT=1` fix-in-passing.
func TestToBool(t *testing.T) {
	truthy := []string{"1", "true", "TRUE", "True", "yes", "Yes", "on", "ON", " 1 ", "\ttrue\n"}
	for _, v := range truthy {
		assert.Truef(t, toBool(v), "toBool(%q) should be true", v)
	}
	falsy := []string{"", "0", "false", "FALSE", "no", "off", "nope", "garbage"}
	for _, v := range falsy {
		assert.Falsef(t, toBool(v), "toBool(%q) should be false", v)
	}
}

func TestMergeConfigMaps(t *testing.T) {
	base := map[string]string{
		"container_manager": "podman",
		"verbose":           "false",
	}
	override := map[string]string{
		"container_manager": "docker",
		"container_name":    "mybox",
	}

	merged := mergeConfigMaps(base, override)

	assert.Equal(t, "docker", merged["container_manager"])
	assert.Equal(t, "false", merged["verbose"])
	assert.Equal(t, "mybox", merged["container_name"])
}

func TestMergeConfigMaps_Empty(t *testing.T) {
	merged := mergeConfigMaps()
	assert.Empty(t, merged)
}

func TestReadConfigFile_KeyValues(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "config.conf")
	content := `
container_manager=docker
container_name=mybox
verbose=true
`
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	result, err := readConfigFile(tmpFile)
	require.NoError(t, err)

	assert.Equal(t, "docker", result["container_manager"])
	assert.Equal(t, "mybox", result["container_name"])
	assert.Equal(t, "true", result["verbose"])
}

func TestReadConfigFile_WithComments(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "config.conf")
	content := `
# This is a comment
container_manager=podman # Inline comment
# Another comment
verbose=false
`
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	result, err := readConfigFile(tmpFile)
	require.NoError(t, err)

	assert.Len(t, result, 2)
	assert.Equal(t, "podman", result["container_manager"])
	assert.Equal(t, "false", result["verbose"])
}

func TestReadConfigFile_NotFound(t *testing.T) {
	result, err := readConfigFile(filepath.Join(t.TempDir(), "nonexistent.conf"))
	require.NoError(t, err)
	assert.Empty(t, result)
}
