package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/89luca89/distrobox/pkg/config"
)

func TestDefaultConfigValues(t *testing.T) {
	cfg := config.DefaultValues()

	assert.Equal(t, "podman", cfg.ContainerManagerType)
	assert.Equal(t, "sudo", cfg.SudoProgram)
	assert.False(t, cfg.Verbose)
	assert.Equal(t, "registry.fedoraproject.org/fedora-toolbox:latest", cfg.DefaultContainerImage)
	assert.Equal(t, "my-distrobox", cfg.DefaultContainerName)
}
