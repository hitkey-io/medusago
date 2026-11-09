// Package utils mirrors @medusajs/link-modules/src/utils.
package utils

import (
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dml"
	"github.com/hitkey-io/medusago/packages/core/utils/src/link"
)

// Column is a column of a link table.
type Column struct {
	// RuntimeType is options.runtimeType of an extra field ("number").
	RuntimeType string
	Name        string
	SQLType     string
	Nullable    bool
	DefaultRaw  *string
	Primary     bool
}

// Index is an index of a link table, either a plain index on Columns or a
// raw Expression.
type Index struct {
	Name       string
	Columns    []string
	Expression string
}

// LinkEntity mirrors the MikroORM EntitySchema generateEntity builds for a
// link definition.
type LinkEntity struct {
	TableName string
	Columns   []Column
	Indexes   []Index
}

// mikroOrmColumnType maps a link column type to the Postgres type the
// MikroORM schema generator emits. Only mappings verified against upstream
// output are supported; an explicit options.columnType always wins.
func mikroOrmColumnType(fieldType string, options map[string]any) (string, error) {
	if ct, ok := options["columnType"].(string); ok && ct != "" {
		return ct, nil
	}
	switch fieldType {
	case "string":
		return "varchar(255)", nil
	default:
		return "", fmt.Errorf("link column type %q is not supported yet", fieldType)
	}
}

// GenerateEntity mirrors generateEntity from generate-entity.ts.
func GenerateEntity(cfg *joiner.ModuleJoinerConfig, primary, foreign joiner.JoinerRelationship) (*LinkEntity, error) {
	primaryKeys := strings.Split(primary.ForeignKey, ",")

	tableName := ""
	if cfg.DatabaseConfig != nil {
		tableName = cfg.DatabaseConfig.TableName
	}
	if tableName == "" {
		tableName = strings.ToLower(link.ComposeTableName(
			primary.ServiceName, primary.ForeignKey, foreign.ServiceName, foreign.ForeignKey,
		))
	}
	hash := common.SimpleHash(tableName)
	table := common.CompressName(tableName)

	columns := []Column{{Name: "id", SQLType: "varchar(255)"}}
	for _, fk := range append(append([]string{}, primaryKeys...), foreign.ForeignKey) {
		columns = append(columns, Column{Name: fk, SQLType: "varchar(255)", Primary: true})
	}

	if cfg.DatabaseConfig != nil {
		for _, extra := range cfg.DatabaseConfig.ExtraFields {
			sqlType, err := mikroOrmColumnType(extra.Type, extra.Options)
			if err != nil {
				return nil, fmt.Errorf("%s.%s: %w", tableName, extra.Name, err)
			}
			runtimeType, _ := extra.Options["runtimeType"].(string)
			columns = append(columns, Column{
				RuntimeType: runtimeType,
				Name:        extra.Name,
				SQLType:     sqlType,
				Nullable:    extra.Nullable,
				DefaultRaw:  extra.DefaultValue,
			})
		}
	}

	currentTimestamp := "CURRENT_TIMESTAMP"
	columns = append(columns,
		Column{Name: "created_at", SQLType: "timestamptz", DefaultRaw: &currentTimestamp},
		Column{Name: "updated_at", SQLType: "timestamptz", DefaultRaw: &currentTimestamp},
		Column{Name: "deleted_at", SQLType: "timestamptz", Nullable: true},
	)

	partial := func(cols string) string {
		return fmt.Sprintf(`CREATE INDEX IF NOT EXISTS "IDX_%s_%s" ON "%s" ("%s") WHERE deleted_at IS NULL`,
			strings.ReplaceAll(cols, ",", "_"), hash, table, cols)
	}

	return &LinkEntity{
		TableName: table,
		Columns:   columns,
		Indexes: []Index{
			{Name: "IDX_id_" + hash, Columns: []string{"id"}},
			{Name: "IDX_" + strings.Join(primaryKeys, "_") + "_" + hash, Expression: partial(strings.Join(primaryKeys, ","))},
			{Name: "IDX_" + foreign.ForeignKey + "_" + hash, Expression: partial(foreign.ForeignKey)},
			{Name: "IDX_deleted_at_" + hash, Columns: []string{"deleted_at"}},
		},
	}, nil
}

// CreateSchemaSQL reproduces the normalized output of MikroORM's
// SchemaGenerator.getCreateSchemaSQL() for the entity.
func (e *LinkEntity) CreateSchemaSQL() string {
	// MikroORM emits primary key columns first, then the others in
	// declaration order.
	ordered := make([]Column, 0, len(e.Columns))
	var pks []string
	for _, c := range e.Columns {
		if c.Primary {
			ordered = append(ordered, c)
			pks = append(pks, `"`+c.Name+`"`)
		}
	}
	for _, c := range e.Columns {
		if !c.Primary {
			ordered = append(ordered, c)
		}
	}

	defs := make([]string, 0, len(ordered)+1)
	for _, c := range ordered {
		def := fmt.Sprintf(`"%s" %s`, c.Name, c.SQLType)
		if c.Nullable {
			def += " null"
		} else {
			def += " not null"
		}
		if c.DefaultRaw != nil {
			def += " default " + *c.DefaultRaw
		}
		defs = append(defs, def)
	}
	defs = append(defs, fmt.Sprintf(`constraint "%s_pkey" primary key (%s)`, e.TableName, strings.Join(pks, ", ")))

	var b strings.Builder
	b.WriteString("set names 'utf8';\n\n")
	fmt.Fprintf(&b, "create table if not exists \"%s\" (%s);\n", e.TableName, strings.Join(defs, ", "))
	for _, idx := range e.Indexes {
		if idx.Expression != "" {
			b.WriteString(idx.Expression + ";\n")
			continue
		}
		quoted := make([]string, len(idx.Columns))
		for i, c := range idx.Columns {
			quoted[i] = `"` + c + `"`
		}
		fmt.Fprintf(&b, "create index if not exists \"%s\" on \"%s\" (%s);\n", idx.Name, e.TableName, strings.Join(quoted, ", "))
	}
	b.WriteString("\n")
	return b.String()
}

// ToDMLEntity describes the link table as a dml.Entity (resolved metadata
// only) so it can be read with the generic repository. Like the MikroORM
// entity upstream generates, the primary key is made of the foreign keys.
func (e *LinkEntity) ToDMLEntity(className string) *dml.Entity {
	ent := &dml.Entity{Name: className, TableName: e.TableName}
	ent.Meta.ClassName = className
	ent.Meta.Collection = e.TableName
	for _, c := range e.Columns {
		if c.Primary {
			ent.Meta.PrimaryKey = append(ent.Meta.PrimaryKey, c.Name)
		}
		columnType := c.SQLType
		if columnType == "varchar(255)" {
			columnType = "text"
		}
		propType := "string"
		if c.RuntimeType != "" {
			propType = c.RuntimeType
		}
		ent.Meta.Properties = append(ent.Meta.Properties, dml.MetaProperty{
			Name:       c.Name,
			Kind:       "scalar",
			Type:       propType,
			ColumnType: columnType,
			FieldNames: []string{c.Name},
			Nullable:   c.Nullable,
			Primary:    c.Primary,
		})
		ent.Properties = append(ent.Properties, dml.Property{Name: c.Name, Field: &dml.Field{DataType: dml.DataType{Name: "text"}, Nullable: c.Nullable}})
	}
	return ent
}
