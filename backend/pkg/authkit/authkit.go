// Package authkit defines allure-hub's RBAC roles and permission constants.
// These strings must match what is declared in policy.yaml exactly.
//
// Roles (defined in policy.yaml):
//
//	admin     — full access (PermAll)
//	developer — PermView + PermUpload
//	viewer    — PermView only (default role for all authenticated users)
package authkit

const (
	// PermView grants read-only access: list environments, projects, and reports.
	PermView = "view"

	// PermUpload grants the ability to upload test results and trigger report generation.
	PermUpload = "upload"

	// PermManage grants full create / edit / delete access over environments,
	// projects, reports, and upload sessions.
	PermManage = "manage"
)

