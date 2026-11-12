// Package workflows mirrors src/workflows/store-credit/workflows.
package workflows

import (
	"fmt"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

func invalidData(msg string) error {
	return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, msg)
}

func get(r *dal.Record, key string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(key)
	return v
}

// first mirrors <query>.data[0] (nil when empty).
func first(res commonsteps.QueryGraphResult) *dal.Record {
	switch t := res.Data.(type) {
	case []any:
		if len(t) > 0 {
			r, _ := t[0].(*dal.Record)
			return r
		}
	case []*dal.Record:
		if len(t) > 0 {
			return t[0]
		}
	}
	return nil
}

// undefinedProperty is the TypeError of reading a property of undefined.
func undefinedProperty(key string) error {
	return fmt.Errorf("Cannot read properties of undefined (reading '%s')", key)
}

// orEmpty mirrors value || "".
func orEmpty(v string) string { return v }
