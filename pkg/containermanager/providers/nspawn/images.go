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
	"fmt"
	"os"
	"strings"

	"github.com/89luca89/distrobox/pkg/ocistore"
)

// ImageExists reports whether the image is present in the local OCI store.
func (n *Nspawn) ImageExists(ctx context.Context, imageName string) bool {
	if n.root {
		// The rootful store may not be readable by the invoking user.
		if os.Geteuid() != 0 {
			return n.runHelper(ctx, nil, "image-exists", "--image", imageName) == nil
		}
	}
	return n.store().Exists(imageName)
}

// PullImage fetches the image from its registry into the local OCI store.
// Unlike podman/docker there is no engine to shell out to: the pull runs
// in-process (or in a root re-exec of ourselves for the rootful store).
func (n *Nspawn) PullImage(ctx context.Context, imageName string, platform string, dryRun bool) error {
	if dryRun {
		command := fmt.Sprintf("<distrobox> %s pull --image %s", HelperCommand, imageName)
		if platform != "" {
			command += " --platform " + platform
		}
		//nolint:forbidigo // Print command in dry-run mode
		fmt.Println(command)
		return nil
	}

	normalized, err := ocistore.NormalizeRef(imageName)
	if err != nil {
		return fmt.Errorf("cannot pull image: %w", err)
	}
	fmt.Fprintf(os.Stdout, "Pulling %s...\n", normalized)

	if n.root && os.Geteuid() != 0 {
		args := []string{"--image", imageName}
		if platform != "" {
			args = append(args, "--platform", platform)
		}
		return n.runHelper(ctx, nil, "pull", args...)
	}

	digest, err := n.store().Pull(ctx, imageName, platform)
	if err != nil {
		return fmt.Errorf("cannot pull image: %w", err)
	}
	fmt.Fprintf(os.Stdout, "Pulled %s (%s)\n", normalized, shortDigest(digest))
	return nil
}

func shortDigest(digest string) string {
	trimmed := strings.TrimPrefix(digest, "sha256:")
	//nolint:mnd // short digest display
	if len(trimmed) > 12 {
		trimmed = trimmed[:12]
	}
	return trimmed
}
