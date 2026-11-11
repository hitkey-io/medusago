#!/usr/bin/env bash
#
# Regenerates seed.sql and the golden results from the upstream Medusa
# application in reference/medusa/.ref-app.
#
# Requires: the reference Postgres container (PG_CONTAINER), a database
# migrated by upstream `medusa db:migrate` (TEMPLATE_DB) and a built upstream
# checkout.
#
#   generate.sh               reseeds and regenerates every golden file
#   generate.sh <name>.json   regenerates the named goldens only (seed.sql kept)
#
# REF_APP and DB_PREFIX allow parallel runs (another reference app copy and
# database names).

set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$HERE/../.." && pwd)"
APP="${REF_APP:-$ROOT/reference/medusa/.ref-app}"
MEDUSA="$APP/../node_modules/.bin/medusa"
SEED_DB="${DB_PREFIX:-}medusa_diff_seed"
RUN_DB="${DB_PREFIX:-}medusa_diff_run"
PG_CONTAINER="${PG_CONTAINER:-medusago-ref-pg}"
PG_URL_BASE="${PG_URL_BASE:-postgres://postgres:postgres@localhost:54329}"
TEMPLATE_DB="${TEMPLATE_DB:-medusa_ref}"

psql_admin() { docker exec "$PG_CONTAINER" psql -U postgres -qc "$1"; }
fresh_db() { psql_admin "drop database if exists $1"; psql_admin "create database $1 template $2"; }

# 1. Seed through the upstream services.
if [ $# -eq 0 ]; then
  fresh_db "$SEED_DB" "$TEMPLATE_DB"
  (cd "$APP" && DATABASE_URL="$PG_URL_BASE/$SEED_DB" LOG_LEVEL=error "$MEDUSA" exec "$HERE/seed.js" >/dev/null)

  docker exec "$PG_CONTAINER" pg_dump -U postgres --data-only --column-inserts --no-owner --no-privileges \
    --exclude-table=mikro_orm_migrations --exclude-table=script_migrations --exclude-table=link_module_migrations \
    "$SEED_DB" | grep -vE '^(\\(un)?restrict|SELECT pg_catalog.set_config)|(mikro_orm_migrations|script_migrations|link_module_migrations)_id_seq' > "$HERE/seed.sql"
fi

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

files=("$@")
[ ${#files[@]} -gt 0 ] || files=("$HERE"/cases/*.json)
for cases in "${files[@]}"; do
  name="$(basename "$cases")"
  cases="$HERE/cases/$name"
  fresh_db "$RUN_DB" "$TEMPLATE_DB"
  load_seed "$RUN_DB"
  (cd "$APP" && DATABASE_URL="$PG_URL_BASE/$RUN_DB" LOG_LEVEL=error "$MEDUSA" exec "$HERE/run-calls.js" "$cases" "$HERE/golden/$name" >/dev/null)
  echo "golden/$name"
done
