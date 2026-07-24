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

package manifest_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/89luca89/distrobox/pkg/manifest"
)

func TestParse_Simple(t *testing.T) {
	rawManifest := `
[distrodev]
 #comment line
image=ubuntu:24.04
pull=true #comment
init=false

start_now=true
`

	manifestPath := t.TempDir() + "/manifest.ini"
	err := os.WriteFile(manifestPath, []byte(rawManifest), 0o644)
	require.NoError(t, err)

	parsed, err := manifest.Parse(t.Context(), manifestPath)
	require.NoError(t, err)
	assert.NotNil(t, parsed)

	assert.Len(t, parsed, 1)

	assert.Equal(t, "distrodev", parsed[0].Name)
	assert.Equal(t, "ubuntu:24.04", parsed[0].Image)
	assert.True(t, parsed[0].AlwaysPull)
	assert.False(t, parsed[0].Init)
	assert.True(t, parsed[0].StartNow)
}

func TestParse_SkipUnknownKeys(t *testing.T) {
	rawManifest := `
[distrodev]
image=ubuntu:24.04
unknown_key=true
`

	manifestPath := t.TempDir() + "/manifest.ini"
	err := os.WriteFile(manifestPath, []byte(rawManifest), 0o644)
	require.NoError(t, err)

	parsed, err := manifest.Parse(t.Context(), manifestPath)
	require.NoError(t, err)
	assert.NotNil(t, parsed)

	assert.Len(t, parsed, 1)
}

func TestParse_EmptyManifest(t *testing.T) {
	rawManifest := ``

	manifestPath := t.TempDir() + "/manifest.ini"
	err := os.WriteFile(manifestPath, []byte(rawManifest), 0o644)
	require.NoError(t, err)

	parsed, err := manifest.Parse(t.Context(), manifestPath)
	require.NoError(t, err)
	assert.NotNil(t, parsed)

	assert.Empty(t, parsed)
}

func TestParse_InvalidManifest(t *testing.T) {
	rawManifest := `
[distrodev
image=ubuntu:24.04
`

	manifestPath := t.TempDir() + "/manifest.ini"
	err := os.WriteFile(manifestPath, []byte(rawManifest), 0o644)
	require.NoError(t, err)

	parsed, err := manifest.Parse(t.Context(), manifestPath)
	require.Error(t, err)
	assert.Nil(t, parsed)
}

func TestParse_InvalidManifestValueOutsideSection(t *testing.T) {
	rawManifest := `
pull=true
[distrodev]
image=ubuntu:24.04
`

	manifestPath := t.TempDir() + "/manifest.ini"
	err := os.WriteFile(manifestPath, []byte(rawManifest), 0o644)
	require.NoError(t, err)

	parsed, err := manifest.Parse(t.Context(), manifestPath)
	require.Error(t, err)
	assert.Nil(t, parsed)
}

func TestParse_WithInclude(t *testing.T) {
	rawManifest := `
[distrodev]
image=ubuntu:24.04
pull=true
init=false
start_now=true

# Install base packages from apt
additional_packages="shellcheck dash make curl wget python3-pip npm git"

# Install additional tools via init hooks
init_hooks="npm install -g markdownlint-cli"
init_hooks="pip3 install --break-system-packages bashate"
init_hooks="curl -L -o /usr/local/bin/shfmt https://github.com/mvdan/sh/releases/download/v3.8.0/shfmt_v3.8.0_linux_amd64 && chmod +x /usr/local/bin/shfmt"
init_hooks="curl -L -o /usr/local/bin/shell-funcheck https://github.com/89luca89/shell-funcheck/releases/download/v0.0.1/shell-funcheck-amd64 && chmod +x /usr/local/bin/shell-funcheck"

[ubuntu]
image=ubuntu:latest
additional_packages="git vim tmux nodejs"
additional_packages="htop iftop iotop"
additional_packages="zsh fish"

[ubuntu-nvidia]
include=ubuntu # test comments tripping, too
nvidia=true
`

	manifestPath := t.TempDir() + "/manifest.ini"
	err := os.WriteFile(manifestPath, []byte(rawManifest), 0o644)
	require.NoError(t, err)

	parsed, err := manifest.Parse(t.Context(), manifestPath)
	require.NoError(t, err)
	assert.NotNil(t, parsed)

	assert.Len(t, parsed, 3)

	// check ubuntu-nvidia
	ubuntuNvidia := parsed[2]
	assert.Equal(t, "ubuntu-nvidia", ubuntuNvidia.Name)
	assert.Equal(t, "ubuntu:latest", ubuntuNvidia.Image)
	assert.True(t, ubuntuNvidia.Nvidia)

	expectedPackages := []string{
		"git", "vim", "tmux", "nodejs",
		"htop", "iftop", "iotop",
		"zsh", "fish",
	}
	assert.Equal(t, expectedPackages, ubuntuNvidia.AdditionalPackages)

	// check distrodev
	distrodev := parsed[0]
	assert.Equal(t, "distrodev", distrodev.Name)
	assert.Equal(t, "ubuntu:24.04", distrodev.Image)
	assert.True(t, distrodev.AlwaysPull)
	assert.False(t, distrodev.Init)
	assert.True(t, distrodev.StartNow)

	expectedDistrodevPackages := []string{
		"shellcheck", "dash", "make", "curl", "wget", "python3-pip", "npm", "git",
	}
	assert.Equal(t, expectedDistrodevPackages, distrodev.AdditionalPackages)

	expectedDistrodevHooks := []string{
		"npm install -g markdownlint-cli",
		"pip3 install --break-system-packages bashate",
		"curl -L -o /usr/local/bin/shfmt https://github.com/mvdan/sh/releases/download/v3.8.0/shfmt_v3.8.0_linux_amd64 && chmod +x /usr/local/bin/shfmt",
		"curl -L -o /usr/local/bin/shell-funcheck https://github.com/89luca89/shell-funcheck/releases/download/v0.0.1/shell-funcheck-amd64 && chmod +x /usr/local/bin/shell-funcheck",
	}
	assert.Equal(t, expectedDistrodevHooks, distrodev.InitHooks)
}

func TestParse_PreserveIncludeOrder(t *testing.T) {
	rawManifest := `
[ubuntu22]
image=ubuntu:22.04
pull=true
root=true

[ubuntu24]
pull=false # this will be overridden by the include
include=ubuntu22
image=ubuntu:22.04 # this will override the included image
`

	manifestPath := t.TempDir() + "/manifest.ini"
	err := os.WriteFile(manifestPath, []byte(rawManifest), 0o644)
	require.NoError(t, err)

	parsed, err := manifest.Parse(t.Context(), manifestPath)
	require.NoError(t, err)
	assert.NotNil(t, parsed)

	assert.Len(t, parsed, 2)

	// Check ubuntu22
	ubuntu22 := parsed[0]
	assert.Equal(t, "ubuntu22", ubuntu22.Name)
	assert.Equal(t, "ubuntu:22.04", ubuntu22.Image)
	assert.True(t, ubuntu22.AlwaysPull)
	assert.True(t, ubuntu22.Root)

	// Check ubuntu24
	ubuntu24 := parsed[1]
	assert.Equal(t, "ubuntu24", ubuntu24.Name)
	assert.Equal(t, "ubuntu:22.04", ubuntu24.Image)
	assert.True(t, ubuntu24.AlwaysPull)
	assert.True(t, ubuntu24.Root)
}

func TestManifest_ParseFromUrl(t *testing.T) {
	rawManifest := `
[distrodev]
 #comment line
image=ubuntu:24.04
pull=true #comment
init=false

start_now=true
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(rawManifest))
	}))
	defer server.Close()
	fileURL := server.URL + "/manifest.ini" // any file name will do

	parsed, err := manifest.Parse(t.Context(), fileURL)
	require.NoError(t, err)
	assert.NotNil(t, parsed)

	assert.Len(t, parsed, 1)

	assert.Equal(t, "distrodev", parsed[0].Name)
	assert.Equal(t, "ubuntu:24.04", parsed[0].Image)
	assert.True(t, parsed[0].AlwaysPull)
	assert.False(t, parsed[0].Init)
	assert.True(t, parsed[0].StartNow)
}

func TestParse_FromUrlServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	fileURL := server.URL + "/manifest.ini" // any file name will do

	_, err := manifest.Parse(t.Context(), fileURL)
	require.Error(t, err)
}

func TestParse_TrimsSectionNameWhitespace(t *testing.T) {
	rawManifest := `
[ leading]
image=ubuntu:24.04

[trailing ]
image=ubuntu:24.04

[ both ]
image=ubuntu:24.04

[none]
image=ubuntu:24.04
`

	manifestPath := t.TempDir() + "/manifest.ini"
	err := os.WriteFile(manifestPath, []byte(rawManifest), 0o644)
	require.NoError(t, err)

	parsed, err := manifest.Parse(t.Context(), manifestPath)
	require.NoError(t, err)
	require.Len(t, parsed, 4)

	assert.Equal(t, "leading", parsed[0].Name)
	assert.Equal(t, "trailing", parsed[1].Name)
	assert.Equal(t, "both", parsed[2].Name)
	assert.Equal(t, "none", parsed[3].Name)
}

func TestParse_ExampleManifest(t *testing.T) {
	parsed, err := manifest.Parse(t.Context(), "../../extras/distrobox-example-manifest.ini")
	require.NoError(t, err)
	require.Len(t, parsed, 5)

	names := make([]string, len(parsed))
	for i, item := range parsed {
		names[i] = item.Name
	}
	assert.Equal(t,
		[]string{"generic1", "generic2", "generic3", "arch", "tumbleweed_distrobox"},
		names,
	)

	generic1 := parsed[0]
	assert.True(t, generic1.UnshareNetns)
	assert.True(t, generic1.UnshareIPC)
	assert.Empty(t, generic1.Image)

	generic2 := parsed[1]
	assert.True(t, generic2.Nvidia)
	assert.Equal(t, []string{"git", "vim", "tmux"}, generic2.AdditionalPackages)

	generic3 := parsed[2]
	assert.Equal(t, "/tmp/home", generic3.Home)
	assert.Equal(t, []string{"git", "vim", "tmux"}, generic3.AdditionalPackages)

	arch := parsed[3]
	assert.Equal(t, "archlinux:latest", arch.Image)
	assert.True(t, arch.AlwaysPull)
	assert.True(t, arch.StartNow)
	assert.True(t, arch.UnshareNetns)
	assert.Equal(t, "/tmp/home", arch.Home)
	assert.Equal(t, []string{"touch /pre-init"}, arch.PreInitHooks)
	assert.Equal(t, []string{"touch /init-normal"}, arch.InitHooks)

	tw := parsed[4]
	assert.Equal(t, "registry.opensuse.org/opensuse/distrobox", tw.Image)
	assert.True(t, tw.AlwaysPull)
	assert.Equal(t, []string{"htop"}, tw.ExportedApps)
	assert.Equal(t, []string{"/usr/bin/htop", "/usr/bin/git"}, tw.ExportedBins)
}

func TestParse_FailCircularInclude(t *testing.T) {
	rawManifest := `
[a]
include=b
image=img_a

[b]
include=a
image=img_b
`

	manifestPath := t.TempDir() + "/manifest.ini"
	err := os.WriteFile(manifestPath, []byte(rawManifest), 0o644)
	require.NoError(t, err)

	parsed, err := manifest.Parse(t.Context(), manifestPath)
	require.ErrorContains(t, err, "circular include detected: a")
	assert.Nil(t, parsed)
}

func TestParse_FailIndirectCircularInclude(t *testing.T) {
	rawManifest := `
[a]
include=b
image=img_a

[b]
include=c
image=img_b

[c]
include=a
image=img_c
`

	manifestPath := t.TempDir() + "/manifest.ini"
	err := os.WriteFile(manifestPath, []byte(rawManifest), 0o644)
	require.NoError(t, err)

	parsed, err := manifest.Parse(t.Context(), manifestPath)
	require.ErrorContains(t, err, "circular include detected: a")
	assert.Nil(t, parsed)
}
