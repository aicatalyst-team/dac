package cmd

import (
	"github.com/bruin-data/dac/pkg/config"
	"github.com/bruin-data/dac/pkg/query"
	"github.com/urfave/cli/v3"
)

// resolveConfig discovers and loads the .bruin.yml config file.
func resolveConfig(cmd *cli.Command, dir string) (string, *config.Config, error) {
	configFile := cmd.Root().String("config")
	if configFile == "" {
		found, err := config.Discover(dir)
		if err != nil {
			return "", nil, err
		}
		configFile = found
	}

	cfg, err := config.Load(configFile)
	if err != nil {
		return configFile, nil, err
	}

	return configFile, cfg, nil
}

// newBackend creates a BruinCLIBackend from the resolved config.
func newBackend(cmd *cli.Command, configFile string) *query.BruinCLIBackend {
	return &query.BruinCLIBackend{
		ConfigFile:  configFile,
		Environment: cmd.Root().String("environment"),
	}
}
