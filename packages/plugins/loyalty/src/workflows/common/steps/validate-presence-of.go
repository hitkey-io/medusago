// Package steps mirrors src/workflows/common/steps.
package steps

import (
	"fmt"
	"reflect"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// ValidatePresenceOfStepInput mirrors ValidatePresenceOfStepInput.
type ValidatePresenceOfStepInput struct {
	Entity     any    `json:"entity"`
	EntityName string `json:"entityName"`
	Identifier string `json:"identifier"`
}

func missing(v any) bool {
	if v == nil {
		return true
	}
	switch t := v.(type) {
	case string:
		return t == ""
	case bool:
		return !t
	case float64:
		return t == 0
	}
	rv := reflect.ValueOf(v)
	return (rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Map || rv.Kind() == reflect.Slice) && rv.IsNil()
}

// ValidatePresenceOfStep mirrors validatePresenceOfStep
// ("validate-presence-of"): a falsy entity is NOT_FOUND.
var ValidatePresenceOfStep = composer.CreateStep("validate-presence-of",
	func(ctx composer.StepExecutionContext, in ValidatePresenceOfStepInput) (composer.StepResponse[any], error) {
		if missing(in.Entity) {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("%s %s not found", in.EntityName, in.Identifier))
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)
