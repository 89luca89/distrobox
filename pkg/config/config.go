package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/ini.v1"
)

type Values struct {
	ContainerManagerType  string
	SudoProgram           string
	Verbose               bool
	DefaultContainerImage string
	DefaultContainerName  string
}

func defaultsMap() map[string]string {
	return map[string]string{
		"container_manager": "podman",
		"sudo_program":      "sudo",
		"verbose":           "false",
		// container_image Fedora toolbox is a sensitive default
		"container_image": "registry.fedoraproject.org/fedora-toolbox:latest",
		"container_name":  "my-distrobox",
	}
}

func DefaultValues() *Values {
	return toStruct(defaultsMap())
}

func LoadValues() (*Values, error) {
	files, err := getConfigFilePaths()
	if err != nil {
		return nil, fmt.Errorf("failed to get config file paths: %w", err)
	}

	configMaps := []map[string]string{defaultsMap()}

	for _, file := range files {
		config, err := readConfigFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file %q: %w", file, err)
		}
		configMaps = append(configMaps, config)
	}

	configMaps = append(configMaps, readEnv())

	merged := mergeConfigMaps(configMaps...)
	return toStruct(merged), nil
}

func toStruct(configMap map[string]string) *Values {
	return &Values{
		ContainerManagerType:  configMap["container_manager"],
		SudoProgram:           configMap["sudo_program"],
		Verbose:               toBool(configMap["verbose"]),
		DefaultContainerImage: configMap["container_image"],
		DefaultContainerName:  configMap["container_name"],
	}
}

func toBool(value string) bool {
	return value == "true"
}

// getConfigFilePaths returns a list of configuration file paths in order of priority.
func getConfigFilePaths() ([]string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate symlinks for executable path: %w", err)
	}

	selfDir := filepath.Dir(execPath)

	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfigHome == "" {
		xdgConfigHome = filepath.Join(os.Getenv("HOME"), ".config")
	}

	home := os.Getenv("HOME")

	// Source configuration files, this is done in an hierarchy so local files have
	// priority over system defaults
	// leave priority to environment variables.
	//
	// On NixOS, for the distrobox derivation to pick up a static config file shipped
	// by the package maintainer the path must be relative to the script itself.
	return []string{
		filepath.Join(selfDir, "..", "share", "distrobox", "distrobox.conf"), // for NixOS
		"/usr/share/distrobox/distrobox.conf",
		"/usr/share/defaults/distrobox/distrobox.conf",
		"/usr/etc/distrobox/distrobox.conf",
		"/usr/local/share/distrobox/distrobox.conf",
		"/etc/distrobox/distrobox.conf",
		filepath.Join(xdgConfigHome, "distrobox", "distrobox.conf"),
		filepath.Join(home, ".distroboxrc"),
	}, nil
}

func mergeConfigMaps(maps ...map[string]string) map[string]string {
	merged := make(map[string]string)
	for _, m := range maps {
		for k, v := range m {
			merged[k] = v
		}
	}
	return merged
}

func readConfigFile(filePath string) (map[string]string, error) {
	cfg, err := ini.Load(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, fmt.Errorf("failed to load config file %q: %w", filePath, err)
	}

	config := make(map[string]string)
	for _, key := range cfg.Section("").Keys() {
		config[key.Name()] = key.String()
	}

	return config, nil
}

func readEnv() map[string]string {
	envConfig := make(map[string]string)

	if value, exists := os.LookupEnv("DBX_CONTAINER_MANAGER"); exists {
		envConfig["container_manager"] = value
	}
	if value, exists := os.LookupEnv("DBX_SUDO_COMMAND"); exists {
		envConfig["sudo_program"] = value
	}
	if value, exists := os.LookupEnv("DBX_SUDO_PROGRAM"); exists {
		envConfig["sudo_program"] = value
	}
	if value, exists := os.LookupEnv("DBX_VERBOSE"); exists {
		envConfig["verbose"] = value
	}

	return envConfig
}
