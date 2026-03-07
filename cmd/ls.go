package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/urfave/cli/v3"
)

func lsCmd() *cli.Command {
	return &cli.Command{
		Name:  "ls",
		Usage: "List discovered dashboards",
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
