package manifest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"gopkg.in/ini.v1"
)

// Item represents a single section in the manifest file.
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
	data, err := readData(ctx, filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	cfg, err := ini.LoadSources(ini.LoadOptions{AllowShadows: true}, data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse INI: %w", err)
	}

	if len(cfg.Section(ini.DefaultSection).Keys()) > 0 {
		return nil, errors.New("key-value pair found outside of a section")
	}

	if err := expandIncludes(cfg); err != nil {
		return nil, fmt.Errorf("failed to expand includes: %w", err)
	}

	items := make([]Item, 0, len(cfg.Sections())-1)
	for _, section := range cfg.Sections() {
		if section.Name() == ini.DefaultSection {
			continue
		}
		items = append(items, sectionToItem(section))
	}

	return items, nil
}

// expandIncludes resolves include= directives by copying keys from referenced sections.
func expandIncludes(cfg *ini.File) error {
	for _, section := range cfg.Sections() {
		if section.Name() == ini.DefaultSection {
			continue
		}

		seen := make(map[string]bool)
		if err := resolveIncludes(cfg, section, seen); err != nil {
			return err
		}
	}

	return nil
}

// resolveIncludes processes include= directives for a given section.
func resolveIncludes(cfg *ini.File, section *ini.Section, seen map[string]bool) error {
	includes := section.Key("include").ValueWithShadows()
	section.DeleteKey("include")

	for _, name := range includes {
		if name == "" {
			continue
		}
		if seen[name] {
			return fmt.Errorf("circular include: %s", name)
		}
		seen[name] = true

		src := cfg.Section(name)
		if src == nil || src.Name() == ini.DefaultSection {
			return fmt.Errorf("included section [%s] not found", name)
		}

		for _, key := range src.Keys() {
			for _, v := range key.ValueWithShadows() {
				if _, err := section.NewKey(key.Name(), v); err != nil {
					return fmt.Errorf("failed to copy key %s from section [%s]: %w", key.Name(), name, err)
				}
			}
		}
	}

	processed[name] = true
	return nil
}

// sectionToItem converts an ini.Section to an Item struct.
func sectionToItem(section *ini.Section) Item {
	return Item{
		Name:             section.Name(),
		Image:            lastString(section, "image"),
		Clone:            lastString(section, "clone"),
		Home:             lastString(section, "home"),
		Hostname:         lastString(section, "hostname"),
		ExportedBinsPath: lastString(section, "exported_bins_path"),

		Init:           lastBool(section, "init"),
		Nvidia:         lastBool(section, "nvidia"),
		Entry:          lastBool(section, "entry"),
		Pull:           lastBool(section, "pull"),
		Root:           lastBool(section, "root"),
		StartNow:       lastBool(section, "start_now"),
		UnshareGroups:  lastBool(section, "unshare_groups"),
		UnshareIPC:     lastBool(section, "unshare_ipc"),
		UnshareNetns:   lastBool(section, "unshare_netns"),
		UnshareProcess: lastBool(section, "unshare_process"),
		UnshareDevsys:  lastBool(section, "unshare_devsys"),
		UnshareAll:     lastBool(section, "unshare_all"),

		AdditionalFlags:    splitString(section, "additional_flags"),
		AdditionalPackages: splitString(section, "additional_packages"),
		ExportedApps:       splitString(section, "exported_apps"),
		ExportedBins:       splitString(section, "exported_bins"),

		InitHooks:    allStrings(section, "init_hooks"),
		PreInitHooks: allStrings(section, "pre_init_hooks"),
		Volumes:      allStrings(section, "volumes"),
	}
}

// lastString returns the final value when a key appears multiple times, or empty string if unset.
func lastString(s *ini.Section, key string) string {
	vals := s.Key(key).ValueWithShadows()
	if len(vals) == 0 || vals[0] == "" {
		return ""
	}

	return vals[len(vals)-1]
}

// lastBool returns the final value when a key appears multiple times, parsed as a boolean (true/1).
func lastBool(s *ini.Section, key string) bool {
	v := lastString(s, key)

	return v == "true" || v == "1"
}

// allStrings returns all values when a key appears multiple times, or nil if unset.
func allStrings(s *ini.Section, key string) []string {
	vals := s.Key(key).ValueWithShadows()
	if len(vals) == 1 && vals[0] == "" {
		return nil
	}

	return vals
}

// splitString splits all values of a key by whitespace and returns a flat slice.
func splitString(s *ini.Section, key string) []string {
	var result []string
	for _, v := range s.Key(key).ValueWithShadows() {
		result = append(result, strings.Fields(v)...)
	}

	return result
}

// readData reads data from a local file or a URL.
func readData(ctx context.Context, path string) ([]byte, error) {
	if u, err := url.Parse(path); err == nil && u.Scheme != "" && u.Host != "" {
		return fetchURL(ctx, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

// fetchURL retrieves data from the specified URL.
func fetchURL(ctx context.Context, u string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var buf bytes.Buffer
	_, err = io.Copy(&buf, resp.Body)

	return buf.Bytes(), err
}
