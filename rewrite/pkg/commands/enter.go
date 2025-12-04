package commands

import (
	"context"
	"fmt"

	"github.com/89luca89/distrobox/pkg/containermanager"
)

type EnterResult struct{}

type EnterCommand struct {
	containerManager containermanager.ContainerManager
	options          containermanager.EnterOptions
}

func NewEnterCommand(cm containermanager.ContainerManager, options containermanager.EnterOptions) *EnterCommand {
	return &EnterCommand{
		containerManager: cm,
		options:          options,
	}
}

func (c *EnterCommand) Execute(ctx context.Context) (*EnterResult, error) {
	err := c.containerManager.Enter(ctx, c.options)
	if err != nil {
		return nil, fmt.Errorf("failed to enter the container: %w", err)
	}

	return &EnterResult{}, nil
}
