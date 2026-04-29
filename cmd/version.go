package cmd

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func versionCmd(build BuildInfo) *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Show DAC version information",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			_ = ctx

			if build.Commit != "" {
				fmt.Printf("%s (%s)\n", build.Version, build.Commit)
				return nil
			}

			fmt.Println(build.Version)
			return nil
		},
	}
}
