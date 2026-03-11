#!/usr/bin/env bash
set -euo pipefail

repo_dir="${1:-.}"
workflows_dir="$repo_dir/.github/workflows"
env_file="$repo_dir/release-naming.env"

[[ -d "$workflows_dir" ]] || { echo "missing workflows dir: $workflows_dir" >&2; exit 1; }
[[ -f "$env_file" ]] || { echo "missing release naming contract: $env_file" >&2; exit 1; }

# shellcheck disable=SC1090
source "$env_file"

for key in CLI_NAME BINARY_NAME TAG_PREFIX ARTIFACT_GLOB BUILD_TARGET; do
  [[ -n "${!key:-}" ]] || { echo "release-naming.env missing value: $key" >&2; exit 1; }
done

if rg -n "your-cli|your-cli-v" "$workflows_dir" >/dev/null; then
  echo "found unreplaced template placeholders in workflows" >&2
  exit 1
fi

rg -n --fixed-strings "${TAG_PREFIX}*" "$workflows_dir/release-on-tag.yml" >/dev/null || {
  echo "release-on-tag.yml trigger does not match TAG_PREFIX pattern: ${TAG_PREFIX}*" >&2
  exit 1
}

for f in "$workflows_dir/release-command.yml" "$workflows_dir/release-on-tag.yml" "$repo_dir/scripts/next-version.sh" "$repo_dir/scripts/print-release-download.sh"; do
  [[ -f "$f" ]] || { echo "missing required file: $f" >&2; exit 1; }
done

echo "release naming audit passed"
