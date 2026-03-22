# Authentication

allure-hub uses Google OAuth for login and a YAML-based RBAC policy for permissions.

## Google OAuth setup

1. Go to [Google Cloud Console](https://console.cloud.google.com) → APIs & Services → Credentials → **Create OAuth 2.0 Client ID**
2. Application type: **Web application**
3. Add authorised redirect URI:
   - Development: `http://localhost:8080/auth/google/callback`
   - Production: `https://your-domain.com/auth/google/callback`
4. Copy the client ID and secret into `backend/.env`:

```bash
GOOGLE_CLIENT_ID=744562771603-....apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GOCSPX-...
BASE_URL=https://your-domain.com
```

!!! note "Development redirect"
    In development, Google redirects directly to `:8080` (bypassing Vite). `AUTH_AFTER_LOGIN_URL=http://localhost:5173/` redirects back to the frontend after the callback completes.

## RBAC policy

Create `backend/policy.yaml` (gitignored). The file is watched and hot-reloaded every 30 seconds — no restart needed.

```yaml
roles:
  admin:
    permissions: ["*"]
    members:
      - alice@example.com

  developer:
    permissions: ["view", "upload"]
    members:
      - bob@example.com

  viewer:
    permissions: ["view"]

default_role: viewer
```

- `default_role` is the fallback for any authenticated user not listed in a role
- Member emails are matched **case-insensitively**
- Use `"*"` to grant all permissions (superuser)

## Permissions

| Permission | Constant | What it grants |
|---|---|---|
| `view` | `PermView` | Read-only: list environments, projects, and reports |
| `upload` | `PermUpload` | Upload results, trigger report generation |
| `manage` | `PermManage` | Create, edit, delete environments, projects, reports, and upload sessions |

## Roles summary

| Role | Permissions | Can upload | Can manage |
|---|---|---|---|
| `admin` | `*` | Yes | Yes |
| `developer` | `view`, `upload` | Yes | No |
| `viewer` | `view` | No | No |

## Auth endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/auth/google` | Start Google OAuth flow |
| `GET` | `/auth/google/callback` | OAuth callback (handled by authkit) |
| `POST` | `/auth/logout` | Clear session, redirect to `AUTH_AFTER_LOGOUT_URL` |
| `GET` | `/auth/me` | Returns current user JSON or `401` |

### `/auth/me` response

```json
{
  "email": "alice@example.com",
  "name": "Alice",
  "avatarUrl": "https://lh3.googleusercontent.com/...",
  "provider": "google",
  "role": "admin"
}
```

## Frontend permission gating

The React frontend mirrors the RBAC logic via `AuthContext`:

```tsx
const { can } = useAuth()

// Show only for admin
{can('manage') && <button>Delete</button>}

// Show for admin and developer
{can('upload') && <UploadButton />}
```

Roles and their frontend permissions are kept in sync with `policy.yaml` via the `ROLE_PERMS` map in `AuthContext.tsx`.

## Session security

- Sessions are stored in an **encrypted cookie** (AES-GCM via gorilla/securecookie)
- Set `SECURE_COOKIE=true` in production — requires HTTPS
- Generate a strong secret: `openssl rand -hex 32`
- The secret must remain stable across restarts (changing it invalidates all sessions)
