// Package service contains the application use-cases for allure-hub.
// Ports (interfaces) defined here are implemented in the infrastructure layer.
package usecase

import "io"

// FileStorage is the port for all filesystem operations.
// The infrastructure adapter lives in internal/storage.
type FileStorage interface {
	InitProject(envID, id string) error
	RemoveProject(envID, id string) error
	SaveResultsStream(envID, projectID, buildID string, r io.Reader) error
	ResultsDir(envID, projectID, buildID string) string
	ReportDir(envID, projectID, buildID string) string
	HistoryDir(envID, projectID string) string
	HistoryFile(envID, projectID string) string // path to the Allure 3 history JSONL file
	ChunkDir(envID, projectID, uploadID string) string
	ChunkPath(envID, projectID, uploadID string, index int) string
	WriteUploadMeta(envID, projectID, uploadID string, meta UploadMeta) error
	ReadUploadMeta(envID, projectID, uploadID string) (UploadMeta, error)
}

// GenerateOptions carries arbitrary allurerc.yml overrides for a single report.
// Overrides is a free-form map whose keys match allurerc.yml field names.
// Fields absent from Overrides keep their value from the base config file.
// The "output" key is always ignored — the backend controls the output path.
type GenerateOptions struct {
	Overrides map[string]any
}

// GenerateResult is the output from one Allure generation run.
type GenerateResult struct {
	// ConfigSnapshot is the effective allurerc.yml config (merged base + overrides)
	// with server-controlled keys removed.
	ConfigSnapshot map[string]any
	// Warnings are non-fatal generation warnings surfaced by the generator adapter.
	Warnings []string
}

// Generator is the port for the Allure CLI report generator.
// The infrastructure adapter lives in internal/allure.
// Generate returns the effective config snapshot (merged base + user overrides,
// without server-controlled keys) so callers can persist it with the build record.
type Generator interface {
	Generate(resultsDir, outputDir, historyPath string, opts GenerateOptions) (GenerateResult, error)
	HistoryDir(reportDir string) string
}

// UploadMeta holds the state of an in-progress chunked upload session.
type UploadMeta struct {
	UploadID    string `json:"uploadId"`
	BuildID     string `json:"buildId"`
	TotalSize   int64  `json:"totalSize"`
	TotalChunks int    `json:"totalChunks"`
}
