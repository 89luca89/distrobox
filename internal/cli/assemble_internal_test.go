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

package cli

import "testing"

func TestResolveManifestPath_FlagOnly(t *testing.T) {
	got := resolveManifestPath("/etc/distrobox.ini", nil)
	if got != "/etc/distrobox.ini" {
		t.Fatalf("expected flag value, got %q", got)
	}
}

func TestResolveManifestPath_PositionalOnly(t *testing.T) {
	got := resolveManifestPath("", []string{"/srv/manifest.ini"})
	if got != "/srv/manifest.ini" {
		t.Fatalf("expected positional value, got %q", got)
	}
}

func TestResolveManifestPath_FlagTakesPrecedenceOverPositional(t *testing.T) {
	got := resolveManifestPath("/etc/distrobox.ini", []string{"/srv/manifest.ini"})
	if got != "/etc/distrobox.ini" {
		t.Fatalf("expected flag value to win, got %q", got)
	}
}

func TestResolveManifestPath_NoInputsReturnsDefault(t *testing.T) {
	got := resolveManifestPath("", nil)
	if got != defaultManifestPath {
		t.Fatalf("expected default %q, got %q", defaultManifestPath, got)
	}
}

func TestResolveManifestPath_EmptyPositionalSliceReturnsDefault(t *testing.T) {
	got := resolveManifestPath("", []string{})
	if got != defaultManifestPath {
		t.Fatalf("expected default %q, got %q", defaultManifestPath, got)
	}
}

func TestResolveManifestPath_EmptyFirstPositionalReturnsDefault(t *testing.T) {
	got := resolveManifestPath("", []string{""})
	if got != defaultManifestPath {
		t.Fatalf("expected default %q, got %q", defaultManifestPath, got)
	}
}

func TestResolveManifestPath_FirstPositionalUsedWhenMultiple(t *testing.T) {
	got := resolveManifestPath("", []string{"/first.ini", "/second.ini"})
	if got != "/first.ini" {
		t.Fatalf("expected first positional, got %q", got)
	}
}

func TestResolveManifestPath_URLFlagValue(t *testing.T) {
	got := resolveManifestPath("https://example.com/manifest.ini", nil)
	if got != "https://example.com/manifest.ini" {
		t.Fatalf("expected URL flag value, got %q", got)
	}
}

func TestResolveManifestPath_URLPositionalValue(t *testing.T) {
	got := resolveManifestPath("", []string{"https://example.com/manifest.ini"})
	if got != "https://example.com/manifest.ini" {
		t.Fatalf("expected URL positional value, got %q", got)
	}
}
