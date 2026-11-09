package migrationscripts

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// assignTaxSystemProviderToTaxRegions mirrors migrate-tax-region-provider.ts:
// parent tax regions without a provider get the system provider.
func (c Context) assignTaxSystemProviderToTaxRegions(ctx context.Context) error {
	if !c.isInstalled(utilsmodulessdk.TAX) {
		return nil
	}
	c.Logger.Info("Assigning tax system provider to tax regions")
	svc := c.service(utilsmodulessdk.TAX).Internal("TaxRegion")
	regions, err := svc.List(ctx, dal.NewRecord(), utilsmodulessdk.FindConfig{Select: []string{"id", "provider_id", "province_code"}})
	if err != nil {
		c.Logger.Error(err.Error())
		return err
	}
	var update []any
	for _, r := range regions {
		province, _ := r.Get("province_code")
		provider, _ := r.Get("provider_id")
		if (province == nil || province == "") && (provider == nil || provider == "") {
			id, _ := r.Get("id")
			u := dal.NewRecord()
			u.Set("id", id)
			u.Set("provider_id", "tp_system")
			update = append(update, u)
		}
	}
	if len(update) > 0 {
		if _, err := svc.Update(ctx, update); err != nil {
			c.Logger.Error(err.Error())
			return err
		}
	}
	c.Logger.Info("System provider assigned to tax regions")
	return nil
}
