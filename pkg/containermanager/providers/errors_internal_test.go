package providers

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsContainerNotFoundError_NilError(t *testing.T) {
	assert.False(t, isContainerNotFoundError(nil))
}

func TestIsContainerNotFoundError_DockerNoSuchObject(t *testing.T) {
	err := errors.New("command execution failed: Error: No such object: foo")
	assert.True(t, isContainerNotFoundError(err))
}

func TestIsContainerNotFoundError_DockerNoSuchContainer(t *testing.T) {
	err := errors.New("command execution failed: Error: No such container: foo")
	assert.True(t, isContainerNotFoundError(err))
}

func TestIsContainerNotFoundError_PodmanInspectingObject(t *testing.T) {
	err := errors.New(`command execution failed: Error: inspecting object: no such container "foo"`)
	assert.True(t, isContainerNotFoundError(err))
}

func TestIsContainerNotFoundError_PodmanNoContainerWithName(t *testing.T) {
	err := errors.New(`command execution failed: Error: no container with name or ID "foo" found: no such container`)
	assert.True(t, isContainerNotFoundError(err))
}

func TestIsContainerNotFoundError_DaemonDown(t *testing.T) {
	err := errors.New("command execution failed: Cannot connect to the Docker daemon at unix:///var/run/docker.sock")
	assert.False(t, isContainerNotFoundError(err))
}

func TestIsContainerNotFoundError_PermissionDenied(t *testing.T) {
	err := errors.New("command execution failed: permission denied while trying to connect to the Docker daemon socket")
	assert.False(t, isContainerNotFoundError(err))
}
