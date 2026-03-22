// Package storage implements the usecase.FileStorage port.
package storage

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/tlmanz/allure-hub/internal/usecase"
)

// ErrUploadTooLarge is returned when a compressed upload exceeds the byte limit.
var ErrUploadTooLarge = errors.New("upload exceeds maximum allowed size")

// ErrDecompressedTooLarge is returned when extracted content exceeds the decompressed size limit (M-03).
var ErrDecompressedTooLarge = errors.New("decompressed content exceeds maximum allowed size")

// ErrTooManyZipEntries is returned when a zip contains more entries than the configured cap (M-04).
var ErrTooManyZipEntries = errors.New("zip contains too many entries")

// PVC layout:
//   {dataDir}/
//     {projectID}/
//       metadata.json
//       history/          ← persisted trend data
//       results/{buildID}/
//       reports/{buildID}/

type Filesystem struct {
	dataDir          string
	maxBytes         int64 // max compressed upload size; 0 = no limit
	maxDecompressed  int64 // max total decompressed bytes across all zip entries; 0 = no limit
	maxZipEntries    int   // max number of entries in a zip; 0 = no limit
}

func NewFilesystem(dataDir string, maxBytes, maxDecompressed int64, maxZipEntries int) *Filesystem {
	abs, err := filepath.Abs(dataDir)
	if err == nil {
		dataDir = abs
	}
	return &Filesystem{
		dataDir:         dataDir,
		maxBytes:        maxBytes,
		maxDecompressed: maxDecompressed,
		maxZipEntries:   maxZipEntries,
	}
}

func (f *Filesystem) ResultsDir(projectID, buildID string) string {
	return filepath.Join(f.dataDir, projectID, "results", buildID)
}

func (f *Filesystem) ReportDir(projectID, buildID string) string {
	return filepath.Join(f.dataDir, projectID, "reports", buildID)
}

func (f *Filesystem) HistoryDir(projectID string) string {
	return filepath.Join(f.dataDir, projectID, "history")
}

func (f *Filesystem) HistoryFile(projectID string) string {
	return filepath.Join(f.dataDir, projectID, "history.jsonl")
}

// SaveResultsStream reads a zip from r and extracts it directly to the
// results directory. r is consumed with io.Copy in 32 KB chunks — the
// full zip is never in memory at once.
func (f *Filesystem) SaveResultsStream(projectID, buildID string, r io.Reader) error {
	dest := f.ResultsDir(projectID, buildID)
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	// We need a seekable reader for zip.NewReader, so we stream to a
	// temp file first (still O(1) memory), then extract.
	// Use dataDir for temp files so we stay on the same filesystem (avoids
	// cross-device issues and tiny /tmp in containers).
	tmp, err := os.CreateTemp(f.dataDir, "allure-upload-*.zip")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	// Limit total bytes written to disk at the storage layer (H-03: defence-in-depth
	// against disk exhaustion regardless of which call path reaches this method).
	// We read up to maxBytes+1 so we can detect when the limit is exceeded.
	var src io.Reader = r
	var limitN int64
	if f.maxBytes > 0 {
		limitN = f.maxBytes + 1
		src = &io.LimitedReader{R: r, N: limitN}
	}
	written, err := io.Copy(tmp, src) // 32 KB copy loop — no full file in RAM
	if err != nil {
		return fmt.Errorf("stream to disk: %w", err)
	}
	if f.maxBytes > 0 && written > f.maxBytes {
		return ErrUploadTooLarge
	}

	tmp.Seek(0, 0)
	return unzip(tmp, written, dest, f.maxDecompressed, f.maxZipEntries)
}

// unzip extracts a zip from rs (size bytes) into dest.
// maxDecompressed caps the total bytes written across all entries (M-03: zip bomb).
// maxEntries caps the number of entries in the archive (M-04: inode exhaustion).
// 0 for either limit means no cap.
func unzip(rs io.ReadSeeker, size int64, dest string, maxDecompressed int64, maxEntries int) error {
	zr, err := zip.NewReader(rs.(interface {
		io.ReaderAt
		io.ReadSeeker
	}), size)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}

	// M-04: reject archives with too many entries before touching the filesystem.
	if maxEntries > 0 && len(zr.File) > maxEntries {
		return ErrTooManyZipEntries
	}

	var totalDecompressed int64

	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		// Only extract flat files — ignore any directory nesting in the zip.
		name := filepath.Base(f.Name)
		// Reject path traversal attempts and hidden files.
		if name == "." || name == ".." || name == "" || name[0] == '.' {
			continue
		}
		target := filepath.Join(dest, name)

		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(target)
		if err != nil {
			rc.Close()
			return err
		}

		// M-03: limit decompressed bytes per entry and in total.
		var src io.Reader = rc
		if maxDecompressed > 0 {
			remaining := maxDecompressed - totalDecompressed
			src = &io.LimitedReader{R: rc, N: remaining + 1}
		}
		n, err := io.Copy(out, src)
		out.Close()
		rc.Close()
		if err != nil {
			return err
		}
		totalDecompressed += n
		if maxDecompressed > 0 && totalDecompressed > maxDecompressed {
			return ErrDecompressedTooLarge
		}
	}
	return nil
}

// InitProject creates the project directory tree on the filesystem.
func (f *Filesystem) InitProject(id string) error {
	return os.MkdirAll(filepath.Join(f.dataDir, id), 0755)
}

// RemoveProject deletes the entire project directory tree from the filesystem.
func (f *Filesystem) RemoveProject(id string) error {
	return os.RemoveAll(filepath.Join(f.dataDir, id))
}

func (f *Filesystem) ChunkDir(projectID, uploadID string) string {
	return filepath.Join(f.dataDir, projectID, "uploads", uploadID)
}

func (f *Filesystem) ChunkPath(projectID, uploadID string, index int) string {
	return filepath.Join(f.ChunkDir(projectID, uploadID), strconv.Itoa(index))
}

func (f *Filesystem) WriteUploadMeta(projectID, uploadID string, meta usecase.UploadMeta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(f.ChunkDir(projectID, uploadID), "meta.json"), data, 0644)
}

func (f *Filesystem) ReadUploadMeta(projectID, uploadID string) (usecase.UploadMeta, error) {
	data, err := os.ReadFile(filepath.Join(f.ChunkDir(projectID, uploadID), "meta.json"))
	if err != nil {
		return usecase.UploadMeta{}, err
	}
	var meta usecase.UploadMeta
	return meta, json.Unmarshal(data, &meta)
}
