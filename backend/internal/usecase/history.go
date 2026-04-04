package usecase

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"
)

// removeHistoryEntry rewrites the history JSONL file, omitting any entry whose
// "timestamp" field (milliseconds since epoch) is within 30 seconds of createdAt.
// This matches the Allure-appended entry for the deleted build run.
func removeHistoryEntry(historyPath string, createdAt time.Time) error {
	data, err := os.ReadFile(historyPath)
	if err != nil {
		return nil // file doesn't exist - nothing to do
	}

	targetMs := createdAt.UnixMilli()
	const windowMs = 30_000 // ±30 s

	var kept [][]byte
	for _, line := range bytes.Split(data, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var entry struct {
			Timestamp int64 `json:"timestamp"`
		}
		if json.Unmarshal(line, &entry) == nil {
			diff := entry.Timestamp - targetMs
			if diff < 0 {
				diff = -diff
			}
			if diff <= windowMs {
				continue // drop this entry
			}
		}
		kept = append(kept, line)
	}

	out := bytes.Join(kept, []byte("\n"))
	if len(out) > 0 {
		out = append(out, '\n')
	}
	return os.WriteFile(historyPath, out, 0644)
}

// copyDir recursively copies src into dst, preserving directory structure.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyFile(path, target)
	})
}

// copyFile copies a single file from src to dst, syncing to disk before close
// so that kernel-buffer write errors are not silently swallowed (M-10).
func copyFile(src, dst string) (retErr error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	// Close explicitly so we can capture its error; the named return lets a
	// close error propagate even when the copy itself succeeded (M-10).
	defer func() {
		if cerr := out.Close(); cerr != nil && retErr == nil {
			retErr = cerr
		}
	}()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
