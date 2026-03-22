# allure-hub

Self-hosted Allure 3 reporting platform. A Go API serves the frontend and all
Allure Awesome reports from a single container — no nginx sidecar required.

## Architecture

Domain-driven design with clean architecture layering:

```
┌─────────────────────────────────────────────────────────────┐
│  Transport  (internal/transport)                            │
│  HTTP router · handlers · middleware                        │
├─────────────────────────────────────────────────────────────┤
│  Application  (internal/service)                            │
│  ProjectService · ReportService · UploadService             │
│  Ports: FileStorage interface · Generator interface         │
├─────────────────────────────────────────────────────────────┤
│  Domain  (internal/domain)                                  │
│  Project entity · Build entity                              │
│  ProjectRepository interface · BuildRepository interface    │
├──────────────────┬──────────────────────────────────────────┤
│  Repository      │  Storage          │  Allure              │
│  (internal/      │  (internal/       │  (internal/allure)   │
│   repository)    │   storage)        │                      │
│  SQL impl of     │  Filesystem impl  │  CLI wrapper impl    │
│  domain repos    │  of FileStorage   │  of Generator        │
│  SQLite/Postgres │  port             │  port                │
└──────────────────┴───────────────────┴──────────────────────┘

Runtime layout:
  PVC  /data/
    allure-hub.db               ← SQLite metadata (or Postgres)
    {projectID}/
      history/                  ← persisted for trend charts
      results/{buildID}/        ← unzipped allure-results
      reports/{buildID}/        ← allure generate output
      uploads/{uploadId}/       ← chunk staging (cleaned after assembly)
```

## Stack

| Layer | Tech |
|---|---|
| Backend | Go 1.22, `net/http` only (no frameworks) |
| Frontend | React 18, TypeScript, Vite 5, react-router-dom v6 |
| Reports | Allure 3 (`allure` npm package v3) |
| Storage | PVC filesystem + SQLite (or PostgreSQL) for metadata |
| Container | Single image (Node 22 Alpine + Go static binary) |
| Kubernetes | Single pod, single container, RWX PVC |

## Quick start

### Prerequisites

- Go 1.22+
- Node 20+
- Allure 3 CLI installed globally (`npm i -g allure`)

### Run locally

```bash
# Terminal 1 — backend (serves on :8080, proxied by Vite for /api)
make dev-backend

# Terminal 2 — frontend dev server (serves on :5173 with API proxy)
make dev-frontend
```

> The frontend Vite dev server proxies `/api/*` to `localhost:8080`.
> For a production-like run, build the frontend first (`make frontend-build`) then
> `WEB_DIR=frontend/dist make dev-backend`.

## Docker

Build context must be the **repo root** so the multi-stage Dockerfile can
reach both `frontend/` and `backend/`.

```bash
make docker-build          # IMAGE=ghcr.io/tlmanz/allure-hub TAG=latest

# Override image name/tag
make docker-build IMAGE=myregistry/allure-hub TAG=v1.2.3
```

## Kubernetes

```bash
# Deploy (applies manifests in dependency order)
make k8s-apply

# Tear down
make k8s-delete
```

Update `k8s/ingress.yaml` with your actual hostname before deploying.
The PVC uses `ReadWriteMany` — ensure your storage class supports it.

## Environment variables

| Variable | Default (container) | Description |
|---|---|---|
| `ADDR` | `:8080` | Listen address |
| `DATA_DIR` | `/data` | PVC mount path |
| `ALLURE_BIN` | `allure` | Path/name of the Allure 3 CLI binary |
| `ALLURE_PROFILE` | `awesome` | Allure report profile (`awesome` / empty for classic) |
| `WEB_DIR` | `/app/web` | Path to built frontend static files |
| `DB_DRIVER` | `sqlite` | Metadata DB driver: `sqlite` or `postgres` |
| `DB_DSN` | `/data/allure-hub.db` | SQLite file path or Postgres connection string |

To switch to PostgreSQL:

```bash
DB_DRIVER=postgres DB_DSN="postgres://user:pass@host:5432/allure_hub?sslmode=disable"
```

Migrations run automatically on startup for both drivers.

## API

```
POST   /api/projects                                  create project
GET    /api/projects                                  list projects
DELETE /api/projects/{id}                             delete project

POST   /api/projects/{id}/results?buildId=<id>        streaming zip upload
POST   /api/projects/{id}/uploads                     init chunked upload
PUT    /api/projects/{id}/uploads/{uploadId}          upload one chunk
POST   /api/projects/{id}/uploads/{uploadId}/complete assemble chunks

POST   /api/projects/{id}/reports                     generate Allure report
GET    /api/projects/{id}/reports                     list reports

GET    /healthz                                       liveness probe
```

### Chunked upload headers

```
PUT /api/projects/{id}/uploads/{uploadId}
X-Chunk-Index:  0          (0-based)
X-Total-Chunks: 10
Body:           raw chunk bytes
```

## Sample Java app

`sample-java-app/` is a self-contained Maven project that runs JUnit 5 tests
with Allure annotations and uploads the results to allure-hub.

### Prerequisites

- Java 17+, Maven 3.8+
- `curl`, `zip` (for the upload script)
- allure-hub running locally (`make dev-backend` or Docker)

### Run end-to-end

```bash
cd sample-java-app

# 1. Run tests — writes target/allure-results/
mvn test

# 2. Upload results and generate the Allure Awesome report
ALLURE_HUB_URL=http://localhost:8080 ./upload-results.sh
```

Or use Make from the repo root:

```bash
# Start allure-hub in another terminal first
make dev-backend

# Run sample tests, upload, generate report
make sample-run ALLURE_HUB_URL=http://localhost:8080
```

The script will print the report URL on completion:

```
Done.
Report: http://localhost:8080/reports/sample-java/20240101-120000/index.html
```

### Upload variables

| Variable | Default | Description |
|---|---|---|
| `ALLURE_HUB_URL` | `http://localhost:8080` | allure-hub base URL |
| `PROJECT_ID` | `sample-java` | Project to upload into |
| `PROJECT_NAME` | `Sample Java App` | Display name (created if absent) |
| `BUILD_ID` | `YYYYmmdd-HHMMSS` | Unique build identifier |
| `RESULTS_DIR` | `target/allure-results` | Path to allure-results directory |

### CI example (GitHub Actions)

```yaml
- name: Run tests
  run: mvn test
  working-directory: sample-java-app

- name: Upload to allure-hub
  env:
    ALLURE_HUB_URL: ${{ secrets.ALLURE_HUB_URL }}
    PROJECT_ID: my-project
    BUILD_ID: ${{ github.run_number }}
  run: bash upload-results.sh
  working-directory: sample-java-app
```

## Design constraints

- Never buffer a full zip in memory — always `io.Copy` to a temp file first.
- Chunk files are named by integer index only (`0`, `1`, `2` …). `meta.json` is the only non-integer file in a chunk directory.
- `history/` stitching happens before every `allure generate` call so trend charts work across builds.
- The frontend never renders Allure HTML — it only links to it via `reportUrl`.
- Repository interfaces are defined in `internal/domain`; SQL implementations in `internal/repository`.
- Infrastructure adapts to application ports (`service.FileStorage`, `service.Generator`) — no framework lock-in.

## Project layout

```
.
├── backend/
│   ├── cmd/server/main.go           entry point — DDD wiring
│   ├── pkg/config/                  goconf + caarlos0/env config
│   └── internal/
│       ├── domain/                  entities + repository interfaces (no deps)
│       │   ├── project.go           Project entity
│       │   ├── build.go             Build entity
│       │   └── repository.go        ProjectRepository + BuildRepository
│       ├── service/                 application use-cases
│       │   ├── ports.go             FileStorage + Generator interfaces
│       │   ├── project.go           ProjectService
│       │   ├── report.go            ReportService
│       │   └── upload.go            UploadService
│       ├── repository/              SQL infrastructure (SQLite / Postgres)
│       │   ├── db.go                Open() factory + placeholder rewriter
│       │   ├── migrate.go           embedded migration runner
│       │   ├── project.go           SQL ProjectRepository impl
│       │   ├── build.go             SQL BuildRepository impl
│       │   └── migrations/          SQL schema files
│       ├── storage/filesystem.go    PVC I/O — implements service.FileStorage
│       ├── allure/generator.go      Allure 3 CLI — implements service.Generator
│       └── transport/               HTTP delivery layer
│           ├── router.go            mux + static serving
│           ├── handler/             project.go · report.go · health.go
│           └── middleware/          logger.go
├── frontend/
│   └── src/
│       ├── api/client.ts
│       ├── components/
│       ├── pages/
│       └── types/
├── docker/
│   └── Dockerfile.backend           multi-stage: frontend → Go → Node runtime
├── k8s/                             Kubernetes manifests
├── sample-java-app/
│   ├── pom.xml                      Maven project (JUnit 5 + Allure)
│   ├── upload-results.sh            zip + POST results, trigger report
│   ├── Makefile
│   └── src/
└── Makefile
```
