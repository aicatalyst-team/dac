package main

import (
	"context"
	"fmt"
	"os"

	"github.com/bruin-data/dac/cmd"
	"github.com/bruin-data/dac/pkg/telemetry"
)

var (
	version      = "dev"
	commit       = ""
	telemetryKey = ""
)

func main() {
	if telemetryKey == "" {
		telemetryKey = os.Getenv("TELEMETRY_KEY")
	}
	telemetry.TelemetryKey = telemetryKey
	telemetry.OptOut = os.Getenv("TELEMETRY_OPTOUT") != "" || os.Getenv("DO_NOT_TRACK") != ""
	telemetry.AppVersion = version
	client := telemetry.Init()
	defer client.Close()

	buildInfo := cmd.BuildInfo{
		Version: version,
		Commit:  commit,
	}

	if err := cmd.Run(context.Background(), os.Args, frontendDistFS(), buildInfo); err != nil {
		// Close manually since defer doesn't run on os.Exit.
		client.Close()
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
