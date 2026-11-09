// Package utils mirrors api/admin/promotions/utils.
package utils

import (
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

func rec(kv ...any) *dal.Record {
	r := dal.NewRecord()
	for i := 0; i+1 < len(kv); i += 2 {
		r.Set(kv[i].(string), kv[i+1])
	}
	return r
}

// OperatorsMap mirrors operatorsMap.
var OperatorsMap = map[string]*dal.Record{
	"in": rec("id", "in", "value", "in", "label", "In"),
	"eq": rec("id", "eq", "value", "eq", "label", "Equals"),
	"ne": rec("id", "ne", "value", "ne", "label", "Not In"),
}

func allOperators() []any {
	return []any{OperatorsMap["in"], OperatorsMap["eq"], OperatorsMap["ne"]}
}

func attribute(id, value, label string) *dal.Record {
	return rec("id", id, "value", value, "label", label, "required", false, "field_type", "multiselect", "operators", allOperators())
}

func ruleAttributes() []*dal.Record {
	return []*dal.Record{
		attribute("customer_group", "customer.groups.id", "Customer Group"),
		attribute("region", "region.id", "Region"),
		attribute("country", "shipping_address.country_code", "Country"),
		attribute("sales_channel", "sales_channel_id", "Sales Channel"),
	}
}

func itemsAttributes() []*dal.Record {
	return []*dal.Record{
		attribute("product", "items.product.id", "Product"),
		attribute("product_category", "items.product.categories.id", "Product Category"),
		attribute("product_collection", "items.product.collection_id", "Product Collection"),
		attribute("product_type", "items.product.type_id", "Product Type"),
		attribute("product_tag", "items.product.tags.id", "Product Tag"),
	}
}

func shippingMethodsAttributes() []*dal.Record {
	return []*dal.Record{attribute("shipping_option_type", "shipping_methods.shipping_option.shipping_option_type_id", "Shipping Option Type")}
}

func currencyRule(required bool) *dal.Record {
	return rec("id", "currency_code", "value", "currency_code", "label", "Currency Code", "field_type", "select",
		"required", required, "disguised", true, "hydrate", true, "operators", []any{OperatorsMap["eq"]})
}

func numberRule(id, label string) *dal.Record {
	return rec("id", id, "value", id, "label", label, "field_type", "number", "required", true, "disguised", true, "operators", []any{OperatorsMap["eq"]})
}

// RuleAttributesMap mirrors getRuleAttributesMap.
func RuleAttributesMap(promotionType, methodType, targetType any) map[string][]*dal.Record {
	m := map[string][]*dal.Record{"rules": ruleAttributes()}
	if targetType == "shipping_methods" {
		m["target-rules"] = shippingMethodsAttributes()
		m["buy-rules"] = shippingMethodsAttributes()
	} else {
		m["target-rules"] = itemsAttributes()
		m["buy-rules"] = itemsAttributes()
	}
	m["rules"] = append(m["rules"], currencyRule(methodType == "fixed"))
	if promotionType == "buyget" {
		m["buy-rules"] = append(m["buy-rules"], numberRule("buy_rules_min_quantity", "Minimum quantity of items"))
		m["target-rules"] = append(m["target-rules"], numberRule("apply_to_quantity", "Quantity of items promotion will apply to"))
	}
	return m
}

// QueryConfiguration mirrors one entry of ruleQueryConfigurations.
type QueryConfiguration struct {
	EntryPoint, LabelAttr, ValueAttr string
}

// RuleQueryConfigurations mirrors ruleQueryConfigurations.
var RuleQueryConfigurations = map[string]QueryConfiguration{
	"region":               {"region", "name", "id"},
	"currency_code":        {"currency", "name", "code"},
	"customer_group":       {"customer_group", "name", "id"},
	"sales_channel":        {"sales_channel", "name", "id"},
	"country":              {"country", "display_name", "iso_2"},
	"product":              {"product", "title", "id"},
	"product_category":     {"product_category", "name", "id"},
	"product_collection":   {"product_collection", "title", "id"},
	"product_type":         {"product_type", "value", "id"},
	"product_tag":          {"product_tag", "value", "id"},
	"shipping_option_type": {"shipping_option_type", "label", "id"},
}

// ValidateRuleType mirrors validateRuleType.
func ValidateRuleType(ruleType string) error {
	switch strings.Join(strings.Split(ruleType, "-"), "_") {
	case "rules", "target_rules", "buy_rules":
		return nil
	}
	return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Invalid param rule_type (%s)", ruleType))
}

// ValidateRuleAttribute mirrors validateRuleAttribute.
func ValidateRuleAttribute(ruleType, ruleAttributeID string, promotionType, methodType, targetType any) error {
	for _, a := range RuleAttributesMap(promotionType, methodType, targetType)[ruleType] {
		if v, _ := a.Get("id"); v == ruleAttributeID {
			return nil
		}
	}
	return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Invalid rule attribute - %s", ruleAttributeID))
}
