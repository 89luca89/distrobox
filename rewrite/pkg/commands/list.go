package commands

import (
	"context"
	"fmt"

	"github.com/89luca89/distrobox/pkg/containermanager"
)

type ListResult struct {
	Containers []containermanager.Container
}

type ListCommand struct {
	containerManager containermanager.ContainerManager
}

func NewListCommand(cm containermanager.ContainerManager) *ListCommand {
	return &ListCommand{
		containerManager: cm,
	}
}

func (c *ListCommand) Execute(ctx context.Context) (*ListResult, error) {
	containers, err := c.containerManager.ListContainers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed while listing contaiers: %w", err)
	}

	var distroboxes []containermanager.Container
	for _, container := range containers {
		if container.IsDistrobox() {
			distroboxes = append(distroboxes, container)
		}
	}

	return &ListResult{Containers: distroboxes}, nil
}

