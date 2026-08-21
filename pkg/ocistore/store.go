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

// Package ocistore is an engine-independent local store for OCI images.
// Images are pulled from remote registries with go-containerregistry and
// kept in an OCI image layout on disk; each stored manifest is annotated
// with the normalized reference it was pulled as, so lookups by reference
// are index-only operations. Blobs are content-addressed, so pulling an
// updated tag only fetches layers not already present.
package ocistore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/match"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

// RefAnnotation is the OCI-standard annotation key used to record which
// reference an image in the layout index was pulled as.
const RefAnnotation = "org.opencontainers.image.ref.name"

// ErrImageNotFound is returned when a reference is not present in the store.
var ErrImageNotFound = errors.New("image not found in local store")

// Store is a local OCI image layout rooted at Dir. The layout (and Dir
// itself) is created lazily on the first Pull.
type Store struct {
	// Dir is the directory holding the OCI image layout.
	Dir string
}

// New returns a Store rooted at dir.
func New(dir string) *Store {
	return &Store{Dir: dir}
}

// NormalizeRef parses ref and returns its fully qualified form
// (e.g. "ubuntu:24.04" -> "index.docker.io/library/ubuntu:24.04").
func NormalizeRef(ref string) (string, error) {
	parsed, err := name.ParseReference(ref)
	if err != nil {
		return "", fmt.Errorf("invalid image reference %q: %w", ref, err)
	}
	return parsed.Name(), nil
}

// Pull fetches the image for ref (optionally for an explicit platform like
// "linux/arm64") and stores it in the layout, replacing any image previously
// stored under the same reference. It returns the image digest.
func (s *Store) Pull(ctx context.Context, ref, platform string) (string, error) {
	parsed, err := name.ParseReference(ref)
	if err != nil {
		return "", fmt.Errorf("invalid image reference %q: %w", ref, err)
	}

	opts := []remote.Option{
		remote.WithContext(ctx),
		remote.WithAuthFromKeychain(authn.DefaultKeychain),
	}
	if platform != "" {
		p, err := v1.ParsePlatform(platform)
		if err != nil {
			return "", fmt.Errorf("invalid platform %q: %w", platform, err)
		}
		opts = append(opts, remote.WithPlatform(*p))
	}

	img, err := remote.Image(parsed, opts...)
	if err != nil {
		return "", fmt.Errorf("cannot fetch image %q: %w", parsed.Name(), err)
	}

	lp, err := s.openOrCreateLayout()
	if err != nil {
		return "", err
	}

	// Drop any previous image stored under this reference, then append the
	// new one annotated with it. ReplaceImage would do both, but matching on
	// annotations directly keeps the ref -> manifest mapping explicit.
	if err := lp.RemoveDescriptors(match.Annotation(RefAnnotation, parsed.Name())); err != nil {
		return "", fmt.Errorf("cannot update image index: %w", err)
	}
	if err := lp.AppendImage(img, layout.WithAnnotations(map[string]string{
		RefAnnotation: parsed.Name(),
	})); err != nil {
		return "", fmt.Errorf("cannot store image %q: %w", parsed.Name(), err)
	}

	digest, err := img.Digest()
	if err != nil {
		return "", fmt.Errorf("cannot compute image digest: %w", err)
	}
	return digest.String(), nil
}

// Exists reports whether ref is present in the store.
func (s *Store) Exists(ref string) bool {
	_, err := s.Image(ref)
	return err == nil
}

// Resolve returns the digest of the image stored under ref, or
// ErrImageNotFound.
func (s *Store) Resolve(ref string) (string, error) {
	img, err := s.Image(ref)
	if err != nil {
		return "", err
	}
	digest, err := img.Digest()
	if err != nil {
		return "", fmt.Errorf("cannot compute image digest: %w", err)
	}
	return digest.String(), nil
}

// Image returns the stored image for ref, or ErrImageNotFound.
func (s *Store) Image(ref string) (v1.Image, error) {
	normalized, err := NormalizeRef(ref)
	if err != nil {
		return nil, err
	}

	lp, err := layout.FromPath(s.layoutDir())
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrImageNotFound, ref)
	}
	index, err := lp.ImageIndex()
	if err != nil {
		return nil, fmt.Errorf("cannot read image index: %w", err)
	}
	manifest, err := index.IndexManifest()
	if err != nil {
		return nil, fmt.Errorf("cannot read image index: %w", err)
	}

	for _, desc := range manifest.Manifests {
		if desc.Annotations[RefAnnotation] != normalized {
			continue
		}
		img, err := index.Image(desc.Digest)
		if err != nil {
			return nil, fmt.Errorf("cannot load image %q: %w", normalized, err)
		}
		return img, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrImageNotFound, ref)
}

// ListRefs returns the references of all stored images.
func (s *Store) ListRefs() ([]string, error) {
	lp, err := layout.FromPath(s.layoutDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, nil
	}
	index, err := lp.ImageIndex()
	if err != nil {
		return nil, fmt.Errorf("cannot read image index: %w", err)
	}
	manifest, err := index.IndexManifest()
	if err != nil {
		return nil, fmt.Errorf("cannot read image index: %w", err)
	}

	refs := make([]string, 0, len(manifest.Manifests))
	for _, desc := range manifest.Manifests {
		if ref, ok := desc.Annotations[RefAnnotation]; ok {
			refs = append(refs, ref)
		}
	}
	return refs, nil
}

func (s *Store) layoutDir() string {
	return filepath.Join(s.Dir, "oci")
}

func (s *Store) openOrCreateLayout() (layout.Path, error) {
	lp, err := layout.FromPath(s.layoutDir())
	if err == nil {
		return lp, nil
	}
	if err := os.MkdirAll(s.Dir, 0o755); err != nil { //nolint:gosec // the store must stay readable for rootful list/inspect by unprivileged users
		return "", fmt.Errorf("cannot create image store directory: %w", err)
	}
	lp, err = layout.Write(s.layoutDir(), empty.Index)
	if err != nil {
		return "", fmt.Errorf("cannot initialize image store: %w", err)
	}
	return lp, nil
}
