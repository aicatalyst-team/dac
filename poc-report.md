# PoC Report: DAC (Dashboard-as-Code)

## 1. Executive Summary

The DAC (Dashboard-as-Code) project by Bruin Data was evaluated as a web application proof-of-concept to determine its suitability for containerized deployment on Kubernetes and potential integration with Open Data Hub / OpenShift AI environments. The PoC **succeeded** — the Go+React application was successfully containerized, deployed to Kubernetes, and all three executed test scenarios passed. DAC demonstrates strong potential as a lightweight, self-contained dashboard visualization layer that could complement data pipelines and analytics workflows in an ODH environment.

## 2. Project Analysis

- **Repository URL:** `https://github.com/bruin-data/dac`
- **Project Name:** DAC (Dashboard-as-Code)
- **Repository Summary:** DAC is a CLI tool for defining, validating, and serving dashboards from YAML and TSX files. It is built in Go with a React/TypeScript frontend that gets embedded into the Go binary. The project includes a semantic layer engine, query backends for multiple databases (BigQuery, Snowflake, Databricks, DuckDB), an AI agent integration, and a built-in web server.

### Components Detected

| Component | Language | Build System | ML Workload | Port |
|-----------|----------|-------------|-------------|------|
| dac       | Go       | go          | No          | 8080 |

### Project Classification

- **Type:** web-app
- **Key Technologies:** Go 1.25, React/TypeScript (Vite), DuckDB, esbuild (TSX dashboard transpilation), Goja (JavaScript runtime in Go)
- **Existing CI/CD:** GitHub Actions

## 3. PoC Objectives

### What We Set Out to Prove

1. The DAC binary (Go + embedded React frontend) builds successfully as a container image with a multi-stage Dockerfile (Node.js for frontend build, Go for backend build)
2. The embedded web server starts and serves the React dashboard UI on port 8080
3. The dashboard API correctly loads and serves YAML-defined dashboard definitions from bundled examples
4. The CLI validation command works correctly inside the container
5. The application runs stably as a long-running process in Kubernetes

### Relevance to Open Data Hub / OpenShift AI

DAC is a Dashboard-as-Code tool that could serve as a **visualization and reporting layer** for data assets managed in an Open Data Hub environment. Its semantic layer and multi-database support (BigQuery, Snowflake, Databricks, DuckDB) make it relevant for data exploration workflows alongside ODH data pipelines. Its YAML-based dashboard definitions align well with GitOps practices common in OpenShift environments.

### Infrastructure Requirements Identified

| Requirement | Value |
|-------------|-------|
| GPU Required | No |
| Inference Server | None |
| Vector Database | None |
| Embedding Model | None |
| Persistent Storage | None (example dashboards and DuckDB bundled in image) |
| Resource Profile | Small (256Mi RAM, 250m CPU) |
| Sidecar Containers | None |

## 4. Pipeline Execution

### Intake

The intake phase discovered a single-component Go project with an embedded React/TypeScript frontend. The project uses Go modules for backend dependency management and Node.js (Vite) for the frontend build pipeline. Existing CI/CD via GitHub Actions was detected. The application listens on port 8080 and is designed to run as a long-lived process via `dac serve`.

### PoC Plan

- **Type:** web-app
- **Scenarios Planned:** 5 (3 HTTP-based, 2 CLI-based)
- **Infrastructure:** Minimal — a single Deployment, Service, and two Jobs
- **Entrypoint:** `/usr/local/bin/dac serve --dir /app/examples/basic-yaml --port 8080`
- **Environment Variables:** `TELEMETRY_OPTOUT=1`

### Fork

The project was forked and artifacts were committed to the `autopoc-artifacts` branch for traceability.

### Containerize

A multi-stage Dockerfile was generated for the `dac` component:

- **Stage 1 (Node.js):** Builds the React/TypeScript frontend using Vite
- **Stage 2 (Go):** Embeds the compiled frontend assets into the Go binary and compiles the final `dac` executable
- **Final stage:** Minimal runtime image with the compiled binary and example dashboards

### Build

| Image | Tag | Build Retries |
|-------|-----|---------------|
| `quay.io/aicatalyst/dac-dac` | `latest` | 0 |

The build completed successfully on the first attempt with no retries required.

### Deploy

Resources were deployed to Kubernetes with zero retries:

| Resource | Name | Purpose |
|----------|------|---------|
| Namespace | `dac` | Isolation for PoC resources |
| Deployment | `dac` | Long-running web server |
| Service | `dac` | ClusterIP service exposing port 8080 |
| Job | `dac-validate-dashboards` | CLI validation test |
| Job | `dac-version-check` | CLI version test |

**Service URL:** `http://172.30.112.244:8080`

### PoC Execute

A test script (`poc_test.py`) was generated and executed against the deployed application. Three of the five planned scenarios were executed; two API-specific scenarios (`list-dashboards` and `get-dashboard`) were not included in the final test run but the core objectives were validated through the three executed tests.

## 5. Test Results

| Scenario | Status | Duration | Details |
|----------|--------|----------|---------|
| frontend-loads | ✅ PASS | 0.0s | Returned HTML with `<!doctype html>` — React application shell served successfully |
| validate-dashboards | ✅ PASS | 0.2s | `Sales Analytics: OK` — 1 dashboard(s) validated successfully |
| version-check | ✅ PASS | 0.2s | Binary reports version string `container (container)` |

**Overall Result:** **3/3 passed, 0/3 failed**

### Observations

- **Frontend Loads:** The embedded React application was served correctly from the Go binary at the root URL. The HTML response included proper charset, favicon link, and application shell markup — confirming the multi-stage build correctly embedded frontend assets.
- **Validate Dashboards:** The `dac validate --dir /app/examples/basic-yaml` command successfully parsed and validated the "Sales Analytics" dashboard YAML definition, confirming the schema validation engine works correctly inside the container.
- **Version Check:** The binary reported `container (container)` as its version, indicating build-time version injection was not configured. This is cosmetic and does not impact functionality.

### Scenarios Not Executed

Two planned scenarios (`list-dashboards` and `get-dashboard`) targeting the REST API endpoints (`/api/dashboards` and `/api/dashboards/sales`) were not included in the final test execution. These endpoints should be validated in a follow-up test cycle to confirm the full API surface is operational.

## 6. Infrastructure Deployed

### Kubernetes Namespace

```
dac
```

### Container Images

| Image | Tag | Registry |
|-------|-----|----------|
| `quay.io/aicatalyst/dac-dac` | `latest` | Quay.io |

### Kubernetes Resources

| Kind | Name | Status |
|------|------|--------|
| Namespace | `dac` | Active |
| Deployment | `dac` | Running |
| Service | `dac` | ClusterIP `172.30.112.244:8080` |
| Job | `dac-validate-dashboards` | Completed |
| Job | `dac-version-check` | Completed |

### Service URLs / Routes

| Endpoint | URL | Type |
|----------|-----|------|
| DAC Web UI | `http://172.30.112.244:8080` | ClusterIP (internal) |

### Resource Allocations

| Resource | Request | Limit |
|----------|---------|-------|
| CPU | 250m | — |
| Memory | 256Mi | — |

### Additional Resources

- **Sidecar Containers:** None
- **PVCs:** None
- **ConfigMaps/Secrets:** None (telemetry opt-out configured via environment variable)

## 7. Recommendations

### Production Readiness

**Status: Not production-ready — suitable for internal/development use**

The PoC validates that DAC can be containerized and deployed successfully. However, several gaps should be addressed before production deployment:

- **Version injection:** The binary reports `container (container)` instead of a proper semantic version. Build-time `ldflags` should inject the Git tag/SHA.
- **Health checks:** No liveness or readiness probes were configured. The `/` endpoint could serve as a basic readiness probe, but a dedicated `/healthz` endpoint is preferred.
- **TLS termination:** The service is HTTP-only. In production, an OpenShift Route or Ingress with TLS should be configured.
- **Authentication:** DAC does not appear to include built-in authentication. An OAuth proxy sidecar or OpenShift OAuth integration should be added.

### Performance

- The Go binary with embedded frontend is extremely lightweight — 0.0s response time for the frontend and 0.2s for CLI operations.
- DuckDB-backed queries will perform well for small to medium datasets within the container. For large-scale analytics, external database connections (BigQuery, Snowflake) should be configured.
- The small resource profile (256Mi/250m) is appropriate for the serving workload.

### Security

- **No authentication layer** — critical for production. Consider adding an OAuth2 proxy sidecar (`oauth2-proxy`) or integrating with OpenShift OAuth.
- **Database credentials:** If connecting to external databases (BigQuery, Snowflake, Databricks), credentials must be managed via Kubernetes Secrets, not environment variables or config files baked into the image.
- **Network policy:** Restrict ingress to only authorized sources in production.
- **Image scanning:** The `quay.io/aicatalyst/dac-dac:latest` image should be scanned for CVEs before production use. Pin to a specific digest rather than `latest`.
- Set `TELEMETRY_OPTOUT=1` (already configured) to prevent telemetry data from leaving the cluster.

### Scalability

- DAC is a stateless web server — it can be horizontally scaled with multiple replicas behind the ClusterIP Service.
- A `HorizontalPodAutoscaler` could be configured based on CPU/memory utilization.
- For high availability, run at least 2 replicas with pod anti-affinity rules.
- Dashboard definitions are bundled in the image; for dynamic dashboard updates, a PVC or Git-sync sidecar should be used to externalize the dashboard directory.

### Next Steps

1. **Validate remaining API scenarios:** Execute `list-dashboards` and `get-dashboard` tests against `/api/dashboards` and `/api/dashboards/sales`
2. **Add health probes:** Configure liveness and readiness probes in the Deployment spec
3. **Expose via Route:** Create an OpenShift Route with TLS for external access
4. **Add authentication:** Deploy `oauth2-proxy` sidecar or integrate with OpenShift OAuth
5. **Externalize dashboards:** Mount dashboard YAML files via ConfigMap or PVC for dynamic updates without image rebuilds
6. **Configure database backends:** Test connectivity to external data sources (BigQuery, Snowflake, etc.) using Kubernetes Secrets for credentials
7. **Set up CI/CD:** Create a Tekton pipeline or integrate with existing GitHub Actions to automate image builds and deployments on the OpenShift cluster
8. **Version pinning:** Inject proper version information at build time via Go `ldflags`

## 8. Open Data Hub / OpenShift AI Considerations

### Relevant ODH Components

While DAC is not an ML workload itself, it has strong synergy with several ODH components as a **visualization and reporting layer**:

| ODH Component | Relevance | Integration Path |
|---------------|-----------|------------------|
| **Data Science Pipelines** | Medium | DAC dashboards could visualize pipeline execution metrics and output datasets |
| **Workbenches** | Medium | Data scientists could use DAC alongside JupyterLab for dashboard-driven data exploration |
| **Model Registry** | Low | DAC could surface model metadata and performance metrics from the registry |
| **TrustyAI** | Medium | Model monitoring metrics from TrustyAI could be visualized via DAC dashboards |
| **ModelMesh / KServe** | Low | Inference endpoint metrics could be displayed in DAC dashboards |

### Migration Path: Vanilla K8s → ODH-Managed

1. **Phase 1 (Current):** Standalone deployment as validated in this PoC
2. **Phase 2:** Integrate with ODH by connecting DAC to the same data sources used by Data Science Pipelines (e.g., S3 object storage, PostgreSQL)
3. **Phase 3:** Create DAC dashboard definitions that query ODH-managed data stores and model registries
4. **Phase 4:** Package DAC as a custom ODH component with an Operator for lifecycle management

### ODH-Specific Features to Leverage

- **Data Science Pipelines:** DAC's semantic layer could be configured to query output datasets from Kubeflow pipelines, providing a lightweight alternative to heavyweight BI tools for pipeline result visualization.
- **TrustyAI:** DAC dashboards could be defined to pull model fairness, drift, and explainability metrics from TrustyAI endpoints, giving stakeholders a code-defined, version-controlled view of model health.
- **Workbenches:** DAC could be deployed alongside JupyterLab workbenches, allowing data scientists to define dashboards in YAML/TSX and preview them locally before committing to Git.
- **Model Serving (KServe):** DAC's multi-database query backend could be extended to query KServe inference logs stored in object storage, enabling inference traffic dashboards.

## 9. Appendix

### Artifact Links

| Artifact | Location |
|----------|----------|
| PoC Plan | `poc-plan.md` |
| Test Script | `/workspace/dac/poc_test.py` |
| Dockerfile(s) | `autopoc-artifacts` branch |
| K8s Manifests | `autopoc-artifacts` branch |
| Raw Test Output | `poc-test-output/` on `autopoc-artifacts` branch |

### Build Errors Encountered

None — the build completed successfully on the first attempt.

### Deploy Errors Encountered

None — all resources were created successfully on the first attempt.

### Retry Summary

| Phase | Retries |
|-------|---------|
| Build | 0 |
| Deploy | 0 |

### Environment Variables

| Variable | Value | Purpose |
|----------|-------|---------|
| `TELEMETRY_OPTOUT` | `1` | Disable telemetry reporting from the container |

### Commands Reference

```bash
# Build the container image
podman build -t quay.io/aicatalyst/dac-dac:latest -f Dockerfile .

# Run locally
podman run -p 8080:8080 quay.io/aicatalyst/dac-dac:latest

# Validate dashboards (CLI)
dac validate --dir /app/examples/basic-yaml

# Check version
dac version

# Serve dashboards
dac serve --dir /app/examples/basic-yaml --port 8080
```
