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

// Package nspawn implements a container manager backed by systemd-nspawn.
// Images are pulled in-process into a local OCI store and flattened into
// per-machine root filesystems; machines run as systemd-run transient units
// wrapping systemd-nspawn. There is no engine-side container database, so
// each machine directory carries its own metadata file.
package nspawn

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// machined machine names must be valid hostnames: at most 64 characters
// from [a-zA-Z0-9-], starting with an alphanumeric.
const maxMachineNameLength = 64

// sanitizeMachineName maps an arbitrary distrobox container name onto a
// machined-acceptable machine name. Names already valid are returned
// unchanged; anything that had to be modified (or truncated) gets a short
// hash of the original appended so distinct container names cannot collide
// after sanitization.
func sanitizeMachineName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_', r == '.':
			b.WriteByte('-')
		}
	}
	sanitized := strings.Trim(collapseDashes(b.String()), "-")

	if sanitized == name && len(sanitized) <= maxMachineNameLength && sanitized != "" {
		return sanitized
	}

	suffix := shortHash(name)
	if sanitized == "" {
		return "distrobox-" + suffix
	}
	maxBase := maxMachineNameLength - len(suffix) - 1
	if len(sanitized) > maxBase {
		sanitized = strings.TrimRight(sanitized[:maxBase], "-")
	}
	return sanitized + "-" + suffix
}

// unitName returns the transient unit name for a sanitized machine name.
func unitName(machineName string) string {
	return "distrobox-" + machineName + ".service"
}

// containerID derives the stable 12-hex-character ID shown in list output
// from the distrobox container name.
func containerID(name string) string {
	sum := sha256.Sum256([]byte(name))
	return hex.EncodeToString(sum[:])[:12]
}

func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:8]
}

func collapseDashes(s string) string {
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return s
}
