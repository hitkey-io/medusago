#!/usr/bin/env bash
#
# Builds the default admin dashboard (packages/admin/dashboard, the original
# React app) with the original @medusajs/admin-bundler and stores it in
# packages/admin/dashboard-bundle/app, which the Go server embeds and serves
# at admin.path (default /app). Users therefore do not need Node for the
# default admin.
#
# The bundle is a "runtime" template: the base path and the values upstream
# compiles in (backendUrl, storefrontUrl, maxUploadFileSize, ADMIN_AUTH_TYPE,
# ADMIN_JWT_TOKEN_STORAGE_KEY) are filled in by the server from the project
# configuration (see packages/admin/dashboard-bundle/build-admin.mjs).
#
# Usage:
#   scripts/build-admin-bundle.sh
#
# Rerun it after every upstream bump (scripts/sync-upstream.sh) and commit the
# result. It uses the repository's yarn release (no global yarn needed);
# YARN_CACHE_FOLDER may point to an existing yarn cache to avoid downloads.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
YARN_RELEASE="$ROOT/.yarn/releases/yarn-4.18.1.cjs"
OUT="$ROOT/packages/admin/dashboard-bundle/app"

cd "$ROOT"

# turbo runs workspace scripts through `yarn`: provide one on PATH.
SHIM_DIR="$(mktemp -d)"
trap 'rm -rf "$SHIM_DIR"' EXIT
printf '#!/bin/sh\nexec node "%s" "$@"\n' "$YARN_RELEASE" > "$SHIM_DIR/yarn"
chmod +x "$SHIM_DIR/yarn"
export PATH="$SHIM_DIR:$PATH"

node "$YARN_RELEASE" install --immutable
node "$YARN_RELEASE" turbo run build --filter=@medusajs/admin-bundler... --no-daemon

# The build runs inside the repository so the generated entry
# (.medusa/client/entry.jsx) resolves @medusajs/dashboard from the workspace.
WORK_DIR="$ROOT/node_modules/.cache/medusa-admin-build"
rm -rf "$WORK_DIR" "$OUT"
mkdir -p "$WORK_DIR"

cat > "$WORK_DIR/options.json" <<EOF
{
  "resolveFrom": "$ROOT",
  "workDir": "$WORK_DIR",
  "outDir": "$OUT",
  "mode": "runtime",
  "sources": [],
  "plugins": []
}
EOF

# NODE_ENV=production: the published dashboard is a production build.
NODE_ENV=production node "$ROOT/packages/admin/dashboard-bundle/build-admin.mjs" "$WORK_DIR/options.json"

rm -rf "$WORK_DIR"

echo "Admin bundle written to ${OUT#"$ROOT/"} ($(du -sh "$OUT" | cut -f1), $(find "$OUT" -type f | wc -l | tr -d ' ') files)"
