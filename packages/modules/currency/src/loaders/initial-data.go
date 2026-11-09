// Package loaders mirrors @medusajs/currency/src/loaders.
package loaders

import (
	"context"
	"fmt"
	"strings"

	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/defaults"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/currency/src/models"
)

// InitialData mirrors initial-data.ts: it upserts the default currencies.
func InitialData(ctx context.Context, opts modulessdk.LoaderOptions) error {
	svc := utilsmodulessdk.NewMedusaService(opts.DB, models.All, nil).Internal("Currency")
	data := make([]*dal.Record, 0, len(defaults.DefaultCurrencies))
	for _, c := range defaults.DefaultCurrencies {
		r := dal.NewRecord()
		r.Set("symbol", c.Symbol)
		r.Set("name", c.Name)
		r.Set("symbol_native", c.SymbolNative)
		r.Set("decimal_digits", float64(c.DecimalDigits))
		r.Set("rounding", c.Rounding)
		r.Set("code", strings.ToLower(c.Code))
		r.Set("name_plural", c.NamePlural)
		data = append(data, r)
	}
	resp, err := svc.Upsert(ctx, data)
	if err != nil {
		opts.Logger.Warn(fmt.Sprintf("Failed to load currencies, skipping loader. Original error: %s", err))
		return nil
	}
	opts.Logger.Debug(fmt.Sprintf("Loaded %d currencies", len(resp)))
	return nil
}
