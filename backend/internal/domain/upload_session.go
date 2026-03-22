package domain

import "time"

// UploadPhase represents the lifecycle stage of an upload session.
type UploadPhase string

const (
	PhaseUploading  UploadPhase = "uploading"  // chunks are being received
	PhaseAssembling UploadPhase = "assembling" // server is stitching chunks
	PhaseGenerating UploadPhase = "generating" // allure generate is running
	PhaseDone       UploadPhase = "done"        // report is ready
	PhaseFailed     UploadPhase = "failed"      // an error occurred
)

// UploadSession tracks a single upload from start to finish regardless of
// whether it originated from the UI or a direct API call (e.g. curl / CI).
type UploadSession struct {
	ID             string      `json:"id"`
	UploadID       string      `json:"uploadId"`
	BuildID        string      `json:"buildId"`
	ProjectID      string      `json:"projectId"`
	EnvID          string      `json:"envId"`
	FileName       string      `json:"fileName"`
	TotalSize      int64       `json:"totalSize"`
	TotalChunks    int         `json:"totalChunks"`
	ReceivedChunks int         `json:"receivedChunks"`
	Phase          UploadPhase `json:"phase"`
	FailedAtPhase  UploadPhase `json:"failedAtPhase,omitempty"`
	Error          string      `json:"error,omitempty"`
	UploadedBy     string      `json:"uploadedBy"`
	StartedAt      time.Time   `json:"startedAt"`
	CompletedAt    *time.Time  `json:"completedAt,omitempty"`
	ReportURL      string      `json:"reportUrl,omitempty"`
	Passed         int         `json:"passed"`
	Failed         int         `json:"failed"`
	Skipped        int         `json:"skipped"`
	Total          int         `json:"total"`
}
