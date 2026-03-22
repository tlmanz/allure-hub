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

| Header | Required | Description |
|---|---|---|
| `Content-Type` | Yes | `application/zip` or `application/octet-stream` |
| `X-Build-Id` | No | Custom build identifier (auto-generated if omitted) |

**Body:** raw zip bytes

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
