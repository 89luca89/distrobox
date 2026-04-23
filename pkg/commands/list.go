package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager"
)

type ListResult struct {
	Containers []containermanager.Container
}

var ErrListContainerNotFound = errors.New("cannot find distrobox")

type ListOptions struct {
	ContainerName string
}

type ListCommand struct {
	cfg              *config.Values
	containerManager containermanager.ContainerManager
}

func NewListCommand(cfg *config.Values, cm containermanager.ContainerManager) *ListCommand {
	return &ListCommand{
		cfg:              cfg,
		containerManager: cm,
	}
}

func (c *ListCommand) Execute(ctx context.Context, opts *ListOptions) (*ListResult, error) {
	containers, err := c.containerManager.ListContainers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed while listing containers: %w", err)
	}

	containerName := ""
	if opts != nil {
		containerName = opts.ContainerName
	}

	var distroboxes []containermanager.Container
	for _, container := range containers {
		if container.IsDistrobox() && (containerName == "" || container.Name == containerName) {
			distroboxes = append(distroboxes, container)
		}
	}

	if containerName != "" && len(distroboxes) == 0 {
		return nil, fmt.Errorf("%w %q", ErrListContainerNotFound, containerName)
	}

	return &ListResult{Containers: distroboxes}, nil
}
