# allure-hub

Self-hosted Allure 3 reporting platform. A Go API serves the React frontend and all Allure reports from a single container — no nginx sidecar required.

## Features

- **Multi-environment / multi-project** — organise reports by environment (staging, production) and project
- **Two upload strategies** — single streaming upload or chunked upload for large files
- **Live upload tracking** — real-time progress via SSE, visible across all connected clients
- **Allure 3 reports** — automatic history stitching for trend charts across builds
- **Google OAuth + RBAC** — role-based access: `admin`, `developer`, `viewer`
- **Single container** — Go binary + Allure CLI + React SPA in one image
- **SQLite or PostgreSQL** — SQLite for single-node; Postgres for HA

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  Transport  (internal/transport)                            │
│  HTTP router · handlers · middleware · auth                 │
├─────────────────────────────────────────────────────────────┤
│  Use-cases  (internal/usecase)                              │
│  EnvironmentService · ProjectService · ReportService        │
│  UploadService · EventBus                                   │
├─────────────────────────────────────────────────────────────┤
│  Domain  (internal/domain)                                  │
│  Environment · Project · Build · UploadSession entities     │
│  Repository interfaces                                      │
├──────────────────┬──────────────────────────────────────────┤
│  Repository      │  Storage          │  Allure              │
│  SQLite/Postgres │  PVC filesystem   │  CLI wrapper         │
└──────────────────┴───────────────────┴──────────────────────┘
```

## Data layout

```
/data/
  allure-hub.db                  ← SQLite metadata
  {projectID}/
    history/                     ← persisted for trend charts
    results/{buildID}/           ← unzipped allure-results
    reports/{buildID}/           ← allure generate output
    uploads/{uploadId}/          ← chunk staging (cleaned after assembly)
```
