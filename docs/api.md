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

## Notifications

Notification APIs are provided by `go-notify` and mounted at `/api/notifications/*`.

| Method | Path | Permission | Description |
|---|---|---|---|
| `GET` | `/api/notifications` | `view` | List notifications (newest first) |
| `GET` | `/api/notifications/stream` | `view` | SSE stream (`event: notification`) |
| `GET` | `/api/notifications/unread` | `view` | Unread count |
| `PATCH` | `/api/notifications/{id}/read` | `view` | Mark one notification as read |
| `POST` | `/api/notifications/read` | `view` | Mark all notifications as read |
| `DELETE` | `/api/notifications/{id}` | `view` | Delete one notification |

### List notifications

```
GET /api/notifications?unread=true&limit=50
```

**Query parameters:**

| Parameter | Required | Description |
|---|---|---|
| `unread` | No | `true` to return unread-only notifications |
| `limit` | No | Max number of notifications to return (`0` means no cap) |

**Response (200):**

```json
[
  {
    "id": "0195f3a2-...",
    "title": "Report ready",
    "body": "Your export has finished.",
    "category": "upload",
    "severity": "success",
    "read": false,
    "created_at": "2026-04-04T10:30:00Z",
    "payload": { "url": "/reports/build1/index.html" }
  }
]
```

### Unread count

```
GET /api/notifications/unread
```

**Response (200):**

```json
{ "count": 3 }
```

### Read-state endpoints

```
PATCH /api/notifications/{id}/read   -> 204 No Content
POST  /api/notifications/read        -> 204 No Content
```

`PATCH /{id}/read` returns `404` when the notification ID does not exist.

### Streaming

SSE endpoint: `GET /api/notifications/stream`

SSE payload format:

```
event: notification
data: {"id":"...","title":"Report ready","severity":"success",...}
```

## Overview

| Method | Path | Permission | Description |
|---|---|---|---|
| `GET` | `/api/overview` | `view` | Dashboard analytics summary, trends, top failing projects, and recent builds |

**Response (200):**

```json
{
  "summary": {
    "totalEnvironments": 3,
    "totalProjects": 18,
    "totalBuilds": 1240,
    "totalPassed": 98210,
    "totalFailed": 4170,
    "overallPassRate": 95
  },
  "dailyTrends": [
    { "date": "2026-04-01", "passed": 1200, "failed": 42, "skipped": 15, "buildCount": 14 }
  ],
  "topFailingProjects": [
    {
      "envId": "staging",
      "projectId": "checkout",
      "projectName": "Checkout Service",
      "envName": "Staging",
      "totalFailed": 310,
      "totalBuilds": 80,
      "passRate": 88
    }
  ],
  "recentBuilds": []
}
```

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
| `PATCH` | `/api/settings/users/{email}/role` | `manage` (admin only) | Override a user's role |
| `DELETE` | `/api/settings/users/{email}/role` | `manage` (admin only) | Reset a user's role to the YAML baseline |

Returns users ordered by most recent login. Each entry includes `email`, `name`, `avatarUrl`, `provider`, `role`, `firstLoginAt`, and `lastLoginAt`.

Role changes take effect on the user's **next login**.

### Disk Usage

| Method | Path | Permission | Description |
|---|---|---|---|
| `GET` | `/api/settings/disk` | `manage` | Storage consumed by the data directory |
| `GET` | `/api/settings/disk/notification-threshold` | `manage` | Get disk-usage alert threshold percentage |
| `PUT` | `/api/settings/disk/notification-threshold` | `manage` | Update disk-usage alert threshold percentage |

```
GET /api/settings/disk
```

Results are cached for 60 seconds on the server to keep the endpoint fast.

**Response (200):**

```json
{
  "usedBytes": 2147483648,
  "freeBytes": 53687091200,
  "totalBytes": 107374182400,
  "breakdown": [
    { "path": "production/checkout", "bytes": 1073741824 },
    { "path": "staging/checkout",    "bytes": 536870912  }
  ]
}
```

| Field | Description |
|---|---|
| `usedBytes` | Total bytes consumed by all files under `DATA_DIR` |
| `freeBytes` | Available bytes on the filesystem (`Bavail × Bsize`). `0` if the stat call fails. |
| `totalBytes` | Total filesystem capacity. `0` if the stat call fails. |
| `breakdown` | Up to 20 env/project pairs sorted by size descending |

#### Disk notification threshold

```
GET /api/settings/disk/notification-threshold
```

**Response (200):**

```json
{ "thresholdPercent": 85 }
```

`85` is the default when no value has been saved yet.

```
PUT /api/settings/disk/notification-threshold
```

**Body (JSON):**

```json
{ "thresholdPercent": 90 }
```

| Status | Reason |
|---|---|
| `204` | Updated successfully |
| `400` | Invalid body or `thresholdPercent` outside `0..100` |
| `500` | Failed to persist the setting |

### Allure CLI

| Method | Path | Permission | Description |
|---|---|---|---|
| `GET` | `/api/settings/allure` | `manage` | Get installed version and latest npm release |
| `PUT` | `/api/settings/allure` | `manage` (admin only) | Install a specific Allure CLI version |

#### Get Allure version

```
GET /api/settings/allure
```

**Response (200):**

```json
{
  "version": "3.3.1",
  "latest": "3.4.0"
}
```

| Field | Description |
|---|---|
| `version` | The Allure CLI version currently installed on the server (`allure --version`) |
| `latest` | The latest version published to npm (`registry.npmjs.org/allure/latest`), cached for 1 hour. Empty string if the registry is unreachable. |

The UI shows an **Update available** banner in **Settings → Allure CLI** when `latest` differs from `version`, and an amber dot on the Settings nav link.

#### Install a version

```
PUT /api/settings/allure
```

**Permission:** `manage` session + `admin` role

Runs `npm install -g allure@<version>` on the server. The updated binary is used immediately for all subsequent report generations — no restart required.

**Body (JSON):**

```json
{ "version": "3.4.0" }
```

`version` must be a valid semver string (`MAJOR.MINOR.PATCH`). Pre-release tags (e.g. `3.4.0-beta.1`) are not accepted. This endpoint works for both upgrades and downgrades.

**Response (200):**

```json
{ "version": "3.4.0" }
```

**Error responses:**

| Status | Reason |
|---|---|
| `400` | Missing or invalid semver version string |
| `403` | Caller is not an admin |
| `500` | `npm install` failed (stderr included in response body) |

### Data Retention

| Method | Path | Permission | Description |
|---|---|---|---|
| `GET` | `/api/settings/retention` | `manage` | Get current retention settings |
| `PUT` | `/api/settings/retention` | `manage` | Update retention settings |
| `GET` | `/api/settings/retention/runs` | `manage` | List recent cleanup sweep runs |

#### Get retention settings

```
GET /api/settings/retention
```

**Response (200):**

```json
{
  "retentionDays": 90,
  "intervalHours": 6,
  "dryRun": false
}
```

#### Update retention settings

```
PUT /api/settings/retention
```

**Body (JSON):**

```json
{
  "retentionDays": 90,
  "intervalHours": 6,
  "dryRun": false
}
```

| Field | Description |
|---|---|
| `retentionDays` | Reports older than this many days are permanently deleted. Minimum: `1`. |
| `intervalHours` | How often the cleanup worker runs, in hours. Minimum: `1`. |
| `dryRun` | If `true`, the worker logs what it would delete but takes no action. |

**Response:** `204 No Content`

#### List cleanup runs

```
GET /api/settings/retention/runs?limit=5
```

Returns the most recent cleanup sweep records, ordered newest first. At most **5 records** are retained in the database; older runs are automatically pruned after each sweep.

**Query parameters:**

| Parameter | Default | Description |
|---|---|---|
| `limit` | `5` | Number of records to return (max `50`) |

**Response (200):**

```json
[
  {
    "id": "01HXY...",
    "startedAt": "2024-01-15T02:00:00Z",
    "finishedAt": "2024-01-15T02:00:03Z",
    "status": "success",
    "deletedCount": 12,
    "skippedCount": 0,
    "dryRun": false
  },
  {
    "id": "01HXX...",
    "startedAt": "2024-01-14T20:00:00Z",
    "finishedAt": "2024-01-14T20:00:00Z",
    "status": "failed",
    "deletedCount": 0,
    "skippedCount": 0,
    "dryRun": false,
    "errorMessage": "cleanup: list expired builds: context deadline exceeded"
  }
]
```

| Field | Description |
|---|---|
| `status` | `success` or `failed` |
| `deletedCount` | Number of builds removed (or would-be removed in dry-run mode) |
| `skippedCount` | Builds that encountered a per-record error and were skipped |
| `dryRun` | Whether this run was a dry-run (no data was actually deleted) |
| `errorMessage` | Present only on `failed` runs — the top-level error that aborted the sweep |

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
