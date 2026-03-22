package handler

import (
	"net/http"
	"regexp"
)

// safePath matches only safe path segment characters: alphanumeric, hyphens, underscores, dots.
// It explicitly rejects "..", empty strings, and any string containing path separators.
var safePath = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// validatePathParam checks that a user-supplied path segment is safe for filesystem use.
// Returns true if valid. Writes a 400 response and returns false if invalid.
func validatePathParam(w http.ResponseWriter, name, value string) bool {
	if value == "" || !safePath.MatchString(value) {
		http.Error(w, name+" contains invalid characters", http.StatusBadRequest)
		return false
	}
	return true
}
