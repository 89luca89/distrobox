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

package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/89luca89/distrobox/pkg/config"
)

func TestDefaultConfigValues(t *testing.T) {
	cfg := config.DefaultValues()

	assert.Equal(t, "autodetect", cfg.ContainerManagerType)
	assert.Equal(t, "sudo", cfg.SudoProgram)
	assert.False(t, cfg.Verbose)
	assert.Equal(t, "registry.fedoraproject.org/fedora-toolbox:latest", cfg.DefaultContainerImage)
	assert.Equal(t, "my-distrobox", cfg.DefaultContainerName)
}
