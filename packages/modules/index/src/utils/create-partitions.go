package utils

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// CreatePartitions mirrors createPartitions: one index_data partition per
// indexed entity with listeners, one index_relation partition per
// (non inverse) parent pivot, their indexes and count_estimate().
func CreatePartitions(ctx context.Context, db dal.Querier, schema *SchemaObjectRepresentation, logger *slog.Logger) error {
	activeSchema := `"public".`

	created := map[string]bool{}
	var partitions []string
	var keys []string
	for _, key := range schema.Keys {
		if len(schema.Entities[key].Listeners) > 0 {
			keys = append(keys, key)
		}
	}
	for _, key := range keys {
		cName := NormalizeTableName(key)
		if created[cName] {
			continue
		}
		created[cName] = true
		partitions = append(partitions, fmt.Sprintf("CREATE TABLE IF NOT EXISTS %scat_%s PARTITION OF %sindex_data FOR VALUES IN ('%s')", activeSchema, cName, activeSchema, key))
		for _, parent := range schema.Entities[key].Parents {
			if parent.IsInverse {
				continue
			}
			pName := GetPivotTableName(parent.Ref.Entity + key)
			if created[pName] {
				continue
			}
			created[pName] = true
			partitions = append(partitions, fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s%s PARTITION OF %sindex_relation FOR VALUES IN ('%s-%s')", activeSchema, pName, activeSchema, parent.Ref.Entity, key))
		}
	}
	if len(partitions) == 0 {
		return nil
	}
	if _, err := db.Exec(ctx, strings.Join(partitions, "; "), pgx.QueryExecModeSimpleProtocol); err != nil {
		return err
	}

	var commands []string
	for _, key := range keys {
		cName := NormalizeTableName(key)
		commands = append(commands,
			fmt.Sprintf(`CREATE INDEX CONCURRENTLY IF NOT EXISTS "IDX_cat_%s_data_gin" ON %scat_%s USING GIN ("data" jsonb_path_ops)`, cName, activeSchema, cName),
			fmt.Sprintf(`CREATE INDEX CONCURRENTLY IF NOT EXISTS "IDX_cat_%s_id" ON %scat_%s ("id")`, cName, activeSchema, cName),
		)
		for _, parent := range schema.Entities[key].Parents {
			if parent.IsInverse {
				continue
			}
			pName := GetPivotTableName(parent.Ref.Entity + key)
			commands = append(commands, fmt.Sprintf(`CREATE INDEX CONCURRENTLY IF NOT EXISTS "IDX_%s_child_id" ON %s%s ("child_id")`, pName, activeSchema, pName))
		}
	}
	for _, cmd := range commands {
		if _, err := db.Exec(ctx, cmd, pgx.QueryExecModeSimpleProtocol); err != nil {
			if logger != nil {
				logger.Error("Failed to create index: " + err.Error())
			}
		}
	}

	_, err := db.Exec(ctx, `
    CREATE OR REPLACE FUNCTION count_estimate(query text) RETURNS bigint AS $$
    DECLARE
        plan jsonb;
    BEGIN
        EXECUTE 'EXPLAIN (FORMAT JSON) ' || query INTO plan;
        RETURN (plan->0->'Plan'->>'Plan Rows')::bigint;
    END;
    $$ LANGUAGE plpgsql;
  `, pgx.QueryExecModeSimpleProtocol)
	return err
}
