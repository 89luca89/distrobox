package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

type StopCommand struct {
	cfg              *config.Values
	containerManager containermanager.ContainerManager
	listCmd          *ListCommand
	prompter         *ui.Prompter
}

type StopOptions struct {
	ContainerNames []string
	NonInteractive bool
	All            bool
}

var ErrEmptyContainerList = errors.New("cannot find containers to stop")
var ErrStopAbortedByUserError = errors.New("stop operation aborted by user")

func NewStopCommand(cfg *config.Values, containerManager containermanager.ContainerManager, prompter *ui.Prompter) *StopCommand {
	return &StopCommand{
		cfg:              cfg,
		containerManager: containerManager,
		listCmd:          NewListCommand(cfg, containerManager),
		prompter:         prompter,
	}
}

func (c *StopCommand) Execute(ctx context.Context, opts *StopOptions) error {
	var containerNames []string
	switch {
	case opts.All:
		containers, err := c.listCmd.Execute(ctx)
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}
		if len(containers.Containers) == 0 {
			return ErrEmptyContainerList
		}
		containerNames = make([]string, 0, len(containers.Containers))
		for _, container := range containers.Containers {
			containerNames = append(containerNames, container.Name)
		}
	case len(opts.ContainerNames) > 0:
		containerNames = opts.ContainerNames
	default:
		containerNames = []string{DefaultCreateContainerName}
	}

	proceed := opts.NonInteractive || c.prompter.Prompt(
		fmt.Sprintf("Do you really want to stop %s?", containerNames),
		true,
	)

	if !proceed {
		return ErrStopAbortedByUserError
	}

	err := c.containerManager.Stop(ctx, containerNames)
	if err != nil {
		return fmt.Errorf("failed to stop containers: %w", err)
	}

	return nil
}
