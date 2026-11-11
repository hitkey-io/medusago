-- The schema of the test module tables (public.brand*): columns, constraints
-- and indexes, one line each, sorted. generate.sh records it with psql -At
-- after each version is migrated upstream; generate_test.go runs the same
-- query on the database the Go migrations were applied to.
SELECT coalesce(string_agg(line, E'\n' ORDER BY line), '') FROM (
  SELECT format('column %s.%s %s %s notnull=%s default=%s', c.relname, a.attnum, a.attname,
      format_type(a.atttypid, a.atttypmod), a.attnotnull, pg_get_expr(d.adbin, d.adrelid)) AS line
    FROM pg_attribute a
    JOIN pg_class c ON c.oid = a.attrelid
    JOIN pg_namespace n ON n.oid = c.relnamespace
    LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
    WHERE n.nspname = 'public' AND c.relkind = 'r' AND c.relname LIKE 'brand%'
      AND a.attnum > 0 AND NOT a.attisdropped
  UNION ALL
  SELECT format('constraint %s.%s %s', c.relname, con.conname, pg_get_constraintdef(con.oid))
    FROM pg_constraint con
    JOIN pg_class c ON c.oid = con.conrelid
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE n.nspname = 'public' AND c.relname LIKE 'brand%'
  UNION ALL
  SELECT format('index %s.%s %s', tablename, indexname, indexdef)
    FROM pg_indexes
    WHERE schemaname = 'public' AND tablename LIKE 'brand%'
  UNION ALL
  SELECT format('sequence %s', c.relname)
    FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE n.nspname = 'public' AND c.relkind = 'S' AND c.relname LIKE 'brand%'
) lines;
