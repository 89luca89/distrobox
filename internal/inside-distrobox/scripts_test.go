package insidedistrobox_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	insidedistrobox "github.com/89luca89/distrobox/internal/inside-distrobox"
)

func TestProvisionScripts_CustomDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DBX_SCRIPTS_DIR", tmpDir)

	scriptsDir, err := insidedistrobox.ProvisionScripts()
	require.NoError(t, err, "ProvisionScripts failed")
	defer os.RemoveAll(scriptsDir)

	require.Equal(t, tmpDir, scriptsDir)

	expectedScripts := []string{
		"distrobox-host-exec",
		"distrobox-init",
		"distrobox-export",
	}

	for _, scriptName := range expectedScripts {
		scriptPath := filepath.Join(scriptsDir, scriptName)
		assert.FileExists(t, scriptPath, "Expected script %s to exist", scriptName)
	}
}

func TestProvisionScripts_HomeDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	scriptsDir, err := insidedistrobox.ProvisionScripts()
	require.NoError(t, err, "ProvisionScripts failed")
	defer os.RemoveAll(scriptsDir)

	expected := filepath.Join(tmpDir, ".local", "share", "distrobox", "v2")
	require.Equal(t, expected, scriptsDir)

	expectedScripts := []string{
		"distrobox-host-exec",
		"distrobox-init",
		"distrobox-export",
	}

	for _, scriptName := range expectedScripts {
		scriptPath := filepath.Join(scriptsDir, scriptName)
		assert.FileExists(t, scriptPath, "Expected script %s to exist", scriptName)
	}
}
