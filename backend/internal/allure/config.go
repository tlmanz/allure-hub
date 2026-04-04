package allure

import (
	"os"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/tlmanz/allure-hub/internal/usecase"
)

// loadBaseConfig reads and parses an allurerc.yml file into a generic map.
// Called once at Generator construction time; the result is cached.
// Returns an empty map if the path is empty or the file cannot be read.
func loadBaseConfig(path string, log *zap.Logger) map[string]any {
	base := map[string]any{}
	if path == "" {
		log.Info("allure base config not set, using empty defaults")
		return base
	}
	data, err := os.ReadFile(path)
	if err != nil {
		log.Warn("allure base config could not be read, using empty defaults",
			zap.String("path", path),
			zap.Error(err),
		)
		return base
	}
	if err := yaml.Unmarshal(data, &base); err != nil {
		log.Warn("allure base config could not be parsed, using empty defaults",
			zap.String("path", path),
			zap.Error(err),
		)
		return base
	}
	log.Info("allure base config loaded",
		zap.String("path", path),
		zap.Int("keys", len(base)),
	)
	return base
}

// serverControlledKeys are allurerc.yml fields that affect the web server's
// file layout and must never be overridden by a caller's request.
var serverControlledKeys = map[string]struct{}{
	"output":      {}, // backend always sets this to the report's output dir
	"historyPath": {}, // backend manages history for trend charts
}

// buildConfig constructs a per-run allurerc.yml as a byte slice and returns
// the user-visible config snapshot (base merged with safe overrides, without
// server-controlled keys so the snapshot is meaningful to the user).
//
// base is the pre-parsed config map cached at startup (read-only - deepMerge
// never mutates it).  The caller's Overrides are merged on top so only the
// fields they supplied change; everything else keeps the base value.
// serverControlledKeys are stripped from Overrides before the merge so callers
// can never influence the server's file layout.
func buildConfig(base map[string]any, outputDir, historyPath string, opts usecase.GenerateOptions) ([]byte, map[string]any, error) {
	safe := make(map[string]any, len(opts.Overrides))
	for k, v := range opts.Overrides {
		if _, blocked := serverControlledKeys[k]; !blocked {
			safe[k] = v
		}
	}

	// snapshot is the user-visible effective config (no server-internal paths).
	snapshot := deepMerge(base, safe)

	// The YAML written to disk adds backend-controlled paths on top.
	forCLI := deepMerge(snapshot, map[string]any{
		"output":      outputDir,
		"historyPath": historyPath,
	})
	data, err := yaml.Marshal(forCLI)
	return data, snapshot, err
}

// deepMerge returns a new map that is base with override applied on top.
// For keys whose values are both maps the merge is applied recursively;
// for all other types the override value wins outright.
func deepMerge(base, override map[string]any) map[string]any {
	result := make(map[string]any, len(base))
	for k, v := range base {
		result[k] = v
	}
	for k, v := range override {
		if baseVal, exists := result[k]; exists {
			if baseMap, ok := asMap(baseVal); ok {
				if overMap, ok := asMap(v); ok {
					result[k] = deepMerge(baseMap, overMap)
					continue
				}
			}
		}
		result[k] = v
	}
	return result
}

func asMap(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}
