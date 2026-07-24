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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleCompatibilityMarkdown = `# Compatibility

## Host Distros

| Distro | Version | Notes |
| --- | --- | --- |
| Alpine Linux | | |
| Void Linux | glibc <br> musl | |

## Containers Distros

| Distro | Version | Images |
| --- | --- | --- |
| AlmaLinux (Toolbox) | 8 <br> 9 | quay.io/toolbx-images/almalinux-toolbox:8 <br> quay.io/toolbx-images/almalinux-toolbox:9 <br> quay.io/toolbx-images/almalinux-toolbox:latest |
| Alpine (Toolbox) | edge | quay.io/toolbx-images/alpine-toolbox:edge |
| Fedora | 39 | quay.io/fedora/fedora:39 |
| Void Linux | glibc <br> musl | ghcr.io/void-linux/void-glibc-full:latest <br> ghcr.io/void-linux/void-musl-full:latest |

Trailing text that should be ignored.
`

func TestParseCompatibilityImages_ReturnsSortedUniqueImagesFromContainerDistrosTable(t *testing.T) {
	images := parseCompatibilityImages(sampleCompatibilityMarkdown)

	expected := []string{
		"ghcr.io/void-linux/void-glibc-full:latest",
		"ghcr.io/void-linux/void-musl-full:latest",
		"quay.io/fedora/fedora:39",
		"quay.io/toolbx-images/almalinux-toolbox:8",
		"quay.io/toolbx-images/almalinux-toolbox:9",
		"quay.io/toolbx-images/almalinux-toolbox:latest",
		"quay.io/toolbx-images/alpine-toolbox:edge",
	}
	assert.Equal(t, expected, images)
}

func TestParseCompatibilityImages_SkipsHostDistrosTable(t *testing.T) {
	// Rows that live in a markdown table BEFORE the `## Containers Distros`
	// heading must never contribute, even when their third column happens
	// to look like an image reference.
	markdown := `## Host Distros

| Distro | Version | Notes |
| --- | --- | --- |
| Alpine | | docker.io/should-not-appear:latest |

## Containers Distros

| Distro | Version | Images |
| --- | --- | --- |
| AlmaLinux | 8 | docker.io/library/almalinux:8 |
`
	images := parseCompatibilityImages(markdown)

	assert.NotContains(t, images, "docker.io/should-not-appear:latest",
		"rows from the Host Distros table must not contribute")
	assert.Contains(t, images, "docker.io/library/almalinux:8")
}

func TestParseCompatibilityImages_StopsAtNonTableLine(t *testing.T) {
	markdown := sampleCompatibilityMarkdown + `
| Wonderlandia | 1 | docker.io/wonderlandia/wonderlandia:1 |
`
	images := parseCompatibilityImages(markdown)

	assert.NotContains(t, images, "docker.io/wonderlandia/wonderlandia:1",
		"rows after the table-terminating prose line should be ignored")
}

func TestParseCompatibilityImages_StopsAtSubsequentHeading(t *testing.T) {
	markdown := `## Containers Distros

| Distro | Version | Images |
| --- | --- | --- |
| AlmaLinux | 8 | docker.io/library/almalinux:8 |

### New Distro support

| Wonderlandia | 1 | docker.io/wonderlandia/wonderlandia:1 |
`
	images := parseCompatibilityImages(markdown)

	assert.Contains(t, images, "docker.io/library/almalinux:8")
	assert.NotContains(t, images, "docker.io/wonderlandia/wonderlandia:1",
		"rows after the next heading should be ignored")
}

func TestParseCompatibilityImages_SkipsTableHeaderRow(t *testing.T) {
	// The literal "Images" cell of the header row must not leak through
	// as if it were an image name. It precedes the separator, so the
	// parser should ignore it.
	markdown := `## Containers Distros

| Distro | Version | Images |
| --- | --- | --- |
| AlmaLinux | 8 | docker.io/library/almalinux:8 |
`
	images := parseCompatibilityImages(markdown)

	assert.NotContains(t, images, "Images")
	assert.Equal(t, []string{"docker.io/library/almalinux:8"}, images)
}

func TestParseCompatibilityImages_DeduplicatesIdenticalEntries(t *testing.T) {
	markdown := `## Containers Distros

| Distro | Version | Images |
| --- | --- | --- |
| AlmaLinux | 8 | docker.io/library/almalinux:8 |
| AlmaLinux 2 | 8 | docker.io/library/almalinux:8 |
| Void Linux | musl | ghcr.io/void-linux/void-musl-full:latest |
`
	images := parseCompatibilityImages(markdown)

	require.Len(t, images, 2)
	assert.Equal(t, []string{
		"docker.io/library/almalinux:8",
		"ghcr.io/void-linux/void-musl-full:latest",
	}, images)
}

func TestParseCompatibilityImages_StripsInternalAndSurroundingWhitespace(t *testing.T) {
	// Some rows in the real doc have stray spaces around the <br>
	// separator inside the image column. The bash pipeline uses
	// `tr -d ' '` to scrub everything, and we mirror that here.
	markdown := `## Containers Distros

| Distro | Version | Images |
| --- | --- | --- |
| AlmaLinux |   |    docker.io/library/almalinux:8   <br>   docker.io/library/almalinux:9 |
| Void Linux | musl | ghcr.io/void-linux/void-musl-full:latest |
`
	images := parseCompatibilityImages(markdown)

	assert.Equal(t, []string{
		"docker.io/library/almalinux:8",
		"docker.io/library/almalinux:9",
		"ghcr.io/void-linux/void-musl-full:latest",
	}, images)
}

func TestParseCompatibilityImages_IgnoresEmptySeparatorRow(t *testing.T) {
	// The real document has `| | | |` between toolbox images and plain
	// images. That row must not yield an empty entry.
	markdown := `## Containers Distros

| Distro | Version | Images |
| --- | --- | --- |
| AlmaLinux | 8 | docker.io/library/almalinux:8 |
| | | |
| Void Linux | musl | ghcr.io/void-linux/void-musl-full:latest |
`
	images := parseCompatibilityImages(markdown)

	for _, image := range images {
		assert.NotEmpty(t, image)
	}
}

func TestParseCompatibilityImages_ReturnsEmptySliceWhenSectionHeaderMissing(t *testing.T) {
	// Without the `## Containers Distros` heading, no table is considered
	// authoritative — even a perfectly shaped table is ignored.
	markdown := `## Host Distros

| Distro | Version | Images |
| --- | --- | --- |
| AlmaLinux | 8 | docker.io/library/almalinux:8 |
`
	images := parseCompatibilityImages(markdown)

	assert.Empty(t, images)
}

func TestParseCompatibilityImages_ReturnsEmptySliceForUnrelatedMarkdown(t *testing.T) {
	markdown := "# Nothing here\n\nNo tables, no images.\n"

	images := parseCompatibilityImages(markdown)

	assert.Empty(t, images)
}

func TestExtractImagesFromRow_ReturnsAllImagesSeparatedByBr(t *testing.T) {
	row := "| Fedora | 38 <br> 39 | quay.io/fedora/fedora:38 <br> quay.io/fedora/fedora:39 |"

	images := extractImagesFromRow(row)

	assert.Equal(t, []string{
		"quay.io/fedora/fedora:38",
		"quay.io/fedora/fedora:39",
	}, images)
}

func TestExtractImagesFromRow_ReturnsNilForNonTableLine(t *testing.T) {
	row := "Some prose, not a table row."

	images := extractImagesFromRow(row)

	assert.Nil(t, images)
}

func TestExtractImagesFromRow_ReturnsNilForRowWithoutImagesColumn(t *testing.T) {
	// `| a | b |` has only 4 fields when split (incl. leading/trailing
	// empties), so the third column does not exist.
	row := "| a | b |"

	images := extractImagesFromRow(row)

	assert.Nil(t, images)
}

func TestIsMarkdownTableSeparator_DetectsCanonicalSeparator(t *testing.T) {
	assert.True(t, isMarkdownTableSeparator("| --- | --- | --- |"))
}

func TestIsMarkdownTableSeparator_DetectsCompactSeparator(t *testing.T) {
	assert.True(t, isMarkdownTableSeparator("|---|---|---|"))
}

func TestIsMarkdownTableSeparator_DetectsAlignmentMarkers(t *testing.T) {
	assert.True(t, isMarkdownTableSeparator("| :--- | :---: | ---: |"))
}

func TestIsMarkdownTableSeparator_RejectsHeaderRow(t *testing.T) {
	assert.False(t, isMarkdownTableSeparator("| Distro | Version | Images |"))
}

func TestIsMarkdownTableSeparator_RejectsDataRow(t *testing.T) {
	assert.False(t, isMarkdownTableSeparator("| AlmaLinux | 8 | docker.io/library/almalinux:8 |"))
}

func TestIsMarkdownTableSeparator_RejectsBlankLine(t *testing.T) {
	assert.False(t, isMarkdownTableSeparator(""))
}

func TestIsMarkdownTableSeparator_RejectsNonTableLine(t *testing.T) {
	assert.False(t, isMarkdownTableSeparator("--- some divider ---"))
}

func TestCompatibilityRef_ReturnsMainForDevVersion(t *testing.T) {
	assert.Equal(t, "main", compatibilityRef("dev"))
}

func TestCompatibilityRef_ReturnsMainForEmptyVersion(t *testing.T) {
	assert.Equal(t, "main", compatibilityRef(""))
}

func TestCompatibilityRef_ReturnsBuildVersionWhenSet(t *testing.T) {
	assert.Equal(t, "1.8.1", compatibilityRef("1.8.1"))
}

func TestCompatibilityRef_StripsGitDescribeSuffix(t *testing.T) {
	// `git describe --tags` on a commit past v1.8.1 yields this shape.
	// We want to fetch docs/compatibility.md from the v1.8.1 tag, which
	// is a real ref upstream, not from "v1.8.1-3-g1bc3554" which is not.
	assert.Equal(t, "v1.8.1", compatibilityRef("v1.8.1-3-g1bc3554"))
}

func TestCompatibilityRef_StripsGitDescribeSuffixWithDirty(t *testing.T) {
	assert.Equal(t, "v1.8.1", compatibilityRef("v1.8.1-3-g1bc3554-dirty"))
}

func TestCompatibilityRef_StripsDirtySuffixAlone(t *testing.T) {
	assert.Equal(t, "v1.8.1", compatibilityRef("v1.8.1-dirty"))
}

func TestCompatibilityRef_FallsBackToMainForBareHash(t *testing.T) {
	// `git describe --always` produces a bare hash when no tag is
	// reachable. That is not a useful ref for the upstream repo.
	assert.Equal(t, "main", compatibilityRef("g1bc3554"))
	assert.Equal(t, "main", compatibilityRef("1bc3554"))
}

func TestCompatibilityRef_PreservesUnconventionalRefs(t *testing.T) {
	// Anything that doesn't look like git describe output should be
	// passed through untouched (e.g., a branch name set via ldflags).
	assert.Equal(t, "release-2.0", compatibilityRef("release-2.0"))
}

func TestSanitizeRefForFilename_PreservesSafeCharacters(t *testing.T) {
	assert.Equal(t, "v1.8.1", sanitizeRefForFilename("v1.8.1"))
	assert.Equal(t, "main", sanitizeRefForFilename("main"))
	assert.Equal(t, "release-2.0", sanitizeRefForFilename("release-2.0"))
}

func TestSanitizeRefForFilename_ReplacesPathSeparator(t *testing.T) {
	// A branch ref like "feature/foo" must not introduce a sub-directory
	// segment in the cache filename.
	assert.Equal(t, "feature_foo", sanitizeRefForFilename("feature/foo"))
}

func TestSanitizeRefForFilename_NeutralisesParentDirSegments(t *testing.T) {
	// "../etc/passwd" must not be able to escape the cache directory.
	assert.Equal(t, ".._etc_passwd", sanitizeRefForFilename("../etc/passwd"))
}

func TestSanitizeRefForFilename_ReplacesDotOnlyRef(t *testing.T) {
	assert.Equal(t, "_", sanitizeRefForFilename("."))
	assert.Equal(t, "_", sanitizeRefForFilename(".."))
}

func TestSanitizeRefForFilename_ReplacesEmptyRef(t *testing.T) {
	assert.Equal(t, "_", sanitizeRefForFilename(""))
}

func TestSanitizeRefForFilename_ReplacesWhitespaceAndShellChars(t *testing.T) {
	assert.Equal(t, "v1_0__rc1_", sanitizeRefForFilename("v1 0 \trc1\n"))
	assert.Equal(t, "ref_with_quote", sanitizeRefForFilename("ref'with\"quote"))
}

func TestCompatibilityCacheDir_HonoursXdgCacheHome(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmp)

	dir, err := compatibilityCacheDir()
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(tmp, "distrobox"), dir)
}

func TestCompatibilityCacheDir_FallsBackToHomeCache(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("HOME", tmp)

	dir, err := compatibilityCacheDir()
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(tmp, ".cache", "distrobox"), dir)
}

func TestReadCompatibilityCache_ReturnsContentsWhenPresent(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "cache")
	require.NoError(t, os.WriteFile(path, []byte("alpine\nfedora\n"), 0o644))

	got, err := readCompatibilityCache(path)
	require.NoError(t, err)
	assert.Equal(t, "alpine\nfedora\n", got)
}

func TestReadCompatibilityCache_ReturnsOsErrNotExistWhenMissing(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "missing")

	_, err := readCompatibilityCache(path)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestReadCompatibilityCache_ReturnsEmptyCacheSentinelForZeroByteFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "empty")
	require.NoError(t, os.WriteFile(path, nil, 0o644))

	_, err := readCompatibilityCache(path)
	assert.ErrorIs(t, err, errEmptyCache)
}

func TestWriteCompatibilityCache_CreatesParentDirectoriesAndFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "nested", "dir", "cache")

	err := writeCompatibilityCache(path, "alpine\nfedora\n")
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "alpine\nfedora\n", string(data))
}

func TestWriteCompatibilityCache_ReplacesExistingFileAtomically(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "cache")
	require.NoError(t, os.WriteFile(path, []byte("old-content"), 0o644))

	err := writeCompatibilityCache(path, "new-content\n")
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "new-content\n", string(data))

	// No leftover ".tmp-*" temp files in the cache directory.
	entries, err := os.ReadDir(tmp)
	require.NoError(t, err)
	for _, e := range entries {
		assert.NotContains(t, e.Name(), ".tmp-",
			"unexpected leftover temp file %q", e.Name())
	}
}

func TestFetchCompatibilityMarkdown_ReturnsBodyForOKResponse(t *testing.T) {
	var seenPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		_, _ = w.Write([]byte("hello from fixture"))
	}))
	defer server.Close()

	// Hit the test server directly by constructing an explicit URL —
	// the real fetcher hard-codes the GitHub URL, so we exercise it
	// via the lower-level request flow by calling through the helper
	// after re-pointing the URL via a custom RoundTripper.
	client := &http.Client{
		Transport: rewriteTransport{base: http.DefaultTransport, target: server.URL},
	}

	body, err := fetchCompatibilityMarkdown(context.Background(), client, "v1.2.3")
	require.NoError(t, err)
	assert.Equal(t, "hello from fixture", body)
	assert.Equal(t, "/89luca89/distrobox/v1.2.3/docs/compatibility.md", seenPath)
}

func TestFetchCompatibilityMarkdown_ReturnsErrorForNonOKResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &http.Client{
		Transport: rewriteTransport{base: http.DefaultTransport, target: server.URL},
	}

	_, err := fetchCompatibilityMarkdown(context.Background(), client, "v1.2.3")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

// rewriteTransport redirects every request to the configured target while
// preserving the original request path, so tests can hit a local httptest
// server without changing the production URL template.
type rewriteTransport struct {
	base   http.RoundTripper
	target string
}

func (t rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Build a new URL that points at the test server but keeps the path
	// and query of the original request so callers can assert on them.
	rewritten := strings.TrimRight(t.target, "/") + req.URL.Path
	if req.URL.RawQuery != "" {
		rewritten += "?" + req.URL.RawQuery
	}
	newReq := req.Clone(req.Context())
	newURL, err := newReq.URL.Parse(rewritten)
	if err != nil {
		return nil, err
	}
	newReq.URL = newURL
	newReq.Host = newURL.Host
	return t.base.RoundTrip(newReq)
}
