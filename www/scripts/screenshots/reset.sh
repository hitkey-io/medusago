#!/usr/bin/env bash
# Fresh screenshot stand: database, admin user, server on :9000 and the demo data.
#   scripts/screenshots/reset.sh   (needs Docker Postgres on :54329 and a built medusa binary in $APP)
set -euo pipefail
APP=${APP:-/tmp/shots-app}
cd "$APP"
pkill -f "medusa start -p 9000" 2>/dev/null || true
docker exec medusago-ref-pg psql -U postgres -q -c "drop database if exists docs_shots" -c "create database docs_shots"
./medusa db:migrate >/dev/null 2>&1
./medusa user -e admin@medusa-test.com -p supersecret >/dev/null 2>&1
(nohup ./medusa start -p 9000 > server.log 2>&1 &)
for i in $(seq 1 30); do curl -s -o /dev/null localhost:9000/health && break; sleep 1; done
cd - >/dev/null
node scripts/screenshots/seed.mjs
node scripts/screenshots/seed-extra.mjs
# spread the order dates over the last weeks (dates only exist in the database)
docker exec medusago-ref-pg psql -U postgres -d docs_shots -q -c "
update \"order\" o set created_at = now() - (interval '1 day' * (n.rn * 2 + 1)) - (interval '1 minute' * (n.rn * 37 % 600)),
  updated_at = now() - (interval '1 day' * (n.rn * 2 + 1))
from (select id, row_number() over (order by display_id desc) as rn from \"order\") n where n.id = o.id"
