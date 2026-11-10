package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
)

func equalBN(a, b any) bool { return totals.Eq(a, b) }

func errNotSupported(what string) error {
	return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, what+" is not supported yet")
}
