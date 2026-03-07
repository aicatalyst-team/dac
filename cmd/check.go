package cmd

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bruin-data/dac/pkg/dashboard"
	"github.com/bruin-data/dac/pkg/query"
	tmpl "github.com/bruin-data/dac/pkg/template"
	"github.com/urfave/cli/v3"
)

type checkJob struct {
	widgetName string
	sql        string
	connection string
}

type checkResult struct {
	widgetName string
	rows       int
	cols       int
	elapsed    time.Duration
	err        error
}

func checkCmd() *cli.Command {
	return &cli.Command{
		Name:  "check",
		Usage: "Validate dashboards and execute all queries to verify they work",
		Flags: []cli.Flag{dirFlag},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			dir := cmd.String("dir")

			configFile, _, err := resolveConfig(cmd, dir)
			if err != nil {
				return fmt.Errorf("config error: %w", err)
			}

			backend := newBackend(cmd, configFile)

			dashboards, err := loadDashboards(dir)
			if err != nil {
				return err
			}
			if dashboards == nil {
				fmt.Println("No dashboard files found in", dir)
				return nil
			}

			totalErrors := 0
			totalWidgets := 0

			for _, d := range dashboards {
				fmt.Printf("\n%s\n", d.Name)

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

				defaults := d.DefaultFilters()

				// Collect jobs, tracking text widgets separately (no query needed).
				var textWidgets []string
				connGroups := make(map[string][]checkJob)

				for _, row := range d.Rows {
					for _, w := range row.Widgets {
						totalWidgets++

						if w.Type == dashboard.WidgetTypeText {
							textWidgets = append(textWidgets, w.Name)
							continue
						}

						sql, conn, err := w.ResolvedQuery(d)
						if err != nil {
							fmt.Printf("  ✗ %-30s  %s\n", w.Name, err)
							totalErrors++
							continue
						}

						if len(defaults) > 0 {
							sql, err = tmpl.Render(sql, defaults)
							if err != nil {
								fmt.Printf("  ✗ %-30s  template error: %s\n", w.Name, err)
								totalErrors++
								continue
							}
						}

						connGroups[conn] = append(connGroups[conn], checkJob{
							widgetName: w.Name,
							sql:        sql,
							connection: conn,
						})
					}
				}

				// Print text widgets.
				for _, name := range textWidgets {
					fmt.Printf("  ✓ %-30s  text\n", name)
				}

				// Execute query groups in parallel (same connection sequential).
				results := executeCheckJobs(ctx, backend, connGroups)
				for _, r := range results {
					if r.err != nil {
						fmt.Printf("  ✗ %-30s  %s\n", r.widgetName, r.err)
						totalErrors++
					} else {
						fmt.Printf("  ✓ %-30s  %d rows, %d cols  (%dms)\n",
							r.widgetName, r.rows, r.cols, r.elapsed.Milliseconds())
					}
				}
			}

			fmt.Println()
			if totalErrors > 0 {
				fmt.Printf("%d error(s) across %d dashboard(s), %d widget(s) checked\n",
					totalErrors, len(dashboards), totalWidgets)
				return fmt.Errorf("check failed with %d error(s)", totalErrors)
			}

			fmt.Printf("%d dashboard(s), %d widget(s) — all passing\n", len(dashboards), totalWidgets)
			return nil
		},
	}
}

// executeCheckJobs runs query jobs grouped by connection. Same connection runs
// sequentially (avoids file lock contention), different connections in parallel.
func executeCheckJobs(ctx context.Context, backend query.Backend, connGroups map[string][]checkJob) []checkResult {
	var allResults []checkResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, jobs := range connGroups {
		wg.Add(1)
		go func(jobs []checkJob) {
			defer wg.Done()
			for _, j := range jobs {
				start := time.Now()
				qr, err := backend.Execute(ctx, j.connection, j.sql)
				elapsed := time.Since(start)

				r := checkResult{widgetName: j.widgetName, elapsed: elapsed, err: err}
				if err == nil {
					r.rows = len(qr.Rows)
					r.cols = len(qr.Columns)
				}

				mu.Lock()
				allResults = append(allResults, r)
				mu.Unlock()
			}
		}(jobs)
	}

	wg.Wait()
	return allResults
}
