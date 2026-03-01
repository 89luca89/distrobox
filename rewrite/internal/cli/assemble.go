package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/89luca89/distrobox/pkg/commands"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/manifest"
	"github.com/89luca89/distrobox/pkg/ui"
)

func newAssembleCommand() *cli.Command {
	fileFlag := &cli.StringFlag{Name: "file", Usage: "path or URL to the distrobox manifest/ini file"}
	nameFlag := &cli.StringFlag{
		Name:    "name",
		Aliases: []string{"n"},
		Usage:   "run against a single entry in the manifest/ini file",
	}
	dryRunFlag := &cli.BoolFlag{
		Name:    "dry-run",
		Aliases: []string{"d"},
		Usage:   "only print the container manager command generated",
	}

	return &cli.Command{
		Name: "assemble",
		UsageText: `
	distrobox assemble create
	distrobox assemble rm
	distrobox assemble create --file /path/to/file.ini
	distrobox assemble rm --file /path/to/file.ini
	distrobox assemble create --replace --file /path/to/file.ini

Options:

	--file:			path or URL to the distrobox manifest/ini file
	--name/-n:		run against a single entry in the manifest/ini file
	--replace/-R:		replace already existing distroboxes with matching names
	--dry-run/-d:		only print the container manager command generated
	--verbose/-v:		show more verbosity
	--version/-V:		show version
`,
		Commands: []*cli.Command{
			{
				Name: "create",
				Flags: []cli.Flag{
					fileFlag,
					nameFlag,
					&cli.BoolFlag{
						Name:    "replace",
						Aliases: []string{"R"},
						Usage:   "replace already existing distroboxes with matching names",
					},
					dryRunFlag,
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return assembleAction(ctx, cmd, false)
				},
			},
			{
				Name: "rm",
				Flags: []cli.Flag{
					fileFlag,
					nameFlag,
					dryRunFlag,
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return assembleAction(ctx, cmd, true)
				},
			},
		},
	}
}

func assembleAction(ctx context.Context, cmd *cli.Command, deleteFlag bool) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	// TODO: handle file name as a positional argument
	// https://github.com/89luca89/distrobox-next/blob/07b3abf2015effafc5596b9dc7f02c35a17eb8a7/distrobox-assemble#L205

	manifestFilePath := cmd.String("file")
	if manifestFilePath == "" {
		manifestFilePath = "./distrobox.ini"
	}

	manifest, err := manifest.Parse(ctx, manifestFilePath)
	if err != nil {
		return fmt.Errorf("failed to parse manifest file: %w", err)
	}

	opts := commands.AssembleOptions{
		Items:   manifest,
		Boxname: cmd.String("name"),
		DryRun:  cmd.Bool("dry-run"),
	}
	if deleteFlag {
		opts.Delete = true
	} else {
		opts.Replace = cmd.Bool("replace")
	}

	prompter := ui.NewPrompter(*bufio.NewReader(os.Stdin), os.Stdout)
	progress := ui.NewProgress(os.Stderr)
	printer := ui.NewPrinter(os.Stdout, true)

	assembleCmd := commands.NewAssembleCommand(containerManager, prompter, progress, printer)

	err = assembleCmd.Execute(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to execute assemble command: %w", err)
	}
	return nil
}
