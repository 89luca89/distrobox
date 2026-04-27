package commands

import (
	"context"
	"fmt"
	"slices"
  "strings"

	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager"
)

type ListResult struct {
	Containers []containermanager.Container
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

	slices.SortFunc(distroboxes, func(a, b containermanager.Container) int {
		return strings.Compare(a.Name, b.Name)
	})

	return &ListResult{Containers: distroboxes}, nil
}
