package commands

import (
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
)

//go:embed assets/desktop_entry.toml.tmpl
var desktopEntryTmpl string

const (
	defaultContainerName   = "my-distrobox"
	defaultEntryIcon       = "https://raw.githubusercontent.com/89luca89/distrobox/main/icons/terminal-distrobox-icon.svg"
	defaultContainerDistro = "terminal-distrobox-icon"
)

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
	listCommand *ListCommand
}

func NewGenerateEntryCommand(listCommand *ListCommand) *GenerateEntryCommand {
	return &GenerateEntryCommand{
		listCommand: listCommand,
	}
}

func (c *GenerateEntryCommand) Execute(
	ctx context.Context,
	opts *GenerateEntryOptions) error {
	// Determine whether is a single or all entries generation
	// If all is set, fetch the list of all containers
	// If not, use the provided container name or the default one
	var containerNames []string
	var icon string
	switch {
	case opts.All:
		// Generate entries for all containers
		listResult, err := c.listCommand.Execute(ctx)
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}

		containerNames = make([]string, 0, len(listResult.Containers))
		for _, container := range listResult.Containers {
			containerNames = append(containerNames, container.Name)
		}
		// Set icon to auto for all entries
		icon = "auto"
	case opts.ContainerName != "":
		containerNames = []string{opts.ContainerName}
		icon = opts.Icon
	default:
		containerNames = []string{defaultContainerName}
		icon = opts.Icon
	}

	if opts.Delete {
		// Delete the desktop entries for all the containers
		for _, containerName := range containerNames {
			if err := c.deleteEntry(containerName, opts.DesktopEntryBaseDir); err != nil {
				return fmt.Errorf("failed to delete desktop entry for container %s: %w", containerName, err)
			}
		}
	} else {
		// Create the desktop entries for all the containers
		for _, containerName := range containerNames {
			if err := c.createEntry(containerName, icon, opts.DesktopEntryBaseDir, opts.DistroboxPath, opts.Root); err != nil {
				return fmt.Errorf("failed to create desktop entry for container %s: %w", containerName, err)
			}
		}
	}

	return nil
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
	containerName string,
	icon string,
	desktopEntryBaseDir string,
	distroboxPath string,
	root bool,
) error {
	desktopEntryAppsDir, _, err := c.ensureDesktopEntryDirExists(desktopEntryBaseDir)
	if err != nil {
		return fmt.Errorf("failed to ensure desktop entry directories exist: %w", err)
	}

	entryFilePath := c.getEntryFilePath(desktopEntryAppsDir, containerName)
	data := c.composeDesktopEntryData(containerName, icon, distroboxPath, root)
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

// composeDesktopEntry generates the desktop entry for a single container
func (c *GenerateEntryCommand) composeDesktopEntryData(
	containerName string,
	icon string,
	distroboxPath string,
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
		"icon":           c.getDesktopIcon(icon),
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

func (c *GenerateEntryCommand) getDesktopIcon(
	icon string,
) string {
	if icon == "auto" {
		// TODO: detect the icon for the current container's distro
		return defaultEntryIcon
	}

	return icon
}
