# Examples

These example projects are intended for real manual testing, not just documentation snippets.

Each example is self-contained:

- its own `.bruin.yml`
- dashboards under `dashboards/`
- shared sample DuckDB data under `examples/data/`
- semantic models under `semantic/` when applicable

## Run an Example

From the repository root:

```bash
make deps
make build
./bin/dac serve --dir examples/basic-yaml
```

DAC uses Bruin connections for query execution, so install the `bruin` CLI too if you want to run these examples locally.

## Included Projects

- `basic-yaml`: standard YAML dashboard with SQL queries, filters, and query files
- `basic-tsx`: TSX dashboard with load-time queries that generate layout from the database
- `semantic-yaml`: YAML dashboard backed by external semantic models in `semantic/`
- `semantic-tsx`: TSX dashboard backed by external semantic models in `semantic/`
