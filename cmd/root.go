package cmd

import (
	"context"
	"io/fs"

	"github.com/urfave/cli/v3"
)

// frontendFS is set by Run to pass the embedded frontend to the serve command.
var frontendFS fs.FS

func NewApp() *cli.Command {
	return &cli.Command{
		Name:  "dac",
		Usage: "Dashboard-as-Code: define, validate, and serve dashboards from YAML",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Path to .bruin.yml config file (default: auto-discover)",
			},
			&cli.StringFlag{
				Name:    "environment",
				Aliases: []string{"e"},
				Usage:   "Target environment name",
			},
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "Enable debug logging",
			},
		},
		Commands: []*cli.Command{
			serveCmd(),
			buildCmd(),
			validateCmd(),
			checkCmd(),
			queryCmd(),
			lsCmd(),
			connectionsCmd(),
		},
	}
}

func Run(ctx context.Context, args []string, frontend fs.FS) error {
	frontendFS = frontend
	return NewApp().Run(ctx, args)
}
