// Package loaders mirrors @medusajs/fulfillment/src/loaders.
package loaders

import (
	"context"
	"fmt"
	"slices"

	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/fulfillment/src/models"
)

// registrationIdentifier mirrors FulfillmentProviderService.getRegistrationIdentifier.
func registrationIdentifier(identifier, id string) string {
	if id == "" {
		return identifier
	}
	return identifier + "_" + id
}

// Providers mirrors syncDatabaseProviders from providers.ts.
func Providers(ctx context.Context, opts modulessdk.LoaderOptions) error {
	var identifiers []string
	for _, p := range opts.Providers {
		identifiers = append(identifiers, registrationIdentifier(p.Identifier, p.ID))
	}

	svc := utilsmodulessdk.NewMedusaService(opts.DB, models.All, nil).Internal("FulfillmentProvider")
	providers, err := svc.List(ctx, dal.NewRecord(), utilsmodulessdk.FindConfig{})
	if err != nil {
		opts.Logger.Error(fmt.Sprintf("Error syncing the fulfillment providers: %s", err))
		return nil
	}
	loaded := map[string]bool{}
	for _, p := range providers {
		id, _ := p.Get("id")
		loaded[id.(string)] = true
	}

	var toCreate []*dal.Record
	var toUpdate []any
	for _, id := range identifiers {
		r := dal.NewRecord()
		r.Set("id", id)
		if !loaded[id] {
			toCreate = append(toCreate, r)
			continue
		}
		r.Set("is_enabled", true)
		toUpdate = append(toUpdate, r)
	}
	for _, p := range providers {
		id, _ := p.Get("id")
		if !slices.Contains(identifiers, id.(string)) {
			r := dal.NewRecord()
			r.Set("id", id)
			r.Set("is_enabled", false)
			toUpdate = append(toUpdate, r)
		}
	}

	if len(toCreate) > 0 {
		if _, err := svc.Create(ctx, toCreate); err != nil {
			opts.Logger.Error(fmt.Sprintf("Error syncing the fulfillment providers: %s", err))
			return nil
		}
	}
	if len(toUpdate) > 0 {
		if _, err := svc.Update(ctx, toUpdate); err != nil {
			opts.Logger.Error(fmt.Sprintf("Error syncing the fulfillment providers: %s", err))
		}
	}
	return nil
}
