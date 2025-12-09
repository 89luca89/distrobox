package commands

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

type RmResult struct {
	Containers []containermanager.Container
}

type RmCommand struct {
	containerManager containermanager.ContainerManager
	listCmd          *ListCommand
	prompter         ui.Prompter
}

type RmOptions struct {
	NoTTY          bool
	Force          bool
	All            bool
	RemoveHome     bool
	ContainerNames []string
}

func NewRmCommand(
	cm containermanager.ContainerManager,
	prompter ui.Prompter,
) *RmCommand {
	return &RmCommand{
		containerManager: cm,
		listCmd:          NewListCommand(cm),
		prompter:         prompter,
	}
}

func (c *RmCommand) Execute(ctx context.Context, options RmOptions) (*RmResult, error) {
	listResult, err := c.listCmd.Execute(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed while listing contaiers: %w", err)
	}

	distroboxesToRemove := getContainersToRemove(listResult.Containers, options.ContainerNames, options.All)

	var removedDistroboxes []containermanager.Container
	for _, currentDistrobox := range distroboxesToRemove {
		err := c.removeContainer(ctx, currentDistrobox, options)
		if err != nil {
			//nolint:forbidigo // waiting for the logger implementation
			fmt.Printf("error deleting %s: %s", currentDistrobox.Name, err)
		}
		removedDistroboxes = append(removedDistroboxes, currentDistrobox)
	}

	return &RmResult{Containers: removedDistroboxes}, nil
}

func (c *RmCommand) removeContainer(
	ctx context.Context,
	container containermanager.Container,
	options RmOptions,
) error {
	if strings.Contains(container.Status, "Up") && !options.NoTTY && !options.Force {
		if c.prompter.Prompt("Container is running, do you want to delete it?", false) {
			err := c.containerManager.Remove(ctx, container.Name, containermanager.RmOptions{
				Force: true,
				NoTTY: true,
			}, c.prompter)
			if err != nil {
				return fmt.Errorf("failed to remove container: %w", err)
			}
		}
		return nil
	}

	cmOptions := containermanager.RmOptions{
		NoTTY:      options.NoTTY,
		Force:      options.Force,
		All:        options.All,
		RemoveHome: options.RemoveHome,
	}
	err := c.containerManager.Remove(ctx, container.Name, cmOptions, c.prompter)
	if err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	return nil
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
