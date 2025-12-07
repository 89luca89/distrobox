package manifest

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
)

type Item struct {
	Name               string
	AdditionalFlags    []string
	AdditionalPackages []string
	Entry              bool
	Home               string
	Hostname           string
	Image              string
	Clone              string
	Init               bool
	Nvidia             bool
	InitHooks          []string
	PreInitHooks       []string
	Pull               bool
	Root               bool
	StartNow           bool
	UnshareGroups      bool
	UnshareIPC         bool
	UnshareNetns       bool
	UnshareProcess     bool
	UnshareDevsys      bool
	UnshareAll         bool
	Volumes            []string
	ExportedApps       []string
	ExportedBins       []string
	ExportedBinsPath   string
}

// Parse reads and parses a manifest file from the given filepath or URL.
// It supports 'include=' directives to include sections within the same file.
// Returns a slice of Item structs representing each section in the manifest.
func Parse(ctx context.Context, filepath string) ([]Item, error) {
	if isURL(filepath) {
		var err error
		filepath, err = fetchIntoTempFile(ctx, filepath)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch manifest from URL: %w", err)
		}
		defer os.Remove(filepath)
	}

	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open manifest file: %w", err)
	}
	defer file.Close()

	sections, err := parseINISections(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse INI sections: %w", err)
	}

	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to seek manifest file: %w", err)
	}

	expanded, err := processINIIncludes(file, sections)
	if err != nil {
		return nil, fmt.Errorf("failed to process includes: %w", err)
	}

	manifest, err := parseManifest(expanded, len(sections))
	if err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return manifest, nil
}

// ParseINISections reads an INI file and returns a map where keys are section names
// and values are the raw, unparsed content of each section (multiline).
// Empty lines are ignored.
func parseINISections(file io.Reader) (map[string]string, error) {
	sections := make(map[string]string)
	var currentSection string
	var currentContent strings.Builder

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// strip comments starting with #
		if idx := strings.Index(line, "#"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check if line is a section header: [section_name]
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			// Save previous section content if exists
			if currentSection != "" {
				sections[currentSection] = currentContent.String()
			}

			// Start new section
			currentSection = trimmed[1 : len(trimmed)-1] // Extract section name
			currentContent.Reset()
		} else if currentSection != "" {
			// Add line to current section content
			if currentContent.Len() > 0 {
				currentContent.WriteString("\n")
			}
			currentContent.WriteString(line)
		}
		// Lines before first section are ignored
	}

	// Save the last section
	if currentSection != "" {
		sections[currentSection] = currentContent.String()
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}

	return sections, nil
}

// ProcessINIIncludes reads an INI file and recursively expands all include= directives
// by replacing them with the content from the corresponding section in the sections map.
// Included sections can themselves contain include= directives, which are also expanded.
// Returns the processed content as a string, or an error if an included section is not found
// or if a circular include is detected.
func processINIIncludes(file io.Reader, sections map[string]string) (io.Reader, error) {
	// Read all lines into a slice that we can append to
	// FIXME: this can be memory intensive for large files, consider streaming approach
	lines := make([]string, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}

	// Process lines (the slice can grow as we append included content)
	// Track which sections we're currently processing to detect circular includes
	// the stack is big at most len(sections), we can avoid re-allocating
	includeStack := make([]string, 0, len(sections))

	var result bytes.Buffer

	// Process lines (the slice can grow as we append included content)
	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Strip comments and leading/trailing whitespace
		line = stripComment(line)
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// if is a section header, just copy as is
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			result.WriteString(fmt.Sprintf("%s\n", line))
			includeStack = includeStack[:0] // clear include stack
			continue
		}

		key, value, err := parseKeyValue(stripComment(line))
		if err != nil {
			// Not a key=value line, just copy as is
			result.WriteString(fmt.Sprintf("%s\n", line))
			continue
		}

		if key != "include" {
			result.WriteString(fmt.Sprintf("%s\n", line))
			continue
		}

		sectionName := value

		// Check for circular includes
		if slices.Contains(includeStack, sectionName) {
			return nil, fmt.Errorf(
				"circular include detected: section [%s] already referenced in %v",
				sectionName,
				includeStack,
			)
		}

		// Mark this section as being processed
		includeStack = append(includeStack, sectionName)

		// Look up the section in the map
		content, exists := sections[sectionName]
		if !exists {
			return nil, fmt.Errorf("included section [%s] not found", sectionName)
		}

		// Split the included content into lines and insert them after current position
		includedLines := strings.Split(content, "\n")

		// Create new slice with: lines[0:i] + includedLines + lines[i+1:]
		// This removes the include line at position i and replaces it with includedLines
		newLines := make([]string, 0, len(lines)+len(includedLines)-1)
		newLines = append(newLines, lines[:i]...)
		newLines = append(newLines, includedLines...)
		newLines = append(newLines, lines[i+1:]...)
		lines = newLines

		// Decrement i so we process the first included line next
		i--
	}

	return bytes.NewReader(result.Bytes()), nil
}

func parseManifest(file io.Reader, size int) ([]Item, error) {
	items := make([]Item, 0, size)
	scanner := bufio.NewScanner(file)
	var currentItem *Item

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		sectionName, err := parseSectionHeader(trimmed)
		if err == nil {
			if currentItem != nil {
				items = append(items, *currentItem)
			}

			currentItem = &Item{Name: sectionName}
			continue
		}

		key, value, err := parseKeyValue(trimmed)
		if err == nil {
			if currentItem == nil {
				return nil, errors.New("key-value pair found outside of a section")
			}
			putValue(currentItem, key, value)
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}

	// Append the last item if it exists
	if currentItem != nil {
		items = append(items, *currentItem)
	}

	return items, nil
}

func parseSectionHeader(line string) (string, error) {
	if line == "" {
		return "", errors.New("unexpected empty line")
	}
	if !strings.HasPrefix(line, "[") || !strings.HasSuffix(line, "]") {
		return "", fmt.Errorf("invalid section header format: %s", line)
	}
	sectionName := strings.TrimSpace(line[1 : len(line)-1])
	return sectionName, nil
}

func parseKeyValue(line string) ( /* key*/ string /* value*/, string, error) {
	if line == "" {
		return "", "", errors.New("unexpected empty line")
	}
	parts := strings.SplitN(line, "=", 2) //nolint:mnd // reason: splitting into 2 parts
	if len(parts) != 2 {                  //nolint:mnd // reason: splitting into 2 parts
		return "", "", fmt.Errorf("invalid line format: %s", line)
	}
	return strings.TrimSpace(parts[0]), stripQuotes(parts[1]), nil
}

func stripQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 { //nolint:mnd // reason: splitting into 2 parts
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func stripComment(s string) string {
	if idx := strings.Index(s, "#"); idx != -1 {
		return strings.TrimSpace(s[:idx])
	}
	return s
}

// putValue sets the appropriate field in ManifestItem based on the key and value provided.
//
//nolint:funlen // reason: straightforward mapping
func putValue(item *Item, key, value string) {
	switch key {
	case "name":
		item.Name = value
	case "image":
		item.Image = value
	case "clone":
		item.Clone = value
	case "init":
		item.Init = atob(value)
	case "nvidia":
		item.Nvidia = atob(value)
	case "entry":
		item.Entry = atob(value)
	case "home":
		item.Home = value
	case "hostname":
		item.Hostname = value
	case "additional_flags":
		item.AdditionalFlags = append(item.AdditionalFlags, strings.Split(value, " ")...)
	case "additional_packages":
		item.AdditionalPackages = append(item.AdditionalPackages, strings.Split(value, " ")...)
	case "init_hooks":
		item.InitHooks = append(item.InitHooks, value)
	case "pre_init_hooks":
		item.PreInitHooks = append(item.PreInitHooks, value)
	case "volumes":
		item.Volumes = append(item.Volumes, value)
	case "exported_apps":
		item.ExportedApps = append(item.ExportedApps, strings.Split(value, " ")...)
	case "exported_bins":
		item.ExportedBins = append(item.ExportedBins, strings.Split(value, " ")...)
	case "exported_bins_path":
		item.ExportedBinsPath = value
	case "pull":
		item.Pull = atob(value)
	case "root":
		item.Root = atob(value)
	case "start_now":
		item.StartNow = atob(value)
	case "unshare_groups":
		item.UnshareGroups = atob(value)
	case "unshare_ipc":
		item.UnshareIPC = atob(value)
	case "unshare_netns":
		item.UnshareNetns = atob(value)
	case "unshare_process":
		item.UnshareProcess = atob(value)
	case "unshare_devsys":
		item.UnshareDevsys = atob(value)
	case "unshare_all":
		item.UnshareAll = atob(value)
	default:
		// Unknown key, ignore
		// TODO: log warning?
	}
}

func atob(value string) bool {
	return value == "true" || value == "1"
}

// fetchIntoTempFile fetches the content
// and saves it to a temporary file, returning the file path.
func fetchIntoTempFile(ctx context.Context, fileURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch URL: status %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "distribox-manifest-fetched-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		_ = os.Remove(tmpFile.Name()) // Cleanup on error
		return "", fmt.Errorf("failed to write content: %w", err)
	}

	return tmpFile.Name(), nil
}

func isURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}

	return u.Scheme != "" && u.Host != ""
}
