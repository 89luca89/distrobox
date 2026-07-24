// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of the distrobox project:
//    https://github.com/89luca89/distrobox
//
// Copyright (C) 2021 distrobox contributors
//
// distrobox is free software; you can redistribute it and/or modify it
// under the terms of the GNU General Public License version 3
// as published by the Free Software Foundation.
//
// distrobox is distributed in the hope that it will be useful, but
// WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU
// General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with distrobox; if not, see <http://www.gnu.org/licenses/>.

package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/89luca89/distrobox/internal/rootful"
	"github.com/89luca89/distrobox/pkg/commands"
	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/manifest"
	"github.com/89luca89/distrobox/pkg/ui"
)

func newAssembleCommand(cfg *config.Values) *cli.Command {
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
	distrobox assemble create /path/to/file.ini
	distrobox assemble rm --file /path/to/file.ini
	distrobox assemble rm /path/to/file.ini
	distrobox assemble create --replace --file /path/to/file.ini

Options:

	--file:			path or URL to the distrobox manifest/ini file
				(may also be supplied as a positional argument; --file takes precedence)
	--name/-n:		run against a single entry in the manifest/ini file
	--replace/-R:		replace already existing distroboxes with matching names
	--dry-run/-d:		only print the container manager command generated
	--verbose/-v:		show more verbosity
	--version/-V:		show version
`,
		Commands: []*cli.Command{
			{
				Name:      "create",
				ArgsUsage: "[manifest-file]",
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
					return assembleAction(ctx, cmd, cfg, false)
				},
			},
			{
				Name:      "rm",
				ArgsUsage: "[manifest-file]",
				Flags: []cli.Flag{
					fileFlag,
					nameFlag,
					dryRunFlag,
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return assembleAction(ctx, cmd, cfg, true)
				},
			},
		},
	}
}

// defaultManifestPath is the path used when neither --file nor a positional
// argument is provided to `distrobox assemble`.
const defaultManifestPath = "./distrobox.ini"

// resolveManifestPath returns the manifest file path to use for an assemble
// invocation. Precedence is:
//  1. the value of the --file flag, if non-empty;
//  2. the first positional argument, if any;
//  3. the default manifest path ("./distrobox.ini").
//
// This mirrors the behavior of the original Bash `distrobox-assemble` script
// while keeping the explicit `--file` flag dominant over the implicit
// positional argument.
func resolveManifestPath(flagValue string, positional []string) string {
	if flagValue != "" {
		return flagValue
	}
	if len(positional) > 0 && positional[0] != "" {
		return positional[0]
	}
	return defaultManifestPath
}

func assembleAction(ctx context.Context, cmd *cli.Command, cfg *config.Values, deleteFlag bool) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	manifestFilePath := resolveManifestPath(cmd.String("file"), cmd.Args().Slice())

	manifest, err := manifest.Parse(ctx, manifestFilePath)
	if err != nil {
		return fmt.Errorf("failed to parse manifest file: %w", err)
	}

	// if at least one item in the manifest requires root, validate sudo before
	// proceeding — unless we are already uid 0, which needs no elevation.
	for _, item := range manifest {
		if item.Root && os.Getuid() != 0 {
			if err := rootful.Validate(ctx, cmd.String("sudo-command")); err != nil {
				return fmt.Errorf("cannot run in root mode: %w", err)
			}
			break
		}
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

	assembleCmd := commands.NewAssembleCommand(cfg, containerManager, prompter, progress, printer)

	err = assembleCmd.Execute(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to execute assemble command: %w", err)
	}
	return nil
}
