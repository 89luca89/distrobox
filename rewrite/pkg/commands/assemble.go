package commands

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/manifest"
	"github.com/89luca89/distrobox/pkg/ui"
)

type AssembleOptions struct {
	Items []manifest.Item
	// Boxname is the name of the box to assemble
	// If specified, the Assemble command will only assemble the given box
	// If empty, the command will assemble all boxes defined in the manifest
	Boxname string
	// Delete indicates whether to delete the existing box before assembling
	// true=delete, false=create or update
	Delete  bool
	Replace bool
	Verbose bool
	DryRun  bool
}

type AssembleCommand struct {
	containermanager containermanager.ContainerManager
	createCmd        *CreateCommand
	rmCmd            *RmCommand
	enterCmd         *EnterCommand
	progress         *ui.Progress
}

func NewAssembleCommand(
	cm containermanager.ContainerManager,
	prompter *ui.Prompter,
	progress *ui.Progress,
	printer *ui.Printer,
) *AssembleCommand {
	return &AssembleCommand{
		containermanager: cm,
		createCmd:        NewCreateCommand(cm, ui.NewDevNullProgress()),
		rmCmd:            NewRmCommand(cm, prompter),
		enterCmd:         NewEnterCommand(cm, progress, printer),
		progress:         progress,
	}
}

func (ac *AssembleCommand) Execute(ctx context.Context, opts AssembleOptions) error {
	var items []manifest.Item
	if opts.Boxname != "" {
		idx := slices.IndexFunc(opts.Items, func(i manifest.Item) bool {
			return i.Name == opts.Boxname
		})
		if idx == -1 {
			return fmt.Errorf("box '%s' not found in manifest", opts.Boxname)
		}
		items = []manifest.Item{opts.Items[idx]}
	} else {
		items = opts.Items
	}

	for _, item := range items {
		switch {
		case opts.Delete:
			if err := ac.deleteItem(ctx, item, opts.DryRun); err != nil {
				return fmt.Errorf("failed to delete item '%s': %w", item.Name, err)
			}
		case opts.Replace:
			if err := ac.replaceItem(ctx, item, opts.DryRun); err != nil {
				return fmt.Errorf("failed to replace item '%s': %w", item.Name, err)
			}
		default:
			if err := ac.createItem(ctx, item, opts.DryRun); err != nil {
				return fmt.Errorf("failed to create item '%s': %w", item.Name, err)
			}
		}
	}

	return nil
}

func (ac *AssembleCommand) deleteItem(ctx context.Context, item manifest.Item, dryRun bool) error {
	ac.progress.Next("Deleting %s...", item.Name)
	opts := RmOptions{
		NoTTY:          dryRun,
		Force:          true,
		All:            false,
		RemoveHome:     false,
		ContainerNames: []string{item.Name},
	}

	_, err := ac.rmCmd.Execute(ctx, opts)
	if err != nil {
		ac.progress.Fail()
		return fmt.Errorf("failed to execute delete item '%s': %w", item.Name, err)
	}
	ac.progress.Done()
	return nil
}

func (ac *AssembleCommand) replaceItem(ctx context.Context, item manifest.Item, dryRun bool) error {
	err := ac.deleteItem(ctx, item, dryRun)
	if err != nil {
		return err
	}

	return ac.createItem(ctx, item, dryRun)
}

func (ac *AssembleCommand) createItem(ctx context.Context, item manifest.Item, dryRun bool) error {
	ac.progress.Next("Creating %s...", item.Name)
	opts := CreateOptions{
		ContainerClone:          item.Clone,
		ContainerName:           item.Name,
		ContainerImage:          item.Image,
		ContainerHostname:       item.Hostname,
		UnshareNetNs:            item.UnshareNetns,
		UnshareDevsys:           item.UnshareDevsys,
		UnshareGroups:           item.UnshareGroups,
		UnshareIpc:              item.UnshareIPC,
		UnshareProcess:          item.UnshareProcess,
		AdditionalFlags:         item.AdditionalFlags,
		AdditionalVolumes:       item.Volumes,
		AdditionalPackages:      item.AdditionalPackages,
		ContainerPreInitHook:    ac.joinHooks(item.PreInitHooks),
		ContainerInitHook:       ac.joinHooks(item.InitHooks),
		ContainerUserCustomHome: item.Home,
		Init:                    item.Init,
		Nvidia:                  item.Nvidia,
		GenerateEntry:           item.Entry,
		Rootful:                 item.Root,
		DryRun:                  dryRun,
		NonInteractive:          true,
		ContainerAlwaysPull:     item.AlwaysPull,
	}

	err := ac.createCmd.Execute(ctx, opts)
	if err != nil {
		ac.progress.Fail()
		return err
	}

	if !dryRun {
		err = ac.setupBox(ctx, item)
		if err != nil {
			ac.progress.Fail()
			return err
		}
	}

	ac.progress.Done()
	return nil
}

func (ac *AssembleCommand) joinHooks(hooks []string) string {
	sb := strings.Builder{}

	for i, hook := range hooks {
		sb.WriteString(hook)

		if i < len(hooks)-1 {
			semicolonRegex := regexp.MustCompile(`;[[:space:]]{0,1}$`)
			andAndRegex := regexp.MustCompile(`&&[[:space:]]{0,1}$`)

			separator := "  " // two spaces just because v1 does that, so it's comparable in regression tests
			if !semicolonRegex.MatchString(hook) && !andAndRegex.MatchString(hook) {
				separator = " && "
			}

			sb.WriteString(separator)
		}
	}

	return sb.String()
}

func (ac *AssembleCommand) setupBox(ctx context.Context, item manifest.Item) error {
	if item.StartNow {
		_, err := ac.enterCmd.Execute(ctx, EnterOptions{
			ContainerName: item.Name,
			NoTTY:         true,
			CustomCommand: "true", // we just want to run the init hooks, so we can skip the shell
			DryRun:        false,
		})
		if err != nil {
			return fmt.Errorf("failed to execute init hooks for item '%s': %w", item.Name, err)
		}
	}

	// validate app name to prevent command injection, since it's used in a custom command
	var validAppName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._+\-]*$`)
	for _, app := range item.ExportedApps {
		if !validAppName.MatchString(app) {
			return fmt.Errorf("invalid app name '%s' for item '%s': must be alphanumeric (with dots, underscores, hyphens)", app, item.Name)
		}
	}
	for _, app := range item.ExportedApps {
		cmd := fmt.Sprintf("distrobox-export --app %s", app)
		_, err := ac.enterCmd.Execute(ctx, EnterOptions{
			ContainerName: item.Name,
			NoTTY:         true,
			CustomCommand: cmd,
			DryRun:        false,
		})
		if err != nil {
			return fmt.Errorf("failed to export app '%s' for item '%s': %w", app, item.Name, err)
		}
	}

	// validate bin path to prevent command injection, since it's used in a custom command
	var validBinPath = regexp.MustCompile(`^/[a-zA-Z0-9._+\-/]+$`)
	if len(item.ExportedBins) > 0 && !validBinPath.MatchString(item.ExportedBinsPath) {
		return fmt.Errorf("invalid exported bins path '%s' for item '%s': must be an absolute path with alphanumeric characters, dots, underscores, or hyphens", item.ExportedBinsPath, item.Name)
	}
	// we allow slashes in bin paths, but we validate each path segment to prevent command injection
	for _, bin := range item.ExportedBins {
		if !validBinPath.MatchString(bin) {
			return fmt.Errorf("invalid bin path '%s' for item '%s': must be an absolute path with alphanumeric characters, dots, underscores, or hyphens", bin, item.Name)
		}
	}
	for _, bin := range item.ExportedBins {
		cmd := fmt.Sprintf("distrobox-export --bin %s --export-path %s", bin, item.ExportedBinsPath)
		_, err := ac.enterCmd.Execute(ctx, EnterOptions{
			ContainerName: item.Name,
			NoTTY:         true,
			CustomCommand: cmd,
			DryRun:        false,
		})
		if err != nil {
			return fmt.Errorf("failed to export bin '%s' for item '%s': %w", bin, item.Name, err)
		}
	}

	return nil
}
