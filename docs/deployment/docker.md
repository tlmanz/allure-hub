# Docker

## Build

The multi-stage Dockerfile builds the React frontend, compiles the Go binary, and packages both with the Allure 3 CLI into a single Node 22 Alpine image.

Build context must be the **repo root**:

```bash
make docker-build
# or with custom image/tag:
make docker-build IMAGE=myregistry/allure-hub TAG=v1.2.3
```

## Run

```bash
docker run -d \
  --name allure-hub \
  -p 8080:8080 \
  -v /srv/allure-hub/data:/data \
  -e SESSION_SECRET="$(openssl rand -hex 32)" \
  -e BASE_URL=https://allure.example.com \
  -e SECURE_COOKIE=true \
  -e GOOGLE_CLIENT_ID=your-client-id \
  -e GOOGLE_CLIENT_SECRET=your-client-secret \
  ghcr.io/tlmanz/allure-hub:latest
```

## Environment variables

Pass all [configuration](../configuration.md) as `-e` flags or via an env file:

```bash
docker run --env-file /etc/allure-hub/env ...
```

Example `/etc/allure-hub/env`:

```bash
SESSION_SECRET=<32-byte hex>
BASE_URL=https://allure.example.com
SECURE_COOKIE=true
GOOGLE_CLIENT_ID=744562771603-....apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GOCSPX-...
AUTH_POLICY_FILE=/data/policy.yaml
DB_DRIVER=sqlite
DB_DSN=/data/allure-hub.db
DATA_DIR=/data
LOG_LEVEL=info
```

## Persistent storage

Mount `/data` to preserve reports and the database across container restarts:

```bash
-v /srv/allure-hub/data:/data
```

The RBAC `policy.yaml` baseline can be stored there and referenced via `AUTH_POLICY_FILE=/data/policy.yaml`. Role overrides set through the Settings UI are stored in the database automatically — no file changes needed at runtime.

## Docker Compose example

```yaml
services:
  allure-hub:
    image: ghcr.io/tlmanz/allure-hub:latest
    ports:
      - "8080:8080"
    volumes:
      - allure_data:/data
    environment:
      SESSION_SECRET: "${SESSION_SECRET}"
      BASE_URL: "https://allure.example.com"
      SECURE_COOKIE: "true"
      GOOGLE_CLIENT_ID: "${GOOGLE_CLIENT_ID}"
      GOOGLE_CLIENT_SECRET: "${GOOGLE_CLIENT_SECRET}"
      AUTH_POLICY_FILE: "/data/policy.yaml"
      DB_DSN: "/data/allure-hub.db"
      DATA_DIR: "/data"
    restart: unless-stopped

volumes:
  allure_data:
```

## Docker Compose with PostgreSQL

For production deployments, PostgreSQL is recommended. All schema migrations (including the `role_overrides` table) run automatically on startup.

```yaml
services:
  allure-hub:
    image: ghcr.io/tlmanz/allure-hub:latest
    ports:
      - "8080:8080"
    volumes:
      - allure_data:/data
    environment:
      SESSION_SECRET: "${SESSION_SECRET}"
      BASE_URL: "https://allure.example.com"
      SECURE_COOKIE: "true"
      GOOGLE_CLIENT_ID: "${GOOGLE_CLIENT_ID}"
      GOOGLE_CLIENT_SECRET: "${GOOGLE_CLIENT_SECRET}"
      DB_DRIVER: "postgres"
      DB_DSN: "postgres://allure:${POSTGRES_PASSWORD}@db:5432/allure_hub?sslmode=disable"
      DATA_DIR: "/data"
    depends_on:
      db:
        condition: service_healthy
    restart: unless-stopped

  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: allure_hub
      POSTGRES_USER: allure
      POSTGRES_PASSWORD: "${POSTGRES_PASSWORD}"
    volumes:
      - pg_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U allure -d allure_hub"]
      interval: 5s
      retries: 5
    restart: unless-stopped

volumes:
  allure_data:
  pg_data:
```

## Push to registry

```bash
make docker-push IMAGE=ghcr.io/tlmanz/allure-hub TAG=v1.2.3
```

This builds (if not already built) and pushes the image.
