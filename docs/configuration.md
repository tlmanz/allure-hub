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

## Database

| Variable | Default | Description |
|---|---|---|
| `DB_DRIVER` | `sqlite` | `sqlite` or `postgres` |
| `DB_DSN` | `./data/allure-hub.db` | SQLite path or Postgres DSN |
| `DB_MAX_OPEN_CONNS` | `25` | Connection pool size |

**PostgreSQL DSN example:**

```
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
| `AUTH_POLICY_FILE` | `./policy.yaml` | Path to RBAC policy file |
| `AUTH_AFTER_LOGIN_URL` | `/` | Redirect URL after successful login |
| `AUTH_AFTER_LOGOUT_URL` | `/login` | Redirect URL after logout |

## Rate limiting

| Variable | Default | Description |
|---|---|---|
| `RATE_LIMIT_RATE` | `20` | Tokens per second per IP |
| `RATE_LIMIT_BURST` | `60` | Burst capacity |
| `TRUST_PROXY` | `false` | Trust `X-Forwarded-For` (enable behind a reverse proxy) |

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
