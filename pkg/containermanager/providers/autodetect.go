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
func NewAutoDetect(root bool, sudoCommand string, verbose, usernsNoLimit bool) (containermanager.ContainerManager, error) {
	if _, err := exec.LookPath("podman"); err == nil {
		return NewPodman(root, sudoCommand, verbose, usernsNoLimit), nil
	}
	if _, err := exec.LookPath("podman-launcher"); err == nil {
		return NewPodmanLauncher(root, sudoCommand, verbose, usernsNoLimit), nil
	}
	if _, err := exec.LookPath("docker"); err == nil {
		return NewDocker(root, sudoCommand, verbose), nil
	}
	return nil, ErrNoContainerManager
}
