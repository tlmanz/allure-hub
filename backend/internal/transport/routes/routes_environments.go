package routes

import (
	"net/http"

	"github.com/tlmanz/allure-hub/internal/transport/handler"
	localauth "github.com/tlmanz/allure-hub/pkg/authkit"
	kit "github.com/tlmanz/authkit"
)

func RegisterEnvironmentRoutes(mux *http.ServeMux, auth *kit.Auth, eh *handler.EnvironmentHandler, ph *handler.ProjectHandler) {
	// Environments - GET allows API keys, mutations are session-only
	mux.Handle("GET /api/environments", auth.Require(localauth.PermView)(http.HandlerFunc(eh.List)))
	mux.Handle("POST /api/environments", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(eh.Create)))
	mux.Handle("PATCH /api/environments/{envId}", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(eh.Update)))
	mux.Handle("DELETE /api/environments/{envId}", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(eh.Delete)))

	// Projects (nested under environments)
	mux.Handle("GET /api/environments/{envId}/projects", auth.Require(localauth.PermView)(http.HandlerFunc(ph.List)))
	mux.Handle("POST /api/environments/{envId}/projects", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(ph.Create)))
	mux.Handle("DELETE /api/environments/{envId}/projects/{projectId}", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(ph.Delete)))
}
