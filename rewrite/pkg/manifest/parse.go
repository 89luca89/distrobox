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
	AlwaysPull         bool
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
	processing := make(map[string]bool) // Currently in call stack (cycle detection)
	processed := make(map[string]bool)  // Already fully resolved (skip duplicates)

	for _, section := range cfg.Sections() {
		if section.Name() == ini.DefaultSection {
			continue
		}

		if err := resolveIncludes(cfg, section, processing, processed); err != nil {
			return err
		}
	}

	return nil
}

func resolveIncludes(cfg *ini.File, section *ini.Section, processing, processed map[string]bool) error {
	name := section.Name()

	if processed[name] {
		return nil
	}
	if processing[name] {
		return fmt.Errorf("circular include detected: %s", name)
	}

	processing[name] = true

	includes := section.Key("include").ValueWithShadows()
	section.DeleteKey("include")

	for _, includeName := range includes {
		if includeName == "" {
			continue
		}

		src := cfg.Section(includeName)
		if src == nil || src.Name() == ini.DefaultSection {
			return fmt.Errorf("included section [%s] not found", includeName)
		}

		if err := resolveIncludes(cfg, src, processing, processed); err != nil {
			return err
		}

		for _, key := range src.Keys() {
			for _, v := range key.ValueWithShadows() {
				if _, err := section.NewKey(key.Name(), v); err != nil {
					return fmt.Errorf("failed to copy key %s from section [%s]: %w", key.Name(), includeName, err)
				}
			}
		}
	}

	processed[name] = true
	return nil
}

// sectionToItem converts an ini.Section to an Item struct.
func sectionToItem(section *ini.Section) Item { //nolint:funlen // Function length is acceptable here.
	item := Item{Name: section.Name()}

	for _, key := range section.Keys() {
		vals := key.ValueWithShadows()
		last := vals[len(vals)-1]

		switch key.Name() {
		case "image":
			item.Image = last
		case "clone":
			item.Clone = last
		case "home":
			item.Home = last
		case "hostname":
			item.Hostname = last
		case "exported_bins_path":
			item.ExportedBinsPath = last

		case "init":
			item.Init = parseBool(last)
		case "nvidia":
			item.Nvidia = parseBool(last)
		case "entry":
			item.Entry = parseBool(last)
		case "pull":
			item.AlwaysPull = parseBool(last)
		case "root":
			item.Root = parseBool(last)
		case "start_now":
			item.StartNow = parseBool(last)
		case "unshare_groups":
			item.UnshareGroups = parseBool(last)
		case "unshare_ipc":
			item.UnshareIPC = parseBool(last)
		case "unshare_netns":
			item.UnshareNetns = parseBool(last)
		case "unshare_process":
			item.UnshareProcess = parseBool(last)
		case "unshare_devsys":
			item.UnshareDevsys = parseBool(last)
		case "unshare_all":
			item.UnshareAll = parseBool(last)

		case "additional_flags":
			for _, v := range vals {
				item.AdditionalFlags = append(item.AdditionalFlags, strings.Fields(v)...)
			}
		case "additional_packages":
			for _, v := range vals {
				item.AdditionalPackages = append(item.AdditionalPackages, strings.Fields(v)...)
			}
		case "exported_apps":
			for _, v := range vals {
				item.ExportedApps = append(item.ExportedApps, strings.Fields(v)...)
			}
		case "exported_bins":
			for _, v := range vals {
				item.ExportedBins = append(item.ExportedBins, strings.Fields(v)...)
			}

		case "init_hooks":
			item.InitHooks = append(item.InitHooks, vals...)
		case "pre_init_hooks":
			item.PreInitHooks = append(item.PreInitHooks, vals...)
		case "volumes":
			item.Volumes = append(item.Volumes, vals...)
		}
	}

	return item
}

// parseBool parses a string as a boolean (true/1).
func parseBool(s string) bool {
	return s == "true" || s == "1"
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
