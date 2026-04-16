package providers_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/89luca89/distrobox/pkg/containermanager/providers"
)

func TestNewAutoDetect(t *testing.T) {
	tests := []struct {
		name         string
		binaries     []string
		expectedName string
		expectError  bool
	}{
		{
			name:         "podman only",
			binaries:     []string{"podman"},
			expectedName: "podman",
		},
		{
			name:         "docker only",
			binaries:     []string{"docker"},
			expectedName: "docker",
		},
		{
			name:         "both present",
			binaries:     []string{"podman", "docker"},
			expectedName: "podman",
		},
		{
			name:         "podman-launcher only",
			binaries:     []string{"podman-launcher"},
			expectedName: "podman-launcher",
		},
		{
			name:        "nothing available",
			binaries:    []string{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, bin := range tt.binaries {
				err := os.WriteFile(filepath.Join(dir, bin), []byte("#!/bin/sh\n"), 0o755)
				require.NoError(t, err)
			}
			t.Setenv("PATH", dir)

			cm, err := providers.NewAutoDetect(false, "sudo", false)
			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, cm)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedName, cm.Name())
			}
		})
	}
}
