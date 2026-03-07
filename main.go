package main

import (
	"context"
	"fmt"
	"os"

	"github.com/bruin-data/dac/cmd"
)

func main() {
	if err := cmd.Run(context.Background(), os.Args, frontendDistFS()); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
