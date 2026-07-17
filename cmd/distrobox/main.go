package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/89luca89/distrobox/internal/cli"
	"github.com/89luca89/distrobox/pkg/config"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.LoadValues()
	if err != nil {
		//nolint:wrapcheck // main reports errors as-is
		return err
	}

	// SIGINT register
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cmd := cli.NewRootCommand(cfg)

	// PrepareArgs lets `enter`/`ephemeral` accept distrobox flags after the
	// container name by splicing a "--" in front of the custom command.
	args := cli.PrepareArgs(cmd, cli.ResolveArgs(os.Args))

	//nolint:wrapcheck // main reports errors as-is
	return cmd.Run(ctx, args)
}
