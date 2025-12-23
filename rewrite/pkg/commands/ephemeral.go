package commands

import (
	"context"
	"fmt"
	"math/rand/v2"

	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

type EphemeralOptions struct {
	CreateOptions
}

type EphemeralCommand struct {
	containerManager containermanager.ContainerManager
	createCmd        *CreateCommand
	enterCmd         *EnterCommand
	rmCmd            *RmCommand
}

func NewEphemeralCommand(
	cm containermanager.ContainerManager,
	progress *ui.Progress,
	printer *ui.Printer,
) *EphemeralCommand {
	return &EphemeralCommand{
		containerManager: cm,
		createCmd:        NewCreateCommand(cm, progress),
		enterCmd:         NewEnterCommand(cm, progress, printer),
		rmCmd:            NewRmCommand(cm, nil),
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
	createOpts.DryRun = false
	// TODO: pull image if needed
	// The feature is still a todo in the CreateCommand. When implemented,
	// remember to set it here as well.
	if err := c.createCmd.Execute(ctx, createOpts); err != nil {
		return fmt.Errorf("failed to create ephemeral container: %w", err)
	}

	// enter into it
	enterOpts := EnterOptions{
		ContainerName: name,
		// TODO: handle enter command
	}
	if _, err := c.enterCmd.Execute(ctx, enterOpts); err != nil {
		return fmt.Errorf("failed to enter ephemeral container: %w", err)
	}

	// remove it
	rmOpts := RmOptions{
		ContainerNames: []string{name},
		Force:          true,
		NoTTY:          true,
		All:            false,
	}
	if _, err := c.rmCmd.Execute(ctx, rmOpts); err != nil {
		return fmt.Errorf("failed to remove ephemeral container: %w", err)
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
