#!/usr/bin/env bash
#
# Copies the original Medusa frontend packages (admin, design-system, js-sdk)
# from an upstream checkout into this repository, 1:1.
#
# Usage:
#   scripts/sync-upstream.sh [--include-enterprise] [UPSTREAM_DIR]
#
# UPSTREAM_DIR defaults to reference/medusa and must be checked out at the tag
# recorded in UPSTREAM_VERSION.
#
# Paths listed in the upstream ENTERPRISE-LICENSE.md are Medusa Enterprise
# Edition materials and are NOT copied unless --include-enterprise is passed.
# Only pass it if you hold a commercial agreement with MedusaJS, Inc.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INCLUDE_ENTERPRISE=0
UPSTREAM=""

for arg in "$@"; do
  case "$arg" in
    --include-enterprise) INCLUDE_ENTERPRISE=1 ;;
    -h|--help) sed -n '2,15p' "$0"; exit 0 ;;
    *) UPSTREAM="$arg" ;;
  esac
done

UPSTREAM="${UPSTREAM:-$ROOT/reference/medusa}"
UPSTREAM="$(cd "$UPSTREAM" && pwd)"
EXPECTED_VERSION="$(tr -d '[:space:]' < "$ROOT/UPSTREAM_VERSION")"

actual_version="$(git -C "$UPSTREAM" describe --tags --exact-match 2>/dev/null || true)"
if [[ "$actual_version" != "$EXPECTED_VERSION" ]]; then
  echo "error: $UPSTREAM is at '${actual_version:-unknown}', expected $EXPECTED_VERSION" >&2
  exit 1
fi

# Frontend packages kept as original JavaScript.
FRONTEND_PATHS=(
  packages/admin/admin-bundler
  packages/admin/admin-sdk
  packages/admin/admin-shared
  packages/admin/admin-vite-plugin
  packages/admin/dashboard
  packages/design-system/icons
  packages/design-system/toolbox
  packages/design-system/ui
  packages/design-system/ui-preset
  packages/core/js-sdk
)

EXCLUDES=(--exclude node_modules --exclude dist --exclude .turbo --exclude '*.tsbuildinfo')

if [[ "$INCLUDE_ENTERPRISE" -eq 0 ]]; then
  # "- `path`" bullet lines of ENTERPRISE-LICENSE.md.
  while IFS= read -r ee_path; do
    for fp in "${FRONTEND_PATHS[@]}"; do
      if [[ "$ee_path" == "$fp/"* ]]; then
        rel="${ee_path#"$fp"}"
        EXCLUDES+=(--exclude "${rel%/}")
      fi
    done
  done < <(sed -n 's/^- `\(.*\)`$/\1/p' "$UPSTREAM/ENTERPRISE-LICENSE.md")
fi

for fp in "${FRONTEND_PATHS[@]}"; do
  mkdir -p "$ROOT/$fp"
  rsync -a --delete "${EXCLUDES[@]}" "$UPSTREAM/$fp/" "$ROOT/$fp/"
done

# Optional local scrub patterns (one extended regex per line, not committed):
# lines matching them are removed from copied Markdown files, and the sync
# fails if any copied file still matches afterwards.
SCRUB_FILE="$ROOT/.git/info/sync-scrub"
if [[ -s "$SCRUB_FILE" ]]; then
  while IFS= read -r -d '' md; do
    if grep -qiEf "$SCRUB_FILE" "$md"; then
      tmp="$(mktemp)"
      grep -viEf "$SCRUB_FILE" "$md" > "$tmp" || true
      mv "$tmp" "$md"
    fi
  done < <(find "${FRONTEND_PATHS[@]/#/$ROOT/}" -name '*.md' -print0)

  leftovers="$(grep -rliEf "$SCRUB_FILE" "${FRONTEND_PATHS[@]/#/$ROOT/}" || true)"
  if [[ -n "$leftovers" ]]; then
    echo "error: copied files still match scrub patterns:" >&2
    echo "$leftovers" >&2
    exit 1
  fi
fi

echo "synced ${#FRONTEND_PATHS[@]} frontend packages from $EXPECTED_VERSION (enterprise: $([[ $INCLUDE_ENTERPRISE -eq 1 ]] && echo included || echo excluded))"
