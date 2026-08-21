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

package ocistore

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

// tarEntry is a declarative tar entry for building test layers.
type tarEntry struct {
	header  tar.Header
	content string
}

func makeLayer(t *testing.T, entries []tarEntry) v1.Layer {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, entry := range entries {
		header := entry.header
		if header.Size == 0 && entry.content != "" {
			header.Size = int64(len(entry.content))
		}
		require.NoError(t, tw.WriteHeader(&header))
		if entry.content != "" {
			_, err := io.WriteString(tw, entry.content)
			require.NoError(t, err)
		}
	}
	require.NoError(t, tw.Close())

	raw := buf.Bytes()
	layer, err := tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(raw)), nil
	})
	require.NoError(t, err)
	return layer
}

func makeImage(t *testing.T, layers ...v1.Layer) v1.Image {
	t.Helper()
	img, err := mutate.AppendLayers(empty.Image, layers...)
	require.NoError(t, err)
	return img
}

// storeImage puts img into a fresh Store under the given ref, bypassing the
// network path of Pull.
func storeImage(t *testing.T, img v1.Image, ref string) *Store {
	t.Helper()
	store := New(t.TempDir())
	normalized, err := NormalizeRef(ref)
	require.NoError(t, err)

	lp, err := layout.Write(store.layoutDir(), empty.Index)
	require.NoError(t, err)
	require.NoError(t, lp.AppendImage(img, layout.WithAnnotations(map[string]string{
		RefAnnotation: normalized,
	})))
	return store
}

func TestNormalizeRef(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"ubuntu:24.04", "index.docker.io/library/ubuntu:24.04"},
		{"ubuntu", "index.docker.io/library/ubuntu:latest"},
		{"registry.fedoraproject.org/fedora-toolbox:latest", "registry.fedoraproject.org/fedora-toolbox:latest"},
		{"quay.io/toolbx/arch-toolbox", "quay.io/toolbx/arch-toolbox:latest"},
	}
	for _, tt := range tests {
		got, err := NormalizeRef(tt.in)
		require.NoError(t, err)
		assert.Equal(t, tt.want, got)
	}
}

func TestNormalizeRefInvalid(t *testing.T) {
	_, err := NormalizeRef("UPPER CASE spaces")
	assert.Error(t, err)
}

func TestExistsAndResolve(t *testing.T) {
	img := makeImage(t, makeLayer(t, []tarEntry{
		{header: tar.Header{Name: "etc/", Typeflag: tar.TypeDir, Mode: 0o755}},
	}))
	store := storeImage(t, img, "example.com/test/image:v1")

	assert.True(t, store.Exists("example.com/test/image:v1"))
	assert.False(t, store.Exists("example.com/test/other:v1"))

	digest, err := store.Resolve("example.com/test/image:v1")
	require.NoError(t, err)
	wantDigest, err := img.Digest()
	require.NoError(t, err)
	assert.Equal(t, wantDigest.String(), digest)

	_, err = store.Resolve("example.com/test/other:v1")
	assert.ErrorIs(t, err, ErrImageNotFound)
}

func TestExistsOnEmptyStoreDir(t *testing.T) {
	store := New(t.TempDir())
	assert.False(t, store.Exists("ubuntu:24.04"))
}

func TestListRefs(t *testing.T) {
	img := makeImage(t, makeLayer(t, []tarEntry{
		{header: tar.Header{Name: "bin/", Typeflag: tar.TypeDir, Mode: 0o755}},
	}))
	store := storeImage(t, img, "example.com/test/image:v1")

	refs, err := store.ListRefs()
	require.NoError(t, err)
	assert.Equal(t, []string{"example.com/test/image:v1"}, refs)

	empty := New(t.TempDir())
	refs, err = empty.ListRefs()
	require.NoError(t, err)
	assert.Empty(t, refs)
}

func TestPullReplacesExistingRef(t *testing.T) {
	// Exercised via the layout primitives Pull uses: append two images under
	// the same ref annotation with RemoveDescriptors in between, then verify
	// only the second remains resolvable.
	imgA := makeImage(t, makeLayer(t, []tarEntry{
		{header: tar.Header{Name: "a", Typeflag: tar.TypeReg, Mode: 0o644}, content: "a"},
	}))
	imgB := makeImage(t, makeLayer(t, []tarEntry{
		{header: tar.Header{Name: "b", Typeflag: tar.TypeReg, Mode: 0o644}, content: "b"},
	}))

	store := storeImage(t, imgA, "example.com/test/image:v1")
	normalized, err := NormalizeRef("example.com/test/image:v1")
	require.NoError(t, err)

	lp, err := layout.FromPath(store.layoutDir())
	require.NoError(t, err)
	require.NoError(t, lp.ReplaceImage(imgB, matchRefAnnotation(normalized), layout.WithAnnotations(map[string]string{
		RefAnnotation: normalized,
	})))

	digest, err := store.Resolve("example.com/test/image:v1")
	require.NoError(t, err)
	wantDigest, err := imgB.Digest()
	require.NoError(t, err)
	assert.Equal(t, wantDigest.String(), digest)

	refs, err := store.ListRefs()
	require.NoError(t, err)
	assert.Len(t, refs, 1)
}

func matchRefAnnotation(ref string) func(desc v1.Descriptor) bool {
	return func(desc v1.Descriptor) bool {
		return desc.Annotations[RefAnnotation] == ref
	}
}

func flattenedRef(t *testing.T, layers ...v1.Layer) (*Store, string) {
	t.Helper()
	const ref = "example.com/test/flatten:v1"
	store := storeImage(t, makeImage(t, layers...), ref)
	return store, ref
}

func TestFlattenBasicTree(t *testing.T) {
	store, ref := flattenedRef(t, makeLayer(t, []tarEntry{
		{header: tar.Header{Name: "etc/", Typeflag: tar.TypeDir, Mode: 0o755}},
		{header: tar.Header{Name: "etc/os-release", Typeflag: tar.TypeReg, Mode: 0o644}, content: "NAME=test\n"},
		{header: tar.Header{Name: "usr/bin/", Typeflag: tar.TypeDir, Mode: 0o755}},
		{header: tar.Header{Name: "usr/bin/sh", Typeflag: tar.TypeReg, Mode: 0o755}, content: "#!/bin/sh\n"},
	}))

	dest := t.TempDir()
	require.NoError(t, store.Flatten(t.Context(), ref, dest, FlattenOptions{}))

	assertFileContent(t, dest+"/etc/os-release", "NAME=test\n")
	assertFileMode(t, dest+"/usr/bin/sh", 0o755)
}

func TestFlattenWhiteoutsApplied(t *testing.T) {
	// Layer 2 deletes a file from layer 1 via a whiteout; mutate.Extract
	// must apply it so the flattened tree does not contain the file.
	store, ref := flattenedRef(t,
		makeLayer(t, []tarEntry{
			{header: tar.Header{Name: "etc/", Typeflag: tar.TypeDir, Mode: 0o755}},
			{header: tar.Header{Name: "etc/deleteme", Typeflag: tar.TypeReg, Mode: 0o644}, content: "x"},
			{header: tar.Header{Name: "etc/keep", Typeflag: tar.TypeReg, Mode: 0o644}, content: "y"},
		}),
		makeLayer(t, []tarEntry{
			{header: tar.Header{Name: "etc/.wh.deleteme", Typeflag: tar.TypeReg, Mode: 0o644}},
		}),
	)

	dest := t.TempDir()
	require.NoError(t, store.Flatten(t.Context(), ref, dest, FlattenOptions{}))

	assert.NoFileExists(t, dest+"/etc/deleteme")
	assert.NoFileExists(t, dest+"/etc/.wh.deleteme")
	assert.FileExists(t, dest+"/etc/keep")
}

func TestFlattenSymlinkAndHardlink(t *testing.T) {
	store, ref := flattenedRef(t, makeLayer(t, []tarEntry{
		{header: tar.Header{Name: "usr/bin/", Typeflag: tar.TypeDir, Mode: 0o755}},
		{header: tar.Header{Name: "usr/bin/busybox", Typeflag: tar.TypeReg, Mode: 0o755}, content: "bb"},
		// Forward hardlink: appears before its target in the stream.
		{header: tar.Header{Name: "usr/bin/early-link", Typeflag: tar.TypeLink, Linkname: "usr/bin/late-target"}},
		{header: tar.Header{Name: "usr/bin/sh", Typeflag: tar.TypeLink, Linkname: "usr/bin/busybox"}},
		{header: tar.Header{Name: "usr/bin/late-target", Typeflag: tar.TypeReg, Mode: 0o644}, content: "late"},
		{header: tar.Header{Name: "usr/bin/dangling", Typeflag: tar.TypeSymlink, Linkname: "/nonexistent"}},
		{header: tar.Header{Name: "usr/bin/relative", Typeflag: tar.TypeSymlink, Linkname: "busybox"}},
	}))

	dest := t.TempDir()
	require.NoError(t, store.Flatten(t.Context(), ref, dest, FlattenOptions{}))

	assertFileContent(t, dest+"/usr/bin/sh", "bb")
	assertFileContent(t, dest+"/usr/bin/early-link", "late")
	assertSymlinkTarget(t, dest+"/usr/bin/dangling", "/nonexistent")
	assertSymlinkTarget(t, dest+"/usr/bin/relative", "busybox")
}

func TestFlattenReadOnlyDirectoryReceivesChildren(t *testing.T) {
	// Fedora's ca-trust directory-hash ships as r-xr-xr-x; its children
	// (symlinks) appear later in the stream. Extraction must keep the
	// directory writable until the end, then restore the exact mode.
	store, ref := flattenedRef(t, makeLayer(t, []tarEntry{
		{header: tar.Header{Name: "etc/hash/", Typeflag: tar.TypeDir, Mode: 0o555}},
		{header: tar.Header{Name: "etc/hash/link.0", Typeflag: tar.TypeSymlink, Linkname: "ca.pem"}},
		{header: tar.Header{Name: "etc/hash/file", Typeflag: tar.TypeReg, Mode: 0o444}, content: "x"},
	}))

	dest := t.TempDir()
	require.NoError(t, store.Flatten(t.Context(), ref, dest, FlattenOptions{}))
	assertSymlinkTarget(t, dest+"/etc/hash/link.0", "ca.pem")
	assertFileContent(t, dest+"/etc/hash/file", "x")
	assertFileMode(t, dest+"/etc/hash", 0o555)

	// Leave the tree removable for TempDir cleanup.
	require.NoError(t, os.Chmod(dest+"/etc/hash", 0o755))
}

func TestFlattenRejectsPathEscape(t *testing.T) {
	store, ref := flattenedRef(t, makeLayer(t, []tarEntry{
		{header: tar.Header{Name: "../../escape", Typeflag: tar.TypeReg, Mode: 0o644}, content: "evil"},
	}))

	// mutate.Extract already drops entries with escaping names; our own
	// securePath guard covers anything that slips through (e.g. hardlink
	// targets). Either way nothing may be written outside the rootfs.
	parent := t.TempDir()
	dest := parent + "/rootfs"
	err := store.Flatten(t.Context(), ref, dest, FlattenOptions{})
	if err != nil {
		assert.Contains(t, err.Error(), "escapes rootfs")
	}
	assert.NoFileExists(t, parent+"/escape")
	assert.NoFileExists(t, parent+"/../escape")
}

func TestSecurePathRejectsEscape(t *testing.T) {
	_, err := securePath("/tmp/rootfs", "../../etc/passwd")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "escapes rootfs")

	inside, err := securePath("/tmp/rootfs", "etc/passwd")
	require.NoError(t, err)
	assert.Equal(t, "/tmp/rootfs/etc/passwd", inside)
}

func TestFlattenSkipsDevicesWithoutAllowDevices(t *testing.T) {
	store, ref := flattenedRef(t, makeLayer(t, []tarEntry{
		{header: tar.Header{Name: "dev/", Typeflag: tar.TypeDir, Mode: 0o755}},
		{header: tar.Header{Name: "dev/null", Typeflag: tar.TypeChar, Mode: 0o666, Devmajor: 1, Devminor: 3}},
	}))

	dest := t.TempDir()
	require.NoError(t, store.Flatten(t.Context(), ref, dest, FlattenOptions{}))
	assert.NoFileExists(t, dest+"/dev/null")
}

func TestFlattenSetuidBitPreserved(t *testing.T) {
	store, ref := flattenedRef(t, makeLayer(t, []tarEntry{
		{header: tar.Header{Name: "usr/bin/", Typeflag: tar.TypeDir, Mode: 0o755}},
		{header: tar.Header{Name: "usr/bin/suid", Typeflag: tar.TypeReg, Mode: 0o4755}, content: "s"},
	}))

	dest := t.TempDir()
	require.NoError(t, store.Flatten(t.Context(), ref, dest, FlattenOptions{}))
	assertFileMode(t, dest+"/usr/bin/suid", 0o4755)
}

func TestFlattenOwnershipSquashedWhenNotPreserving(t *testing.T) {
	store, ref := flattenedRef(t, makeLayer(t, []tarEntry{
		{header: tar.Header{Name: "var/", Typeflag: tar.TypeDir, Mode: 0o755, Uid: 8, Gid: 8}},
		{header: tar.Header{Name: "var/mail", Typeflag: tar.TypeReg, Mode: 0o644, Uid: 8, Gid: 8}, content: "m"},
	}))

	dest := t.TempDir()
	require.NoError(t, store.Flatten(t.Context(), ref, dest, FlattenOptions{PreserveOwnership: false}))
	// Without PreserveOwnership no chown happens: the extracting user owns
	// everything (tests never run as uid 8).
	assertOwnedByCurrentUser(t, dest+"/var/mail")
}

func TestFlattenOverwritesConflictingEntryTypes(t *testing.T) {
	// Layer 2 replaces a symlink with a regular file and a file with a
	// directory; the extractor must replace, not follow or fail.
	store, ref := flattenedRef(t,
		makeLayer(t, []tarEntry{
			{header: tar.Header{Name: "link", Typeflag: tar.TypeSymlink, Linkname: "/etc"}},
			{header: tar.Header{Name: "file", Typeflag: tar.TypeReg, Mode: 0o644}, content: "old"},
		}),
		makeLayer(t, []tarEntry{
			{header: tar.Header{Name: "link", Typeflag: tar.TypeReg, Mode: 0o644}, content: "now-a-file"},
			{header: tar.Header{Name: "file/", Typeflag: tar.TypeDir, Mode: 0o755}},
			{header: tar.Header{Name: "file/child", Typeflag: tar.TypeReg, Mode: 0o644}, content: "c"},
		}),
	)

	dest := t.TempDir()
	require.NoError(t, store.Flatten(t.Context(), ref, dest, FlattenOptions{}))
	assertFileContent(t, dest+"/link", "now-a-file")
	assertFileContent(t, dest+"/file/child", "c")
}

func TestFlattenXattrs(t *testing.T) {
	store, ref := flattenedRef(t, makeLayer(t, []tarEntry{
		{header: tar.Header{
			Name:     "usr/bin/ping",
			Typeflag: tar.TypeReg,
			Mode:     0o755,
			PAXRecords: map[string]string{
				"SCHILY.xattr.user.distrobox.test": "value",
				// Requires privileges; must be skipped, not fatal.
				"SCHILY.xattr.security.capability": "\x01\x00\x00\x02",
			},
		}, content: "p"},
	}))

	dest := t.TempDir()
	require.NoError(t, store.Flatten(t.Context(), ref, dest, FlattenOptions{}))
	assertXattr(t, dest+"/usr/bin/ping", "user.distrobox.test", "value")
}

func TestFlattenUnknownImage(t *testing.T) {
	store := New(t.TempDir())
	err := store.Flatten(t.Context(), "example.com/missing:v1", t.TempDir(), FlattenOptions{})
	assert.ErrorIs(t, err, ErrImageNotFound)
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, want, string(raw))
}

func assertFileMode(t *testing.T, path string, want uint32) {
	t.Helper()
	var stat unix.Stat_t
	require.NoError(t, unix.Lstat(path, &stat))
	assert.Equal(t, fmt.Sprintf("%o", want), fmt.Sprintf("%o", stat.Mode&0o7777))
}

func assertSymlinkTarget(t *testing.T, path, want string) {
	t.Helper()
	target, err := os.Readlink(path)
	require.NoError(t, err)
	assert.Equal(t, want, target)
}

func assertOwnedByCurrentUser(t *testing.T, path string) {
	t.Helper()
	var stat unix.Stat_t
	require.NoError(t, unix.Lstat(path, &stat))
	assert.Equal(t, uint32(os.Getuid()), stat.Uid)
	assert.Equal(t, uint32(os.Getgid()), stat.Gid)
}

func assertXattr(t *testing.T, path, attr, want string) {
	t.Helper()
	buf := make([]byte, 256)
	n, err := unix.Lgetxattr(path, attr, buf)
	if errors.Is(err, unix.ENOTSUP) || errors.Is(err, unix.EOPNOTSUPP) {
		t.Skipf("filesystem does not support xattrs")
	}
	require.NoError(t, err)
	assert.Equal(t, want, string(buf[:n]))
}
