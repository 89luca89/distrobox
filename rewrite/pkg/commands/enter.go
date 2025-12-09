package commands

import (
	"context"
	"fmt"

	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

type EnterResult struct{}

type EnterCommand struct {
	containerManager containermanager.ContainerManager
	options          containermanager.EnterOptions
	progress         *ui.Progress
	printer          *ui.Printer
}

func NewEnterCommand(
	cm containermanager.ContainerManager,
	options containermanager.EnterOptions,
	progress *ui.Progress,
	printer *ui.Printer,
) *EnterCommand {
	return &EnterCommand{
		containerManager: cm,
		options:          options,
		progress:         progress,
		printer:          printer,
	}
}

func (c *EnterCommand) Execute(ctx context.Context) (*EnterResult, error) {
	err := c.containerManager.Enter(ctx, c.options, c.progress, c.printer)
	if err != nil {
		return nil, fmt.Errorf("failed to enter the container: %w", err)
	}

	return &EnterResult{}, nil
}
