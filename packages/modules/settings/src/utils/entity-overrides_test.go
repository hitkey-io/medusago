package utils

import (
	"reflect"
	"testing"
)

// Port of the EntityOverrideRegistry cases of
// integration-tests/__tests__/entity-discovery.spec.ts.

func withFreshRegistry(t *testing.T) *EntityOverrideRegistry {
	t.Helper()
	ResetEntityOverrideRegistry()
	t.Cleanup(ResetEntityOverrideRegistry)
	return GetEntityOverrideRegistry()
}

func TestEntityOverrideRegistryInitialization(t *testing.T) {
	registry := withFreshRegistry(t)
	names := registry.GetEntityNames()
	for _, want := range []string{"Order", "Product", "Customer", "User", "Region", "SalesChannel"} {
		if !includes(names, want) {
			t.Errorf("entity names lack %s", want)
		}
	}
	if GetEntityOverrideRegistry() != registry {
		t.Error("registry is not a singleton")
	}
	registry.Register("TestEntity", EntityOverride{ExcludeFields: []string{"test"}})
	ResetEntityOverrideRegistry()
	if GetEntityOverrideRegistry().Has("TestEntity") {
		t.Error("reset kept TestEntity")
	}
}

func TestEntityOverrideRegistryRegister(t *testing.T) {
	registry := withFreshRegistry(t)

	registry.Register("CustomEntity", EntityOverride{
		ExcludeFields:        []string{"internal_field"},
		ExcludePrefixes:      []string{"_"},
		DefaultVisibleFields: []string{"name", "status"},
		DefaultFieldOrdering: map[string]float64{"name": 100, "status": 200},
	})
	got := registry.Get("CustomEntity")
	if got == nil || !includes(got.ExcludeFields, "internal_field") || !includes(got.ExcludePrefixes, "_") ||
		!reflect.DeepEqual(got.DefaultVisibleFields, []string{"name", "status"}) {
		t.Fatalf("CustomEntity = %+v", got)
	}

	registry.Register("Order", EntityOverride{
		ExcludeFields:        []string{"new_excluded_field"},
		DefaultFieldOrdering: map[string]float64{"new_field": 50},
	})
	merged := registry.Get("Order")
	if !includes(merged.ExcludeFields, "order_change") || !includes(merged.ExcludeFields, "new_excluded_field") {
		t.Errorf("excludeFields = %v", merged.ExcludeFields)
	}
	if merged.DefaultFieldOrdering["display_id"] != 100 || merged.DefaultFieldOrdering["new_field"] != 50 {
		t.Errorf("defaultFieldOrdering = %v", merged.DefaultFieldOrdering)
	}

	registry.Register("Order", EntityOverride{ExcludePrefixes: []string{"custom_"}})
	if got := registry.Get("Order").ExcludePrefixes; !reflect.DeepEqual(got, []string{"custom_"}) {
		t.Errorf("excludePrefixes = %v", got)
	}

	registry.Register("Order", EntityOverride{AdditionalTypes: []string{"OrderSummary"}})
	if got := registry.Get("Order").AdditionalTypes; !includes(got, "OrderDetail") || !includes(got, "OrderSummary") {
		t.Errorf("additionalTypes = %v", got)
	}

	registry.Register("Order", EntityOverride{NonFilterableFields: []string{"custom_computed_field"}})
	got = registry.Get("Order")
	for _, f := range []string{"payment_status", "fulfillment_status", "custom_computed_field"} {
		if !includes(got.NonFilterableFields, f) {
			t.Errorf("nonFilterableFields lack %s: %v", f, got.NonFilterableFields)
		}
	}
}

func TestEntityOverrideRegistryLookups(t *testing.T) {
	registry := withFreshRegistry(t)
	if o := registry.Get("Order"); o == nil || !includes(o.ExcludeFields, "order_change") {
		t.Errorf("Order override = %+v", o)
	}
	if registry.Get("NonExistentEntity") != nil || registry.Has("NonExistentEntity") {
		t.Error("NonExistentEntity has an override")
	}
	if !registry.Has("Order") || !registry.Has("Product") {
		t.Error("built-in overrides missing")
	}
	names := registry.GetEntityNames()
	for _, builtin := range BuiltinEntityOverrideNames() {
		if !includes(names, builtin) {
			t.Errorf("entity names lack %s", builtin)
		}
	}
	registry.Register("NewCustomEntity", EntityOverride{DefaultVisibleFields: []string{"field1"}})
	if !includes(registry.GetEntityNames(), "NewCustomEntity") {
		t.Error("NewCustomEntity not listed")
	}
	registry.Register("AnotherCustomEntity", EntityOverride{ExcludeFields: []string{"secret"}})
	all := registry.GetAll()
	if all["Order"] == nil || all["Product"] == nil || !includes(all["AnotherCustomEntity"].ExcludeFields, "secret") {
		t.Errorf("getAll = %v", all)
	}
}

func TestEntityOverrideHelpers(t *testing.T) {
	registry := withFreshRegistry(t)

	registry.Register("TestEntity", EntityOverride{ExcludeFields: []string{"secret_field"}, ExcludePrefixes: []string{"internal_"}})
	rules := GetFieldFilterRules("TestEntity", nil)
	if !includes(rules.ExcludeFields, "secret_field") || !includes(rules.ExcludePrefixes, "internal_") {
		t.Errorf("filter rules = %+v", rules)
	}
	// excludeSuffixes falls back to the default
	if !reflect.DeepEqual(rules.ExcludeSuffixes, []string{"_link"}) {
		t.Errorf("excludeSuffixes = %v", rules.ExcludeSuffixes)
	}

	registry.Register("VisibilityTestEntity", EntityOverride{DefaultVisibleFields: []string{"name", "status", "created_at"}})
	if got := GetDefaultVisibleFields("VisibilityTestEntity", nil); !reflect.DeepEqual(got, []string{"name", "status", "created_at"}) {
		t.Errorf("visible fields = %v", got)
	}

	registry.Register("OrderingTestEntity", EntityOverride{DefaultFieldOrdering: map[string]float64{"name": 100, "status": 200, "created_at": 300}})
	ordering := GetFieldOrdering("OrderingTestEntity", nil)
	if ordering["name"] != 100 || ordering["status"] != 200 || ordering["created_at"] != 300 {
		t.Errorf("ordering = %v", ordering)
	}

	registry.Register("TypesTestEntity", EntityOverride{AdditionalTypes: []string{"TypesTestEntityDetail", "TypesTestEntitySummary"}})
	if got := GetAdditionalTypes("TypesTestEntity", nil); !includes(got, "TypesTestEntityDetail") || !includes(got, "TypesTestEntitySummary") {
		t.Errorf("additional types = %v", got)
	}
}

func TestSpreadEntityOverride(t *testing.T) {
	registry := withFreshRegistry(t)
	option := map[string]any{
		"excludeFields":        []any{"email"},
		"defaultFieldOrdering": map[string]any{"display_id": 1},
		"computedColumns": []any{map[string]any{
			"id": "custom", "name": "Custom", "renderMode": "text", "requiredFields": []any{"id"},
		}},
	}
	merged, computed := SpreadEntityOverride(registry.Get("Order"), option)
	registry.Register("Order", merged)
	got := registry.Get("Order")
	if !includes(got.ExcludeFields, "email") || !includes(got.ExcludeFields, "order_change") {
		t.Errorf("excludeFields = %v", got.ExcludeFields)
	}
	if got.DefaultFieldOrdering["display_id"] != 1 {
		t.Errorf("defaultFieldOrdering = %v", got.DefaultFieldOrdering)
	}
	if len(computed) != 1 || computed[0].ID != "custom" || !reflect.DeepEqual(computed[0].RequiredFields, []string{"id"}) {
		t.Errorf("computed = %+v", computed)
	}
}
