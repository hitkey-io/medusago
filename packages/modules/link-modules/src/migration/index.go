// Package migration mirrors @medusajs/link-modules/src/migration: it keeps the
// link tables in sync with the link definitions and tracks them in the
// link_module_migrations table exactly like upstream.
package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	"github.com/hitkey-io/medusago/packages/modules/link-modules/src/utils"
)

const tableName = "link_module_migrations"

// externallyManagedTables are link tables of Medusa Enterprise Edition
// features, which are not part of this port. A database created by Medusa
// has them; they are left untouched (never planned for deletion).
var externallyManagedTables = []string{"invite_rbac_role", "user_rbac_role"}

// LinkDescriptor mirrors PlannerActionLinkDescriptor.
type LinkDescriptor struct {
	FromModule string `json:"fromModule"`
	ToModule   string `json:"toModule"`
	FromModel  string `json:"fromModel"`
	ToModel    string `json:"toModel"`
}

// Action is one planned change: create, update, delete, noop or notify.
type Action struct {
	Action         string
	TableName      string
	LinkDescriptor LinkDescriptor
	SQL            string
	// Reason explains a notify action.
	Reason string
}

type linkEntity struct {
	entity     *utils.LinkEntity
	descriptor LinkDescriptor
}

// Planner mirrors MigrationsExecutionPlanner.
type Planner struct {
	db       *pgxpool.Pool
	schema   string
	entities []linkEntity
}

// NewPlanner builds the planner for the given (non read-only) link configs.
func NewPlanner(db *pgxpool.Pool, configs []*joiner.ModuleJoinerConfig) (*Planner, error) {
	p := &Planner{db: db, schema: "public"}
	for _, cfg := range configs {
		if cfg.IsReadOnlyLink {
			continue
		}
		if len(cfg.Relationships) != 2 {
			return nil, fmt.Errorf("link %s must have exactly 2 relationships", cfg.ServiceName)
		}
		primary, foreign := cfg.Relationships[0], cfg.Relationships[1]
		entity, err := utils.GenerateEntity(cfg, primary, foreign)
		if err != nil {
			return nil, err
		}
		p.entities = append(p.entities, linkEntity{
			entity: entity,
			descriptor: LinkDescriptor{
				FromModule: primary.ServiceName,
				ToModule:   foreign.ServiceName,
				FromModel:  primary.Alias,
				ToModel:    foreign.Alias,
			},
		})
	}
	return p, nil
}

func (p *Planner) ensureMigrationsTable(ctx context.Context) error {
	_, err := p.db.Exec(ctx, fmt.Sprintf(`
      CREATE TABLE IF NOT EXISTS "%s"."%s" (
        id SERIAL PRIMARY KEY,
        table_name VARCHAR(255) NOT NULL UNIQUE,
        link_descriptor JSONB NOT NULL DEFAULT '{}'::jsonb,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
      )
    `, p.schema, tableName))
	return err
}

// ensureMigrationsTableUpToDate tracks link tables that already exist but are
// not recorded yet.
func (p *Planner) ensureMigrationsTableUpToDate(ctx context.Context) error {
	rows, err := p.db.Query(ctx, `SELECT table_name FROM information_schema.tables WHERE table_schema = $1`, p.schema)
	if err != nil {
		return err
	}
	existing, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}

	for _, name := range existing {
		for _, le := range p.entities {
			if le.entity.TableName != name {
				continue
			}
			descriptor, _ := json.Marshal(le.descriptor)
			if _, err := p.db.Exec(ctx, fmt.Sprintf(
				`INSERT INTO "%s"."%s" (table_name, link_descriptor) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				p.schema, tableName), name, descriptor); err != nil {
				return err
			}
		}
	}
	return nil
}

type trackedTable struct {
	TableName      string
	LinkDescriptor LinkDescriptor
}

func (p *Planner) trackedTables(ctx context.Context) ([]trackedTable, error) {
	rows, err := p.db.Query(ctx, fmt.Sprintf(`SELECT table_name, link_descriptor FROM "%s"."%s"`, p.schema, tableName))
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (trackedTable, error) {
		var t trackedTable
		var raw []byte
		if err := row.Scan(&t.TableName, &raw); err != nil {
			return t, err
		}
		return t, json.Unmarshal(raw, &t.LinkDescriptor)
	})
}

// migrateOldTables renames tracked tables whose link now uses another name.
func (p *Planner) migrateOldTables(ctx context.Context, tracked []trackedTable) ([]trackedTable, error) {
	out := make([]trackedTable, 0, len(tracked))
	for _, t := range tracked {
		var match *linkEntity
		for i, le := range p.entities {
			d := le.descriptor
			if d.FromModel == t.LinkDescriptor.FromModel &&
				d.ToModel == t.LinkDescriptor.ToModel &&
				strings.EqualFold(d.FromModule, t.LinkDescriptor.FromModule) &&
				strings.EqualFold(d.ToModule, t.LinkDescriptor.ToModule) {
				match = &p.entities[i]
				break
			}
		}
		if match != nil && match.entity.TableName != t.TableName {
			descriptor, _ := json.Marshal(match.descriptor)
			if _, err := p.db.Exec(ctx, fmt.Sprintf(
				`ALTER TABLE "%s"."%s" RENAME TO "%s"`, p.schema, t.TableName, match.entity.TableName)); err != nil {
				return nil, err
			}
			if _, err := p.db.Exec(ctx, fmt.Sprintf(
				`UPDATE "%s"."%s" SET table_name = $1, link_descriptor = $2 WHERE table_name = $3`, p.schema, tableName),
				match.entity.TableName, descriptor, t.TableName); err != nil {
				return nil, err
			}
			t.TableName = match.entity.TableName
		}
		out = append(out, t)
	}
	return out, nil
}

// CreatePlan mirrors createPlan.
func (p *Planner) CreatePlan(ctx context.Context) ([]Action, error) {
	if err := p.ensureMigrationsTable(ctx); err != nil {
		return nil, err
	}
	if err := p.ensureMigrationsTableUpToDate(ctx); err != nil {
		return nil, err
	}
	tracked, err := p.trackedTables(ctx)
	if err != nil {
		return nil, err
	}
	tracked, err = p.migrateOldTables(ctx, tracked)
	if err != nil {
		return nil, err
	}
	trackedNames := make([]string, len(tracked))
	for i, t := range tracked {
		trackedNames[i] = t.TableName
	}

	var actions []Action
	var linkNames []string
	for _, le := range p.entities {
		linkNames = append(linkNames, le.entity.TableName)
		if !slices.Contains(trackedNames, le.entity.TableName) {
			actions = append(actions, Action{
				Action:         "create",
				TableName:      le.entity.TableName,
				LinkDescriptor: le.descriptor,
				SQL:            le.entity.CreateSchemaSQL(),
			})
			continue
		}
		reason, err := p.diffExistingTable(ctx, le.entity)
		if err != nil {
			return nil, err
		}
		action := Action{Action: "noop", TableName: le.entity.TableName, LinkDescriptor: le.descriptor}
		if reason != "" {
			action.Action = "notify"
			action.Reason = reason
		}
		actions = append(actions, action)
	}

	for _, t := range tracked {
		if !slices.Contains(linkNames, t.TableName) && !slices.Contains(externallyManagedTables, t.TableName) {
			actions = append(actions, Action{Action: "delete", TableName: t.TableName, LinkDescriptor: t.LinkDescriptor})
		}
	}
	return actions, nil
}

// diffExistingTable compares a tracked table with its expected definition.
// Upstream computes an update diff with the MikroORM schema generator; this
// port reports any difference as a notify action instead of altering the
// table automatically.
func (p *Planner) diffExistingTable(ctx context.Context, e *utils.LinkEntity) (string, error) {
	rows, err := p.db.Query(ctx, `
		SELECT column_name, is_nullable = 'YES'
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2`, p.schema, e.TableName)
	if err != nil {
		return "", err
	}
	type col struct {
		name     string
		nullable bool
	}
	actual, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (col, error) {
		var c col
		return c, r.Scan(&c.name, &c.nullable)
	})
	if err != nil {
		return "", err
	}
	actualByName := map[string]bool{}
	for _, c := range actual {
		actualByName[c.name] = c.nullable
	}

	var diffs []string
	for _, c := range e.Columns {
		nullable, ok := actualByName[c.Name]
		switch {
		case !ok:
			diffs = append(diffs, fmt.Sprintf("missing column %q", c.Name))
		case nullable != c.Nullable:
			diffs = append(diffs, fmt.Sprintf("column %q nullability differs", c.Name))
		}
		delete(actualByName, c.Name)
	}
	for name := range actualByName {
		diffs = append(diffs, fmt.Sprintf("unexpected column %q", name))
	}

	idxRows, err := p.db.Query(ctx, `SELECT indexname FROM pg_indexes WHERE schemaname = $1 AND tablename = $2`, p.schema, e.TableName)
	if err != nil {
		return "", err
	}
	indexes, err := pgx.CollectRows(idxRows, pgx.RowTo[string])
	if err != nil {
		return "", err
	}
	for _, idx := range e.Indexes {
		if !slices.Contains(indexes, idx.Name) {
			diffs = append(diffs, fmt.Sprintf("missing index %q", idx.Name))
		}
	}

	slices.Sort(diffs)
	return strings.Join(diffs, "; "), nil
}

// ExecutePlan mirrors executePlan: create, update and delete actions are
// applied, each under the same advisory lock key as upstream so that a
// concurrent Medusa process and this one never apply the same change twice.
func (p *Planner) ExecutePlan(ctx context.Context, actions []Action) error {
	for _, action := range actions {
		if action.Action != "create" && action.Action != "update" && action.Action != "delete" {
			continue
		}
		err := pgx.BeginFunc(ctx, p.db, func(tx pgx.Tx) error {
			lockKey := "db-link-migration:" + action.TableName
			if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, lockKey); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, fmt.Sprintf(`SET LOCAL search_path TO "%s"`, p.schema)); err != nil {
				return err
			}

			switch action.Action {
			case "create":
				descriptor, _ := json.Marshal(action.LinkDescriptor)
				if _, err := tx.Exec(ctx, fmt.Sprintf(
					`INSERT INTO "%s" (table_name, link_descriptor) VALUES ($1, $2) ON CONFLICT DO NOTHING`, tableName),
					action.TableName, descriptor); err != nil {
					return err
				}
				_, err := tx.Exec(ctx, action.SQL, pgx.QueryExecModeSimpleProtocol)
				return err
			case "update":
				_, err := tx.Exec(ctx, action.SQL, pgx.QueryExecModeSimpleProtocol)
				return err
			case "delete":
				if _, err := tx.Exec(ctx, fmt.Sprintf(`DROP TABLE IF EXISTS "%s"`, action.TableName)); err != nil {
					return err
				}
				_, err := tx.Exec(ctx, fmt.Sprintf(`DELETE FROM "%s" WHERE table_name = $1`, tableName), action.TableName)
				return err
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("link table %s: %w", action.TableName, err)
		}
	}
	return nil
}
