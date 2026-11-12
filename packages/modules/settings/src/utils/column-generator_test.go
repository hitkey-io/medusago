package utils

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

func TestFormatFieldName(t *testing.T) {
	cases := map[string]string{
		"display_id":        "Display Id",
		"sales_channel":     "Sales Channel",
		"createdAt":         "Created At",
		"URL":               "U R L",
		"_internal":         "Internal",
		"raw__double":       "Raw  Double",
		"custom_display_id": "Custom Display Id",
	}
	for in, want := range cases {
		if got := formatFieldName(in); got != want {
			t.Errorf("formatFieldName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestInferRenderModeAndDataType(t *testing.T) {
	str := NamedType{Name: "String", Kind: "ScalarTypeDefinition"}
	enum := NamedType{Name: "Status", Kind: "EnumTypeDefinition"}
	cases := []struct {
		field                    string
		typ                      NamedType
		mode, semantic, dataType string
	}{
		{"created_at", str, "datetime", "timestamp", "date"},
		{"total", NamedType{Name: "Float", Kind: "ScalarTypeDefinition"}, "currency", "currency", "currency"},
		{"status", enum, "status", "status", "enum"},
		{"kind", enum, "status", "enum", "enum"},
		{"is_active", NamedType{Name: "Boolean", Kind: "ScalarTypeDefinition"}, "boolean", "boolean", "boolean"},
		{"metadata", NamedType{Name: "JSON", Kind: "ScalarTypeDefinition"}, "json", "metadata", "object"},
		{"settings", NamedType{Name: "JSON", Kind: "ScalarTypeDefinition"}, "json", "json", "object"},
		{"rank", NamedType{Name: "Int", Kind: "ScalarTypeDefinition"}, "number", "int", "number"},
		{"title", str, "text", "string", "string"},
	}
	for _, c := range cases {
		mode, semantic := InferRenderMode(c.field, c.typ)
		if mode != c.mode || semantic != c.semantic {
			t.Errorf("InferRenderMode(%s) = %s/%s, want %s/%s", c.field, mode, semantic, c.mode, c.semantic)
		}
		if dt := InferDataType(c.typ, c.field); dt != c.dataType {
			t.Errorf("InferDataType(%s) = %s, want %s", c.field, dt, c.dataType)
		}
	}
}

const testSchema = `
directive @enumValue(value: String) on ENUM_VALUE

enum WidgetStatusEnum {
  DRAFT @enumValue(value: "draft")
  LIVE @enumValue(value: "live")
}

type Maker {
  id: ID!
  title: String
  code: String
  widgets: [Widget]
}

type Part {
  id: ID!
  label: String
}

type Widget {
  id: ID!
  name: String!
  status: WidgetStatusEnum!
  maker_id: String
  maker: Maker
  parts: [Part]
  raw_amount: JSON
  metadata: JSON
  created_at: DateTime!
}

type WidgetDetail {
  id: ID!
  parts: [Part]
  score: Float
}
`

func TestGenerateEntityColumns(t *testing.T) {
	ResetEntityOverrideRegistry()
	t.Cleanup(ResetEntityOverrideRegistry)
	GetEntityOverrideRegistry().Register("Widget", EntityOverride{
		AdditionalTypes:      []string{"WidgetDetail"},
		DefaultVisibleFields: []string{"name"},
		DefaultFieldOrdering: map[string]float64{"name": 100},
	})

	discovery := &EntityDiscoveryService{}
	discovery.Initialize([]*joiner.ModuleJoinerConfig{{ServiceName: "widget", Schema: testSchema}})

	description := "The widget name"
	columns := GenerateEntityColumns(discovery, "widgets", map[string]PropertyLabel{
		"name": {ID: "plbl_1", Entity: "Widget", Property: "name", Label: "Title", Description: &description},
	}, nil)
	if columns == nil {
		t.Fatal("no columns")
	}
	var ids []string
	byID := map[string]*dal.Record{}
	for _, c := range columns {
		id, _ := c.Get("id")
		ids = append(ids, id.(string))
		byID[id.(string)] = c
	}
	// name first (ordering 100), scalars (900), the maker relationship
	// scalars (950), array filters (980; parts twice, from Widget and
	// WidgetDetail). maker_id is covered by maker.id, raw_ fields excluded.
	want := "name,id,status,metadata,created_at,score,maker.id,maker.title,maker.code,parts_filter,parts_filter"
	if got := strings.Join(ids, ","); got != want {
		t.Fatalf("columns = %s\nwant      %s", got, want)
	}

	encode := func(id string) string {
		b, err := json.Marshal(byID[id])
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	if got := encode("name"); got != `{"id":"name","name":"Title","description":"The widget name","field":"name","sortable":true,"hideable":true,"default_visible":true,"data_type":"string","semantic_type":"string","context":"both","render_mode":"text","default_order":100,"category":"field","filter":{"enabled":true,"operators":["eq","ne","contains","startsWith","endsWith","in","nin"]},"source":{"module":"widget","entity":"Widget"},"custom_label":true,"label_id":"plbl_1"}` {
		t.Errorf("name = %s", got)
	}
	if got := encode("status"); !strings.Contains(got, `"enumValues":["draft","live"]`) {
		t.Errorf("status = %s", got)
	}
	if got := encode("maker.id"); !strings.Contains(got, `"relationship":{"entity":"Maker","value_field":"id","display_field":"title","multiple":false,"endpoint":"/admin/makers"}`) {
		t.Errorf("maker.id = %s", got)
	}
	if got := encode("metadata"); !strings.Contains(got, `"sortable":false`) || !strings.Contains(got, `"filter":{"enabled":false}`) {
		t.Errorf("metadata = %s", got)
	}

	if GenerateEntityColumns(discovery, "unknown", nil, nil) != nil {
		t.Error("columns of an unknown entity")
	}
	if GenerateEntityColumns(&EntityDiscoveryService{}, "widgets", nil, nil) != nil {
		t.Error("columns without initialisation")
	}
}
