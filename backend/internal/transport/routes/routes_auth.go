package routes

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/tlmanz/allure-hub/internal/domain"
	kit "github.com/tlmanz/authkit"
)

func providersHandler(names []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if names == nil {
			names = []string{}
		}
		json.NewEncoder(w).Encode(names)
	}
}

func RegisterAuthRoutes(mux *http.ServeMux, auth *kit.Auth, userRepo domain.TrackedUserRepository, providerNames []string) {
	mux.HandleFunc("GET /auth/providers", providersHandler(providerNames))
	mux.HandleFunc("GET /auth/{provider}", auth.BeginAuth)
	mux.HandleFunc("GET /auth/{provider}/callback", auth.Callback)
	mux.HandleFunc("POST /auth/logout", auth.Logout)
	// /auth/me: session-only, track the user on access
	mux.Handle("GET /auth/me", auth.RequireSessionAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := kit.UserFromCtx(r.Context())
		go upsertTrackedUser(context.WithoutCancel(r.Context()), userRepo, u)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(u)
	})))
}
