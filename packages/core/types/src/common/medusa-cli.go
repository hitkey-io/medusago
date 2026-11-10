// Package common mirrors @medusajs/types/src/common.
package common

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/framework/src/container"
)

// ExecArgs mirrors ExecArgs: what `medusa exec` passes to a script.
type ExecArgs struct {
	Container *container.MedusaContainer
	Args      []string
}

// ExecScript is a script run by `medusa exec` (the default export of a script
// file upstream).
type ExecScript func(ctx context.Context, args ExecArgs) error
