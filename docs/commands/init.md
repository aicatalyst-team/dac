# dac init

Create a new DAC project scaffold.

```shell
dac init [path] [flags]
```

If `path` is omitted, DAC initializes the current directory.

`dac init` initializes the generated project as a Git repository by default. Bruin uses the Git repository root for project discovery, so generated projects can run `dac query`, `dac check`, and `dac serve` immediately.

## Flags

| Flag | Alias | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--template` | `-t` | string | `starter` | Project template: `starter`, `sql`, `semantic`, or `tsx` |
| `--force` | `-f` | bool | `false` | Overwrite scaffold files if they already exist |

## Examples

```shell
# Create the default starter project
dac init my-dashboards

# Create only a SQL-backed YAML dashboard
dac init my-sql-dashboard --template sql

# Create a semantic YAML dashboard and model
dac init my-semantic-dashboard --template semantic

# Create a semantic TSX dashboard and model
dac init my-tsx-dashboard --template tsx
```

## Generated Project

The default `starter` template creates:

```text
my-dashboards/
├── .git/
├── .bruin.yml
├── .claude/
│   └── skills/
│       └── create-dashboard/
│           └── SKILL.md
├── .codex/
│   └── skills/
│       └── create-dashboard -> ../../.claude/skills/create-dashboard
├── README.md
├── data/
│   └── dac-demo.duckdb
├── dashboards/
│   ├── sales.yml
│   └── semantic-sales.yml
└── semantic/
    └── sales.yml
```

The generated dashboards use a local DuckDB connection named `local_duckdb`. Starter queries include inline sample data, so there is no separate seed step.

`dac init` also installs DAC's bundled `create-dashboard` agent skill. Claude gets the real skill file under `.claude/skills/`; Codex gets a symlink under `.codex/skills/` pointing at the same content.

## Next Steps

```shell
cd my-dashboards
dac validate --dir .
dac serve --dir . --open
```

To inspect a generated semantic widget from the command line:

```shell
dac query --dir . --dashboard "Semantic Sales" --widget "Revenue"
```
