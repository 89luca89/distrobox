package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

type Values struct {
	ContainerManagerType  string
	SudoProgram           string
	Verbose               bool
	DefaultContainerImage string
	DefaultContainerName  string

	// Env-set values; empty when the corresponding DBX_* var is not exported.
	// Consumers prefer these over the Default* counterparts when non-empty,
	// matching the shell's `[ -n "$DBX_X" ] && var=$DBX_X` pattern.
	ContainerImage      string
	ContainerName       string
	ContainerHostname   string
	ContainerCustomHome string
	ContainerHomePrefix string
	ContainerAlwaysPull bool
	NonInteractive      bool
	GenerateEntry       bool
	CleanPath           bool
	SkipWorkDir         bool
	UsernsNoLimit       bool
	RmCustomHome        bool

	// ScriptsDir is the directory path where the helper scripts
	// (distrobox-init, distrobox-export, distrobox-host-exec) are stored.
	// Resolved from DBX_SCRIPTS_DIR env var first, then the directory of the
	// running binary, then ~/.local/bin, falling back to /usr/bin.
	ScriptsDir string
}

func defaultsMap() map[string]string {
	return map[string]string{
		"container_manager": "autodetect",
		"sudo_program":      "sudo",
		"verbose":           "false",
		// container_image Fedora toolbox is a sensitive default
		"container_image":          "registry.fedoraproject.org/fedora-toolbox:latest",
		"container_name":           "my-distrobox",
		"container_always_pull":    "false",
		"non_interactive":          "false",
		"container_generate_entry": "true",
		"container_clean_path":     "false",
		"container_skip_workdir":   "false",
		"userns_nolimit":           "false",
		"rm_home":                  "false",
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
		ContainerImage:        configMap["container_image_env"],
		ContainerName:         configMap["container_name_env"],
		ContainerHostname:     configMap["container_hostname"],
		ContainerCustomHome:   configMap["container_user_custom_home"],
		ContainerHomePrefix:   configMap["container_home_prefix"],
		ContainerAlwaysPull:   toBool(configMap["container_always_pull"]),
		NonInteractive:        toBool(configMap["non_interactive"]),
		GenerateEntry:         toBool(configMap["container_generate_entry"]),
		CleanPath:             toBool(configMap["container_clean_path"]),
		SkipWorkDir:           toBool(configMap["container_skip_workdir"]),
		UsernsNoLimit:         toBool(configMap["userns_nolimit"]),
		RmCustomHome:          toBool(configMap["rm_home"]),
		ScriptsDir:            resolveScriptsDir(),
	}
}

// toBool recognizes the values the shell distrobox treats as truthy
// (`1`, `true`, `yes`, `on`, case-insensitive). Anything else, including the
// empty string, is false. The previous narrower `value == "true"` silently
// ignored `DBX_VERBOSE=1` and similar — this fixes it.
func toBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes", "on":
		return true
	default:
		return false
	}
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

	// The reference shell uses the config key `distrobox_sudo_program`; map it
	// onto our `sudo_program` slot so .conf/.distroboxrc files keep working.
	// Note: .distroboxrc is parsed as INI here, not sourced as shell.
	if v, ok := config["distrobox_sudo_program"]; ok {
		if _, has := config["sudo_program"]; !has {
			config["sudo_program"] = v
		}
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
	if value, exists := os.LookupEnv("DBX_CONTAINER_IMAGE"); exists {
		envConfig["container_image_env"] = value
	}
	if value, exists := os.LookupEnv("DBX_CONTAINER_NAME"); exists {
		envConfig["container_name_env"] = value
	}
	if value, exists := os.LookupEnv("DBX_CONTAINER_HOSTNAME"); exists {
		envConfig["container_hostname"] = value
	}
	if value, exists := os.LookupEnv("DBX_CONTAINER_CUSTOM_HOME"); exists {
		envConfig["container_user_custom_home"] = value
	}
	if value, exists := os.LookupEnv("DBX_CONTAINER_HOME_PREFIX"); exists {
		envConfig["container_home_prefix"] = value
	}
	if value, exists := os.LookupEnv("DBX_CONTAINER_ALWAYS_PULL"); exists {
		envConfig["container_always_pull"] = value
	}
	if value, exists := os.LookupEnv("DBX_NON_INTERACTIVE"); exists {
		envConfig["non_interactive"] = value
	}
	if value, exists := os.LookupEnv("DBX_CONTAINER_GENERATE_ENTRY"); exists {
		envConfig["container_generate_entry"] = value
	}
	if value, exists := os.LookupEnv("DBX_CONTAINER_CLEAN_PATH"); exists {
		envConfig["container_clean_path"] = value
	}
	if value, exists := os.LookupEnv("DBX_SKIP_WORKDIR"); exists {
		envConfig["container_skip_workdir"] = value
	}
	if value, exists := os.LookupEnv("DBX_USERNS_NOLIMIT"); exists {
		envConfig["userns_nolimit"] = value
	}
	if value, exists := os.LookupEnv("DBX_CONTAINER_RM_CUSTOM_HOME"); exists {
		envConfig["rm_home"] = value
	}

	return envConfig
}

// resolveScriptsDir returns the directory path where the helper scripts should
// be stored.
func resolveScriptsDir() string {
	// First check DBX_SCRIPTS_DIR env var
	if dir := os.Getenv("DBX_SCRIPTS_DIR"); dir != "" {
		return dir
	}

	// then check the path where main distrobox is installed
	if exe, err := os.Executable(); err == nil {
		return filepath.Dir(exe)
	}

	// Then, check host's HOME env var
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".local", "bin")
	}

	// Fallback to default path
	return "/usr/bin"
}
