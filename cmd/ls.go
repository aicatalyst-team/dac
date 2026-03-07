package cmd

import (
	"context"
	"fmt"
	"text/tabwriter"
	"os"

	"github.com/bruin-data/dac/pkg/dashboard"
	"github.com/urfave/cli/v3"
)

func lsCmd() *cli.Command {
	return &cli.Command{
		Name:  "ls",
		Usage: "List discovered dashboards",
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

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "NAME\tWIDGETS\tFILTERS\tCONNECTION")

			for _, d := range dashboards {
				widgetCount := 0
				for _, row := range d.Rows {
					widgetCount += len(row.Widgets)
				}

				conn := d.Connection
				if conn == "" {
					conn = "-"
				}

				fmt.Fprintf(w, "%s\t%d\t%d\t%s\n",
					d.Name,
					widgetCount,
					len(d.Filters),
					conn,
				)
			}

			return w.Flush()
		},
	}
}
