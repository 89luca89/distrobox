package commands

import (
	"context"
	"fmt"

	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

type EnterResult struct{}

type EnterOptions struct {
	ContainerName   string
	AdditionalFlags string
	NoTTY           bool
	Verbose         bool
	CleanPath       bool
}

type EnterCommand struct {
	containerManager containermanager.ContainerManager
	progress         *ui.Progress
	printer          *ui.Printer
}

func NewEnterCommand(
	cm containermanager.ContainerManager,
	progress *ui.Progress,
	printer *ui.Printer,
) *EnterCommand {
	return &EnterCommand{
		containerManager: cm,
		progress:         progress,
		printer:          printer,
	}
}

func (c *EnterCommand) Execute(ctx context.Context, opts EnterOptions) (*EnterResult, error) {
	cmdOpts := containermanager.EnterOptions{
		ContainerName:   opts.ContainerName,
		AdditionalFlags: opts.AdditionalFlags,
		NoTTY:           opts.NoTTY,
		Verbose:         opts.Verbose,
		CleanPath:       opts.CleanPath,
	}

	err := c.containerManager.Enter(ctx, cmdOpts, c.progress, c.printer)
	if err != nil {
		return nil, fmt.Errorf("failed to enter the container: %w", err)
	}

	return &EnterResult{}, nil
}
