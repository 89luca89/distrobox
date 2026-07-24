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

package commands

import "strings"

// distroIconBaseURL is the upstream raw asset root for the bundled distro
// logos shipped with the distrobox project.
const distroIconBaseURL = "https://raw.githubusercontent.com/89luca89/distrobox/main/docs/assets/png/distros/"

// distroIconEntry maps a substring expected to appear in a container image
// or container name to the matching desktop icon URL.
//
// The list is ordered from most specific to most generic so that, when
// scanning a hint string for the first matching key, more specific variants
// (e.g. opensuse-tumbleweed, kdeneon) win over short, ambiguous prefixes.
type distroIconEntry struct {
	key string
	url string
}

// distroIconMap mirrors the DISTRO_ICON_MAP from the upstream
// distrobox-generate-entry shell script. Keep entries in sync when new
// distro icons are added under docs/assets/png/distros.
//
//nolint:gochecknoglobals // static lookup table, behaves like a constant
var distroIconMap = []distroIconEntry{
	{"opensuse-tumbleweed", distroIconBaseURL + "opensuse-distrobox.png"},
	{"opensuse-leap", distroIconBaseURL + "opensuse-distrobox.png"},
	{"opensuse", distroIconBaseURL + "opensuse-distrobox.png"},
	{"tumbleweed", distroIconBaseURL + "opensuse-distrobox.png"},
	{"kdeneon", distroIconBaseURL + "kdeneon-distrobox.png"},
	{"archlinux", distroIconBaseURL + "arch-distrobox.png"},
	{"alpinelinux", distroIconBaseURL + "alpine-distrobox.png"},
	{"kalilinux", distroIconBaseURL + "kali-distrobox.png"},
	{"clear", distroIconBaseURL + "clear-distrobox.png"},
	{"alma", distroIconBaseURL + "alma-distrobox.png"},
	{"alpine", distroIconBaseURL + "alpine-distrobox.png"},
	{"alt", distroIconBaseURL + "alt-distrobox.png"},
	{"arch", distroIconBaseURL + "arch-distrobox.png"},
	{"centos", distroIconBaseURL + "centos-distrobox.png"},
	{"debian", distroIconBaseURL + "debian-distrobox.png"},
	{"deepin", distroIconBaseURL + "deepin-distrobox.png"},
	{"fedora", distroIconBaseURL + "fedora-distrobox.png"},
	{"gentoo", distroIconBaseURL + "gentoo-distrobox.png"},
	{"kali", distroIconBaseURL + "kali-distrobox.png"},
	{"rhel", distroIconBaseURL + "redhat-distrobox.png"},
	{"redhat", distroIconBaseURL + "redhat-distrobox.png"},
	{"rocky", distroIconBaseURL + "rocky-distrobox.png"},
	{"ubuntu", distroIconBaseURL + "ubuntu-distrobox.png"},
	{"vanilla", distroIconBaseURL + "vanilla-distrobox.png"},
	{"void", distroIconBaseURL + "void-distrobox.png"},
}

// lookupDistroIcon returns the icon URL for the first distro key found as a
// substring of hint (case-insensitive). When no key matches it returns the
// empty string so callers can apply their own fallback.
func lookupDistroIcon(hint string) string {
	if hint == "" {
		return ""
	}
	lower := strings.ToLower(hint)
	for _, entry := range distroIconMap {
		if strings.Contains(lower, entry.key) {
			return entry.url
		}
	}
	return ""
}
