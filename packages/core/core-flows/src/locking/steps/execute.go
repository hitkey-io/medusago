package steps

import (
	"context"
	"fmt"

	lockingtypes "github.com/hitkey-io/medusago/packages/core/types/src/locking"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// UniqueKeys mirrors Array.from(new Set(keys)) over string keys (first
// occurrence order; non-string values are formatted like String(v)).
func UniqueKeys(keys []any) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, k := range keys {
		var s string
		switch v := k.(type) {
		case string:
			s = v
		case nil:
			s = "undefined"
		default:
			s = fmt.Sprint(v)
		}
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// Execute runs fn with the keys held (locking.execute(keys, job) with the
// default provider and options), as the reservation and inventory steps do.
func Execute(ctx context.Context, c composer.Container, keys []string, fn func(ctx context.Context) error) error {
	locking, err := LockingModule(c)
	if err != nil {
		return err
	}
	_, err = locking.Execute(ctx, keys, func(ctx context.Context) (any, error) {
		return nil, fn(ctx)
	}, lockingtypes.ExecuteArgs{})
	return err
}
