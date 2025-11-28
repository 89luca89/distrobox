package main

import (
	"context"
	"log"
	"os"

	"github.com/89luca89/distrobox/internal/cli"
)

func main() {
	cmd := cli.NewRootCommand()

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
