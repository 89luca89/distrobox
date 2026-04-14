package providers

import (
	"errors"
	"os/exec"

	"github.com/89luca89/distrobox/pkg/containermanager"
)

// ErrNoContainerManager is returned when no supported container runtime is found.
var ErrNoContainerManager = errors.New("no container manager found")

// NewAutoDetect returns a ContainerManager for the first available container runtime.
// Priority order: podman > podman-launcher > docker.
func NewAutoDetect(root bool, sudoCommand string, verbose bool) (containermanager.ContainerManager, error) {
	if _, err := exec.LookPath("podman"); err == nil {
		return NewPodman(root, sudoCommand, verbose), nil
	}
	if _, err := exec.LookPath("podman-launcher"); err == nil {
		return NewPodmanLauncher(root, sudoCommand, verbose), nil
	}
	if _, err := exec.LookPath("docker"); err == nil {
		return NewDocker(root, sudoCommand, verbose), nil
	}
	return nil, ErrNoContainerManager
}
