package routes

import (
	"net/http"

	"github.com/tlmanz/allure-hub/internal/transport/handler"
	localauth "github.com/tlmanz/allure-hub/pkg/authkit"
	kit "github.com/tlmanz/authkit"
)

func RegisterSystemRoutes(mux *http.ServeMux, auth *kit.Auth, oh *handler.OverviewHandler, hh *handler.HealthHandler) {
	// Overview analytics dashboard
	mux.Handle("GET /api/overview", auth.Require(localauth.PermView)(http.HandlerFunc(oh.GetStats)))

	// Health + version (unprotected)
	mux.HandleFunc("GET /api/healthz", hh.Check)
	mux.HandleFunc("GET /api/version", hh.Info)
}
