package commands

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/89luca89/distrobox/internal/prompt"
	"github.com/89luca89/distrobox/pkg/containermanager"
)

type RmResult struct {
	Containers []containermanager.Container
}

type RmCommand struct {
	containerManager containermanager.ContainerManager
	options          containermanager.RmOptions
	prompter         prompt.Prompter
	listCmd          *ListCommand
}

func NewRmCommand(
	cm containermanager.ContainerManager,
	options containermanager.RmOptions,
	prompter prompt.Prompter,
) *RmCommand {
	return &RmCommand{
		containerManager: cm,
		options:          options,
		prompter:         prompter,
		listCmd:          NewListCommand(cm),
	}
}

func (c *RmCommand) Execute(ctx context.Context, containerNames []string) (*RmResult, error) {
	listResult, err := c.listCmd.Execute(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed while listing contaiers: %w", err)
	}

	distroboxesToRemove := getContainersToRemove(listResult.Containers, containerNames, c.options.All)

	var removedDistroboxes []containermanager.Container
	for _, currentDistrobox := range distroboxesToRemove {
		err := c.removeContainer(ctx, currentDistrobox, c.options)
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
	options containermanager.RmOptions,
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

	err := c.containerManager.Remove(ctx, container.Name, options, c.prompter)
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
