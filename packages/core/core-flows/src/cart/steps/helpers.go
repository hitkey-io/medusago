package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
)

func multiply(a, b any) *totals.BN { return totals.Mult(a, b) }

func notAllowedInventory() error {
	return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Some variant does not have the required inventory", common.MedusaErrorCodes.INSUFFICIENT_INVENTORY)
}
