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
		"container_manager": "docker",
		"sudo_program":      "doas",
		"verbose":           "true",
		"container_image":   "ubuntu:latest",
		"container_name":    "mybox",
	}

	cfg := toStruct(input)

	assert.Equal(t, "docker", cfg.ContainerManagerType)
	assert.Equal(t, "doas", cfg.SudoProgram)
	assert.True(t, cfg.Verbose)
	assert.Equal(t, "ubuntu:latest", cfg.DefaultContainerImage)
	assert.Equal(t, "mybox", cfg.DefaultContainerName)
}

func TestToStruct_MissingKeys(t *testing.T) {
	cfg := toStruct(map[string]string{})

	assert.Empty(t, cfg.ContainerManagerType)
	assert.Empty(t, cfg.SudoProgram)
	assert.False(t, cfg.Verbose)
	assert.Empty(t, cfg.DefaultContainerImage)
	assert.Empty(t, cfg.DefaultContainerName)
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
