// Package loaders mirrors @medusajs/region/src/loaders.
package loaders

import (
	"context"
	"fmt"
	"strings"

	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/defaults"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/region/src/models"
)

// Defaults mirrors defaults.ts: it upserts the default countries.
func Defaults(ctx context.Context, opts modulessdk.LoaderOptions) error {
	svc := utilsmodulessdk.NewMedusaService(opts.DB, models.All, nil).Internal("Country")
	data := make([]*dal.Record, 0, len(defaults.DefaultCountries))
	for _, c := range defaults.DefaultCountries {
		r := dal.NewRecord()
		r.Set("iso_2", strings.ToLower(c.Alpha2))
		r.Set("iso_3", strings.ToLower(c.Alpha3))
		r.Set("num_code", c.Numeric)
		r.Set("name", strings.ToUpper(c.Name))
		r.Set("display_name", c.Name)
		data = append(data, r)
	}
	resp, err := svc.Upsert(ctx, data)
	if err != nil {
		opts.Logger.Warn(fmt.Sprintf("Failed to load countries, skipping loader. Original error: %s", err))
		return nil
	}
	opts.Logger.Debug(fmt.Sprintf("Loaded %d countries", len(resp)))
	return nil
}
