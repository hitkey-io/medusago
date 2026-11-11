package modules

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dml"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// upstreamJoinerOptions are the defineJoinerConfig options of the modules
// that pass more than their models (joiner-config.ts of each module).
var upstreamJoinerOptions = map[string]func(models func(...string) []*dml.Entity) utilsmodulessdk.DefineJoinerConfigOptions{
	"api_key": func(func(...string) []*dml.Entity) utilsmodulessdk.DefineJoinerConfigOptions {
		return utilsmodulessdk.DefineJoinerConfigOptions{LinkableKeys: joiner.OrderedStringMap{{Key: "publishable_key_id", Value: "ApiKey"}}}
	},
	"auth": func(m func(...string) []*dml.Entity) utilsmodulessdk.DefineJoinerConfigOptions {
		return utilsmodulessdk.DefineJoinerConfigOptions{Models: m("AuthIdentity", "ProviderIdentity")}
	},
	"file": func(func(...string) []*dml.Entity) utilsmodulessdk.DefineJoinerConfigOptions {
		return utilsmodulessdk.DefineJoinerConfigOptions{Models: []*dml.Entity{}, ModelNames: []string{"File"}}
	},
	"fulfillment": func(func(...string) []*dml.Entity) utilsmodulessdk.DefineJoinerConfigOptions {
		return utilsmodulessdk.DefineJoinerConfigOptions{LinkableKeys: joiner.OrderedStringMap{
			{Key: "fulfillment_id", Value: "Fulfillment"},
			{Key: "fulfillment_set_id", Value: "FulfillmentSet"},
			{Key: "shipping_option_id", Value: "ShippingOption"},
			{Key: "shipping_option_rule_id", Value: "ShippingOptionRule"},
			{Key: "fulfillment_provider_id", Value: "FulfillmentProvider"},
		}}
	},
	"inventory": func(func(...string) []*dml.Entity) utilsmodulessdk.DefineJoinerConfigOptions {
		return utilsmodulessdk.DefineJoinerConfigOptions{Alias: []joiner.JoinerServiceConfigAlias{
			{Name: joiner.StringOrStrings{"inventory_items", "inventory_item", "inventory"}, Entity: "InventoryItem", Args: map[string]any{"methodSuffix": "InventoryItems"}},
			{Name: joiner.StringOrStrings{"reservation", "reservations", "reservation_item", "reservation_items"}, Entity: "ReservationItem", Args: map[string]any{"methodSuffix": "ReservationItems"}},
		}}
	},
	"order": func(m func(...string) []*dml.Entity) utilsmodulessdk.DefineJoinerConfigOptions {
		return utilsmodulessdk.DefineJoinerConfigOptions{
			LinkableKeys: joiner.OrderedStringMap{{Key: "claim_id", Value: "OrderClaim"}, {Key: "exchange_id", Value: "OrderExchange"}},
			Models:       m("Order", "OrderAddress", "OrderChange", "OrderClaim", "OrderExchange", "OrderItem", "OrderLineItem", "OrderShippingMethod", "OrderTransaction", "Return", "ReturnReason"),
		}
	},
	"payment": func(func(...string) []*dml.Entity) utilsmodulessdk.DefineJoinerConfigOptions {
		return utilsmodulessdk.DefineJoinerConfigOptions{
			LinkableKeys: joiner.OrderedStringMap{
				{Key: "payment_id", Value: "Payment"},
				{Key: "payment_collection_id", Value: "PaymentCollection"},
				{Key: "payment_provider_id", Value: "PaymentProvider"},
				{Key: "refund_reason_id", Value: "RefundReason"},
				{Key: "account_holder_id", Value: "AccountHolder"},
			},
			Alias: []joiner.JoinerServiceConfigAlias{{Name: joiner.StringOrStrings{"payment_method", "payment_methods"}, Entity: "PaymentMethod", Args: map[string]any{"methodSuffix": "PaymentMethods"}}},
		}
	},
	"pricing": func(m func(...string) []*dml.Entity) utilsmodulessdk.DefineJoinerConfigOptions {
		return utilsmodulessdk.DefineJoinerConfigOptions{Models: m("PriceSet", "PriceList", "Price", "PricePreference")}
	},
	"product": func(m func(...string) []*dml.Entity) utilsmodulessdk.DefineJoinerConfigOptions {
		return utilsmodulessdk.DefineJoinerConfigOptions{
			Models:       m("Product", "ProductVariant", "ProductOption", "ProductOptionValue", "ProductType", "ProductTag", "ProductCollection", "ProductCategory", "ProductImage"),
			LinkableKeys: joiner.OrderedStringMap{{Key: "variant_id", Value: "ProductVariant"}},
			PrimaryKeys:  []string{"id", "handle"},
			Alias:        []joiner.JoinerServiceConfigAlias{{Name: joiner.StringOrStrings{"product_variant", "product_variants", "variant", "variants"}, Entity: "ProductVariant", Args: map[string]any{"methodSuffix": "ProductVariants"}}},
		}
	},
	"search": func(func(...string) []*dml.Entity) utilsmodulessdk.DefineJoinerConfigOptions {
		return utilsmodulessdk.DefineJoinerConfigOptions{Models: []*dml.Entity{}}
	},
	"stock_location": func(func(...string) []*dml.Entity) utilsmodulessdk.DefineJoinerConfigOptions {
		return utilsmodulessdk.DefineJoinerConfigOptions{LinkableKeys: joiner.OrderedStringMap{{Key: "stock_location_id", Value: "StockLocation"}, {Key: "location_id", Value: "StockLocation"}}}
	},
}

// serviceJoinerConfig are the modules whose service declares __joinerConfig
// (returning the module joiner config object).
var serviceJoinerConfig = map[string]bool{
	"api_key": true, "auth": true, "customer": true, "file": true, "fulfillment": true, "inventory": true,
	"order": true, "payment": true, "pricing": true, "product": true, "sales_channel": true, "search": true,
	"stock_location": true,
}

// customSchemaModules use a hand-written schema.ts instead of the generated
// GraphQL schema.
var customSchemaModules = []string{"fulfillment", "inventory", "order", "product", "sales_channel", "stock_location"}

// noServiceModels are the modules whose service is built without model
// objects (MedusaService({})), their joiner config has none.
var noServiceModels = map[string]bool{"index": true, "search": true}

func canonicalJoinerConfig(t *testing.T, c *joiner.ModuleJoinerConfig) string {
	t.Helper()
	type kv struct{ K, V string }
	lk := []kv{}
	for _, x := range c.LinkableKeys {
		lk = append(lk, kv{x.Key, x.Value})
	}
	aliases := []any{}
	for _, a := range c.Alias {
		var internal any
		if len(a.Internal) > 0 {
			var buf bytes.Buffer
			_ = json.Compact(&buf, a.Internal)
			internal = buf.String()
		}
		aliases = append(aliases, map[string]any{"name": []string(a.Name), "entity": a.Entity, "args": a.Args, "internal": internal})
	}
	b, _ := json.MarshalIndent(map[string]any{
		"serviceName":          c.ServiceName,
		"primaryKeys":          c.PrimaryKeys,
		"idPrefixToEntityName": c.IDPrefixToEntityName,
		"linkableKeys":         lk,
		"alias":                aliases,
	}, "", " ")
	return string(b)
}

func TestDefineJoinerConfigMatchesUpstream(t *testing.T) {
	checked := 0
	var customSchemas []string
	for pkg, exp := range Registry {
		if exp.JoinerConfig == nil || exp.JoinerConfig.ServiceName == "" {
			continue
		}
		want := exp.JoinerConfig
		byName := map[string]*dml.Entity{}
		for _, m := range exp.Models {
			byName[m.Name] = m
		}
		pick := func(names ...string) []*dml.Entity {
			out := []*dml.Entity{}
			for _, n := range names {
				if m, ok := byName[n]; ok {
					out = append(out, m)
				} else {
					t.Fatalf("%s: no model %s", pkg, n)
				}
			}
			return out
		}
		var opts utilsmodulessdk.DefineJoinerConfigOptions
		if fn, ok := upstreamJoinerOptions[want.ServiceName]; ok {
			opts = fn(pick)
		}
		if noServiceModels[want.ServiceName] && opts.Models == nil {
			opts.Models = []*dml.Entity{}
		}
		if opts.Models == nil {
			// the models in upstream order, as revealed by their default
			// linkable keys; the module's other models after them
			// (the service model objects are the models with an alias)
			var aliased []string
			for _, a := range want.Alias {
				if _, ok := byName[a.Entity]; ok {
					aliased = append(aliased, a.Entity)
				}
			}
			var names []string
			for _, kv := range want.LinkableKeys {
				if containsName(aliased, kv.Value) && kv.Key == strings.ToLower(common.CamelToSnakeCase(kv.Value))+"_id" && !containsName(names, kv.Value) {
					names = append(names, kv.Value)
				}
			}
			for _, n := range aliased {
				if !containsName(names, n) {
					names = append(names, n)
				}
			}
			opts.Models = pick(names...)
		}
		opts.Entities = exp.Models
		if serviceJoinerConfig[want.ServiceName] {
			// Module() assigns the prefixes of the service model objects to
			// the joiner config object the service returns.
			// (the service model objects differ per module: use the result)
			opts.IDPrefixToEntityName = want.IDPrefixToEntityName
		}
		got := utilsmodulessdk.DefineJoinerConfig(want.ServiceName, opts)
		if got.Schema != want.Schema {
			customSchemas = append(customSchemas, want.ServiceName)
		}
		if a, b := canonicalJoinerConfig(t, want), canonicalJoinerConfig(t, got); a != b {
			t.Errorf("%s: joiner config differs\nwant: %s\ngot:  %s", pkg, a, b)
		}
		checked++
	}
	sort.Strings(customSchemas)
	if strings.Join(customSchemas, ",") != strings.Join(customSchemaModules, ",") {
		t.Errorf("generated schema differs for %v, want only %v", customSchemas, customSchemaModules)
	}
	if checked < 20 {
		t.Fatalf("only %d joiner configs checked", checked)
	}
}

func containsName(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
