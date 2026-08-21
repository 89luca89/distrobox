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
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"golang.org/x/sys/unix"
)

// FlattenOptions controls how the flattened image filesystem is written to
// disk.
type FlattenOptions struct {
	// PreserveOwnership chowns every entry to the uid/gid recorded in the
	// image. Requires running as root; when false every entry is owned by
	// the invoking user (the rootless/unprivileged-nspawn model).
	PreserveOwnership bool
	// AllowDevices creates character/block device nodes recorded in the
	// image. Requires running as root; when false device entries are
	// skipped.
	AllowDevices bool
}

// Flatten extracts the image stored under ref into destDir as a root
// filesystem. Layer whiteouts are already applied by mutate.Extract, so the
// stream contains only the files of the final merged filesystem.
func (s *Store) Flatten(ctx context.Context, ref, destDir string, opts FlattenOptions) error {
	img, err := s.Image(ref)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil { //nolint:gosec // rootfs directories need standard world-traversable permissions
		return fmt.Errorf("cannot create rootfs directory: %w", err)
	}

	reader := mutate.Extract(img)
	defer reader.Close()

	return extractTar(ctx, tar.NewReader(reader), destDir, opts)
}

// deferredMeta is metadata applied after all entries are written, so that
// e.g. extracting files inside a read-only directory works and directory
// mtimes are not clobbered by their children.
type deferredMeta struct {
	path   string
	header *tar.Header
}

//nolint:gocognit // sequential per-entry-type dispatch; splitting further would obscure the extraction order invariants
func extractTar(ctx context.Context, tr *tar.Reader, destDir string, opts FlattenOptions) error {
	var dirs []deferredMeta
	var hardlinks []deferredMeta

	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("extraction interrupted: %w", err)
		}
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("cannot read image content: %w", err)
		}

		target, err := securePath(destDir, header.Name)
		if err != nil {
			return err
		}
		if target == destDir {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := extractDir(target, header); err != nil {
				return err
			}
			dirs = append(dirs, deferredMeta{path: target, header: header})
			continue
		case tar.TypeLink:
			// Targets may appear later in the stream; resolve at the end.
			hardlinks = append(hardlinks, deferredMeta{path: target, header: header})
			continue
		default:
			extracted, err := extractEntry(target, header, tr, opts)
			if err != nil {
				return err
			}
			if !extracted {
				continue
			}
		}

		if err := applyMeta(target, header, opts); err != nil {
			return err
		}
	}

	for _, link := range hardlinks {
		source, err := securePath(destDir, link.header.Linkname)
		if err != nil {
			return err
		}
		if err := replaceEntry(link.path, func() error {
			return os.Link(source, link.path)
		}); err != nil {
			return fmt.Errorf("cannot create hardlink %s: %w", link.header.Name, err)
		}
	}

	// Children are all in place: apply directory metadata deepest-first so
	// parent mtime/mode fixups do not disturb already-fixed children.
	for i := len(dirs) - 1; i >= 0; i-- {
		if err := applyMeta(dirs[i].path, dirs[i].header, opts); err != nil {
			return err
		}
	}
	return nil
}

// extractEntry writes one non-directory, non-hardlink tar entry to disk.
// The boolean reports whether anything was created (device nodes may be
// skipped).
func extractEntry(target string, header *tar.Header, content io.Reader, opts FlattenOptions) (bool, error) {
	switch header.Typeflag {
	case tar.TypeReg:
		return true, extractFile(target, header, content)
	case tar.TypeSymlink:
		// Created verbatim and never followed by the extractor, so a link
		// pointing outside the rootfs is inert until resolved inside the
		// container, where it cannot escape.
		if err := replaceEntry(target, func() error {
			return os.Symlink(header.Linkname, target)
		}); err != nil {
			return false, fmt.Errorf("cannot create symlink %s: %w", header.Name, err)
		}
		return true, nil
	case tar.TypeFifo:
		if err := replaceEntry(target, func() error {
			return unix.Mkfifo(target, uint32(header.Mode&0o7777)) //nolint:gosec // masked to permission bits
		}); err != nil {
			return false, fmt.Errorf("cannot create fifo %s: %w", header.Name, err)
		}
		return true, nil
	case tar.TypeChar, tar.TypeBlock:
		if !opts.AllowDevices {
			return false, nil
		}
		return true, extractDevice(target, header)
	default:
		return false, nil
	}
}

// securePath resolves a tar entry name inside destDir, rejecting entries
// that would escape it.
func securePath(destDir, entryName string) (string, error) {
	cleaned := filepath.Clean(filepath.Join(destDir, entryName))
	if cleaned != destDir && !strings.HasPrefix(cleaned, destDir+string(os.PathSeparator)) {
		return "", fmt.Errorf("image content escapes rootfs: %q", entryName)
	}
	return cleaned, nil
}

// replaceEntry removes whatever occupies target (never following symlinks)
// and runs create. The parent directory is guaranteed to exist because tar
// streams list directories before their content; guard anyway for layers
// with missing parents.
func replaceEntry(target string, create func() error) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil { //nolint:gosec // rootfs directories need standard permissions
		return fmt.Errorf("cannot create parent directory: %w", err)
	}
	if _, err := os.Lstat(target); err == nil {
		if err := os.RemoveAll(target); err != nil {
			return fmt.Errorf("cannot replace existing entry: %w", err)
		}
	}
	return create()
}

func extractDir(target string, header *tar.Header) error {
	info, err := os.Lstat(target)
	if err == nil && !info.IsDir() {
		if err := os.RemoveAll(target); err != nil {
			return fmt.Errorf("cannot replace %s with directory: %w", header.Name, err)
		}
	}
	// Keep directories owner-writable during extraction: read-only
	// directories (e.g. Fedora's ca-trust directory-hash, mode r-x) would
	// reject their own children otherwise. The deferred metadata pass
	// applies the exact recorded mode once all children are in place.
	mode := os.FileMode(header.Mode&0o7777) | 0o700 //nolint:gosec // masked to permission bits
	if err := os.MkdirAll(target, mode); err != nil {
		return fmt.Errorf("cannot create directory %s: %w", header.Name, err)
	}
	// MkdirAll neither updates modes of pre-existing directories nor
	// escapes the umask; chmod explicitly.
	if err := os.Chmod(target, mode); err != nil {
		return fmt.Errorf("cannot create directory %s: %w", header.Name, err)
	}
	return nil
}

func extractFile(target string, header *tar.Header, content io.Reader) error {
	err := replaceEntry(target, func() error {
		file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, os.FileMode(header.Mode&0o7777)) //nolint:gosec // masked to permission bits
		if err != nil {
			return fmt.Errorf("cannot open file: %w", err)
		}
		defer file.Close()
		if _, err := io.Copy(file, content); err != nil {
			return fmt.Errorf("cannot write file content: %w", err)
		}
		return file.Close()
	})
	if err != nil {
		return fmt.Errorf("cannot extract %s: %w", header.Name, err)
	}
	return nil
}

func extractDevice(target string, header *tar.Header) error {
	mode := uint32(header.Mode & 0o7777) //nolint:gosec // masked to permission bits
	if header.Typeflag == tar.TypeChar {
		mode |= unix.S_IFCHR
	} else {
		mode |= unix.S_IFBLK
	}
	//nolint:gosec // tar device numbers are far below any overflow boundary
	dev := unix.Mkdev(uint32(header.Devmajor), uint32(header.Devminor))
	if err := replaceEntry(target, func() error {
		return unix.Mknod(target, mode, int(dev)) //nolint:gosec // dev fits in int on linux
	}); err != nil {
		return fmt.Errorf("cannot create device %s: %w", header.Name, err)
	}
	return nil
}

// applyMeta sets ownership, mode (incl. setuid/setgid/sticky), xattrs and
// times on an extracted entry.
func applyMeta(target string, header *tar.Header, opts FlattenOptions) error {
	if opts.PreserveOwnership {
		if err := os.Lchown(target, header.Uid, header.Gid); err != nil {
			return fmt.Errorf("cannot chown %s: %w", header.Name, err)
		}
	}

	// Symlink modes are meaningless on Linux and Chmod would follow the
	// link; ownership above is the only metadata symlinks carry.
	if header.Typeflag != tar.TypeSymlink {
		// Chmod after chown: chown clears setuid/setgid bits. FileInfo()
		// maps tar permission bits (0o4000 etc.) onto os.FileMode flags.
		mode := header.FileInfo().Mode() & (os.ModePerm | os.ModeSetuid | os.ModeSetgid | os.ModeSticky)
		if err := os.Chmod(target, mode); err != nil {
			return fmt.Errorf("cannot chmod %s: %w", header.Name, err)
		}
	}

	if err := applyXattrs(target, header); err != nil {
		return err
	}

	if header.Typeflag != tar.TypeSymlink && !header.ModTime.IsZero() {
		mtime := header.ModTime
		atime := header.AccessTime
		if atime.IsZero() {
			atime = mtime
		}
		if err := os.Chtimes(target, atime, mtime); err != nil {
			return fmt.Errorf("cannot set times on %s: %w", header.Name, err)
		}
	}
	return nil
}

// applyXattrs restores the extended attributes recorded in the entry's PAX
// records.
func applyXattrs(target string, header *tar.Header) error {
	for key, value := range header.PAXRecords {
		attr, ok := strings.CutPrefix(key, "SCHILY.xattr.")
		if !ok {
			continue
		}
		if err := unix.Lsetxattr(target, attr, []byte(value), 0); err != nil {
			// Only root can set security.* / trusted.* attributes (and the
			// kernel validates their values); the filesystem may not
			// support xattrs at all. Neither is fatal for a dev-sandbox
			// rootfs, but a failing user.* attribute is a real error.
			if !strings.HasPrefix(attr, "user.") ||
				errors.Is(err, unix.ENOTSUP) || errors.Is(err, unix.EOPNOTSUPP) {
				continue
			}
			return fmt.Errorf("cannot set xattr %s on %s: %w", attr, header.Name, err)
		}
	}
	return nil
}
