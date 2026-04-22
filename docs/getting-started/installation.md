# Installation

DAC is distributed as part of the [Bruin CLI](https://github.com/bruin-data/bruin). Install Bruin to get access to `dac`.

## Install Bruin

::: code-group

```shell [macOS (Homebrew)]
brew install bruin-data/tap/bruin
```

```shell [Go]
go install github.com/bruin-data/bruin@latest
```

:::

## Build from Source

If you're working on DAC itself:

```shell
git clone https://github.com/bruin-data/bruin.git
cd bruin/internal/dac
make deps
make build
```

The binary is output to `bin/dac`.

## Verify Installation

```shell
dac --help
```

You should see:

```
NAME:
   dac - Dashboard-as-Code: define, validate, and serve dashboards from YAML

USAGE:
   dac [global options] command [command options]

COMMANDS:
   serve        Start development server with live reload
   build        Build static dashboard with baked-in query results
   validate     Validate dashboard YAML definitions
   check        Validate dashboards and execute all queries
   query        Run a SQL query against a connection
   ls           List discovered dashboards
   connections  Test database connections
   export       Export dashboards to external formats
   help, h      Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --config value, -c value       Path to .bruin.yml config file
   --environment value, -e value  Target environment name
   --debug                        Enable debug logging
   --help, -h                     show help
```
