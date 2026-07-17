package insidedistrobox_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	insidedistrobox "github.com/89luca89/distrobox/internal/inside-distrobox"
)

// expectedScripts is the canonical triad of in-container helper scripts.
// ProvisionScripts must always end with all three present in the returned
// directory, regardless of which resolution branch produced it.
//
//nolint:gochecknoglobals // shared fixture across the suite, behaves like a constant
var expectedScripts = []string{
	"distrobox-host-exec",
	"distrobox-init",
	"distrobox-export",
}

func assertAllScripts(t *testing.T, dir string) {
	t.Helper()
	for _, name := range expectedScripts {
		assert.FileExists(t, filepath.Join(dir, name), "expected %s in %s", name, dir)
	}
}

// TestProvisionScripts_CustomDir checks the DBX_SCRIPTS_DIR override:
// when set to an empty directory, ProvisionScripts writes all three
// scripts there and returns that directory.
func TestProvisionScripts_CustomDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DBX_SCRIPTS_DIR", tmpDir)

	dir, err := insidedistrobox.ProvisionScripts()
	require.NoError(t, err)
	require.Equal(t, tmpDir, dir)
	assertAllScripts(t, dir)
}

// TestProvisionScripts_ExtractsAdjacentToBinary verifies the default
// resolution: with no DBX_SCRIPTS_DIR override,
// ProvisionScripts writes to the directory containing the running
// binary. That is the layout a fresh `go install` or curl-only deploy
// produces.
func TestProvisionScripts_ExtractsAdjacentToBinary(t *testing.T) {
	t.Setenv("DBX_SCRIPTS_DIR", "")
	t.Setenv("HOME", t.TempDir())

	exe, err := os.Executable()
	require.NoError(t, err)
	exeDir := filepath.Dir(exe)
	if !isDirWritable(exeDir) {
		t.Skipf("binary-adjacent dir %s is not writable; cannot exercise this branch here", exeDir)
	}
	t.Cleanup(func() {
		for _, name := range expectedScripts {
			_ = os.Remove(filepath.Join(exeDir, name))
		}
	})

	dir, err := insidedistrobox.ProvisionScripts()
	require.NoError(t, err)
	require.Equal(t, exeDir, dir)
	assertAllScripts(t, dir)
}

// isDirWritable does a probing-write into dir to determine whether a
// non-root user can create files there. Used by the extraction-adjacent
// test as a defensive skip when the binary's directory isn't writable
// (e.g. read-only test harness, distros with hardened tmpfs).
func isDirWritable(dir string) bool {
	probe, err := os.CreateTemp(dir, ".dbx-write-probe-")
	if err != nil {
		return false
	}
	defer os.Remove(probe.Name())
	defer probe.Close()
	return true
}
