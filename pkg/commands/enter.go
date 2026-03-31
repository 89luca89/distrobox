package commands

import (
	"context"
	"fmt"

	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

type EnterResult struct{}

type EnterOptions struct {
	ContainerName   string
	AdditionalFlags string
	CustomCommand   string
	DryRun          bool
	NoTTY           bool
	Verbose         bool
	CleanPath       bool
}

type EnterCommand struct {
	cfg              *config.Values
	containerManager containermanager.ContainerManager
	progress         *ui.Progress
	printer          *ui.Printer
}

func NewEnterCommand(
	cfg *config.Values,
	cm containermanager.ContainerManager,
	progress *ui.Progress,
	printer *ui.Printer,
) *EnterCommand {
	return &EnterCommand{
		cfg:              cfg,
		containerManager: cm,
		progress:         progress,
		printer:          printer,
	}
}

func (c *EnterCommand) Execute(ctx context.Context, opts EnterOptions) (*EnterResult, error) {
	cmdOpts := containermanager.EnterOptions{
		ContainerName:   opts.ContainerName,
		AdditionalFlags: opts.AdditionalFlags,
		CustomCommand:   opts.CustomCommand,
		DryRun:          opts.DryRun,
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
