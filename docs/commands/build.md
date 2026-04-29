# dac build

Build a self-contained static dashboard with baked-in query results. The output is a directory with an HTML file and assets that can be deployed anywhere — no server required.

```shell
dac build [flags]
```

## Flags

| Flag | Alias | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--dashboard` | `-n` | string | *required* | Dashboard name |
| `--output` | `-o` | string | `build` | Output directory |
| `--dir` | `-d` | string | `.` | Dashboard definitions directory |
| `--template` | `-t` | string | `bruin` | Theme name or path to YAML file |
| `--filters` | | string | | JSON string with filter overrides |

## Examples

```shell
# Build the "Sales Analytics" dashboard
dac build --dashboard "Sales Analytics"

# Custom output directory and theme
dac build --dashboard "Sales Analytics" --output dist --template bruin-dark

# Build with specific filter values baked in
dac build --dashboard "Sales Analytics" --filters '{"region": "Europe", "date_range": "last_30_days"}'
```

## Output

The build produces a self-contained directory:

```
build/
├── index.html
└── assets/
    ├── index-[hash].js
    └── index-[hash].css
```

The HTML file includes a `window.__DAC_STATIC__` payload containing:
- The full dashboard definition
- Pre-computed query results for all widgets
- Theme tokens
- Filter defaults

Semantic widgets are compiled to SQL during the build, then the generated SQL results are baked into the static payload.

Open `index.html` in a browser — no server needed. Deploy to any static hosting (S3, Netlify, GitHub Pages, etc.).

## Use Cases

- **Scheduled reports**: Build on a cron, upload to S3, share a link
- **Offline viewing**: Send dashboards to stakeholders who don't have database access
- **Embedding**: Include dashboard HTML in other applications
- **Archival**: Snapshot a dashboard's state at a point in time
