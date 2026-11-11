package dml

import (
	"regexp"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
)

// graphqlTypes mirrors GRAPHQL_TYPES of graphql-builder/get-attribute.ts.
var graphqlTypes = map[string]string{
	"boolean":   "Boolean",
	"dateTime":  "DateTime",
	"number":    "Int",
	"bigNumber": "Float",
	"float":     "Float",
	"text":      "String",
	"json":      "JSON",
	"array":     "[String]",
	"id":        "ID",
}

var enumValueRe = regexp.MustCompile(`(?i)[^a-z0-9_]`)

// graphqlAttribute mirrors getGraphQLAttributeFromDMLPropety.
func graphqlAttribute(modelName, propertyName string, f *Field) (enum, attribute string) {
	fieldType := f.DataType.Name
	if f.PrimaryKey {
		fieldType = "id"
	}
	var typ string
	switch fieldType {
	case "enum":
		enumName := common.ToPascalCase(modelName + "_" + propertyName + "Enum")
		var values []string
		if f.DataType.Options != nil {
			for _, c := range f.DataType.Options.Choices {
				value := jsString(c)
				values = append(values, "  "+strings.ToUpper(enumValueRe.ReplaceAllString(value, "_"))+` @enumValue(value: "`+value+`")`)
			}
		}
		enum = "enum " + enumName + " {\n" + strings.Join(values, "\n") + "\n}"
		typ = enumName
	case "id":
		typ = graphqlTypes["id"]
	default:
		var ok bool
		if typ, ok = graphqlTypes[fieldType]; !ok {
			typ = "String"
		}
	}
	if !f.Nullable {
		typ += "!"
	}
	return enum, propertyName + ": " + typ
}

// GenerateGraphQLFromEntity mirrors generateGraphQLFromEntity of
// create-graphql.ts.
func GenerateGraphQLFromEntity(e *Entity, entities map[string]*Entity) string {
	modelName := parseEntityName(e).modelName
	var extra, gql []string
	for i := range e.Properties {
		p := &e.Properties[i]
		if p.Field != nil {
			enum, attr := graphqlAttribute(modelName, p.Name, p.Field)
			if enum != "" {
				extra = append(extra, enum)
			}
			gql = append(gql, attr)
			continue
		}
		rel := p.Relation
		if rel.Type == "belongsTo" || rel.Type == "hasOneWithFK" {
			_, attr := graphqlAttribute(modelName, p.Name, &Field{DataType: DataType{Name: "text"}, Nullable: rel.Nullable})
			// the attribute is named after the foreign key
			attr = foreignKeyName(p) + attr[len(p.Name):]
			gql = append(gql, attr)
		}
		related := common.UpperCaseFirst(common.ToCamelCase(rel.Entity))
		if t, ok := entities[related]; ok {
			related = parseEntityName(t).modelName
		}
		if rel.Options.MappedBy != "" && rel.Type == "hasOne" {
			extra = append(extra, "extend type "+related+" {\n  "+rel.Options.MappedBy+": "+modelName+"!\n}")
		}
		isArray := rel.Type == "hasMany" || rel.Type == "manyToMany"
		attr := p.Name + ": "
		if isArray {
			attr += "[" + related + "]"
		} else {
			attr += related
		}
		if !rel.Nullable {
			attr += "!"
		}
		gql = append(gql, attr)
	}
	return "\n      " + strings.Join(extra, "\n") + "\n      type " + modelName + " {\n        " + strings.Join(gql, "\n") + "\n      }\n    "
}

// ToGraphQLSchema mirrors toGraphQLSchema of create-graphql.ts.
func ToGraphQLSchema(entities []*Entity) string {
	byName := map[string]*Entity{}
	for _, e := range entities {
		byName[parseEntityName(e).modelName] = e
	}
	var schemas []string
	for _, e := range entities {
		schemas = append(schemas, GenerateGraphQLFromEntity(e, byName))
	}
	prefix := ""
	if len(schemas) > 0 {
		prefix = "\n    scalar DateTime\n    scalar JSON\n    directive @enumValue(value: String) on ENUM_VALUE\n  "
	}
	return prefix + strings.Join(schemas, "\n")
}
