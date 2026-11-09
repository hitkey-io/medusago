package migrationscripts

import (
	"context"
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// assignProductsToShippingProfile mirrors migrate-product-shipping-profile.ts:
// every product is linked to the default shipping profile, created when no
// shipping profile exists. Errors are logged, not thrown, like upstream.
func (c Context) assignProductsToShippingProfile(ctx context.Context) error {
	if !c.isInstalled(utilsmodulessdk.FULFILLMENT) || !c.isInstalled(utilsmodulessdk.PRODUCT) {
		return nil
	}
	c.Logger.Info("Assigning products to shipping profile")
	if err := c.assignProducts(ctx); err != nil {
		c.Logger.Error(err.Error())
		return nil
	}
	c.Logger.Info("Products assigned to shipping profile")
	return nil
}

func (c Context) assignProducts(ctx context.Context) error {
	fulfillment := c.service(utilsmodulessdk.FULFILLMENT).Internal("ShippingProfile")
	product := c.service(utilsmodulessdk.PRODUCT).Internal("Product")

	profiles, err := fulfillment.List(ctx, dal.NewRecord(), utilsmodulessdk.FindConfig{Select: []string{"id", "name"}})
	if err != nil {
		return err
	}
	products, err := product.List(ctx, dal.NewRecord(), utilsmodulessdk.FindConfig{Select: []string{"id"}})
	if err != nil {
		return err
	}

	var profileID any
	for _, p := range profiles {
		name, _ := p.Get("name")
		if strings.Contains(strings.ToLower(fmt.Sprint(name)), "default") {
			profileID, _ = p.Get("id")
			break
		}
	}
	if profileID == nil && len(profiles) > 0 {
		profileID, _ = profiles[0].Get("id")
	}
	if profileID == nil {
		data := dal.NewRecord()
		data.Set("name", "Default Shipping Profile")
		data.Set("type", "default")
		created, err := fulfillment.Create(ctx, []*dal.Record{data})
		if err != nil {
			return err
		}
		profileID, _ = created[0].Get("id")
	}

	for _, p := range products {
		productID, _ := p.Get("id")
		if _, err := c.DB.Exec(ctx, `insert into "product_shipping_profile" ("id", "product_id", "shipping_profile_id") values ($1, $2, $3) on conflict do nothing`,
			common.GenerateEntityID("", "prodsp"), productID, profileID); err != nil {
			return err
		}
	}
	return nil
}
