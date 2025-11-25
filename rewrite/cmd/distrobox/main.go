package main

import (
	"context"
	"os"

	"github.com/urfave/cli/v3"
)

func main() {
	(&cli.Command{}).Run(context.Background(), os.Args)
}
