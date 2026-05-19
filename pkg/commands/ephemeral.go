package commands

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

const ephemeralCleanupTimeout = 30 * time.Second

type EphemeralOptions struct {
	CreateOptions

	DryRun bool
}

type EphemeralCommand struct {
	cfg              *config.Values
	containerManager containermanager.ContainerManager
	createCmd        *CreateCommand
	enterCmd         *EnterCommand
	rmCmd            *RmCommand
	printer          *ui.Printer
}

func NewEphemeralCommand(
	cfg *config.Values,
	cm containermanager.ContainerManager,
	progress *ui.Progress,
	printer *ui.Printer,
	prompter *ui.Prompter,
) *EphemeralCommand {
	return &EphemeralCommand{
		cfg:              cfg,
		containerManager: cm,
		createCmd:        NewCreateCommand(cfg, cm, progress, prompter),
		enterCmd:         NewEnterCommand(cfg, cm, progress, printer),
		rmCmd:            NewRmCommand(cfg, cm, printer, prompter),
		printer:          printer,
	}
}

func (c *EphemeralCommand) Execute(ctx context.Context, opts EphemeralOptions) error {
	name := opts.ContainerName
	if name == "" {
		name = makeRandomName()
	}

	// create ephemeral container
	createOpts := opts.CreateOptions
	createOpts.ContainerName = name
	// override options not relevant for creating ephemeral containers
	createOpts.GenerateEntry = false
	createOpts.DryRun = opts.DryRun
	createOpts.NonInteractive = true
	if _, createErr := c.createCmd.Execute(ctx, createOpts); createErr != nil {
		return fmt.Errorf("ephemeral: %w", createErr)
	}

	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), ephemeralCleanupTimeout)
		defer cancel()
		rmOpts := RmOptions{
			ContainerNames: []string{name},
			Force:          true,
			NoTTY:          true,
		}
		if _, rmErr := c.rmCmd.Execute(cleanupCtx, rmOpts); rmErr != nil {
			c.printer.PrintWarningln("warning: %s: %s", name, rmErr)
		}
	}()

	// enter into it
	enterOpts := EnterOptions{
		ContainerName: name,
		DryRun:        opts.DryRun,
		// TODO: handle enter command
	}
	if _, enterErr := c.enterCmd.Execute(ctx, enterOpts); enterErr != nil {
		return fmt.Errorf("ephemeral: %w", enterErr)
	}

	return nil
}

func makeRandomName() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	l := len(charset)
	b := make([]byte, 10) //nolint:mnd // length of random part
	for i := range b {
		b[i] = charset[rand.IntN(l)] //nolint:gosec // cryptographic security not needed
	}
	// FIXME: avoid collisions
	return fmt.Sprintf("distrobox-%s", string(b))
}
