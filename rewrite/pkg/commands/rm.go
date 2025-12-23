package commands

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/89luca89/distrobox/internal/userenv"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

type RmResult struct {
	Containers []containermanager.Container
}

type RmCommand struct {
	containerManager containermanager.ContainerManager
	listCmd          *ListCommand
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
	cm containermanager.ContainerManager,
	prompter *ui.Prompter,
) *RmCommand {
	return &RmCommand{
		containerManager: cm,
		listCmd:          NewListCommand(cm),
		prompter:         prompter,
	}
}

func (c *RmCommand) Execute(ctx context.Context, options RmOptions) (*RmResult, error) {
	if !options.NoTTY && c.prompter == nil {
		return nil, errors.New("prompter is required for interactive mode")
	}

	listResult, err := c.listCmd.Execute(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed while listing contaiers: %w", err)
	}

	distroboxesToRemove := getContainersToRemove(listResult.Containers, options.ContainerNames, options.All)

	userEnv := userenv.LoadUserEnvironment(ctx)
	userHome := userEnv.Home

	var removedDistroboxes []containermanager.Container
	for _, currentDistrobox := range distroboxesToRemove {
		err := c.removeContainer(ctx, currentDistrobox, options.Force, options.NoTTY, userHome)
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
		Force:      forceRemove,
		RemoveHome: removeHome,
	}
	err = c.containerManager.Remove(ctx, container.Name, cmOptions)
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
