package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/oklog/ulid/v2"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// LinkInput is one link to create: the values of the two foreign keys (in
// the order of the relationships; a composite primary side is not
// supported) and the extra column values.
type LinkInput struct {
	Primary any
	Foreign any
	Extra   *dal.Record
}

func quote(s string) string { return `"` + strings.ReplaceAll(s, `"`, `""`) + `"` }

func (s *LinkModuleService) keys() (string, string) {
	return s.Config.Relationships[0].ForeignKey, s.Config.Relationships[1].ForeignKey
}

func (s *LinkModuleService) idPrefix() string {
	if s.Config.DatabaseConfig != nil && s.Config.DatabaseConfig.IDPrefix != "" {
		return s.Config.DatabaseConfig.IDPrefix
	}
	return "link"
}

// serializeRow orders a link row like the MikroORM serializer: primary keys
// first, then the other columns in entity order; numeric extra columns with
// runtimeType number become numbers.
func (s *LinkModuleService) serializeRow(row dal.Row) *dal.Record {
	e := s.repository.Entity
	out := dal.NewRecord()
	for _, pk := range e.Meta.PrimaryKey {
		out.Set(pk, row[pk])
	}
	for _, p := range e.Meta.Properties {
		if out.Has(p.Name) {
			continue
		}
		v := row[p.Name]
		if p.Type == "number" {
			if str, ok := v.(string); ok {
				var f float64
				if _, err := fmt.Sscan(str, &f); err == nil {
					v = f
				}
			}
		}
		out.Set(p.Name, v)
	}
	return out
}

// Create mirrors create: an upsert on the foreign key pair (a new id, the
// extra columns and deleted_at = null), returning the rows.
func (s *LinkModuleService) Create(ctx context.Context, links []LinkInput) ([]*dal.Record, error) {
	if len(links) == 0 {
		return []*dal.Record{}, nil
	}
	fkA, fkB := s.keys()
	var extraCols []string
	if s.Config.DatabaseConfig != nil {
		for _, x := range s.Config.DatabaseConfig.ExtraFields {
			extraCols = append(extraCols, x.Name)
		}
	}
	cols := append([]string{fkA, fkB, "id"}, extraCols...)
	cols = append(cols, "deleted_at")
	var args []any
	var values []string
	type defaultValue struct{}
	for _, l := range links {
		row := []any{l.Primary, l.Foreign, s.idPrefix() + "_" + ulid.Make().String()}
		for _, c := range extraCols {
			var v any = defaultValue{}
			if l.Extra != nil {
				if x, ok := l.Extra.Get(c); ok {
					v = x
				}
			}
			row = append(row, v)
		}
		row = append(row, nil)
		ph := make([]string, len(row))
		for i, v := range row {
			if _, ok := v.(defaultValue); ok {
				ph[i] = "default"
				continue
			}
			args = append(args, v)
			ph[i] = fmt.Sprintf("$%d", len(args))
		}
		values = append(values, "("+strings.Join(ph, ", ")+")")
	}
	quoted := make([]string, len(cols))
	var set []string
	for i, c := range cols {
		quoted[i] = quote(c)
		if c != fkA && c != fkB {
			set = append(set, fmt.Sprintf("%s = excluded.%s", quote(c), quote(c)))
		}
	}
	// Extra columns not provided keep their default on insert.
	sql := fmt.Sprintf(`insert into %s (%s) values %s on conflict (%s, %s) do update set %s returning *`,
		quote(s.repository.Entity.Meta.Collection), strings.Join(quoted, ", "), strings.Join(values, ", "),
		quote(fkA), quote(fkB), strings.Join(set, ", "))
	rows, err := dal.QueryRows(ctx, s.repository.DB, sql, args...)
	if err != nil {
		return nil, dal.MapDBError(err)
	}
	out := make([]*dal.Record, len(rows))
	for i, r := range rows {
		out[i] = s.serializeRow(r)
	}
	return out, nil
}

// Dismiss mirrors dismiss: the matching links are soft deleted and returned
// as they were before the update (with deleted_at set).
func (s *LinkModuleService) Dismiss(ctx context.Context, pairs [][2]any) ([]*dal.Record, error) {
	if len(pairs) == 0 {
		return []*dal.Record{}, nil
	}
	fkA, fkB := s.keys()
	var args []any
	var conds []string
	for _, p := range pairs {
		args = append(args, p[0], p[1])
		conds = append(conds, fmt.Sprintf("(%s = $%d and %s = $%d)", quote(fkA), len(args)-1, quote(fkB), len(args)))
	}
	table := quote(s.repository.Entity.Meta.Collection)
	now := dal.Now()
	args = append(args, now)
	sql := fmt.Sprintf(`with "old" as (select * from %s where %s) update %s as "t" set "deleted_at" = $%d, "updated_at" = $%d from "old" where "t".%s = "old".%s and "t".%s = "old".%s returning "old".*`,
		table, strings.Join(conds, " or "), table, len(args), len(args), quote(fkA), quote(fkA), quote(fkB), quote(fkB))
	rows, err := dal.QueryRows(ctx, s.repository.DB, sql, args...)
	if err != nil {
		return nil, dal.MapDBError(err)
	}
	out := make([]*dal.Record, len(rows))
	for i, r := range rows {
		r["deleted_at"] = now
		out[i] = s.serializeRow(r)
	}
	return out, nil
}
