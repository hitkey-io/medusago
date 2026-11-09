// Package loaders mirrors @medusajs/tax/src/loaders.
package loaders

import (
	"context"
	"slices"

	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/tax/src/models"
)

// providerKey mirrors the registration key `tp_<identifier>[_<id>]`.
func providerKey(identifier, id string) string {
	key := "tp_" + identifier
	if id != "" {
		key += "_" + id
	}
	return key
}

// Providers mirrors providers.ts: the local (system) provider and the
// configured providers are registered enabled.
func Providers(ctx context.Context, opts modulessdk.LoaderOptions) error {
	providersToLoad := []string{providerKey("system", "")}
	for _, p := range opts.Providers {
		providersToLoad = append(providersToLoad, providerKey(p.Identifier, p.ID))
	}

	svc := utilsmodulessdk.NewMedusaService(opts.DB, models.All, nil).Internal("TaxProvider")
	filter := dal.NewRecord()
	ids := make([]any, len(providersToLoad))
	for i, id := range providersToLoad {
		ids[i] = id
	}
	filter.Set("id", ids)
	existing, err := svc.List(ctx, filter, utilsmodulessdk.FindConfig{})
	if err != nil {
		return err
	}
	var upsert []*dal.Record
	for _, e := range existing {
		id, _ := e.Get("id")
		if !slices.Contains(providersToLoad, id.(string)) {
			r := dal.NewRecord()
			r.Set("id", id)
			r.Set("is_enabled", false)
			upsert = append(upsert, r)
		}
	}
	for _, id := range providersToLoad {
		r := dal.NewRecord()
		r.Set("id", id)
		r.Set("is_enabled", true)
		upsert = append(upsert, r)
	}
	_, err = svc.Upsert(ctx, upsert)
	return err
}
