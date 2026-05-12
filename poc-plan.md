# PoC Plan: dac

## Project Classification
- **Type:** web-app
- **Key Technologies:** Go 1.25, React/TypeScript (Vite), DuckDB, esbuild (for TSX dashboard transpilation), Goja (JavaScript runtime in Go)
- **ODH Relevance:** DAC is a Dashboard-as-Code tool that could serve as a visualization/reporting layer for data assets managed in an Open Data Hub environment. Its semantic layer and multi-database support (BigQuery, Snowflake, Databricks, etc.) make it relevant for data exploration workflows alongside ODH data pipelines.

## PoC Objectives
What we want to prove:
1. The DAC binary (Go + embedded React frontend) builds successfully as a container image with a multi-stage Dockerfile (Node.js for frontend build, Go for backend build)
2. The embedded web server starts and serves the React dashboard UI
3. The dashboard API correctly loads and serves YAML-defined dashboard definitions from bundled examples
4. The CLI validation command works correctly inside the container
5. The application runs stably as a long-running process in Kubernetes

## Infrastructure Requirements
- **Inference Server:** none
- **Vector Database:** none
- **Embedding Model:** none
- **GPU Required:** no
- **Persistent Storage:** none (example dashboards and DuckDB file are bundled in the image)
- **Resource Profile:** small (256Mi RAM, 250m CPU — lightweight Go binary serving static frontend + YAML parsing)
- **Sidecar Containers:** none

## Test Scenarios

### Scenario 1: Frontend Loads
- **Description:** Verify the embedded React frontend is served at the root URL
- **Type:** http
- **Input:** GET /
- **Expected:** Returns 200 OK with HTML content containing the React application shell
- **Timeout:** 30 seconds

### Scenario 2: List Dashboards API
- **Description:** Verify the REST API lists available dashboards from the example project
- **Type:** http
- **Input:** GET /api/dashboards
- **Expected:** Returns 200 OK with a JSON array containing at least one dashboard entry (the "sales" dashboard from the basic-yaml example)
- **Timeout:** 15 seconds

### Scenario 3: Get Dashboard Detail
- **Description:** Verify a specific dashboard can be retrieved via the API with its full definition
- **Type:** http
- **Input:** GET /api/dashboards/sales
- **Expected:** Returns 200 OK with JSON containing the dashboard definition including name, rows, and widget configurations
- **Timeout:** 15 seconds

### Scenario 4: Validate Dashboards (CLI)
- **Description:** Run the `dac validate` command against the bundled example dashboards to verify schema validation works
- **Type:** cli
- **Input:** `dac validate --dir /app/examples/basic-yaml`
- **Expected:** Job exits 0, output indicates dashboards are valid with no errors
- **Timeout:** 30 seconds

### Scenario 5: Version Check (CLI)
- **Description:** Verify the built binary reports version information correctly
- **Type:** cli
- **Input:** `dac version`
- **Expected:** Job exits 0, outputs version string
- **Timeout:** 10 seconds

## Dockerfile Considerations

This is a Go application with an embedded React frontend. The build requires a **multi-stage Dockerfile**:

1. **Stage 1 — Frontend build:** Use a Node.js 20+ image. Copy `frontend/` directory, run `npm ci` and `npm run build` to produce `frontend/dist/`.
2. **Stage 2 — Go build:** Use a Go 1.25+ image (or `golang:1.25`). Copy the entire source including the `frontend/dist/` from stage 1 (which gets embedded via `embed.go`). Run `go build -o /dac .` to produce the binary.
3. **Stage 3 — Runtime:** Use a minimal image (e.g., `gcr.io/distroless/base-debian12` or `alpine`). Copy the `dac` binary and the `examples/` directory (for serving demo dashboards). Set `TELEMETRY_OPTOUT=1` environment variable.

**ENTRYPOINT** should be `["/usr/local/bin/dac"]` with **CMD** `["serve", "--dir", "/app/examples/basic-yaml", "--port", "8080"]`.

**Add `EXPOSE 8080`** — this is a web server that listens on a port.

The `examples/data/test.db` (DuckDB file, ~10MB) should be included in the image so the example dashboards have data to reference (though actual query execution requires the `bruin` CLI which won't be available — the dashboard definitions and UI will still load).

Key build notes:
- The `Makefile` target `make build` runs `cd frontend && npm ci && npm run build` then `go build`. The Dockerfile should replicate this flow.
- Go 1.25.0 is specified in `go.mod` — ensure the Go builder image supports this version.
- The `embed.go` file uses `//go:embed` to embed `frontend/dist` into the binary, so the frontend build output MUST be present before `go build`.

## Deployment Considerations

**Deploy as a Kubernetes Deployment** with 1 replica. This is a long-running web server.

**Create a Service** on port 8080 — the `dac serve` command starts an HTTP server.

**Test via HTTP requests** to the Service:
- `GET /` — verify the frontend loads (200 + HTML)
- `GET /api/dashboards` — verify the API returns dashboard listings (200 + JSON)
- `GET /api/dashboards/sales` — verify individual dashboard retrieval

For CLI test scenarios (validate, version), run them as one-off `kubectl run --rm` Jobs against the same container image.

**Environment variables:**
- `TELEMETRY_OPTOUT=1` — Disable telemetry in the container (prevents external network calls to analytics services)
- `DO_NOT_TRACK=1` — Alternative telemetry opt-out

**No bruin CLI dependency for the PoC:** The `bruin` CLI is needed for executing queries against real databases, but the PoC focuses on dashboard loading, validation, and UI serving. The dashboard definitions will load and display correctly; actual query execution against databases is out of scope for this PoC. If bruin is needed later, it can be added as an init container or bundled in the image.

**Liveness/readiness probes:** Use HTTP GET on `/` or `/api/dashboards` on port 8080.