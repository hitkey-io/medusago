#!/usr/bin/env bash
# Records the goldens of the db:generate harness from upstream Medusa.
#
#   integration-tests/db-generate/generate.sh
#
# The brand test module (app/) goes through the model versions of
# versions/v1, v2, ... in order: for each one, upstream `medusa db:generate
# brand` runs in a copy of the reference application (REF_APP, default
# reference/medusa/.ref-app-gen, created from .ref-app), the generated
# migration and the snapshot are recorded in golden/<version>.json, the
# migration is applied with `medusa db:migrate` and the schema of the module
# tables (schema.sql) is recorded in golden/<version>.schema.txt.
# generate_test.go replays the versions with the Go implementation.
#
# The database (GEN_DB) is created from TEMPLATE_DB, a database migrated by
# upstream (the core modules), on the reference Postgres container.
# extract-module-vectors.mjs records golden/modules.json (every default
# module from an empty snapshot).
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$HERE/../.." && pwd)"
APP="${REF_APP:-$ROOT/reference/medusa/.ref-app-gen}"
MEDUSA="$APP/../node_modules/.bin/medusa"
PG_CONTAINER="${PG_CONTAINER:-medusago-ref-pg}"
PG_URL_BASE="${PG_URL_BASE:-postgres://postgres:postgres@localhost:54329}"
TEMPLATE_DB="${TEMPLATE_DB:-medusa_ref}"
DB="${GEN_DB:-gen_upstream}"

psql_db() { docker exec -i "$PG_CONTAINER" psql -U postgres -v ON_ERROR_STOP=1 -q "$@"; }

if [ ! -d "$APP" ]; then
  cp -R "$(dirname "$APP")/.ref-app" "$APP"
fi
cp "$HERE/app/medusa-config.js" "$APP/medusa-config.js"
rm -rf "$APP/src"
mkdir -p "$APP/src/modules/brand/models"
cp "$HERE/app/src/modules/brand/index.js" "$APP/src/modules/brand/index.js"
MODELS="$APP/src/modules/brand/models"
MIGRATIONS="$APP/src/modules/brand/migrations"

psql_db -c "drop database if exists $DB with (force)" -c "create database $DB template $TEMPLATE_DB"

run_medusa() {
  (cd "$APP" && NODE_ENV=development LOG_LEVEL=error DATABASE_URL="$PG_URL_BASE/$DB" "$MEDUSA" "$@")
}

mkdir -p "$HERE/golden"
for dir in $(ls -d "$HERE"/versions/v* | sort -V); do
  version="$(basename "$dir")"
  echo "== $version"
  rm -f "$MODELS"/*
  cp "$dir"/*.js "$MODELS/"
  before="$(ls "$MIGRATIONS"/Migration*.ts 2>/dev/null || true)"
  run_medusa db:generate brand
  generated=""
  for f in "$MIGRATIONS"/Migration*.ts; do
    [ -e "$f" ] || continue
    case "$before" in *"$f"*) ;; *) generated="$f" ;; esac
  done
  node "$HERE/record-version.mjs" "$HERE/golden/$version.json" "$MIGRATIONS" "$MIGRATIONS/.snapshot-brand.json" $generated
  run_medusa db:migrate --skip-links --skip-scripts
  docker exec -i "$PG_CONTAINER" psql -U postgres -d "$DB" -At -v ON_ERROR_STOP=1 < "$HERE/schema.sql" > "$HERE/golden/$version.schema.txt"
done
