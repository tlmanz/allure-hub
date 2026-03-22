package usecase

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseSummary_ValidFile(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "data")
	_ = os.Mkdir(sub, 0o755)

	doc := map[string]any{
		"stats": map[string]any{
			"total": 20, "passed": 15, "failed": 3, "skipped": 1, "broken": 1,
		},
		"status": "failed",
	}
	b, _ := json.Marshal(doc)
	_ = os.WriteFile(filepath.Join(sub, "summary.json"), b, 0o644)

	passed, failed, skipped, total, status := parseSummary(dir)
	if passed != 15 {
		t.Errorf("passed = %d, want 15", passed)
	}
	if failed != 4 {
		t.Errorf("failed = %d, want 4 (failed+broken)", failed)
	}
	if skipped != 1 {
		t.Errorf("skipped = %d, want 1", skipped)
	}
	if total != 20 {
		t.Errorf("total = %d, want 20", total)
	}
	if status != "failed" {
		t.Errorf("status = %q, want failed", status)
	}
}

func TestParseSummary_MissingDir(t *testing.T) {
	passed, failed, skipped, total, status := parseSummary(t.TempDir())
	if passed != 0 || failed != 0 || skipped != 0 || total != 0 || status != "" {
		t.Errorf("expected all-zero values, got p=%d f=%d sk=%d tot=%d st=%q",
			passed, failed, skipped, total, status)
	}
}

func TestParseSummary_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "summary.json"), []byte("not-json{{{"), 0o644)

	passed, failed, _, _, _ := parseSummary(dir)
	if passed != 0 || failed != 0 {
		t.Errorf("expected zeros on bad JSON, got passed=%d failed=%d", passed, failed)
	}
}

func TestParseSummary_BrokenCountsAsFailed(t *testing.T) {
	dir := t.TempDir()
	doc := map[string]any{
		"stats": map[string]any{
			"total": 5, "passed": 3, "failed": 0, "skipped": 0, "broken": 2,
		},
		"status": "broken",
	}
	b, _ := json.Marshal(doc)
	_ = os.WriteFile(filepath.Join(dir, "summary.json"), b, 0o644)

	_, failed, _, _, _ := parseSummary(dir)
	if failed != 2 {
		t.Errorf("failed = %d, want 2 (broken only)", failed)
	}
}

func TestParseSummary_NestedFile(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c")
	_ = os.MkdirAll(nested, 0o755)

	doc := map[string]any{
		"stats":  map[string]any{"total": 3, "passed": 3, "failed": 0, "skipped": 0, "broken": 0},
		"status": "passed",
	}
	b, _ := json.Marshal(doc)
	_ = os.WriteFile(filepath.Join(nested, "summary.json"), b, 0o644)

	passed, _, _, total, status := parseSummary(dir)
	if passed != 3 || total != 3 || status != "passed" {
		t.Errorf("nested summary not found: passed=%d total=%d status=%q", passed, total, status)
	}
}
