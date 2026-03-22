# Getting Started

## Prerequisites

| Tool | Version |
|---|---|
| Go | 1.22+ |
| Node | 20+ |
| Allure CLI | 3.x (`npm i -g allure`) |

## Local development

**Terminal 1 — backend**

```bash
cd backend
cp .env.example .env   # fill in SESSION_SECRET and Google OAuth credentials
make run
```

The backend listens on `:8080`. It serves `/api/*`, `/auth/*`, and `/reports/*`.

**Terminal 2 — frontend**

```bash
make dev-frontend
```

Vite dev server starts on `:5173` and proxies `/api`, `/auth`, and `/reports` to `:8080`.

Open [http://localhost:5173](http://localhost:5173) and sign in with Google.

## Backend `.env`

Create `backend/.env` (gitignored):

```bash
SESSION_SECRET=<openssl rand -hex 32>
BASE_URL=http://localhost:8080
SECURE_COOKIE=false
GOOGLE_CLIENT_ID=<your-google-client-id>
GOOGLE_CLIENT_SECRET=<your-google-client-secret>
AUTH_POLICY_FILE=./policy.yaml
AUTH_AFTER_LOGIN_URL=http://localhost:5173/
AUTH_AFTER_LOGOUT_URL=http://localhost:5173/login
```

## RBAC policy

Create `backend/policy.yaml` (gitignored):

```yaml
roles:
  admin:
    permissions: ["*"]
    members:
      - you@example.com

  developer:
    permissions: ["view", "upload"]

  viewer:
    permissions: ["view"]

default_role: viewer
```

See [Authentication](authentication.md) for full RBAC details.

## Production build

```bash
# Build frontend + backend binary
make build

# Or build + push Docker image
make docker-build IMAGE=myregistry/allure-hub TAG=v1.0.0
make docker-push IMAGE=myregistry/allure-hub TAG=v1.0.0
```

## Sample Java app

`sample-java-app/` is a self-contained Maven project with JUnit 5 + Allure that uploads results to allure-hub.

```bash
# Run tests and upload to local instance
make sample-run ALLURE_HUB_URL=http://localhost:8080
```

### GitHub Actions example

```yaml
- name: Run tests
  run: mvn test
  working-directory: sample-java-app

- name: Upload to allure-hub
  env:
    ALLURE_HUB_URL: ${{ secrets.ALLURE_HUB_URL }}
    ALLURE_HUB_TOKEN: ${{ secrets.ALLURE_HUB_TOKEN }}   # API key
    PROJECT_ID: my-project
    BUILD_ID: ${{ github.run_number }}
  run: bash upload-results.sh
  working-directory: sample-java-app
```

Create an API key from **Settings → API Keys** (requires `manage` permission) and store the plaintext value as a repository secret. See [API key authentication](authentication.md#api-key-authentication) for details.
