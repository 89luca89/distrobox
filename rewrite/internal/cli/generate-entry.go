package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"

	"github.com/89luca89/distrobox/pkg/commands"
	"github.com/89luca89/distrobox/pkg/containermanager"
)

func newGenerateEntryCommand() *cli.Command {
	return &cli.Command{
		Name:    "generate-entry",
		Usage:   "Generate or delete distrobox entries",
		Version: "1.0.0",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "delete",
				Aliases: []string{"d"},
				Usage:   "delete the entry",
			},
			&cli.StringFlag{
				Name:    "icon",
				Aliases: []string{"i"},
				Usage:   "specify a custom icon (default auto)",
				Value:   "auto",
			},
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
				Usage:   "perform for all distroboxes",
			},
			&cli.BoolFlag{
				Name:    "root",
				Aliases: []string{"r"},
				Usage:   "perform on rootful distroboxes",
			},
		},
		ArgsUsage: "container-name",
		Action:    generateEntryAction,
	}
}

func generateEntryAction(ctx context.Context, cmd *cli.Command) error {
	// The current executable is used as distrobox path
	distroboxPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get distrobox executable path: %w", err)
	}

	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	listCmd := commands.NewListCommand(containerManager)

	opts := &commands.GenerateEntryOptions{
		Verbose:             cmd.Bool("verbose"),
		Delete:              cmd.Bool("delete"),
		Root:                cmd.Bool("root"),
		DesktopEntryBaseDir: getDesktopEntryDir(),
		DistroboxPath:       distroboxPath,
	}
	if cmd.Bool("all") {
		opts.All = true
	} else {
		opts.ContainerName = cmd.Args().First()
		opts.Icon = cmd.String("icon")
	}

	genEntryCmd := commands.NewGenerateEntryCommand(listCmd)

	err = genEntryCmd.Execute(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to execute generate entry command: %w", err)
	}

	return nil
}

// getDesktopEntryDir resolves the system path for the desktop entry file
func getDesktopEntryDir() string {
	xdgDataHome := os.Getenv("XDG_DATA_HOME")
	if xdgDataHome == "" {
		home := os.Getenv("HOME")
		return filepath.Join(home, ".local", "share")
	}
	return xdgDataHome
}
