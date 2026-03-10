package cmd

import (
	"context"

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
			dirFlag,
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
			&cli.StringFlag{
				Name:  "password",
				Usage: "Admin password for management API (admin endpoints disabled if not set)",
			},
			&cli.StringFlag{
				Name:  "agent-effort",
				Usage: "Reasoning effort for AI agent (low, medium, high)",
				Value: "medium",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			dir := cmd.String("dir")
			configFile := resolveConfigOptional(cmd, dir)

			srv, err := server.New(server.Config{
				Host:          cmd.String("host"),
				Port:          int(cmd.Int("port")),
				DashboardDir:  dir,
				TemplateName:  cmd.String("template"),
				ConfigFile:    configFile,
				Environment:   cmd.Root().String("environment"),
				AdminPassword: cmd.String("password"),
				AgentEffort:   cmd.String("agent-effort"),
				Frontend:      frontendFS,
			})
			if err != nil {
				return err
			}

			return srv.Start()
		},
	}
}
