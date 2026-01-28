package main

import (
	"context"
	"fmt"
	"os"

	"github.com/fuabioo/xlq/internal/cli"
)

var (
	version = ""
	commit  = ""
	date    = ""
)

func main() {
	if err := cli.Execute(context.Background(),
		version,
		commit,
		date,
	); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
