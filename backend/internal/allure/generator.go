// Package allure implements the usecase.Generator port using the Allure 3 CLI.
package allure

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/tlmanz/allure-hub/internal/usecase"
)

// Generator wraps the Allure 3 CLI (@allurereport/cli).
type Generator struct {
	bin        string
	baseConfig map[string]any // parsed allurerc.yml, cached at construction; never mutated
	sem        chan struct{}  // bounds concurrent allure subprocess invocations
	timeout    time.Duration  // per-invocation deadline; 0 means no timeout
	log        *zap.Logger
}

// NewGenerator parses the base config file once and returns a Generator ready
// for concurrent use.  maxConcurrency caps how many allure processes run at the
// same time; values ≤ 0 default to 1. timeout is the per-invocation deadline;
// 0 disables it.
func NewGenerator(bin, configPath string, maxConcurrency int, timeout time.Duration, log *zap.Logger) *Generator {
	if maxConcurrency <= 0 {
		maxConcurrency = 1
	}
	return &Generator{
		bin:        bin,
		baseConfig: loadBaseConfig(configPath, log),
		sem:        make(chan struct{}, maxConcurrency),
		timeout:    timeout,
		log:        log,
	}
}

// Generate runs: allure generate <resultsDir> --output <outputDir> --config <tempConfig>
// A temporary allurerc.yml is written per invocation by merging the base config
// file (if any) with opts, so each report can have its own name / quality gates.
// History must already be injected into resultsDir/history/ before calling.
// Returns the user-visible config snapshot (effective config without server-internal keys).
func (g *Generator) Generate(resultsDir, outputDir, historyPath string, opts usecase.GenerateOptions) (usecase.GenerateResult, error) {
	// Validate paths inside Generate itself so this method is safe regardless
	// of which call path reaches it (C-03: defence-in-depth).
	if err := validateGeneratorPath(resultsDir); err != nil {
		return usecase.GenerateResult{}, fmt.Errorf("invalid resultsDir: %w", err)
	}
	if err := validateGeneratorPath(outputDir); err != nil {
		return usecase.GenerateResult{}, fmt.Errorf("invalid outputDir: %w", err)
	}

	// Allure 3 does not have --clean; remove the output dir manually.
	if err := os.RemoveAll(outputDir); err != nil {
		return usecase.GenerateResult{}, fmt.Errorf("clear output dir: %w", err)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return usecase.GenerateResult{}, fmt.Errorf("create output dir: %w", err)
	}

	// Write a per-run config file that merges the cached base with request opts.
	cfgData, snapshot, err := buildConfig(g.baseConfig, outputDir, historyPath, opts)
	if err != nil {
		return usecase.GenerateResult{}, fmt.Errorf("build allure config: %w", err)
	}
	tmpCfg, err := os.CreateTemp("", "allurerc-*.yml")
	if err != nil {
		return usecase.GenerateResult{}, fmt.Errorf("create temp config: %w", err)
	}
	defer os.Remove(tmpCfg.Name())
	if _, err := tmpCfg.Write(cfgData); err != nil {
		tmpCfg.Close()
		return usecase.GenerateResult{}, fmt.Errorf("write temp config: %w", err)
	}
	tmpCfg.Close()

	// Acquire semaphore slot — blocks if maxConcurrency allure processes are
	// already running, providing back-pressure under high load.
	g.sem <- struct{}{}
	defer func() { <-g.sem }()

	// Build a context with a per-invocation deadline (H-01: prevents a hung
	// allure process from blocking the semaphore indefinitely).
	ctx := context.Background()
	if g.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, g.timeout)
		defer cancel()
	}

	// allure generate <resultsDir> --output <outputDir> --config <tempConfig>
	args := []string{"generate", resultsDir, "--output", outputDir, "--config", tmpCfg.Name()}
	cmd := exec.CommandContext(ctx, g.bin, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	// Forward each non-empty line from the CLI through zap so they appear
	// as structured JSON log entries rather than raw text in the log stream.
	for _, line := range splitLines(stdout.String()) {
		g.log.Debug("allure", zap.String("output", line))
	}
	for _, line := range splitLines(stderr.String()) {
		g.log.Warn("allure stderr", zap.String("output", line))
	}
	warnings := collectKnownAllureParseWarnings(stderr.String())

	if err != nil {
		if shouldIgnoreAllureParseErrors(stderr.String()) {
			g.log.Warn("allure generate returned non-zero exit due to known parse errors; continuing",
				zap.Error(err),
			)
			return usecase.GenerateResult{ConfigSnapshot: snapshot, Warnings: warnings}, nil
		}
		return usecase.GenerateResult{}, fmt.Errorf("allure generate: %w", err)
	}
	return usecase.GenerateResult{ConfigSnapshot: snapshot, Warnings: warnings}, nil
}

// HistoryDir returns the history subdirectory of a generated report,
// satisfying the usecase.Generator port interface.
func (g *Generator) HistoryDir(reportDir string) string {
	return filepath.Join(reportDir, "history")
}

// validateGeneratorPath rejects paths that are not absolute or that contain
// null bytes or ".." components, providing defence-in-depth against path
// traversal regardless of which call path reaches Generate (C-03).
func validateGeneratorPath(p string) error {
	if !filepath.IsAbs(p) {
		return fmt.Errorf("path must be absolute, got %q", p)
	}
	if strings.Contains(p, "\x00") {
		return fmt.Errorf("path contains null byte")
	}
	cleaned := filepath.Clean(p)
	for _, part := range strings.Split(cleaned, string(filepath.Separator)) {
		if part == ".." {
			return fmt.Errorf("path contains traversal sequence")
		}
	}
	return nil
}

func splitLines(s string) []string {
	var lines []string
	for _, l := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(l); t != "" {
			lines = append(lines, t)
		}
	}
	return lines
}

// shouldIgnoreAllureParseErrors returns true when stderr contains only known
// non-fatal parse errors from Allure 3's testrun/history inputs that we treat
// as warnings.
func shouldIgnoreAllureParseErrors(stderr string) bool {
	lines := collectKnownAllureParseWarnings(stderr)
	if len(lines) == 0 {
		return false
	}
	// Ignore only when all non-empty stderr lines are known warnings.
	return len(lines) == len(splitLines(stderr))
}

func isKnownAllureParseWarning(line string) bool {
	l := strings.ToLower(strings.TrimSpace(line))
	if !strings.Contains(l, "error parsing ") {
		return false
	}
	if !strings.Contains(l, "typeerror: parsed is not iterable") {
		return false
	}
	return strings.Contains(l, "testrun.json") || strings.Contains(l, "history.json")
}

func collectKnownAllureParseWarnings(stderr string) []string {
	lines := splitLines(stderr)
	if len(lines) == 0 {
		return nil
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if isKnownAllureParseWarning(line) {
			out = append(out, line)
		}
	}
	return out
}
