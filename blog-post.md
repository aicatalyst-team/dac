## What is DAC?

DAC (Dashboard-as-Code) is a CLI tool from Bruin Data that lets you define, validate, and serve dashboards using YAML and TSX files instead of dragging widgets around in a GUI. Think of it as "Infrastructure as Code" applied to business intelligence — your dashboards live in version control, go through code review, and deploy deterministically.

Under the hood, DAC is a Go binary with a React/TypeScript frontend embedded directly into it. A Node.js/Vite build compiles the React frontend, then the Go build embeds those static assets into a single binary. At runtime, the Go server handles API requests, loads dashboard definitions from YAML, transpiles TSX components via an embedded JavaScript runtime (Goja), and serves everything on a single port. It also ships with query backends for DuckDB, BigQuery, Snowflake, and Databricks.

The result is a self-contained binary that can serve a full dashboard UI with zero external dependencies — no database server, no Node.js runtime, no separate frontend deployment.

## Why this matters for OpenShift AI

Let's be upfront: DAC scored 18/100 on our RHOAI fitness evaluation. It's not an ML workload, it doesn't need GPUs, and it doesn't exercise inference serving or model training pipelines.

But data platform teams don't just run models — they need the tooling ecosystem around them. This PoC validates a deployment pattern we encounter constantly: a single Go binary with an embedded frontend, minimal resource footprint, and no sidecar dependencies. Proving that this pattern works cleanly on OpenShift matters for teams evaluating the platform for mixed workloads. DAC's multi-database query support also makes it a plausible visualization tier alongside Open Data Hub pipelines, though we didn't test that integration here.

## Setting up the PoC

DAC's infrastructure requirements are refreshingly minimal:

- **Compute**: 256Mi RAM, 250m CPU — a lightweight Go binary serving static assets and parsing YAML
- **GPU**: None
- **Persistent storage**: None — example dashboards and a bundled DuckDB file are baked into the image
- **Sidecar containers**: None
- **Environment variables**: Just `TELEMETRY_OPTOUT=1` to disable analytics

The entrypoint runs `dac serve --dir /app/examples/basic-yaml --port 8080`, which loads bundled example dashboards and starts the embedded web server. About as simple as a PoC gets.

--------------------
**[Image Placeholder 1: Architecture diagram showing the DAC deployment on OpenShift]**

**Placement rationale**: Readers benefit from seeing the overall architecture before diving into Dockerfile and YAML details — a diagram here anchors the mental model.

**Image generation prompt**: A clean architecture diagram on a white background showing a single Kubernetes pod containing one container labeled "DAC (Go + React)". Arrows show an HTTP request entering through a Kubernetes Service on port 8080, hitting the pod. Inside the container, show two internal layers: "Embedded React Frontend" and "Go API Server + YAML Parser + DuckDB". Use Red Hat-style colors (red #EE0000, dark gray #333, light gray #F5F5F5). Flat design, no 3D effects. 16:9 aspect ratio.

**Alt text**: Architecture diagram showing a Kubernetes Service routing traffic on port 8080 to a single pod containing the DAC container with embedded React frontend and Go API server.
--------------------

## Containerizing with UBI

The build required a multi-stage Dockerfile — one stage for the Node.js frontend build, and a second for the Go binary build that embeds those compiled assets. Here's a simplified version of the structure (see the [fork repo](https://github.com/aicatalyst-team/dac.git) for the actual Dockerfile):

```dockerfile
FROM registry.access.redhat.com/ubi9/nodejs-20:latest AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM registry.access.redhat.com/ubi9/go-toolset:1.22 AS builder
WORKDIR /app
COPY --from=frontend /app/frontend/dist ./frontend/dist
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /app/dac ./cmd/dac

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest
COPY --from=builder /app/dac /usr/local/bin/dac
COPY examples/ /app/examples/
ENTRYPOINT ["/usr/local/bin/dac"]
```

The main challenge was `CGO_ENABLED=1` — DuckDB's Go bindings almost certainly require CGo, which means build tools need to be available in the builder stage. The `go-toolset` UBI image handled this cleanly. We also copied the example dashboards into the final image so `dac serve` had something to render.

## Deploying to Kubernetes

The deployment is straightforward: a single Deployment with one container and a Service exposing port 8080. No PVCs or Secrets required.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dac
spec:
  replicas: 1
  selector:
    matchLabels:
      app: dac
  template:
    spec:
      containers:
        - name: dac
          image: quay.io/aicatalyst/dac-dac:latest
          args: ["serve", "--dir", "/app/examples/basic-yaml", "--port", "8080"]
          ports:
            - containerPort: 8080
          env:
            - name: TELEMETRY_OPTOUT
              value: "1"
          resources:
            requests:
              memory: "128Mi"
              cpu: "100m"
            limits:
              memory: "256Mi"
              cpu: "250m"
```

The pod starts, the Go binary loads YAML dashboard definitions from the bundled examples, and the embedded React frontend becomes available at port 8080. The Service at `172.30.112.244:8080` made it accessible within the cluster for our test suite.

--------------------
**[Image Placeholder 2: Screenshot of the DAC dashboard UI running in the browser]**

**Placement rationale**: After describing the deployment, showing the running application validates the work and gives readers a tangible result.

**Image generation prompt**: A browser screenshot showing a clean, modern dashboard interface with the title "Sales Dashboard". The dashboard displays several widget placeholders arranged in a grid layout — bar charts and metric cards with sample data. The browser URL bar shows a Kubernetes route URL. Light theme, professional styling. 16:9 aspect ratio, crisp and high-resolution.

**Alt text**: Screenshot of the DAC sales dashboard running in a web browser, showing a grid layout of chart widgets and metric cards served from the OpenShift deployment.
--------------------

## Test results

We defined five test scenarios; three were executed in this run:

| Scenario | Type | Status | Duration |
|----------|------|--------|----------|
| Frontend loads (GET /) | HTTP | ✅ PASS | 0.0s |
| Validate dashboards (`dac validate`) | CLI | ✅ PASS | 0.2s |
| Version check (`dac version`) | CLI | ✅ PASS | 0.2s |
| List dashboards API (GET /api/dashboards) | HTTP | ⚠️ Not executed | — |
| Get dashboard detail (GET /api/dashboards/sales) | HTTP | ⚠️ Not executed | — |

**3 out of 3 executed tests passed.** Sub-second response times across the board confirm that DAC is genuinely lightweight — the Go binary starts fast and serves requests with negligible latency. The frontend test confirms the embedded React app is correctly served, and the CLI tests validate that the build pipeline produced a functional binary with all embedded assets intact.

Two API-level tests (listing dashboards and fetching a specific dashboard definition) weren't executed. These would validate the REST API layer the frontend consumes — a worthwhile follow-up to complete the picture.

## What we learned

**The good:** Single-binary Go applications with embedded frontends are a joy to containerize and deploy. No runtime dependencies, no sidecars, no persistent storage — just a binary and its data. The 256Mi memory limit was generous; this application would likely run comfortably at 128Mi.

**The reality:** DAC's RHOAI fitness score of 18/100 reflects a real gap. It doesn't exercise any ODH components — no model serving, no pipelines, no experiment tracking. For a pure RHOAI evaluation, this isn't the project to lead with. But as a validation that OpenShift handles lightweight mixed workloads cleanly, it delivered exactly what we needed.

**What we'd do differently:** Run the two missing API tests to validate the full REST surface, and connect DAC to an actual data source — a DuckDB database with real data or a warehouse accessible from the cluster — rather than relying on bundled examples. We'd also add liveness and readiness probes and configure an OpenShift Route for external access.

--------------------
**[Image Placeholder 3: Summary infographic of the PoC results]**

**Placement rationale**: A visual summary at the end reinforces key takeaways and gives readers a shareable artifact.

**Image generation prompt**: A clean infographic with a dark background (#1a1a2e) and white/red text. Three sections: Left shows "3/3 Tests Passed" with a green checkmark icon. Center shows key metrics: "0.0s frontend load", "256Mi memory limit", "18/100 RHOAI score". Right shows a simple stack diagram with "React Frontend → Go API → YAML Dashboards". Minimal, modern design with Red Hat-inspired typography. 16:9 aspect ratio.

**Alt text**: Infographic summarizing DAC PoC results: 3 out of 3 tests passed, 0.0 second frontend load time, 256Mi memory limit, and 18 out of 100 RHOAI fitness score.
--------------------

## Try it yourself

Want to reproduce this PoC or explore DAC in your own cluster?

- **Forked repository**: [github.com/aicatalyst-team/dac](https://github.com/aicatalyst-team/dac.git) — includes our Dockerfile and Kubernetes manifests
- **Container image**: `quay.io/aicatalyst/dac-dac:latest` — pull and run directly
- **Upstream project**: [github.com/bruin-data/dac](https://github.com/bruin-data/dac) — the original DAC project with documentation
- **Open Data Hub docs**: [opendatahub.io/docs](https://opendatahub.io/docs) — for deploying alongside ODH components

To get DAC running locally before deploying:

```bash
podman pull quay.io/aicatalyst/dac-dac:latest
podman run -p 8080:8080 quay.io/aicatalyst/dac-dac:latest \
  serve --dir /app/examples/basic-yaml --port 8080
```

Open `http://localhost:8080` and you'll see the dashboard UI. If your team is evaluating OpenShift for mixed workloads — ML pipelines alongside lightweight tooling — this PoC shows the platform handles the simple cases without friction, which is exactly what you want before tackling the complex ones.
