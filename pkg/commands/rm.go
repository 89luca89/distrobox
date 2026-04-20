package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/89luca89/distrobox/internal/userenv"
	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

type RmResult struct {
	Containers []containermanager.Container
}

type RmCommand struct {
	cfg              *config.Values
	containerManager containermanager.ContainerManager
	listCmd          *ListCommand
	generateEntryCmd *GenerateEntryCommand
	prompter         *ui.Prompter
}

type RmOptions struct {
	NoTTY          bool
	Force          bool
	All            bool
	RemoveHome     bool
	ContainerNames []string
}

func NewRmCommand(
	cfg *config.Values,
	cm containermanager.ContainerManager,
	prompter *ui.Prompter,
) *RmCommand {
	listCmd := NewListCommand(cfg, cm)
	generateEntryCmd := NewGenerateEntryCommand(cfg, listCmd)
	return &RmCommand{
		cfg:              cfg,
		containerManager: cm,
		listCmd:          listCmd,
		generateEntryCmd: generateEntryCmd,
		prompter:         prompter,
	}
}

func removeValue(slice []string, valueToRemove string) []string {
	for i, value := range slice {
		if value == valueToRemove {
			return slices.Delete(slice, i, i+1)
		}
	}
	return slice
}

func (c *RmCommand) Execute(ctx context.Context, options RmOptions) (*RmResult, error) {
	if !options.NoTTY && c.prompter == nil {
		return nil, errors.New("prompter is required for interactive mode")
	}

	listResult, err := c.listCmd.Execute(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed while listing contaiers: %w", err)
	}

	explicitelyRequested := options.ContainerNames
	distroboxesToRemove := getContainersToRemove(listResult.Containers, options.ContainerNames, options.All)

	userEnv := userenv.LoadUserEnvironment(ctx)
	userHome := userEnv.Home

	// Remove containers
	var removedDistroboxes []containermanager.Container
	for _, currentDistrobox := range distroboxesToRemove {
		explicitelyRequested = removeValue(explicitelyRequested, currentDistrobox.Name)

		err := c.removeContainer(ctx, currentDistrobox, options.Force, options.NoTTY, userHome)
		if err != nil {
			//nolint:forbidigo // waiting for the logger implementation
			fmt.Printf("error deleting %s: %s", currentDistrobox.Name, err)
		}
		removedDistroboxes = append(removedDistroboxes, currentDistrobox)
	}

	// Clean up exported files of all remaining explicitely requested distroboxes,
	// even if the container doesn't exist anymore
	for _, containerName := range explicitelyRequested {
		c.cleanup(ctx, userHome, containerName)
	}

	return &RmResult{Containers: removedDistroboxes}, nil
}

func (c *RmCommand) removeContainer(
	ctx context.Context,
	container containermanager.Container,
	force bool,
	noTTY bool,
	userHome string,
) error {
	forceRemove := force
	if !forceRemove && !noTTY && strings.Contains(container.Status, "Up") {
		if c.prompter.Prompt("Container is running, do you want to force delete it?", false) {
			forceRemove = true
		} else {
			return nil
		}
	}

	inspectOutput, err := c.containerManager.InspectContainer(ctx, container.Name)
	if err != nil {
		return fmt.Errorf("error inspecting the container: %w", err)
	}

	removeHome := false
	if !noTTY && inspectOutput.ContainerHome != userHome {
		question := fmt.Sprintf(
			"Do you really want to remove custom home of container %s (%s)?",
			container.Name,
			inspectOutput.ContainerHome,
		)
		answer := c.prompter.Prompt(question, false)
		removeHome = answer
	}

	cmOptions := containermanager.RmOptions{
		Force:         forceRemove,
		RemoveHome:    removeHome,
		ContainerHome: inspectOutput.ContainerHome,
	}
	err = c.containerManager.Remove(ctx, container.Name, cmOptions)
	if err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	c.cleanup(ctx, userHome, container.Name)

	return nil
}

func (c *RmCommand) cleanup(ctx context.Context, userHome, containerName string) {
	if containerName == "" {
		panic("Refusing to run cleanup for empty container name")
	}

	bins := findExportedBinaries(userHome, containerName)
	desktopApps := findExportedDesktopApps(userHome, containerName)

	toDelete := slices.Concat(bins, desktopApps)

	for _, path := range toDelete {
		if err := os.Remove(path); err != nil {
			//nolint:forbidigo // FIXME: use logger instead of fmt.Printf when available
			fmt.Printf("warning: failed to remove file '%s': %s\n", path, err)
		}
	}

	err := c.generateEntryCmd.Execute(
		ctx,
		&GenerateEntryOptions{
			ContainerName: containerName,
			Delete:        true,
			// TODO: handle verbose
			Verbose: false,
		},
	)
	if err != nil {
		//nolint:forbidigo // FIXME: use logger instead of fmt.Printf when available
		fmt.Printf("warning: failed to remove desktop entry for container '%s': %s\n", containerName, err)
	}
}

func getContainersToRemove(
	containers []containermanager.Container,
	names []string,
	all bool,
) []containermanager.Container {
	if all {
		return containers
	}

	var filtered []containermanager.Container
	for _, container := range containers {
		if slices.ContainsFunc(names, func(name string) bool {
			return container.Name == name
		}) {
			filtered = append(filtered, container)
		}
	}

	return filtered
}

func findExportedBinaries(userHome, containerName string) []string {
	binDir := filepath.Join(userHome, ".local", "bin")

	entries, err := os.ReadDir(binDir)
	if err != nil {
		return nil
	}

	var files []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := filepath.Join(binDir, entry.Name())

		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		content := string(data)
		if strings.Contains(content, "# distrobox_binary") &&
			strings.Contains(content, "# name: "+containerName+"\n") {
			absPath, err := filepath.Abs(path)
			if err != nil {
				continue
			}

			files = append(files, absPath)
		}
	}

	return files
}

func findExportedDesktopApps(userHome, containerName string) []string {
	appsPattern := filepath.Join(userHome, ".local", "share", "applications", containerName+"*")

	matches, err := filepath.Glob(appsPattern)
	if err != nil {
		//nolint:forbidigo // FIXME: use logger instead of fmt.Printf when available
		fmt.Printf("warning: failed to glob desktop apps: %s\n", err)
		return []string{}
	}

	var files []string

	for _, desktopFile := range matches {
		iconValue, ok := parseDesktopExport(desktopFile, containerName)
		if !ok {
			continue
		}

		absDesktop, err := filepath.Abs(desktopFile)
		if err != nil {
			continue
		}

		files = append(files, absDesktop)

		if iconValue != "" {
			files = append(files, findIconFiles(userHome, iconValue)...)
		}
	}

	return files
}

func parseDesktopExport(desktopFile, containerName string) (string, bool) {
	data, err := os.ReadFile(desktopFile)
	if err != nil {
		return "", false
	}

	hasExecMatch := false
	var iconValue string

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "Exec=") && strings.Contains(line, containerName+" ") {
			hasExecMatch = true
		}

		if strings.HasPrefix(line, "Icon=") {
			iconValue = strings.TrimPrefix(line, "Icon=")
		}
	}

	return iconValue, hasExecMatch
}

func findIconFiles(userHome, iconName string) []string {
	iconsDir := filepath.Join(userHome, ".local", "share", "icons")
	iconPrefix := iconName + "."

	var files []string

	_ = filepath.WalkDir(iconsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // skip unreadable directories
		}

		if d.IsDir() {
			return nil
		}

		if strings.HasPrefix(d.Name(), iconPrefix) {
			absIcon, err := filepath.Abs(path)
			if err == nil {
				files = append(files, absIcon)
			}
		}

		return nil
	})

	return files
}
