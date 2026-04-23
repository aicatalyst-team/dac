package main

import (
	"context"
	"fmt"
	"os"

	"github.com/bruin-data/dac/cmd"
)

var (
	version = "dev"
	commit  = ""
)

func main() {
	buildInfo := cmd.BuildInfo{
		Version: version,
		Commit:  commit,
	}

	if err := cmd.Run(context.Background(), os.Args, frontendDistFS(), buildInfo); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
