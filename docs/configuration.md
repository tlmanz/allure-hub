# Configuration

All settings are loaded from environment variables. In local development, place them in `backend/.env` — the Makefile exports them automatically.

## Server

| Variable | Default | Description |
|---|---|---|
| `ADDR` | `:8080` | Listen address |
| `WEB_DIR` | `./app/web` | Path to compiled frontend static files |
| `READ_TIMEOUT` | `2h` | Must exceed the longest possible upload |
| `WRITE_TIMEOUT` | `10m` | Covers report serving; SSE clients reconnect on expiry |
| `SHUTDOWN_TIMEOUT` | `30s` | Graceful drain on SIGTERM |

## Storage

| Variable | Default | Description |
|---|---|---|
| `DATA_DIR` | `./data` | Root directory for reports, results, and uploads |
| `ASSEMBLE_TEMP_DIR` | `./temp` | Staging directory for chunk assembly |
| `MAX_CHUNK_BYTES` | `52428800` | 50 MB per chunk |
| `MAX_UPLOAD_BYTES` | `1073741824` | 1 GB compressed upload cap |
| `MAX_DECOMPRESSED_BYTES` | `1610612736` | 1.5 GB decompressed cap (zip-bomb protection) |
| `MAX_ZIP_ENTRIES` | `10000` | Max files in a zip (inode exhaustion protection) |

!!! warning "Upload limits"
    If uploads fail with `413` or timeout errors, increase `MAX_UPLOAD_BYTES`, `MAX_DECOMPRESSED_BYTES`, and `READ_TIMEOUT` accordingly.

## Database

| Variable | Default | Description |
|---|---|---|
| `DB_DRIVER` | `sqlite` | `sqlite` or `postgres` |
| `DB_DSN` | `./data/allure-hub.db` | SQLite path or Postgres DSN |
| `DB_MAX_OPEN_CONNS` | `25` | Connection pool size |

=== "SQLite (default)"

    ```bash
    DB_DRIVER=sqlite
    DB_DSN=/data/allure-hub.db
    ```

=== "PostgreSQL"

    ```bash
    DB_DRIVER=postgres
    DB_DSN=postgres://user:pass@host:5432/allure_hub?sslmode=require
    ```

Migrations run automatically on startup for both drivers.

## Allure CLI

| Variable | Default | Description |
|---|---|---|
| `ALLURE_BIN` | `allure` | Path/name of the Allure 3 CLI binary |
| `ALLURE_CONFIG` | `./settings/allurerc.yml` | Allure config file |
| `ALLURE_MAX_CONCURRENCY` | `4` | Max parallel `allure generate` invocations |
| `ALLURE_TIMEOUT` | `10m` | Per-invocation deadline |

## Authentication

| Variable | Default | Description |
|---|---|---|
| `SESSION_SECRET` | — **(required)** | 32-byte hex secret for cookie encryption |
| `BASE_URL` | — | Public base URL (e.g. `https://allure.example.com`) |
| `SECURE_COOKIE` | `false` | Set `true` in production (HTTPS only) |
| `GOOGLE_CLIENT_ID` | — | Google OAuth client ID |
| `GOOGLE_CLIENT_SECRET` | — | Google OAuth client secret |
| `AUTH_POLICY_FILE` | `./policy.yaml` | Path to RBAC policy file (used as baseline — see below) |
| `AUTH_AFTER_LOGIN_URL` | `/` | Redirect URL after successful login |
| `AUTH_AFTER_LOGOUT_URL` | `/login` | Redirect URL after logout |

## RBAC

Allure Hub uses a **layered RBAC** system:

```mermaid
flowchart LR
    A["policy.yaml\n(baseline)"] --> C{"Role resolver"}
    B["role_overrides\n(database)"] --> C
    C --> D["Effective role"]
    style B fill:#f9a825,color:#000
```

1. `policy.yaml` defines roles, their permissions, and initial members. This is the baseline and is hot-reloaded every 30 seconds.
2. Role overrides set through the Settings UI are stored in the `role_overrides` database table and take precedence over the YAML baseline.

!!! tip
    You can bootstrap roles via `policy.yaml` and manage individual user roles at runtime through the UI without editing files or redeploying.

**Changing a user's role** requires the `admin` role. The change takes effect on the user's next login — their current session is invalidated immediately so they are forced to re-authenticate with the new role.

**Resetting a user to the YAML baseline** is done by removing their override (the "Reset to default" action in the Settings UI).

??? note "role_overrides table schema"

    The migration that creates this table runs automatically on startup. For reference:

    ```sql
    CREATE TABLE role_overrides (
        email       TEXT PRIMARY KEY,
        role        TEXT NOT NULL,
        permissions JSONB NOT NULL DEFAULT '[]'
    );
    ```

    For PostgreSQL deployments this table is created in the same database as the rest of the schema.

## Cleanup Worker

The cleanup worker runs as a background goroutine and periodically deletes reports older than the configured retention period.

| Variable | Default | Description |
|---|---|---|
| `CLEANUP_RETENTION_DAYS` | `90` | Seed value for retention period in days. Reports older than this are permanently deleted. |
| `CLEANUP_INTERVAL` | `6h` | How often the cleanup worker runs. Set to `0` to disable the worker entirely. |
| `CLEANUP_DRY_RUN` | `false` | Seed value for dry-run mode. If `true`, the worker logs what it would delete but takes no action. |

!!! info "Runtime overrides"
    These are **startup seed values** — they are written to the `system_settings` database table on first run and can be changed at any time via **Settings → Data Retention** in the UI (or the `PUT /api/settings/retention` endpoint) without restarting the server. Database values always take precedence over environment variables after the first boot.

!!! tip "Disabling the worker"
    Set `CLEANUP_INTERVAL=0` to disable the cleanup worker entirely. Data retention settings in the UI will still be saved but no automatic deletion will occur.

The outcome of each sweep is recorded. The last **5 runs** are visible in **Settings → Data Retention → Recent Cleanup Runs**, including status (`success`/`failed`), number of builds deleted, skipped count, duration, and the error message if the sweep failed.

## Notifications

Allure Hub uses [`go-notify`](https://github.com/tlmanz/go-notify) for in-app notifications (SSE/WebSocket stream + repository-backed storage).

| Variable | Default | Description |
|---|---|---|
| `NOTIFY_REDIS_URL` | `` | Redis URL for persistent/shared notification storage. Leave empty to use in-memory storage. |
| `NOTIFY_REDIS_KEY_PREFIX` | `allure_hub_notify` | Redis key prefix for notification keys. |
| `NOTIFY_RETENTION_DAYS` | `30` | Notification retention window in days. `0` disables pruning. |

!!! info "Storage backend behavior"
    If `NOTIFY_REDIS_URL` is set, notifications are stored in Redis via the official `go-notify/redis` adapter. If it is unset, Allure Hub falls back to in-memory storage (good for local/dev, not durable across restarts).

!!! info "Logging"
    `go-notify` internal events are forwarded through the app's configured logger (`LOG_LEVEL`, `LOG_FORMAT`, `LOG_OUTPUT`) with `component=go-notify`.

## Rate limiting

| Variable | Default | Description |
|---|---|---|
| `RATE_LIMIT_RATE` | `20` | Tokens per second per IP |
| `RATE_LIMIT_BURST` | `60` | Burst capacity |
| `TRUST_PROXY` | `false` | Trust `X-Forwarded-For` (enable behind a reverse proxy) |

!!! warning
    Always set `TRUST_PROXY=true` when running behind a reverse proxy (nginx, Caddy, cloud LB), otherwise rate limiting will apply to the proxy IP instead of individual clients.

## CORS

| Variable | Default | Description |
|---|---|---|
| `CORS_ALLOWED_ORIGINS` | `` | Comma-separated allowed origins; empty = same-origin only |

In development with Vite proxy, CORS is not needed. In production with a CDN or separate domain, set this to your frontend origin.

## Logging

| Variable | Default | Description |
|---|---|---|
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, or `error` |
| `LOG_FORMAT` | `json` | `json` or `console` |
| `LOG_OUTPUT` | `stdout` | `stdout` or `stderr` |
