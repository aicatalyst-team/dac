package cmd

import (
	"context"

	"github.com/bruin-data/dac/pkg/config"
	"github.com/bruin-data/dac/pkg/server"
	"github.com/urfave/cli/v3"
)

func serveCmd() *cli.Command {
	return &cli.Command{
		Name:  "serve",
		Usage: "Start development server with live reload",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Usage:   "Port to listen on",
				Value:   8321,
			},
			&cli.StringFlag{
				Name:    "dir",
				Aliases: []string{"d"},
				Usage:   "Dashboard definitions directory",
				Value:   ".",
			},
			&cli.StringFlag{
				Name:    "template",
				Aliases: []string{"t"},
				Usage:   "Template name (bruin, bruin-dark) or path to a theme YAML file",
				Value:   "bruin",
			},
			&cli.StringFlag{
				Name:  "host",
				Usage: "Host to bind to",
				Value: "localhost",
			},
			&cli.BoolFlag{
				Name:  "open",
				Usage: "Open browser automatically",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			dir := cmd.String("dir")

			// Discover .bruin.yml config.
			configFile := cmd.Root().String("config")
			if configFile == "" {
				found, err := config.Discover(dir)
				if err != nil {
					// Config is optional for serve — queries will fail but dashboards still render.
					configFile = ""
					_ = err
				} else {
					configFile = found
				}
			}

			srv, err := server.New(server.Config{
				Host:         cmd.String("host"),
				Port:         int(cmd.Int("port")),
				DashboardDir: dir,
				TemplateName: cmd.String("template"),
				ConfigFile:   configFile,
				Environment:  cmd.Root().String("environment"),
				Frontend:     frontendFS,
			})
			if err != nil {
				return err
			}

			return srv.Start()
		},
	}
}
