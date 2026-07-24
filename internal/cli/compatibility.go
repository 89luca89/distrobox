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

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/89luca89/distrobox/pkg/version"
)

const (
	// compatibilityURLTemplate is the upstream URL for docs/compatibility.md.
	// %s is the git ref (tag or branch) to fetch from.
	compatibilityURLTemplate = "https://raw.githubusercontent.com/89luca89/distrobox/%s/docs/compatibility.md"

	// compatibilityFetchTimeout caps the time we wait for the HTTP request.
	compatibilityFetchTimeout = 15 * time.Second

	// devFallbackRef is used in place of the "dev" version (set on local
	// builds) to fetch the latest upstream compatibility list.
	devFallbackRef = "main"

	// containersDistrosHeader is the markdown heading that precedes the
	// "Containers Distros" table in docs/compatibility.md. We parse the
	// first markdown table that follows it; everything else (including
	// the "Host Distros" table above it) is ignored.
	containersDistrosHeader = "## Containers Distros"
)

// showCompatibility prints the list of container images known to work with
// distrobox. The list is fetched from the upstream docs/compatibility.md (for
// the current version) and cached on disk so subsequent invocations are
// offline-friendly. The provided context is used to cancel the upstream HTTP
// fetch (e.g., via SIGINT); local cache reads/writes are intentionally
// uncancellable since they are bounded and very fast.
//
// This is the Go port of the bash show_compatibility helper:
// https://github.com/89luca89/distrobox/blob/main/distrobox-create#L254
func showCompatibility(ctx context.Context) error {
	ref := compatibilityRef(version.Version)

	cacheDir, err := compatibilityCacheDir()
	if err != nil {
		return fmt.Errorf("resolve cache directory: %w", err)
	}
	// Use the sanitized ref in the cache filename so that an unusual
	// build-time version string (e.g. one containing "/" or "..") can
	// never escape the cache directory.
	cachePath := filepath.Join(cacheDir, "distrobox-compatibility-"+sanitizeRefForFilename(ref))

	content, err := readCompatibilityCache(cachePath)
	if err == nil {
		fmt.Print(content) //nolint:forbidigo // CLI output by design
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, errEmptyCache) {
		return fmt.Errorf("read compatibility cache: %w", err)
	}

	fetchCtx, cancel := context.WithTimeout(ctx, compatibilityFetchTimeout)
	defer cancel()

	markdown, err := fetchCompatibilityMarkdown(fetchCtx, http.DefaultClient, ref)
	if err != nil {
		return fmt.Errorf("fetch compatibility list: %w", err)
	}

	images := parseCompatibilityImages(markdown)
	if len(images) == 0 {
		return errors.New("parse compatibility list: no images found")
	}

	rendered := strings.Join(images, "\n") + "\n"

	if err := writeCompatibilityCache(cachePath, rendered); err != nil {
		// Cache write failure is not fatal: we still got the data and
		// the user is entitled to see it.
		fmt.Fprintf(os.Stderr, "warning: could not write compatibility cache to %s: %v\n", cachePath, err)
	}

	fmt.Print(rendered) //nolint:forbidigo // CLI output by design
	return nil
}

// gitDescribeSuffixRE matches the trailing `-N-gHEX` segment that
// `git describe --tags --always` appends to a tag when HEAD is N commits
// past the tag (e.g. `v1.8.1-3-g1bc3554`). Stripping it gives us the tag
// itself, which is a valid GitHub ref upstream.
var gitDescribeSuffixRE = regexp.MustCompile(`-\d+-g[0-9a-f]{4,40}$`)

// compatibilityRef returns the git ref to fetch docs/compatibility.md from.
//
//   - "" and "dev" (local builds without ldflags overrides) fall back to "main".
//   - A `git describe` output like "v1.8.1-3-g1bc3554" or
//     "v1.8.1-3-g1bc3554-dirty" is normalized to the nearest tag ("v1.8.1"),
//     which is a real ref upstream.
//   - A bare commit-only output like "g1bc3554" (no tag prefix) also falls
//     back to "main".
//   - Anything else is used as-is.
func compatibilityRef(buildVersion string) string {
	if buildVersion == "" || buildVersion == "dev" {
		return devFallbackRef
	}

	// `git describe --dirty` appends "-dirty"; drop it first so the
	// suffix regex sees the canonical `-N-gHEX` shape.
	ref := strings.TrimSuffix(buildVersion, "-dirty")
	ref = gitDescribeSuffixRE.ReplaceAllString(ref, "")

	// `git describe --always` falls back to a bare hash for untagged
	// repositories. That is not a useful ref for upstream, so fall back
	// to main.
	if isGitHashOnly(ref) {
		return devFallbackRef
	}

	return ref
}

// sanitizeRefForFilename returns a version of ref safe to embed in a
// filename: every character that is not [A-Za-z0-9._-] is replaced with
// `_`. This prevents a ref containing path separators (e.g.
// "feature/foo") or dot-segments (e.g. "../etc/passwd") from creating
// nested paths or escaping the cache directory when concatenated into
// the cache file name.
func sanitizeRefForFilename(ref string) string {
	if ref == "" {
		return "_"
	}
	mapped := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '.', r == '_', r == '-':
			return r
		default:
			return '_'
		}
	}, ref)
	// A ref of "." or ".." would still be problematic after the per-char
	// mapping (it stays as-is), so explicitly neutralize it.
	if mapped == "." || mapped == ".." {
		return "_"
	}
	return mapped
}

// isGitHashOnly reports whether ref looks like a bare git hash with no
// surrounding tag information (matches the "g<hex>" or plain "<hex>"
// shapes that `git describe --always` produces when no tag is reachable).
func isGitHashOnly(ref string) bool {
	candidate := strings.TrimPrefix(ref, "g")
	if len(candidate) < 4 {
		return false
	}
	for _, r := range candidate {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

// compatibilityCacheDir returns the per-user cache directory used to store
// the parsed compatibility list. It honours XDG_CACHE_HOME and falls back to
// $HOME/.cache.
func compatibilityCacheDir() (string, error) {
	base := os.Getenv("XDG_CACHE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("user home directory: %w", err)
		}
		base = filepath.Join(home, ".cache")
	}
	return filepath.Join(base, "distrobox"), nil
}

// errEmptyCache is returned by readCompatibilityCache when the cache file
// exists but contains no data. The caller treats this the same as a miss.
var errEmptyCache = errors.New("compatibility cache file is empty")

// readCompatibilityCache returns the cached compatibility list if a non-empty
// cache file exists. It returns os.ErrNotExist when the cache is absent and
// errEmptyCache when the file is present but empty.
func readCompatibilityCache(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err //nolint:wrapcheck // sentinel errors are propagated as-is for the caller
	}
	if info.Size() == 0 {
		return "", errEmptyCache
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read cache file: %w", err)
	}
	return string(data), nil
}

// writeCompatibilityCache atomically writes the parsed compatibility list
// to the on-disk cache, creating the parent directory if needed. The write
// goes to a temp file in the same directory and is then renamed into place
// so that a crashed or concurrent writer can never leave a partially
// populated cache file visible to readCompatibilityCache.
func writeCompatibilityCache(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create cache directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp cache file: %w", err)
	}
	tmpPath := tmp.Name()
	// If anything below fails after the temp file is opened, do our best
	// to leave no garbage behind.
	cleanup := func() { _ = os.Remove(tmpPath) }

	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write temp cache file: %w", err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("chmod temp cache file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp cache file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		cleanup()
		return fmt.Errorf("rename temp cache file into place: %w", err)
	}
	return nil
}

// fetchCompatibilityMarkdown downloads docs/compatibility.md for the given
// git ref using the provided HTTP client.
func fetchCompatibilityMarkdown(ctx context.Context, client *http.Client, ref string) (string, error) {
	url := fmt.Sprintf(compatibilityURLTemplate, ref)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected HTTP status %d fetching %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}
	return string(body), nil
}

// parseCompatibilityImages extracts the container image references from the
// "Containers Distros" table in compatibility.md. It walks the document to
// the `## Containers Distros` heading, locates the first markdown table that
// follows (recognised by its `| --- |`-style separator row), then harvests
// the third column of each subsequent row until the table ends (a non-table,
// non-blank line) or another heading is reached. The header row and the
// separator row are skipped automatically because they appear before
// pastSeparator becomes true. Returned slice is sorted and deduplicated,
// mirroring the `sort -u` behaviour of the original bash implementation.
func parseCompatibilityImages(markdown string) []string {
	seen := make(map[string]struct{})

	var (
		inSection     bool
		pastSeparator bool
	)
	for _, line := range strings.Split(markdown, "\n") {
		trimmed := strings.TrimSpace(line)

		if !inSection {
			if trimmed == containersDistrosHeader {
				inSection = true
			}
			continue
		}

		// A subsequent heading at any level closes our section.
		if strings.HasPrefix(trimmed, "#") {
			break
		}

		if !pastSeparator {
			if isMarkdownTableSeparator(trimmed) {
				pastSeparator = true
			}
			continue
		}

		// Past the separator the table ends as soon as we see a non-blank
		// line that is not a row. Blank lines and empty-cell rows are
		// tolerated and simply yield no images.
		if trimmed != "" && !strings.HasPrefix(trimmed, "|") {
			break
		}

		for _, image := range extractImagesFromRow(line) {
			seen[image] = struct{}{}
		}
	}

	images := make([]string, 0, len(seen))
	for image := range seen {
		images = append(images, image)
	}
	sort.Strings(images)
	return images
}

// isMarkdownTableSeparator reports whether line is the separator row of a
// markdown table (e.g., `| --- | --- | --- |`, optionally with alignment
// colons such as `| :--- | :---: | ---: |`). The line is expected to be
// already whitespace-trimmed.
func isMarkdownTableSeparator(line string) bool {
	if !strings.HasPrefix(line, "|") || !strings.Contains(line, "---") {
		return false
	}
	for _, r := range line {
		switch r {
		case '|', '-', ':', ' ', '\t':
			continue
		default:
			return false
		}
	}
	return true
}

// extractImagesFromRow returns the image references from a single markdown
// table row. The original bash pipeline cuts on `|`, takes the fourth field
// (column 3 of the table when counting from 1), splits on `<br>`, strips
// whitespace and drops empty entries.
func extractImagesFromRow(line string) []string {
	fields := strings.Split(line, "|")
	// `| a | b | c |` splits into ["", " a ", " b ", " c ", ""], so the
	// fourth field (index 4 with `cut -f 4`, index 3 here) holds the
	// images column. Anything shorter is not a table row we care about.
	if len(fields) < 5 {
		return nil
	}

	cell := fields[3]
	var images []string
	for _, part := range strings.Split(cell, "<br>") {
		image := strings.TrimSpace(part)
		// Mirror the bash `tr -d ' '` and the later `sort -u`: also drop
		// any stray whitespace within the entry (some rows have stray
		// spaces inside the image name).
		image = strings.Map(stripWhitespace, image)
		if image == "" {
			continue
		}
		images = append(images, image)
	}
	return images
}

// stripWhitespace is a strings.Map helper that drops every whitespace rune.
func stripWhitespace(r rune) rune {
	if unicode.IsSpace(r) {
		return -1
	}
	return r
}
