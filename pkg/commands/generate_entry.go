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
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/89luca89/distrobox/internal/userenv"
	pkgconfig "github.com/89luca89/distrobox/pkg/config"
)

//go:embed assets/desktop_entry.toml.tmpl
var desktopEntryTmpl string

const (
	defaultContainerName = "my-distrobox"
	// fallbackIconName is a freedesktop theme icon installed by distrobox
	// (icons/hicolor/**/terminal-distrobox-icon). Used when no distro is
	// detected or its logo can't be downloaded.
	fallbackIconName = "terminal-distrobox-icon"
)

// iconDownloadTimeout bounds the per-icon HTTP download.
const iconDownloadTimeout = 10 * time.Second

type GenerateEntryOptions struct {
	Verbose             bool
	Delete              bool
	Root                bool
	DesktopEntryBaseDir string
	DistroboxPath       string
	All                 bool
	Icon                string // ignored when All=true
	ContainerName       string // ignored when All=true
}

type GenerateEntryCommand struct {
	cfg         *pkgconfig.Values
	listCommand *ListCommand
	// fetchIcon downloads url into destPath. It is a field so tests can inject
	// a fake that writes bytes without touching the network.
	fetchIcon func(ctx context.Context, url, destPath string) error
}

func NewGenerateEntryCommand(cfg *pkgconfig.Values, listCommand *ListCommand) *GenerateEntryCommand {
	return &GenerateEntryCommand{
		cfg:         cfg,
		listCommand: listCommand,
		fetchIcon:   downloadIconFile,
	}
}

func (c *GenerateEntryCommand) Execute(
	ctx context.Context,
	opts *GenerateEntryOptions) error {
	// containerImages maps a container name to its image; it is consulted
	// when icon == "auto" to pick the matching distro logo. The map may be
	// missing entries when the container manager can't be reached (e.g. in
	// tests), in which case the container name itself is used as the
	// distro hint.
	containerNames, containerImages, icon, err := c.resolveTargets(ctx, opts)
	if err != nil {
		return err
	}

	// Determine the desktop entry base dir
	desktopEntryBaseDir := opts.DesktopEntryBaseDir
	if desktopEntryBaseDir == "" {
		userEnv := userenv.LoadUserEnvironment(ctx)
		desktopEntryBaseDir = userEnv.DesktopEntryBaseDir
	}

	if opts.Delete {
		// Delete the desktop entries for all the containers
		for _, containerName := range containerNames {
			if err := c.deleteEntry(containerName, desktopEntryBaseDir); err != nil {
				return fmt.Errorf("failed to delete desktop entry for container %s: %w", containerName, err)
			}
		}

		return nil
	}

	// Determine DistroboxPath
	distroboxPath := opts.DistroboxPath
	if distroboxPath == "" {
		p, err := os.Executable()
		if err != nil {
			return fmt.Errorf("cannot read distrobox path, %w", err)
		}
		distroboxPath = p
	}

	// Create the desktop entries for all the containers
	for _, containerName := range containerNames {
		// Prefer the container's image as the distro hint when known;
		// fall back to the container name so auto-detection still has
		// something to match against.
		distroHint := containerName
		if image, ok := containerImages[containerName]; ok && image != "" {
			distroHint = image
		}
		if err := c.createEntry(ctx, containerName, icon, distroHint, desktopEntryBaseDir, distroboxPath, opts.Root); err != nil {
			return fmt.Errorf("failed to create desktop entry for container %s: %w", containerName, err)
		}
	}

	return nil
}

// resolveTargets determines which containers need entries written, which
// images back them (if known), and which icon string to use.
//
// When All is set the container manager is queried so the resulting images
// can later feed distro auto-detection. For single-container modes the icon
// comes from the user.
func (c *GenerateEntryCommand) resolveTargets(
	ctx context.Context,
	opts *GenerateEntryOptions,
) ([]string, map[string]string, string, error) {
	switch {
	case opts.All:
		listResult, err := c.listCommand.Execute(ctx)
		if err != nil {
			return nil, nil, "", fmt.Errorf("failed to list containers: %w", err)
		}
		names := make([]string, 0, len(listResult.Containers))
		images := make(map[string]string, len(listResult.Containers))
		for _, container := range listResult.Containers {
			names = append(names, container.Name)
			images[container.Name] = container.Image
		}
		return names, images, "auto", nil
	case opts.ContainerName != "":
		// Look up the container's image so auto-detection keys off the distro
		// (image) rather than the box name: a box named "dev" running
		// Ubuntu should still get the Ubuntu icon. Non-fatal if the manager
		// can't be reached — we fall back to the name in Execute.
		images := map[string]string{}
		if listResult, err := c.listCommand.Execute(ctx); err == nil {
			found := false
			for _, container := range listResult.Containers {
				if container.Name == opts.ContainerName {
					images[opts.ContainerName] = container.Image
					found = true
					break
				}
			}
			if !found && !opts.Delete {
				return nil, nil, "", fmt.Errorf("cannot find container %s, please create it first", opts.ContainerName)
			}
		}
		return []string{opts.ContainerName}, images, opts.Icon, nil
	default:
		return []string{defaultContainerName}, nil, opts.Icon, nil
	}
}

func (c *GenerateEntryCommand) deleteEntry(containerName string, desktopEntryBaseDir string) error {
	desktopEntryAppsDir := filepath.Join(desktopEntryBaseDir, "applications")
	entryFilePath := c.getEntryFilePath(desktopEntryAppsDir, containerName)
	if _, err := os.Stat(entryFilePath); os.IsNotExist(err) {
		return nil
	}
	if err := os.Remove(entryFilePath); err != nil {
		return fmt.Errorf("failed to delete desktop entry for container %s: %w", containerName, err)
	}
	return nil
}

func (c *GenerateEntryCommand) createEntry(
	ctx context.Context,
	containerName string,
	icon string,
	distroHint string,
	desktopEntryBaseDir string,
	distroboxPath string,
	root bool,
) error {
	desktopEntryAppsDir, desktopEntryIconsDir, err := c.ensureDesktopEntryDirExists(desktopEntryBaseDir)
	if err != nil {
		return fmt.Errorf("failed to ensure desktop entry directories exist: %w", err)
	}

	entryFilePath := c.getEntryFilePath(desktopEntryAppsDir, containerName)
	data := c.composeDesktopEntryData(ctx, containerName, icon, distroHint, distroboxPath, desktopEntryIconsDir, root)
	if err := c.writeDesktopEntryFile(entryFilePath, data); err != nil {
		return fmt.Errorf("failed to write desktop entry file for container %s: %w", containerName, err)
	}

	return nil
}

func (c *GenerateEntryCommand) ensureDesktopEntryDirExists(desktopEntryBaseDir string) (string, string, error) {
	// Ensure the needed targets directories exist
	desktopEntryAppsDir := filepath.Join(desktopEntryBaseDir, "applications")
	if err := os.MkdirAll(desktopEntryAppsDir, 0750); err != nil {
		return "", "", fmt.Errorf("failed to create desktop entry applications directory: %w", err)
	}
	desktopEntryIconsDir := filepath.Join(desktopEntryBaseDir, "icons")
	if err := os.MkdirAll(desktopEntryIconsDir, 0750); err != nil {
		return "", "", fmt.Errorf("failed to create desktop entry icons directory: %w", err)
	}
	return desktopEntryAppsDir, desktopEntryIconsDir, nil
}

// composeDesktopEntry generates the desktop entry for a single container.
// distroHint is consulted only when icon == "auto" to pick a distro-specific
// logo; pass the container's image name when available, otherwise the
// container name itself.
func (c *GenerateEntryCommand) composeDesktopEntryData(
	ctx context.Context,
	containerName string,
	icon string,
	distroHint string,
	distroboxPath string,
	iconsDir string,
	root bool,
) map[string]string {
	extraFlags := ""
	if root {
		extraFlags += "--root"
	}

	return map[string]string{
		"entry_name":     getEntryName(containerName),
		"container_name": containerName,
		"distrobox_path": distroboxPath,
		"icon":           c.resolveIcon(ctx, icon, distroHint, iconsDir),
		"extra_flags":    extraFlags,
	}
}

// getEntryName returns the formatted entry name for the desktop entry
// based on the container name, capitalizing the first letter.
func getEntryName(containerName string) string {
	if containerName == "" {
		return ""
	}
	first := strings.ToUpper(containerName[:1])
	if len(containerName) > 1 {
		return first + containerName[1:]
	}
	return first
}

func (c *GenerateEntryCommand) writeDesktopEntryFile(
	entryFilePath string,
	data map[string]string,
) error {
	//nolint:gosec // 644 is common permission for desktop entry files
	destFileWriter, err := os.OpenFile(entryFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create desktop entry file: %w", err)
	}
	defer destFileWriter.Close()

	t, err := template.New("desktopEntry").Parse(desktopEntryTmpl)
	if err != nil {
		return fmt.Errorf("failed to parse desktop entry template: %w", err)
	}
	err = t.Execute(destFileWriter, data)
	if err != nil {
		return fmt.Errorf("failed to execute desktop entry template: %w", err)
	}

	return nil
}

func (c *GenerateEntryCommand) getEntryFilePath(desktopEntryDir, containerName string) string {
	return filepath.Join(desktopEntryDir, containerName+".desktop")
}

// resolveIcon resolves the Icon= value written to the desktop entry.
//
// An explicit icon (anything other than ""/"auto") is returned unchanged. For
// "auto", the distro is detected from distroHint and its bundled logo is cached
// under <iconsDir>/distrobox and downloaded only once: a non-empty cache file is
// reused, a miss is fetched, and a fetch failure falls back to the generic
// terminal theme icon WITHOUT clobbering any existing cache file.
func (c *GenerateEntryCommand) resolveIcon(ctx context.Context, icon, distroHint, iconsDir string) string {
	if icon != "" && icon != "auto" {
		return icon
	}

	url := lookupDistroIcon(distroHint)
	if url == "" {
		return fallbackIconName
	}

	localPath := filepath.Join(iconsDir, "distrobox", path.Base(url))

	// Cache hit: reuse a previously downloaded icon, so we neither re-download
	// nor risk truncating it (works offline once cached).
	if info, err := os.Stat(localPath); err == nil && info.Size() > 0 {
		return localPath
	}

	if err := c.fetchIcon(ctx, url, localPath); err != nil {
		return fallbackIconName
	}
	return localPath
}

// downloadIconFile downloads url into destPath, creating the parent directory.
// It writes to a temp file and renames atomically, so a partial or failed
// download never leaves a truncated icon in place.
func downloadIconFile(ctx context.Context, url, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o750); err != nil {
		return fmt.Errorf("failed to create icon directory: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, iconDownloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to build icon request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download icon: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download icon: status %d", resp.StatusCode)
	}

	tmp, err := os.CreateTemp(filepath.Dir(destPath), ".icon-*")
	if err != nil {
		return fmt.Errorf("failed to create temp icon file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once renamed

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write icon: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close icon file: %w", err)
	}

	if err := os.Rename(tmpName, destPath); err != nil {
		return fmt.Errorf("failed to finalize icon: %w", err)
	}
	return nil
}
