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


# Default plugins: defineConfig adds @medusajs/draft-order to every
# application, so its admin extension (the Draft Orders section) is part of the
# stock dashboard. Optional plugins with an admin extension get a variant of
# the dashboard (the default plugins and the plugin), stored as
# app-<plugin dir>.tar.gz and served when the plugin is configured. Plugin
# admin extensions are compiled from the synced sources like
# `medusa plugin:build` (admin-bundler's plugin command, output in
# .medusa/server) and bundled through their "./admin" export.
DEFAULT_PLUGINS=(packages/plugins/draft-order)
OPTIONAL_PLUGINS=(packages/plugins/loyalty)

# build_plugin_admin <plugin dir>: like `medusa plugin:build`, which runs in
# the plugin's directory; prints the "<name>/admin" import.
build_plugin_admin() {
  (cd "$ROOT/$1" && node -e '
    const bundler = require(require.resolve("@medusajs/admin-bundler", { paths: [process.argv[1]] }))
    bundler.plugin({ root: process.argv[2], outDir: ".medusa/server" }).catch((e) => {
      console.error(e)
      process.exit(1)
    })
  ' "$ROOT" "$ROOT/$1") >&2
  printf '"%s/admin"' "$(node -p 'require(process.argv[1]).name' "$ROOT/$1/package.json")"
}

# build_dashboard <out dir> <plugins JSON>. The build runs inside the
# repository so the generated entry (.medusa/client/entry.jsx) resolves
# @medusajs/dashboard from the workspace.
build_dashboard() {
  local work_dir="$ROOT/node_modules/.cache/medusa-admin-build"
  rm -rf "$work_dir" "$1"
  mkdir -p "$work_dir"
  cat > "$work_dir/options.json" <<OPTIONS
{
  "resolveFrom": "$ROOT",
  "workDir": "$work_dir",
  "outDir": "$1",
  "mode": "runtime",
  "sources": [],
  "plugins": $2
}
OPTIONS
  # NODE_ENV=production: the published dashboard is a production build.
  NODE_ENV=production node "$ROOT/packages/admin/dashboard-bundle/build-admin.mjs" "$work_dir/options.json"
  rm -rf "$work_dir"
}

ADMIN_PLUGINS=()
for plugin_dir in "${DEFAULT_PLUGINS[@]}"; do
  ADMIN_PLUGINS+=("$(build_plugin_admin "$plugin_dir")")
done
build_dashboard "$OUT" "[$(IFS=,; echo "${ADMIN_PLUGINS[*]}")]"
echo "Admin bundle written to ${OUT#"$ROOT/"} ($(du -sh "$OUT" | cut -f1), $(find "$OUT" -type f | wc -l | tr -d ' ') files)"

for plugin_dir in "${OPTIONAL_PLUGINS[@]}"; do
  plugin="$(build_plugin_admin "$plugin_dir")"
  variant_dir="$(mktemp -d)"
  build_dashboard "$variant_dir/app" "[$(IFS=,; echo "${ADMIN_PLUGINS[*]}"),$plugin]"
  archive="$ROOT/packages/admin/dashboard-bundle/app-$(basename "$plugin_dir").tar.gz"
  node "$ROOT/scripts/pack-dir.mjs" "$variant_dir/app" "$archive"
  rm -rf "$variant_dir"
  echo "Admin bundle with $plugin written to ${archive#"$ROOT/"} ($(du -sh "$archive" | cut -f1))"
done
