package middleware

import "net/http"

var stateMethods = map[string]bool{
	http.MethodPost:   true,
	http.MethodPut:    true,
	http.MethodPatch:  true,
	http.MethodDelete: true,
}

// CSRF rejects browser-initiated state-changing requests that lack the
// X-Requested-With header. Only requests that carry an Origin header are
// checked — direct API calls (curl, CI pipelines) have no Origin and cannot
// be CSRF attacks, so they are passed through unconditionally.
func CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if stateMethods[r.Method] &&
			r.Header.Get("Origin") != "" &&
			r.Header.Get("X-Requested-With") == "" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
