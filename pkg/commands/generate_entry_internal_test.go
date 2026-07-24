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

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newIconTestCmd(fetch func(context.Context, string, string) error) *GenerateEntryCommand {
	c := NewGenerateEntryCommand(nil, nil)
	if fetch != nil {
		c.fetchIcon = fetch
	}
	return c
}

// fakeIconWriter simulates a successful download by writing bytes to dest.
func fakeIconWriter(t *testing.T) func(context.Context, string, string) error {
	t.Helper()
	return func(_ context.Context, _ string, dest string) error {
		if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
			return err
		}
		return os.WriteFile(dest, []byte("png"), 0o644)
	}
}

func TestResolveIcon_ExplicitIconPassthrough(t *testing.T) {
	c := newIconTestCmd(nil)
	got := c.resolveIcon(context.Background(), "custom-url", "fedora", t.TempDir())
	assert.Equal(t, "custom-url", got)
}

func TestResolveIcon_AutoDownloadsDistroIconOnce(t *testing.T) {
	dir := t.TempDir()
	calls := 0
	c := newIconTestCmd(func(_ context.Context, _ string, dest string) error {
		calls++
		require.NoError(t, os.MkdirAll(filepath.Dir(dest), 0o750))
		return os.WriteFile(dest, []byte("png"), 0o644)
	})

	want := filepath.Join(dir, "distrobox", "fedora-distrobox.png")

	got := c.resolveIcon(context.Background(), "auto", "fedora-39", dir)
	assert.Equal(t, want, got)
	assert.FileExists(t, want)
	assert.Equal(t, 1, calls)

	// Second resolution is a cache hit: no second download.
	got2 := c.resolveIcon(context.Background(), "auto", "fedora-39", dir)
	assert.Equal(t, want, got2)
	assert.Equal(t, 1, calls, "cached icon must not be re-downloaded")
}

func TestResolveIcon_DownloadFailureFallsBackWithoutClobber(t *testing.T) {
	dir := t.TempDir()
	c := newIconTestCmd(func(_ context.Context, _ string, _ string) error {
		return errors.New("offline")
	})

	got := c.resolveIcon(context.Background(), "auto", "ubuntu-22", dir)
	assert.Equal(t, fallbackIconName, got)
	assert.NoFileExists(t, filepath.Join(dir, "distrobox", "ubuntu-distrobox.png"))
}

func TestResolveIcon_CacheHitReusedOffline(t *testing.T) {
	dir := t.TempDir()
	cached := filepath.Join(dir, "distrobox", "arch-distrobox.png")
	require.NoError(t, os.MkdirAll(filepath.Dir(cached), 0o750))
	require.NoError(t, os.WriteFile(cached, []byte("png"), 0o644))

	c := newIconTestCmd(func(_ context.Context, _ string, _ string) error {
		t.Fatal("must not download when a cached icon exists")
		return nil
	})

	got := c.resolveIcon(context.Background(), "auto", "docker.io/library/archlinux:latest", dir)
	assert.Equal(t, cached, got)
}

func TestResolveIcon_UnknownDistroUsesFallback(t *testing.T) {
	c := newIconTestCmd(func(_ context.Context, _ string, _ string) error {
		t.Fatal("must not download for an unknown distro")
		return nil
	})

	assert.Equal(t, fallbackIconName, c.resolveIcon(context.Background(), "auto", "unknown-distro", t.TempDir()))
	assert.Equal(t, fallbackIconName, c.resolveIcon(context.Background(), "auto", "", t.TempDir()))
}

func TestResolveIcon_AutoDetectsCaseInsensitiveAndSpecific(t *testing.T) {
	dir := t.TempDir()
	c := newIconTestCmd(fakeIconWriter(t))

	assert.Equal(t, filepath.Join(dir, "distrobox", "fedora-distrobox.png"),
		c.resolveIcon(context.Background(), "auto", "Fedora-Toolbox", dir))
	// opensuse-tumbleweed must match the specific variant, not a generic miss.
	assert.Equal(t, filepath.Join(dir, "distrobox", "opensuse-distrobox.png"),
		c.resolveIcon(context.Background(), "auto", "registry.opensuse.org/opensuse/tumbleweed:latest", dir))
}
