package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/89luca89/distrobox/internal/config"
)

func TestLoadConfig_NoConfigFiles(t *testing.T) {
	// LoadConfig should not error when no config files exist
	err := config.LoadConfig()
	assert.NoError(t, err, "LoadConfig() should not error when no config files exist")
}

func TestLoadConfig_LoadsEnvFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, ".distroboxrc")

	content := []byte("TEST_VAR=hello_from_config\n")
	err := os.WriteFile(configFile, content, 0o644)
	require.NoError(t, err, "failed to write temp config")

	t.Setenv("HOME", tmpDir)
	os.Unsetenv("TEST_VAR")

	err = config.LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, "hello_from_config", os.Getenv("TEST_VAR"))
}

func TestLoadConfig_EnvVarOverridesFile(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, ".distroboxrc")

	content := []byte("TEST_VAR=hello_from_config\n")
	err := os.WriteFile(configFile, content, 0o644)
	require.NoError(t, err, "failed to write temp config")

	t.Setenv("HOME", tmpDir)
	t.Setenv("TEST_VAR", "hello_from_env")

	err = config.LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, "hello_from_env", os.Getenv("TEST_VAR"))
}
