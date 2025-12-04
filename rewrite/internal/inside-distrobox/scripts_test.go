package insidedistrobox_test

import (
	"os"
	"path/filepath"
	"testing"

	insidedistrobox "github.com/89luca89/distrobox/internal/inside-distrobox"
)

func TestProvisionScripts_CustomDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DBX_SCRIPTS_DIR", tmpDir)

	scriptsDir, err := insidedistrobox.ProvisionScripts()
	if err != nil {
		t.Fatalf("ProvisionScripts failed: %v", err)
	}
	defer os.RemoveAll(scriptsDir)

	if scriptsDir != tmpDir {
		t.Fatalf("Expected scripts directory to be %s, but got %s", tmpDir, scriptsDir)
	}

	expectedScripts := []string{
		"distrobox-host-exec",
		"distrobox-init",
		"distrobox-export",
	}

	for _, scriptName := range expectedScripts {
		scriptPath := filepath.Join(scriptsDir, scriptName)
		_, err := os.Stat(scriptPath)
		if err != nil {
			t.Errorf("Expected script %s to exist, but got error: %v", scriptName, err)
		}
	}
}

func TestProvisionScripts_HomeDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	scriptsDir, err := insidedistrobox.ProvisionScripts()
	if err != nil {
		t.Fatalf("ProvisionScripts failed: %v", err)
	}
	defer os.RemoveAll(scriptsDir)

	expected := filepath.Join(tmpDir, ".local", "share", "distrobox", "v2")
	if scriptsDir != expected {
		t.Fatalf("Expected scripts directory to be %s, but got %s", expected, scriptsDir)
	}

	expectedScripts := []string{
		"distrobox-host-exec",
		"distrobox-init",
		"distrobox-export",
	}

	for _, scriptName := range expectedScripts {
		scriptPath := filepath.Join(scriptsDir, scriptName)
		_, err := os.Stat(scriptPath)
		if err != nil {
			t.Errorf("Expected script %s to exist, but got error: %v", scriptName, err)
		}
	}
}
