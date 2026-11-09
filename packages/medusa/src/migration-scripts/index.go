// Package migrationscripts mirrors @medusajs/medusa/src/migration-scripts.
package migrationscripts

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/packages/core/framework/src/migrations"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/medusa/src/modules"
)

// Context is what the scripts receive (ExecArgs upstream).
type Context struct {
	DB     *pgxpool.Pool
	Config config.ConfigModule
	Logger *slog.Logger
}

// isInstalled mirrors MedusaModule.isInstalled.
func (c Context) isInstalled(key string) bool {
	for _, m := range c.Config.Modules {
		if m.Key == key && !m.Disable {
			return true
		}
	}
	return false
}

// service returns the generic service of an installed module.
func (c Context) service(key string) *utilsmodulessdk.MedusaService {
	for _, m := range c.Config.Modules {
		if m.Key != key {
			continue
		}
		if exp, ok := modules.Registry[m.Resolve]; ok {
			return utilsmodulessdk.NewMedusaService(c.DB, exp.Models, exp.JoinerConfig)
		}
	}
	return nil
}

// Scripts lists the migration scripts in upstream order (file names sorted).
// The RBAC scripts are Medusa Enterprise Edition materials and not ported.
func Scripts(c Context) []migrations.Script {
	return []migrations.Script{
		{Name: "backfill-refund-status.js", Run: c.backfillRefundStatus},
		{Name: "migrate-normalize-currency-codes-normalization.js", Run: c.migrateNormalizeCurrencyCodes},
		{Name: "migrate-product-option-link-ids.js", Run: c.migrateProductOptionLinkIDs},
		{Name: "migrate-product-shipping-profile.js", Run: c.assignProductsToShippingProfile},
		{Name: "migrate-tax-region-provider.js", Run: c.assignTaxSystemProviderToTaxRegions},
		{Name: "reconcile-inventory-reserved-quantity.js", Run: c.reconcileInventoryReservedQuantity},
	}
}
