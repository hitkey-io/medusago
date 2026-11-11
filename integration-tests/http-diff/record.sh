#!/usr/bin/env bash
# Records the golden responses of the HTTP scenarios from upstream Medusa.
#
# For each scenario: a fresh database (the Node-migrated reference schema
# with the data of seed.sql), an upstream server on it, the scenario
# recorded with `go test -run TestRecord`, the server stopped.
#
#   integration-tests/http-diff/record.sh [scenario...]
#
# Requires the reference app (reference/medusa/.ref-app) and the reference
# Postgres container. TEMPLATE_DB is a database migrated by upstream (a
# scenario's .env file may name another one with a TEMPLATE_DB line).
# REF_APP, REC_DB and PORT allow parallel recordings (another checkout,
# another database and port).
#
# Scenarios with fake third-party services (MEDUSA_TEST_FAKE_HOSTS,
# MEDUSA_TEST_MODULES, see fakes_test.go) need a reference application
# whose medusa-config.js requires "$MEDUSA_HTTP_DIFF_DIR/fake-hosts.js" when
# MEDUSA_TEST_FAKE_HOSTS is set and adds the modules of MEDUSA_TEST_MODULES
# (JSON) to its modules.
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$HERE/../.." && pwd)"
APP="${REF_APP:-$ROOT/reference/medusa/.ref-app}"
MEDUSA="$APP/../node_modules/.bin/medusa"
PG_CONTAINER="${PG_CONTAINER:-medusago-ref-pg}"
PG_URL_BASE="${PG_URL_BASE:-postgres://postgres:postgres@localhost:54329}"
TEMPLATE_DB="${TEMPLATE_DB:-medusa_ref}"
PORT="${PORT:-9101}"
DB="${REC_DB:-medusa_http_rec}"
# shared with runner_test.go (mfaEncryptionKey)
MFA_KEY=http-diff-mfa-encryption-key

psql_db() { docker exec -i "$PG_CONTAINER" psql -U postgres -v ON_ERROR_STOP=1 -q "$@"; }

scenarios=("$@")
if [ ${#scenarios[@]} -eq 0 ]; then
  for f in "$HERE"/scenarios/*.json; do scenarios+=("$(basename "$f" .json)"); done
fi

for scenario in "${scenarios[@]}"; do
  echo "== $scenario"
  # A TEMPLATE_DB line of the scenario environment selects another template
  # (one migrated by upstream with a feature flagged module enabled).
  scenario_env=()
  template="$TEMPLATE_DB"
  if [ -f "$HERE/scenarios/$scenario.env" ]; then
    while IFS= read -r line; do
      case "$line" in
        ''|'#'*) continue ;;
        TEMPLATE_DB=*) template="${line#TEMPLATE_DB=}"; continue ;;
      esac
      scenario_env+=("$line")
    done < "$HERE/scenarios/$scenario.env"
  fi
  psql_db -c "drop database if exists $DB with (force)" -c "create database $DB template $template"
  psql_db -d "$DB" -c "DO \$\$ DECLARE t text; BEGIN FOR t IN SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tablename NOT IN ('mikro_orm_migrations', 'script_migrations', 'link_module_migrations') LOOP EXECUTE format('TRUNCATE %I CASCADE', t); END LOOP; END \$\$;"
  psql_db -d "$DB" < "$HERE/seed.sql" >/dev/null
  # SEED_MIGRATE_SEARCH=true: the search indexes are migrated again on the
  # seeded data (the truncation above emptied their bookkeeping).
  migrate_search=false
  for line in "${scenario_env[@]:-}"; do
    [ "$line" = "SEED_MIGRATE_SEARCH=true" ] && migrate_search=true
  done
  if [ "$migrate_search" = true ]; then
    (cd "$APP" && export "${scenario_env[@]}" && NODE_ENV=development LOG_LEVEL=error DATABASE_URL="$PG_URL_BASE/$DB" \
      "$MEDUSA" db:migrate:search >/dev/null)
  fi
  (cd "$APP" && { [ ${#scenario_env[@]} -eq 0 ] || export "${scenario_env[@]}"; } && NODE_ENV=development LOG_LEVEL=error AUTH_MFA_ENCRYPTION_KEY="$MFA_KEY" MEDUSA_HTTP_DIFF_DIR="$HERE" DATABASE_URL="$PG_URL_BASE/$DB" \
    ADMIN_CORS=http://localhost:5173 AUTH_CORS=http://localhost:5173 STORE_CORS=http://localhost:8000 \
    exec "$MEDUSA" start -p "$PORT" >"$HERE/.record-server-$PORT.log" 2>&1) &
  pid=$!
  trap 'kill $pid 2>/dev/null || true' EXIT
  for _ in $(seq 1 120); do
    curl -sf "http://localhost:$PORT/health" >/dev/null && break
    sleep 1
  done
  (cd "$ROOT" && MEDUSA_HTTP_RECORD_URL="http://localhost:$PORT" MEDUSA_HTTP_SCENARIO="$scenario" \
    go test ./integration-tests/http-diff/ -run TestRecord -count=1 >/dev/null)
  kill $pid; wait $pid 2>/dev/null || true
  trap - EXIT
done
