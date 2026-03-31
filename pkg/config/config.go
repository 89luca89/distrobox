package config

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
