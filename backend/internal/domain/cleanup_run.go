package domain

import "time"

// CleanupRun records the outcome of a single cleanup sweep.
type CleanupRun struct {
	ID           string    `json:"id"`
	StartedAt    time.Time `json:"startedAt"`
	FinishedAt   time.Time `json:"finishedAt"`
	Status       string    `json:"status"` // "success" | "failed"
	DeletedCount int       `json:"deletedCount"`
	SkippedCount int       `json:"skippedCount"`
	DryRun       bool      `json:"dryRun"`
	ErrorMessage string    `json:"errorMessage,omitempty"`
}
