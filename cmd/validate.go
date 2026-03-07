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
		Flags: []cli.Flag{dirFlag},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			dashboards, err := loadDashboards(cmd.String("dir"))
			if err != nil {
				return err
			}
			if dashboards == nil {
				fmt.Println("No dashboard files found in", cmd.String("dir"))
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
