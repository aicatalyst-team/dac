package cmd

import (
	"context"
	"fmt"

	"github.com/bruin-data/dac/pkg/dashboard"
	"github.com/urfave/cli/v3"
)

func validateCmd() *cli.Command {
	return &cli.Command{
		Name:  "validate",
		Usage: "Validate dashboard YAML definitions",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "dir",
				Aliases: []string{"d"},
				Usage:   "Dashboard definitions directory",
				Value:   ".",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			dir := cmd.String("dir")

			dashboards, err := dashboard.LoadDir(dir)
			if err != nil {
				return fmt.Errorf("failed to load dashboards: %w", err)
			}

			if len(dashboards) == 0 {
				fmt.Println("No dashboard files found in", dir)
				return nil
			}

			hasErrors := false
			for _, d := range dashboards {
				if err := dashboard.Validate(d); err != nil {
					fmt.Println(err)
					hasErrors = true
				} else {
					fmt.Printf("  %s: OK\n", d.Name)
				}
			}

			if hasErrors {
				return fmt.Errorf("validation failed")
			}

			fmt.Printf("\n%d dashboard(s) validated successfully.\n", len(dashboards))
			return nil
		},
	}
}
