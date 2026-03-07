package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/urfave/cli/v3"
)

func connectionsCmd() *cli.Command {
	return &cli.Command{
		Name:  "connections",
		Usage: "Test database connections from .bruin.yml",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "dir",
				Aliases: []string{"d"},
				Usage:   "Dashboard definitions directory (for config discovery)",
				Value:   ".",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			dir := cmd.String("dir")

			configFile, cfg, err := resolveConfig(cmd, dir)
			if err != nil {
				return fmt.Errorf("config error: %w", err)
			}

			envName := cmd.Root().String("environment")
			env, err := cfg.GetEnvironment(envName)
			if err != nil {
				return err
			}

			backend := newBackend(cmd, configFile)

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "NAME\tTYPE\tSTATUS")

			hasErrors := false
			for connType, conns := range env.Connections {
				for _, conn := range conns {
					_, err := backend.Execute(ctx, conn.Name, "SELECT 1")
					status := "✓ connected"
					if err != nil {
						status = "✗ " + summarizeError(err)
						hasErrors = true
					}
					fmt.Fprintf(w, "%s\t%s\t%s\n", conn.Name, connType, status)
				}
			}

			w.Flush()

			if hasErrors {
				return fmt.Errorf("some connections failed")
			}
			return nil
		},
	}
}

// summarizeError extracts a short message from a bruin query error.
func summarizeError(err error) string {
	s := err.Error()
	// Truncate long error messages.
	if len(s) > 120 {
		return s[:117] + "..."
	}
	return s
}
