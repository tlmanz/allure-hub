# API Reference

All `/api/*` routes require authentication. Permission requirements are noted per endpoint.

## Environments

| Method | Path | Permission | Description |
|---|---|---|---|
| `GET` | `/api/environments` | `view` | List all environments |
| `POST` | `/api/environments` | `manage` | Create an environment |
| `PATCH` | `/api/environments/{envId}` | `manage` | Update an environment |
| `DELETE` | `/api/environments/{envId}` | `manage` | Delete an environment |

## Projects

| Method | Path | Permission | Description |
|---|---|---|---|
| `GET` | `/api/environments/{envId}/projects` | `view` | List projects in an environment |
| `POST` | `/api/environments/{envId}/projects` | `manage` | Create a project |
| `DELETE` | `/api/environments/{envId}/projects/{projectId}` | `manage` | Delete a project |

## Upload — Strategy A: streaming

Upload a single zip file in one request.

```
POST /api/environments/{envId}/projects/{projectId}/results
```

**Permission:** `upload`

**Headers:**

**Query parameters:**

| Parameter | Required | Description |
|---|---|---|
| `buildId` | Yes | Unique build identifier |

**Headers:**

| Header | Required | Description |
|---|---|---|
| `Content-Type` | Yes | `application/zip` or `application/octet-stream` |

**Body:** raw zip bytes

Report generation is triggered automatically after the upload completes. The response is returned once the results are saved; generation runs server-side.

## Upload — Strategy B: chunked

For large files (>50 MB). Initialise, send chunks, then complete.

### 1. Initialise

```
POST /api/environments/{envId}/projects/{projectId}/uploads
```

**Permission:** `upload`

**Body (JSON):**

```json
{
  "fileName": "results.zip",
  "totalSize": 104857600,
  "totalChunks": 3,
  "buildId": "2024-01-15-001"
}
```

**Response (201):**

```json
{ "uploadId": "01HXY..." }
```

### 2. Upload a chunk

```
PUT /api/environments/{envId}/projects/{projectId}/uploads/{uploadId}
```

**Permission:** `upload`

**Headers:**

| Header | Description |
|---|---|
| `X-Chunk-Index` | Zero-based chunk index |
| `X-Total-Chunks` | Total number of chunks |

**Body:** raw chunk bytes

### 3. Complete

```
POST /api/environments/{envId}/projects/{projectId}/uploads/{uploadId}/complete
```

**Permission:** `upload`

Assembles all chunks, extracts the zip, and triggers report generation.

## Reports

| Method | Path | Permission | Description |
|---|---|---|---|
| `POST` | `/api/environments/{envId}/projects/{projectId}/reports` | `upload` | Trigger report generation for a build |
| `GET` | `/api/environments/{envId}/projects/{projectId}/reports` | `view` | List reports |
| `GET` | `/api/environments/{envId}/projects/{projectId}/reports/stats` | `view` | Aggregated pass/fail stats |
| `DELETE` | `/api/environments/{envId}/projects/{projectId}/reports/{buildId}` | `manage` | Delete a report and its files |

### Trigger report generation

```
POST /api/environments/{envId}/projects/{projectId}/reports
```

**Permission:** `upload`

**Body (JSON):**

```json
{
  "buildId": "2024-01-15-001",
  "reportConfig": {
    "name": "My Report",
    "qualityGate": {
      "rules": [
        { "maxFailures": 2, "fastFail": true }
      ]
    },
    "plugins": {
      "allure2": {
        "options": {
          "reportName": "My Report",
          "singleFile": false,
          "reportLanguage": "en"
        }
      },
      "dashboard": {
        "options": {
          "reportName": "My Dashboard",
          "singleFile": false,
          "reportLanguage": "en"
        }
      },
      "csv": {
        "options": {
          "fileName": "allure-report.csv"
        }
      }
    },
    "variables": {
      "env": "production"
    }
  }
}
```

`buildId` must match the one used during upload. `reportConfig` is optional — if omitted, the server's default `allurerc.yml` is used.

The `reportConfig` body maps directly to the [allurerc.yml schema](https://allurereport.org/docs/reference-allurerc/). The server merges your overrides on top of its base config. Two keys are always server-controlled and cannot be overridden: `output` (report output path) and `historyPath` (trend chart history).

**Response (202):**

```json
{ "reportUrl": "/reports/env-1/project-1/2024-01-15-001/index.html" }
```

#### Sending `allurerc.yml` from CI

If you manage report config as an `allurerc.yml` file in your repo, parse and inline it into the `reportConfig` field when calling this endpoint:

=== "curl + yq"

    ```bash
    REPORT_CONFIG=$(yq -o=json '.' allurerc.yml)

    curl -X POST \
      -H "Authorization: Bearer $TOKEN" \
      -H "Content-Type: application/json" \
      -d "{\"buildId\": \"$BUILD_ID\", \"reportConfig\": $REPORT_CONFIG}" \
      https://your-host/api/environments/$ENV_ID/projects/$PROJECT_ID/reports
    ```

=== "Python"

    ```python
    import yaml, json, httpx

    with open("allurerc.yml") as f:
        report_config = yaml.safe_load(f)

    httpx.post(
        f"{BASE_URL}/api/environments/{env_id}/projects/{project_id}/reports",
        headers={"Authorization": f"Bearer {token}"},
        json={"buildId": build_id, "reportConfig": report_config},
    )
    ```

## Upload sessions

| Method | Path | Permission | Description |
|---|---|---|---|
| `GET` | `/api/uploads` | `view` | List all upload sessions (all envs/projects) |
| `GET` | `/api/uploads/stream` | `view` | SSE stream of live upload events |
| `DELETE` | `/api/uploads/{id}` | `manage` | Delete a session record and associated files |

### SSE stream

`GET /api/uploads/stream` returns a persistent `text/event-stream` connection. Each event is a JSON-encoded `UploadSession` object:

```
event: update
data: {"id":"01HXY...","phase":"uploading","receivedChunks":1,"totalChunks":3,...}
```

Phases: `uploading` → `assembling` → `generating` → `done` / `failed`

## Health

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/api/healthz` | None | Liveness probe — returns `200 OK` |
| `GET` | `/api/version` | None | Build version, time, and Go version |

### `/api/version` response

```json
{
  "version": "v1.2.3",
  "buildTime": "2024-01-15T10:00:00Z",
  "goVersion": "go1.22.0"
}
```

## Settings

Settings endpoints require an active **session** (OAuth login). API keys cannot be used to manage settings.

### API Keys

| Method | Path | Permission | Description |
|---|---|---|---|
| `GET` | `/api/settings/apikeys` | `manage` | List all API keys (hashes never returned) |
| `POST` | `/api/settings/apikeys` | `manage` | Create a new API key |
| `DELETE` | `/api/settings/apikeys/{id}` | `manage` | Revoke or permanently delete a key |

#### Create API key

```
POST /api/settings/apikeys
```

**Body (JSON):**

```json
{
  "name": "ci-pipeline",
  "role": "developer"
}
```

`role` must be one of `admin`, `developer`, or `viewer`.

**Response (201):**

```json
{
  "id": "01HXY...",
  "name": "ci-pipeline",
  "role": "developer",
  "key": "ah_a3f9..."
}
```

!!! warning "Save the key now"
    The plaintext key is returned **once** at creation time. The server stores only its SHA-256 hash and cannot recover the original value.

#### Delete or revoke API key

```
DELETE /api/settings/apikeys/{id}
DELETE /api/settings/apikeys/{id}?action=delete
```

- Default (`action` omitted or `action=revoke`): soft-delete — key is marked inactive but the record is retained.
- `action=delete`: permanently removes the key record.

### Users

| Method | Path | Permission | Description |
|---|---|---|---|
| `GET` | `/api/settings/users` | `manage` | List all OAuth users who have signed in |

Returns users ordered by most recent login. Each entry includes `email`, `name`, `avatarUrl`, `provider`, `role`, `firstLoginAt`, and `lastLoginAt`.

## Auth endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/auth/{provider}` | Start OAuth flow (`provider` = `google`) |
| `GET` | `/auth/{provider}/callback` | OAuth callback |
| `POST` | `/auth/logout` | Clear session |
| `GET` | `/auth/me` | Current user or `401` |

## Error responses

All error responses use JSON:

```json
{ "error": "description" }
```

| Status | Meaning |
|---|---|
| `400` | Invalid request body or parameters |
| `401` | Not authenticated |
| `403` | Authenticated but missing permission |
| `404` | Resource not found |
| `409` | Conflict (e.g. duplicate build ID) |
| `500` | Internal server error |
