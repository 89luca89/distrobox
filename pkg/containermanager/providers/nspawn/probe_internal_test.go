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

package nspawn

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeProber simulates a host: which binaries exist and what systemctl
// answers.
type fakeProber struct {
	missingBinaries map[string]bool
	responses       map[string]string
}

func (f fakeProber) prober() prober {
	return prober{
		lookPath: func(file string) (string, error) {
			if f.missingBinaries[file] {
				return "", errors.New("not found")
			}
			return "/usr/bin/" + file, nil
		},
		runner: func(_ context.Context, argv ...string) (string, error) {
			key := ""
			for _, arg := range argv {
				if key != "" {
					key += " "
				}
				key += arg
			}
			if out, ok := f.responses[key]; ok {
				return out, nil
			}
			return "", errors.New("command failed")
		},
	}
}

func supportedHost() fakeProber {
	return fakeProber{
		responses: map[string]string{
			"systemctl --version":                             "systemd 257 (257.1-arch1)\n+PAM +AUDIT",
			"systemctl is-enabled systemd-mountfsd.socket":    "enabled",
			"systemctl is-enabled systemd-nsresourced.socket": "enabled",
			"systemctl --user is-system-running":              "running",
		},
	}
}

func TestProbeSupportedHost(t *testing.T) {
	assert.NoError(t, runProbe(t.Context(), supportedHost().prober()))
}

func TestProbeOldSystemd(t *testing.T) {
	host := supportedHost()
	host.responses["systemctl --version"] = "systemd 254 (254.5)"

	err := runProbe(t.Context(), host.prober())
	require.ErrorIs(t, err, ErrRootlessUnsupported)
	assert.Contains(t, err.Error(), "systemd 254 found, need >= 257")
	assert.Contains(t, err.Error(), "--root")
}

func TestProbeMissingSockets(t *testing.T) {
	host := supportedHost()
	host.responses["systemctl is-enabled systemd-mountfsd.socket"] = "disabled"
	delete(host.responses, "systemctl is-enabled systemd-nsresourced.socket")

	err := runProbe(t.Context(), host.prober())
	require.ErrorIs(t, err, ErrRootlessUnsupported)
	assert.Contains(t, err.Error(), "systemd-mountfsd.socket: not enabled")
	assert.Contains(t, err.Error(), "systemd-nsresourced.socket: not enabled")
	assert.Contains(t, err.Error(), "systemctl enable --now")
}

func TestProbeActiveSocketsCount(t *testing.T) {
	host := supportedHost()
	// Sockets manually started but not enabled still work for this boot.
	host.responses["systemctl is-enabled systemd-mountfsd.socket"] = "disabled"
	host.responses["systemctl is-active systemd-mountfsd.socket"] = "active"

	assert.NoError(t, runProbe(t.Context(), host.prober()))
}

func TestProbeMissingNspawnBinary(t *testing.T) {
	host := supportedHost()
	host.missingBinaries = map[string]bool{"systemd-nspawn": true}

	err := runProbe(t.Context(), host.prober())
	require.ErrorIs(t, err, ErrRootlessUnsupported)
	assert.Contains(t, err.Error(), "systemd-container")
}

func TestProbeMissingPasta(t *testing.T) {
	host := supportedHost()
	host.missingBinaries = map[string]bool{"pasta": true}

	err := runProbe(t.Context(), host.prober())
	require.ErrorIs(t, err, ErrRootlessUnsupported)
	assert.Contains(t, err.Error(), "install passt")
}

func TestSystemdVersionParsing(t *testing.T) {
	host := supportedHost()

	version, err := systemdVersion(t.Context(), host.prober())
	require.NoError(t, err)
	assert.Equal(t, 257, version)

	host.responses["systemctl --version"] = "garbage"
	_, err = systemdVersion(t.Context(), host.prober())
	assert.Error(t, err)
}
