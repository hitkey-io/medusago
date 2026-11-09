// Package utils mirrors @medusajs/fulfillment/src/utils.
package utils

import (
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// AvailableOperators mirrors Object.values(RuleOperator).
var AvailableOperators = []string{"in", "eq", "ne", "gt", "gte", "lt", "lte", "nin"}

func invalid(msg string) error {
	return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, msg)
}

func truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	case bool:
		return t
	case float64:
		return t != 0
	}
	return true
}

// ValidateRule mirrors validateRule.
func ValidateRule(rule *dal.Record) error {
	attribute, _ := rule.Get("attribute")
	operator, _ := rule.Get("operator")
	value, _ := rule.Get("value")
	if !truthy(attribute) || !truthy(operator) || !truthy(value) {
		return invalid("Rule must have an attribute, an operator and a value")
	}
	if _, ok := attribute.(string); !ok {
		return invalid("Rule attribute must be a string")
	}
	op, ok := operator.(string)
	if !ok {
		return invalid("Rule operator must be a string")
	}
	found := false
	for _, o := range AvailableOperators {
		if o == op {
			found = true
		}
	}
	if !found {
		return invalid(fmt.Sprintf("Rule operator %s is not supported. Must be one of %s", op, strings.Join(AvailableOperators, ", ")))
	}
	_, isArray := value.([]any)
	_, isObject := value.(*dal.Record)
	if op == "in" || op == "nin" {
		if !isArray {
			return invalid("Rule value must be an array for in/nin operators")
		}
	} else if isArray || isObject {
		return invalid(fmt.Sprintf("Rule value must be a string, bool, number value for the selected operator %s", op))
	}
	return nil
}

// ValidateAndNormalizeRules mirrors validateAndNormalizeRules (booleans
// become "true"/"false").
func ValidateAndNormalizeRules(rules []*dal.Record) error {
	for _, r := range rules {
		if err := ValidateRule(r); err != nil {
			return err
		}
	}
	for _, r := range rules {
		if v, _ := r.Get("value"); v == true {
			r.Set("value", "true")
		} else if v == false {
			r.Set("value", "false")
		}
	}
	return nil
}
