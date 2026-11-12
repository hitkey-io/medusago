package loyalty

import (
	"context"

	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// LoyaltyModuleService mirrors service.ts: MedusaService({ GiftCard }) with
// the plugin options.
type LoyaltyModuleService struct {
	*utilsmodulessdk.MedusaService
	options map[string]any
}

// GetOptions mirrors getOptions: the plugin options.
func (s *LoyaltyModuleService) GetOptions(_ context.Context) (map[string]any, error) {
	return s.options, nil
}
