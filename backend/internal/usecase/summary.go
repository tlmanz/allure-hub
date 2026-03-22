package usecase

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// parseSummary walks the report output directory and returns stats from the
// first summary.json found at any depth.
// Returns all-zero values on any error so report generation is never blocked.
func parseSummary(reportDir string) (passed, failed, skipped, total int, status string) {
	var data []byte
	filepath.WalkDir(reportDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "summary.json" {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		data = b
		return filepath.SkipAll
	})
	if data == nil {
		return
	}
	var doc struct {
		Stats struct {
			Total   int `json:"total"`
			Passed  int `json:"passed"`
			Failed  int `json:"failed"`
			Skipped int `json:"skipped"`
			Broken  int `json:"broken"`
		} `json:"stats"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return
	}
	return doc.Stats.Passed,
		doc.Stats.Failed + doc.Stats.Broken,
		doc.Stats.Skipped,
		doc.Stats.Total,
		doc.Status
}
