#!/usr/bin/env bash
#
# Regenerates seed.sql and the golden results from the upstream Medusa
# application in reference/medusa/.ref-app.
#
# Requires: the reference Postgres container (PG_CONTAINER), a database
# migrated by upstream `medusa db:migrate` (TEMPLATE_DB) and a built upstream
# checkout.

set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$HERE/../.." && pwd)"
APP="$ROOT/reference/medusa/.ref-app"
MEDUSA="$ROOT/reference/medusa/node_modules/.bin/medusa"
PG_CONTAINER="${PG_CONTAINER:-medusago-ref-pg}"
PG_URL_BASE="${PG_URL_BASE:-postgres://postgres:postgres@localhost:54329}"
TEMPLATE_DB="${TEMPLATE_DB:-medusa_ref}"

psql_admin() { docker exec "$PG_CONTAINER" psql -U postgres -qc "$1"; }
fresh_db() { psql_admin "drop database if exists $1"; psql_admin "create database $1 template $2"; }

# 1. Seed through the upstream services.
fresh_db medusa_diff_seed "$TEMPLATE_DB"
(cd "$APP" && DATABASE_URL="$PG_URL_BASE/medusa_diff_seed" LOG_LEVEL=error "$MEDUSA" exec "$HERE/seed.js" >/dev/null)

docker exec "$PG_CONTAINER" pg_dump -U postgres --data-only --column-inserts --no-owner --no-privileges \
  --exclude-table=mikro_orm_migrations --exclude-table=script_migrations --exclude-table=link_module_migrations \
  medusa_diff_seed | grep -vE '^(\\(un)?restrict|SELECT pg_catalog.set_config)|(mikro_orm_migrations|script_migrations|link_module_migrations)_id_seq' > "$HERE/seed.sql"

# 2. Golden results, each case file on a fresh database holding exactly
#    seed.sql (the same data, in the same physical order, as the Go tests).
load_seed() {
  docker exec "$PG_CONTAINER" psql -U postgres -d "$1" -qv ON_ERROR_STOP=1 -c "
    DO \$\$ DECLARE t text; BEGIN
      FOR t IN SELECT tablename FROM pg_tables WHERE schemaname = 'public'
        AND tablename NOT IN ('mikro_orm_migrations', 'script_migrations', 'link_module_migrations')
      LOOP EXECUTE format('TRUNCATE %I CASCADE', t); END LOOP;
    END \$\$;"
  docker exec -i "$PG_CONTAINER" psql -U postgres -d "$1" -qv ON_ERROR_STOP=1 < "$HERE/seed.sql" >/dev/null
}

for cases in "$HERE"/cases/*.json; do
  name="$(basename "$cases")"
  fresh_db medusa_diff_run "$TEMPLATE_DB"
  load_seed medusa_diff_run
  (cd "$APP" && DATABASE_URL="$PG_URL_BASE/medusa_diff_run" LOG_LEVEL=error "$MEDUSA" exec "$HERE/run-calls.js" "$cases" "$HERE/golden/$name" >/dev/null)
  echo "golden/$name"
done
