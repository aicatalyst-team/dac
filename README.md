# DAC

DAC is a Dashboard-as-Code tool for defining, validating, and serving dashboards from YAML and TSX.

It is built for analytics engineers who want dashboards to live in version control, and for business users who need a fast, dense, browser-based view of the data.

## What DAC Does

- Define dashboards in YAML or TSX.
- Reuse connections from `.bruin.yml`.
- Validate dashboards and semantic models in CI before they break production.
- Validate dashboard, semantic model, and theme YAML against versioned Bruin schemas.
- Compile semantic widgets to SQL in the backend instead of inside dashboard files.
- Serve a single embedded frontend from one DAC binary.

## Install

Install the latest stable DAC release:

```bash
curl -fsSL https://raw.githubusercontent.com/bruin-data/dac/main/install.sh | bash
```

Install the latest edge build from `main`:

```bash
curl -fsSL https://raw.githubusercontent.com/bruin-data/dac/main/install.sh | bash -s -- --channel edge
```

DAC uses your existing Bruin connections and currently shells out to `bruin query` for query execution. Install the Bruin CLI as well if you want to run `dac serve`, `dac query`, or `dac check` against live data.

## Quickstart

Create a new starter project:

```bash
dac init my-dashboards
cd my-dashboards
dac validate --dir .
dac serve --dir . --open
```

The starter includes a SQL-backed YAML dashboard, a semantic YAML dashboard, and a semantic model under `semantic/`.

`dac init` also installs DAC's bundled dashboard authoring skill for Claude and Codex:

```bash
ls .claude/skills/create-dashboard/SKILL.md
ls .codex/skills/create-dashboard
```

For existing projects, run `dac skills install --dir .`.

If you cloned the repository and have `dac` installed, you can also run one of the bundled example projects:

```bash
dac serve --dir examples/basic-yaml
```

## Examples

The repository includes four self-contained example projects under [`examples/`](examples):

| Example | What it shows |
| --- | --- |
| [`examples/basic-yaml`](examples/basic-yaml) | A standard YAML dashboard with filters, SQL queries, and query files. |
| [`examples/basic-tsx`](examples/basic-tsx) | A TSX dashboard that uses load-time queries to generate layout from the database. |
| [`examples/semantic-yaml`](examples/semantic-yaml) | A YAML dashboard that reads semantic models from `semantic/` and compiles widgets in the backend. |
| [`examples/semantic-tsx`](examples/semantic-tsx) | A TSX dashboard using external semantic models and backend semantic query compilation. |

## Project Layout

```text
.
├── cmd/         CLI entrypoints
├── pkg/         Dashboard loading, semantic engine, server, query backends
├── frontend/    React frontend embedded into the DAC binary
├── docs/        VitePress documentation source
├── examples/    Runnable example projects for YAML, TSX, and semantic dashboards
└── testdata/    Internal fixtures used by tests
```

## Development

```bash
make deps
make test
make build
make dev
```

The main development commands are defined in the [`Makefile`](Makefile). Use `make` targets rather than ad-hoc `go build` or `npm run build` commands so frontend embedding and build flags stay consistent.

## Documentation

- Docs source: [`docs/`](docs)
- Example projects: [`examples/`](examples)
- Contribution guide: [`CONTRIBUTING.md`](CONTRIBUTING.md)
- Security policy: [`SECURITY.md`](SECURITY.md)

## License

AGPL-3.0-only. See [`LICENSE`](LICENSE).
