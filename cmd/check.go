package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/bruin-data/dac/pkg/dashboard"
	tmpl "github.com/bruin-data/dac/pkg/template"
	"github.com/urfave/cli/v3"
)

func checkCmd() *cli.Command {
	return &cli.Command{
		Name:  "check",
		Usage: "Validate dashboards and execute all queries to verify they work",
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

			configFile, _, err := resolveConfig(cmd, dir)
			if err != nil {
				return fmt.Errorf("config error: %w", err)
			}

			backend := newBackend(cmd, configFile)

			dashboards, err := dashboard.LoadDir(dir)
			if err != nil {
				return fmt.Errorf("failed to load dashboards: %w", err)
			}

			if len(dashboards) == 0 {
				fmt.Println("No dashboard files found in", dir)
				return nil
			}

			totalErrors := 0
			totalWidgets := 0
			totalDashboards := len(dashboards)

			for _, d := range dashboards {
				fmt.Printf("\n%s\n", d.Name)

				// YAML validation first.
				if err := dashboard.Validate(d); err != nil {
					fmt.Printf("  ✗ YAML validation failed\n")
					if ve, ok := err.(*dashboard.ValidationError); ok {
						for _, e := range ve.Errors {
							fmt.Printf("    %s\n", e)
						}
					} else {
						fmt.Printf("    %s\n", err)
					}
					totalErrors++
					continue
				}

				defaults := buildDefaultFilters(d)

				for rowIdx, row := range d.Rows {
					for widgetIdx, w := range row.Widgets {
						totalWidgets++

						if w.Type == "text" {
							fmt.Printf("  ✓ %-30s  text\n", w.Name)
							continue
						}

						sql, conn, err := w.ResolvedQuery(d)
						if err != nil {
							fmt.Printf("  ✗ %-30s  %s\n", w.Name, err)
							totalErrors++
							continue
						}

						// Apply default filter values.
						if len(defaults) > 0 {
							sql, err = tmpl.Render(sql, defaults)
							if err != nil {
								fmt.Printf("  ✗ %-30s  template error: %s\n", w.Name, err)
								totalErrors++
								continue
							}
						}

						_ = rowIdx
						_ = widgetIdx

						start := time.Now()
						result, err := backend.Execute(ctx, conn, sql)
						elapsed := time.Since(start)

						if err != nil {
							fmt.Printf("  ✗ %-30s  %s\n", w.Name, err)
							totalErrors++
							continue
						}

						fmt.Printf("  ✓ %-30s  %d rows, %d cols  (%dms)\n",
							w.Name,
							len(result.Rows),
							len(result.Columns),
							elapsed.Milliseconds(),
						)
					}
				}
			}

			fmt.Println()
			if totalErrors > 0 {
				fmt.Printf("%d error(s) across %d dashboard(s), %d widget(s) checked\n",
					totalErrors, totalDashboards, totalWidgets)
				return fmt.Errorf("check failed with %d error(s)", totalErrors)
			}

			fmt.Printf("%d dashboard(s), %d widget(s) — all passing\n", totalDashboards, totalWidgets)
			return nil
		},
	}
}
