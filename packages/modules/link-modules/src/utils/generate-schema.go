package utils

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/link"
)

// ModuleConfigs resolves the joiner config of a loaded module by service
// name (MedusaModule.getJoinerConfig / getModuleInstance).
type ModuleConfigs func(serviceName string) *joiner.ModuleJoinerConfig

func linkableKeyOf(cfg *joiner.ModuleJoinerConfig, key string) string {
	if cfg == nil {
		return ""
	}
	for _, kv := range cfg.LinkableKeys {
		if kv.Key == key {
			return kv.Value
		}
	}
	return ""
}

// LinkEntityName returns the entity name of a link table:
// toPascalCase("Link_" + table name).
func LinkEntityName(cfg *joiner.ModuleJoinerConfig, primary, foreign joiner.JoinerRelationship) string {
	table := ""
	if cfg.DatabaseConfig != nil {
		table = cfg.DatabaseConfig.TableName
	}
	if table == "" {
		table = link.ComposeTableName(primary.ServiceName, primary.ForeignKey, foreign.ServiceName, foreign.ForeignKey)
	}
	return common.ToPascalCase("Link_" + table)
}

// GenerateGraphQLSchema mirrors generateGraphQLSchema of generate-schema.ts.
func GenerateGraphQLSchema(cfg *joiner.ModuleJoinerConfig, primary, foreign *joiner.JoinerRelationship, modules ModuleConfigs, logger *slog.Logger) (string, error) {
	if logger == nil {
		logger = slog.Default()
	}
	var fieldNames []string
	entityName := ""
	readOnly := cfg.IsReadOnlyLink
	if !readOnly {
		fieldNames = append(strings.Split(primary.ForeignKey, ","), foreign.ForeignKey)
		entityName = LinkEntityName(cfg, *primary, *foreign)
	}

	var typeDef strings.Builder
	for _, extend := range cfg.Extends {
		extendedModule := modules(extend.ServiceName)
		if extendedModule == nil {
			return "", fmt.Errorf("Module %s not found. Please verify that the module is configured and installed, also the module must be loaded before the link modules.", extend.ServiceName)
		}
		extendedEntityName := linkableKeyOf(extendedModule, extend.Relationship.PrimaryKey)
		if !readOnly && (primary == nil || foreign == nil || extendedEntityName == "") {
			logger.Warn(fmt.Sprintf("Link modules schema: No linkable key found for %s on module %s.", extend.Relationship.PrimaryKey, extend.ServiceName))
			continue
		}
		fieldName := common.CamelToSnakeCase(common.LowerCaseFirst(extend.Relationship.Alias))
		typ := entityName
		if extend.Relationship.IsList {
			typ = "[" + entityName + "]"
		}
		if readOnly {
			if extend.Entity == "" {
				continue
			}
			rel := extend.Relationship
			relModule := modules(rel.ServiceName)
			hasSchema := relModule != nil && relModule.Schema != ""
			relEntity := rel.Entity
			if relEntity == "" {
				relEntity = linkableKeyOf(relModule, rel.PrimaryKey)
			}
			if relEntity == "" || !hasSchema {
				continue
			}
			typ = relEntity
			if rel.IsList {
				typ = "[" + relEntity + "]"
			}
			extendedEntityName = extend.Entity
		}

		var aliases []string
		// Object.entries(extend.fieldAlias): declaration order
		for _, name := range extend.FieldAliasNames() {
			alias := extend.FieldAlias[name]
			// config.isList ?? relationship.isList (an absent isList and
			// false are not distinguished)
			isList := alias.IsList || extend.Relationship.IsList
			parts := strings.Split(alias.Path, ".")
			target := parts[len(parts)-1]
			var targetRel *joiner.JoinerRelationship
			for i := range cfg.Relationships {
				if cfg.Relationships[i].Alias == target {
					targetRel = &cfg.Relationships[i]
					break
				}
			}
			if targetRel == nil {
				continue
			}
			targetEntity := linkableKeyOf(modules(targetRel.ServiceName), targetRel.ForeignKey)
			if targetEntity == "" {
				logger.Warn(fmt.Sprintf("Link modules schema: No linkable key found for %s on module %s.", targetRel.ForeignKey, targetRel.ServiceName))
				continue
			}
			t := targetEntity
			if isList {
				t = "[" + targetEntity + "]"
			}
			aliases = append(aliases, name+": "+t)
		}
		fmt.Fprintf(&typeDef, "    \n      extend type %s {\n        %s: %s\n        \n        %s\n      }\n    ", extendedEntityName, fieldName, typ, strings.Join(aliases, "\n"))
	}
	if readOnly {
		return typeDef.String(), nil
	}

	type field struct {
		name, typ string
		nullable  bool
	}
	var fields []field
	for _, f := range fieldNames {
		fields = append(fields, field{f, "String", false})
	}
	if cfg.DatabaseConfig != nil {
		for _, extra := range cfg.DatabaseConfig.ExtraFields {
			replaced := false
			for i := range fields {
				if fields[i].name == extra.Name {
					fields[i] = field{extra.Name, graphQLType(extra.Type), extra.Nullable}
					replaced = true
				}
			}
			if !replaced {
				fields = append(fields, field{extra.Name, graphQLType(extra.Type), extra.Nullable})
			}
		}
	}
	primaryField, foreignField := "", ""
	if m := modules(primary.ServiceName); m != nil && m.Schema != "" {
		e := primary.Entity
		if e == "" {
			e = link.ComposeTableName(primary.ServiceName)
		}
		primaryField = common.CamelToSnakeCase(primary.Alias) + ": " + common.ToPascalCase(e)
	}
	if m := modules(foreign.ServiceName); m != nil && m.Schema != "" {
		e := foreign.Entity
		if e == "" {
			e = link.ComposeTableName(foreign.ServiceName)
		}
		foreignField = common.CamelToSnakeCase(foreign.Alias) + ": " + common.ToPascalCase(e)
	}
	var lines []string
	for _, f := range fields {
		t := f.typ
		if !f.nullable {
			t += "!"
		}
		lines = append(lines, f.name+": "+t)
	}
	fmt.Fprintf(&typeDef, "\n    type %s {\n      %s\n        \n      %s\n      %s\n      \n      createdAt: String!\n      updatedAt: String!\n      deletedAt: String\n    }\n  ", entityName, strings.Join(lines, "\n      "), primaryField, foreignField)
	return typeDef.String(), nil
}

// graphQLType mirrors getGraphQLType of generate-schema.ts.
func graphQLType(t string) string {
	switch t {
	case "numeric", "float", "double", "decimal":
		return "Float"
	case "integer", "smallint", "tinyint", "mediumint":
		return "Int"
	case "boolean":
		return "Boolean"
	case "string", "text", "enum":
		return "String"
	case "uuid":
		return "ID"
	case "date":
		return "Date"
	case "time":
		return "Time"
	case "datetime":
		return "DateTime"
	case "bigint":
		return "BigInt"
	case "blob":
		return "Blob"
	case "uint8array":
		return "[Int]"
	case "array", "enumArray":
		return "[String]"
	case "json", "jsonb":
		return "JSON"
	}
	return "String"
}
