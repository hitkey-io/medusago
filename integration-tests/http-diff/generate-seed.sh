#!/usr/bin/env bash
# Regenerates seed.sql: a database migrated by upstream, seeded through the
# upstream workflows (seed.js) plus the admin user, dumped as data only.
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$HERE/../.." && pwd)"
APP="$ROOT/reference/medusa/.ref-app"
MEDUSA="$APP/../node_modules/.bin/medusa"
PG_CONTAINER="${PG_CONTAINER:-medusago-ref-pg}"
PG_URL_BASE="${PG_URL_BASE:-postgres://postgres:postgres@localhost:54329}"
TEMPLATE_DB="${TEMPLATE_DB:-medusa_ref}"
DB=medusa_http_seed

psql_admin() { docker exec "$PG_CONTAINER" psql -U postgres -v ON_ERROR_STOP=1 -qc "$1"; }
psql_admin "drop database if exists $DB with (force)"
psql_admin "create database $DB template $TEMPLATE_DB"

cd "$APP"
DATABASE_URL="$PG_URL_BASE/$DB" LOG_LEVEL=error "$MEDUSA" exec "$HERE/seed.js" >/dev/null
DATABASE_URL="$PG_URL_BASE/$DB" LOG_LEVEL=error "$MEDUSA" user -e admin@medusa-test.com -p supersecret >/dev/null

docker exec "$PG_CONTAINER" pg_dump -U postgres --data-only --column-inserts --no-owner --no-privileges \
  --exclude-table=mikro_orm_migrations --exclude-table=script_migrations --exclude-table=link_module_migrations \
  "$DB" 2>/dev/null | grep -vE '^(\\(un)?restrict|SELECT pg_catalog.set_config)|(mikro_orm_migrations|script_migrations|link_module_migrations)_id_seq' > "$HERE/seed.sql"
echo "seed.sql: $(grep -c '^INSERT' "$HERE/seed.sql") rows"
