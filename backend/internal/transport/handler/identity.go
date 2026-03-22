package handler

import (
	"net/http"

	kit "github.com/tlmanz/authkit"
)

// callerIdentity returns the identity string of the authenticated caller:
//   - OAuth session:  the user's email  (e.g. "alice@example.com")
//   - API key:        "apikey:<name>"   (e.g. "apikey:ci-pipeline")
//   - Unauthenticated: ""
//
// Both OAuth and API key users are now injected into the same context key by
// authkit, so a single UserFromCtx call covers both paths.
func callerIdentity(r *http.Request) string {
	if u := kit.UserFromCtx(r.Context()); u != nil {
		return u.Email
	}
	return ""
}
