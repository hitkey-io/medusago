// Package loaders mirrors @medusajs/notification/src/loaders.
package loaders

import (
	"context"
	"fmt"

	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/notification/src/models"
)

// Providers mirrors syncDatabaseProviders from providers.ts.
func Providers(ctx context.Context, opts modulessdk.LoaderOptions) error {
	var normalized []*dal.Record
	ids := map[string]bool{}
	for _, p := range opts.Providers {
		if p.ID == "" {
			return fmt.Errorf("An entry in the provider config is required to initialize notification providers")
		}
		channels := []any{}
		if raw, ok := p.Options["channels"].([]string); ok {
			for _, c := range raw {
				channels = append(channels, c)
			}
		}
		r := dal.NewRecord()
		r.Set("id", p.ID)
		r.Set("handle", p.ID)
		r.Set("name", p.ID)
		r.Set("is_enabled", true)
		r.Set("channels", channels)
		normalized = append(normalized, r)
		ids[p.ID] = true
	}

	svc := utilsmodulessdk.NewMedusaService(opts.DB, models.All, nil).Internal("NotificationProvider")
	inDB, err := svc.List(ctx, dal.NewRecord(), utilsmodulessdk.FindConfig{})
	if err != nil {
		opts.Logger.Error(fmt.Sprintf("Error syncing the notification providers: %s", err))
		return nil
	}
	if len(normalized) > 0 {
		if _, err := svc.Upsert(ctx, normalized); err != nil {
			opts.Logger.Error(fmt.Sprintf("Error syncing the notification providers: %s", err))
			return nil
		}
	}
	var disable []any
	for _, p := range inDB {
		id, _ := p.Get("id")
		if !ids[id.(string)] {
			r := dal.NewRecord()
			r.Set("id", id)
			r.Set("is_enabled", false)
			disable = append(disable, r)
		}
	}
	if len(disable) > 0 {
		if _, err := svc.Update(ctx, disable); err != nil {
			opts.Logger.Error(fmt.Sprintf("Error syncing the notification providers: %s", err))
		}
	}
	return nil
}
