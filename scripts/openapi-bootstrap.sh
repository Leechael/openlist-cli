#!/usr/bin/env bash
set -euo pipefail

spec="${1:-}"
out_dir="${2:-docs/openapi}"

if [[ -z "$spec" ]]; then
  echo "Usage: scripts/openapi-bootstrap.sh <openapi.json> [output-dir]" >&2
  exit 1
fi

if [[ ! -f "$spec" ]]; then
  echo "OpenAPI file not found: $spec" >&2
  exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required: https://jqlang.org/" >&2
  exit 1
fi

mkdir -p "$out_dir"

summary_md="$out_dir/openapi-summary.md"
ops_tsv="$out_dir/openapi-operations.tsv"
plan_md="$out_dir/openapi-command-plan.md"
test_md="$out_dir/openapi-test-matrix.md"

title=$(jq -r '.info.title // "unknown"' "$spec")
version=$(jq -r '.info.version // "unknown"' "$spec")

jq -r '
  def fallback_id($method; $path):
    (($method + "_" + $path)
      | ascii_downcase
      | gsub("[^a-z0-9]+"; "_")
      | gsub("^_+|_+$"; ""));

  (.paths // {})
  | to_entries[] as $p
  | ($p.value | to_entries[] | select(.key | test("^(get|post|put|patch|delete|options|head)$"))) as $op
  | [
      ($op.value.tags[0] // "default"),
      ($op.value.operationId // fallback_id($op.key; $p.key)),
      ($op.key | ascii_upcase),
      $p.key,
      ($op.value.summary // $op.value.description // ""),
      (if ($op.value.requestBody? != null) then "yes" else "no" end)
    ]
  | @tsv
' "$spec" | LC_ALL=C sort > "$ops_tsv"

op_count=$(wc -l < "$ops_tsv" | tr -d ' ')

{
  echo "# OpenAPI Summary"
  echo
  echo "- Spec file: \
\`$spec\`"
  echo "- API title: \
\`$title\`"
  echo "- API version: \
\`$version\`"
  echo "- Total operations: \
\`$op_count\`"
} > "$summary_md"

{
  echo "# OpenAPI Command Plan"
  echo
  echo "| Tag | Operation ID | Endpoint | Suggested CLI Command | Request Body |"
  echo "| --- | --- | --- | --- | --- |"
  awk -F'\t' '{ printf "| `%s` | `%s` | `%s %s` | `openlist-cli call %s` | %s |\n", $1, $2, $3, $4, $2, $6 }' "$ops_tsv"
} > "$plan_md"

{
  echo "# OpenAPI Test Matrix"
  echo
  echo "| Operation ID | Endpoint | Output Contract |"
  echo "| --- | --- | --- |"
  awk -F'\t' '{ printf "| `%s` | `%s %s` | `--json` parseable, `--plain` stable, `--jq` JSON-only |\n", $2, $3, $4 }' "$ops_tsv"
} > "$test_md"

echo "Generated OpenAPI bootstrap artifacts in $out_dir"
