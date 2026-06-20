package commands

import (
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/89luca89/distrobox/internal/userenv"
	pkgconfig "github.com/89luca89/distrobox/pkg/config"
)

//go:embed assets/desktop_entry.toml.tmpl
var desktopEntryTmpl string

const (
	defaultContainerName = "my-distrobox"
	defaultEntryIcon     = "https://raw.githubusercontent.com/89luca89/distrobox/main/icons/terminal-distrobox-icon.svg"
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
	cfg         *pkgconfig.Values
	listCommand *ListCommand
}

func NewGenerateEntryCommand(cfg *pkgconfig.Values, listCommand *ListCommand) *GenerateEntryCommand {
	return &GenerateEntryCommand{
		cfg:         cfg,
		listCommand: listCommand,
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
		if err := c.createEntry(containerName, icon, distroHint, desktopEntryBaseDir, distroboxPath, opts.Root); err != nil {
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
			for _, container := range listResult.Containers {
				if container.Name == opts.ContainerName {
					images[opts.ContainerName] = container.Image
					break
				}
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
	containerName string,
	icon string,
	distroHint string,
	desktopEntryBaseDir string,
	distroboxPath string,
	root bool,
) error {
	desktopEntryAppsDir, _, err := c.ensureDesktopEntryDirExists(desktopEntryBaseDir)
	if err != nil {
		return fmt.Errorf("failed to ensure desktop entry directories exist: %w", err)
	}

	entryFilePath := c.getEntryFilePath(desktopEntryAppsDir, containerName)
	data := c.composeDesktopEntryData(containerName, icon, distroHint, distroboxPath, root)
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
	containerName string,
	icon string,
	distroHint string,
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
		"icon":           c.getDesktopIcon(icon, distroHint),
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

// getDesktopIcon resolves the icon URL written to the desktop entry.
//
// When the caller passes a concrete icon it is returned unchanged. When the
// special value "auto" is used, distroHint (typically the container's image
// name, falling back to its name) is matched against the known distro icon
// map. The generic terminal icon is returned if no distro can be detected.
func (c *GenerateEntryCommand) getDesktopIcon(icon, distroHint string) string {
	if icon != "auto" {
		return icon
	}

	if url := lookupDistroIcon(distroHint); url != "" {
		return url
	}
	return defaultEntryIcon
}
