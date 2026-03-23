#!/usr/bin/env bash
# Upload allure-results to allure-hub and trigger report generation.
#
# Usage:
#   ./upload-results.sh
#
# Environment variables (all optional):
#   ALLURE_HUB_URL   Base URL of the allure-hub API  (default: http://localhost:8080)
#   ENV_ID           Environment ID                   (default: default)
#   ENV_NAME         Human-readable environment name  (default: Default)
#   PROJECT_ID       Project ID to upload results to  (default: sample-java)
#   PROJECT_NAME     Human-readable project name      (default: Sample Java App)
#   BUILD_ID         Unique build identifier          (default: YYYYmmdd-HHMMSS)
#   RESULTS_DIR      Path to allure-results directory (default: target/allure-results)

set -euo pipefail

ALLURE_HUB_URL="${ALLURE_HUB_URL:-http://localhost:8080}"
ENV_ID="${ENV_ID:-default}"
ENV_NAME="${ENV_NAME:-Default}"
PROJECT_ID="${PROJECT_ID:-sample-java}"
PROJECT_NAME="${PROJECT_NAME:-Sample Java App}"
BUILD_ID="${BUILD_ID:-$(date +%Y%m%d-%H%M%S)}"
RESULTS_DIR="${RESULTS_DIR:-target/allure-results}"
BEARER_TOKEN="${BEARER_TOKEN:-}"

AUTH_HEADER=()
if [[ -n "$BEARER_TOKEN" ]]; then
  AUTH_HEADER=(-H "Authorization: Bearer $BEARER_TOKEN")
fi

# ── Validate ──────────────────────────────────────────────────────────────────

if [[ ! -d "$RESULTS_DIR" ]]; then
  echo "Error: results directory not found: $RESULTS_DIR" >&2
  echo "Run 'mvn test' first to generate allure-results." >&2
  exit 1
fi

if [[ -z "$(ls -A "$RESULTS_DIR")" ]]; then
  echo "Error: results directory is empty: $RESULTS_DIR" >&2
  exit 1
fi

echo "allure-hub  : $ALLURE_HUB_URL"
echo "environment : $ENV_ID  ($ENV_NAME)"
echo "project     : $PROJECT_ID  ($PROJECT_NAME)"
echo "build       : $BUILD_ID"
echo "results     : $RESULTS_DIR"
echo ""

# ── Zip results ───────────────────────────────────────────────────────────────

tmp_zip="$(mktemp -u /tmp/allure-results-XXXXXX.zip)"
trap 'rm -f "$tmp_zip"' EXIT

echo "→ Zipping $(find "$RESULTS_DIR" -type f | wc -l | tr -d ' ') result files..."
# -j junk paths so all files land flat in the zip root; -r recurse into subdirs (e.g. attachments/)
zip -jqr "$tmp_zip" "$RESULTS_DIR"

zip_size="$(du -sh "$tmp_zip" | cut -f1)"
echo "   $zip_size compressed"

# ── Stream upload ─────────────────────────────────────────────────────────────

echo "→ Uploading..."
curl -s -X POST \
  "$ALLURE_HUB_URL/api/environments/$ENV_ID/projects/$PROJECT_ID/results?buildId=$BUILD_ID" \
  "${AUTH_HEADER[@]}" \
  -H "Content-Type: application/zip" \
  --data-binary "@$tmp_zip" \
  -w "\n   HTTP %{http_code}\n"


echo ""
echo "Done."
