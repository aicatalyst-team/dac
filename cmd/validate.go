package cmd

import (
	"context"
	"fmt"

	"github.com/bruin-data/dac/pkg/dashboard"
	"github.com/bruin-data/dac/pkg/telemetry"
	analytics "github.com/rudderlabs/analytics-go/v4"
	"github.com/urfave/cli/v3"
)

func validateCmd() *cli.Command {
	return &cli.Command{
		Name:  "validate",
		Usage: "Validate dashboard definitions and semantic model references",
		Flags: []cli.Flag{dirFlag},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			dashboards, err := loadDashboards(cmd.String("dir"))
			if err != nil {
				return err
			}
			if dashboards == nil {
				telemetry.SendEvent("dashboards_loaded", analytics.Properties{
					"count":         0,
					"valid_count":   0,
					"invalid_count": 0,
				})
				fmt.Println("No dashboard files found in", cmd.String("dir"))
				return nil
			}

			validCount := 0
			invalidCount := 0
			for _, d := range dashboards {
				if err := dashboard.Validate(d); err != nil {
					fmt.Println(err)
					invalidCount++
				} else {
					fmt.Printf("  %s: OK\n", d.Name)
					validCount++
				}
			}

			telemetry.SendEvent("dashboards_loaded", analytics.Properties{
				"count":         len(dashboards),
				"valid_count":   validCount,
				"invalid_count": invalidCount,
			})

			if invalidCount > 0 {
				return fmt.Errorf("validation failed")
			}

			fmt.Printf("\n%d dashboard(s) validated successfully.\n", len(dashboards))
			return nil
		},
	}
}
