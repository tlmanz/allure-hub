package routes

import (
	"net/http"

	"github.com/tlmanz/allure-hub/internal/transport/handler"
	localauth "github.com/tlmanz/allure-hub/pkg/authkit"
	kit "github.com/tlmanz/authkit"
)

// registerSettingsRoutes registers all /api/settings/* routes.
// All are session-only - API keys must not manage themselves.
func RegisterSettingsRoutes(mux *http.ServeMux, auth *kit.Auth, sh *handler.SettingsHandler) {
	mux.Handle("GET /api/settings/apikeys", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(sh.ListAPIKeys)))
	mux.Handle("POST /api/settings/apikeys", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(sh.CreateAPIKey)))
	mux.Handle("DELETE /api/settings/apikeys/{id}", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(sh.RevokeAPIKey)))
	mux.Handle("GET /api/settings/users", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(sh.ListUsers)))
	mux.Handle("PATCH /api/settings/users/{email}/role", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(sh.SetUserRole)))
	mux.Handle("DELETE /api/settings/users/{email}/role", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(sh.ResetUserRole)))
	mux.Handle("GET /api/settings/retention", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(sh.GetRetention)))
	mux.Handle("PUT /api/settings/retention", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(sh.SetRetention)))
	mux.Handle("GET /api/settings/retention/runs", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(sh.GetCleanupRuns)))
	mux.Handle("GET /api/settings/allure", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(sh.GetAllureVersion)))
	mux.Handle("PUT /api/settings/allure", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(sh.UpdateAllureVersion)))
	mux.Handle("GET /api/settings/disk", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(sh.GetDiskUsage)))
	mux.Handle("GET /api/settings/disk/notification-threshold", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(sh.GetDiskNotificationThreshold)))
	mux.Handle("PUT /api/settings/disk/notification-threshold", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(sh.SetDiskNotificationThreshold)))
	mux.Handle("GET /api/settings/publishing", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(sh.GetPublishingSettings)))
	mux.Handle("PUT /api/settings/publishing", auth.RequireSession(localauth.PermManage)(http.HandlerFunc(sh.SetPublishingSettings)))
}
