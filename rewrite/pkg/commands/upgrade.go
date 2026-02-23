package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

//nolint:lll // upgrade command mirrors the shell version
const upgradeCommand = `sh -c "command -v su-exec 2>/dev/null && su-exec root /usr/bin/entrypoint --upgrade || command -v doas 2>/dev/null && doas /usr/bin/entrypoint --upgrade || sudo -S /usr/bin/entrypoint --upgrade"`

type UpgradeOptions struct {
	ContainerNames []string
	All            bool
	Running        bool
	NonInteractive bool
}

type UpgradeCommand struct {
	containerManager containermanager.ContainerManager
	listCmd          *ListCommand
	enterCmd         *EnterCommand
	prompter         *ui.Prompter
}

var ErrUpgradeAbortedByUser = errors.New("upgrade operation aborted by user")

func NewUpgradeCommand(
	cm containermanager.ContainerManager,
	progress *ui.Progress,
	printer *ui.Printer,
	prompter *ui.Prompter,
) *UpgradeCommand {
	return &UpgradeCommand{
		containerManager: cm,
		listCmd:          NewListCommand(cm),
		enterCmd:         NewEnterCommand(cm, progress, printer),
		prompter:         prompter,
	}
}

func (c *UpgradeCommand) Execute(ctx context.Context, opts *UpgradeOptions) error {
	var containerNames []string

	switch {
	case opts.All, opts.Running:
		containers, err := c.listCmd.Execute(ctx)
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}

		if len(containers.Containers) == 0 {
			return ErrEmptyContainerList
		}

		containerNames = make([]string, 0, len(containers.Containers))
		for _, container := range containers.Containers {
			if opts.Running && !container.IsRunning() {
				continue
			}

			containerNames = append(containerNames, container.Name)
		}

		if len(containerNames) == 0 {
			return ErrEmptyContainerList
		}
	case len(opts.ContainerNames) > 0:
		containerNames = opts.ContainerNames
	default:
		containerNames = []string{DefaultCreateContainerName}
	}

	proceed := opts.NonInteractive || c.canProceed(containerNames)

	if !proceed {
		return ErrUpgradeAbortedByUser
	}

	var lastErr error

	for _, name := range containerNames {
		if err := c.upgradeContainer(ctx, name); err != nil {
			//nolint:forbidigo // FIXME: waiting for the logger implementation
			fmt.Printf("error upgrading %s: %s\n", name, err)

			lastErr = err

			continue
		}
	}

	return lastErr
}

func (c *UpgradeCommand) upgradeContainer(ctx context.Context, name string) error {
	enterOpts := EnterOptions{
		ContainerName: name,
		CustomCommand: upgradeCommand,
	}

	if _, err := c.enterCmd.Execute(ctx, enterOpts); err != nil {
		return fmt.Errorf("failed to upgrade container %s: %w", name, err)
	}

	return nil
}

func (c *UpgradeCommand) canProceed(containerNames []string) bool {
	return c.prompter.Prompt(
		fmt.Sprintf("Do you really want to upgrade %s?", containerNames),
		true,
	)
}
