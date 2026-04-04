[![CI](https://github.com/tlmanz/allure-hub/actions/workflows/ci.yml/badge.svg)](https://github.com/tlmanz/allure-hub/actions/workflows/ci.yml)
[![CodeQL](https://github.com/tlmanz/allure-hub/actions/workflows/codequality.yml/badge.svg)](https://github.com/tlmanz/allure-hub/actions/workflows/codequality.yml)
[![Coverage Status](https://coveralls.io/repos/github/tlmanz/allure-hub/badge.svg)](https://coveralls.io/github/tlmanz/allure-hub)
![Open Issues](https://img.shields.io/github/issues/tlmanz/allure-hub)
[![Go Report Card](https://goreportcard.com/badge/github.com/tlmanz/allure-hub)](https://goreportcard.com/report/github.com/tlmanz/allure-hub)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/tlmanz/allure-hub)

# allure-hub

Self-hosted Allure 3 reporting platform - upload test results via API or UI, generate and browse reports across environments and projects, with Google OAuth and role-based access control.

![Allure Hub UI](docs/image.png)

## Quick start

```bash
# Backend (serves on :8080)
make dev-backend

# Frontend dev server (proxies /api, /auth, /reports to :8080)
make dev-frontend
```

See the [documentation](https://tlmanz.github.io/allure-hub) for full setup, configuration, and deployment guides.

## Stack

| Layer | Tech |
|---|---|
| Backend | Go 1.22+, `net/http` |
| Frontend | React 18, TypeScript, Vite 5 |
| Reports | Allure 3 CLI |
| Database | SQLite (default) or PostgreSQL |
| Auth | Google OAuth + YAML RBAC |
| Container | Single Docker image (Node 22 Alpine + Go) |

## License

MIT
