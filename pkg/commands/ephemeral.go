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

const (
	ephemeralCleanupTimeout       = 30 * time.Second
	ephemeralMaxNameGenAttempts   = 10
	ephemeralRandomNameSuffixSize = 10
)

type EphemeralOptions struct {
	CreateOptions

	// CustomCommand is the command (and its arguments) to execute inside the
	// ephemeral container instead of the default login shell. It is forwarded
	// to the underlying enter command.
	CustomCommand []string
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
		rmCmd:            NewRmCommand(cfg, cm, prompter, printer),
		printer:          printer,
	}
}

func (c *EphemeralCommand) Execute(ctx context.Context, opts EphemeralOptions) error {
	name := opts.ContainerName
	if name == "" {
		generatedName, err := c.makeUniqueRandomName(ctx, opts.DryRun)
		if err != nil {
			return fmt.Errorf("ephemeral: %w", err)
		}
		name = generatedName
	}

	// create ephemeral container
	createOpts := opts.CreateOptions
	createOpts.ContainerName = name
	// override options not relevant for creating ephemeral containers
	createOpts.GenerateEntry = false
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
		CustomCommand: opts.CustomCommand,
		DryRun:        opts.DryRun,
	}
	if _, enterErr := c.enterCmd.Execute(ctx, enterOpts); enterErr != nil {
		return fmt.Errorf("ephemeral: %w", enterErr)
	}

	return nil
}

// makeUniqueRandomName generates a random container name that does not
// collide with an existing container. When dryRun is true, the existence
// check is skipped (mirroring the rest of the dry-run pipeline, where no
// container lookup is performed).
func (c *EphemeralCommand) makeUniqueRandomName(ctx context.Context, dryRun bool) (string, error) {
	for range ephemeralMaxNameGenAttempts {
		name := makeRandomName()
		if dryRun || !c.containerManager.Exists(ctx, name) {
			return name, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique ephemeral container name after %d attempts", ephemeralMaxNameGenAttempts)
}

func makeRandomName() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	l := len(charset)
	b := make([]byte, ephemeralRandomNameSuffixSize)
	for i := range b {
		b[i] = charset[rand.IntN(l)] //nolint:gosec // cryptographic security not needed
	}
	return fmt.Sprintf("distrobox-%s", string(b))
}
