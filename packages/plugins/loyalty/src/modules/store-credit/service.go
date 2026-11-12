package storecredit

import (
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// StoreCreditModuleService mirrors service.ts (StoreCreditService):
// MedusaService({ StoreCreditAccount, AccountTransaction }) with the
// account balance methods.
type StoreCreditModuleService struct {
	*utilsmodulessdk.MedusaService
}
