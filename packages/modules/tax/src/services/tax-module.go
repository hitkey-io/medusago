// Package services mirrors @medusajs/tax/src/services.
package services

import (
	"errors"
	"log/slog"

	"context"
	"fmt"
	taxtypes "github.com/hitkey-io/medusago/packages/core/types/src/tax"
	"github.com/hitkey-io/medusago/packages/modules/tax/src/providers"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/tax/src/models"
)

// TaxModuleService mirrors TaxModuleService (writes).
type TaxModuleService struct {
	*utilsmodulessdk.MedusaService
	// providers are the registered tax providers by their container key
	// (tp_system, tp_<identifier>_<id>).
	providers map[string]taxtypes.ITaxProvider
	logger    *slog.Logger
}

// NewTaxModuleService builds the service with the system provider and the
// configured providers.
func NewTaxModuleService(opts modulessdk.ServiceOptions, joinerConfig *joiner.ModuleJoinerConfig) (*TaxModuleService, error) {
	s := &TaxModuleService{
		MedusaService: utilsmodulessdk.NewMedusaService(opts.DB, models.All, joinerConfig),
		providers:     map[string]taxtypes.ITaxProvider{"tp_" + providers.SystemTaxIdentifier: providers.SystemTaxService{}},
		logger:        opts.Logger,
	}
	if s.logger == nil {
		s.logger = slog.Default()
	}
	for _, p := range opts.Providers {
		if inst, ok := p.Instance.(taxtypes.ITaxProvider); ok {
			key := "tp_" + p.Identifier
			if p.ID != "" {
				key += "_" + p.ID
			}
			s.providers[key] = inst
		}
	}
	return s, nil
}

// retrieveProvider mirrors TaxProviderService.retrieveProvider.
func (s *TaxModuleService) retrieveProvider(providerID string) (taxtypes.ITaxProvider, error) {
	if p, ok := s.providers[providerID]; ok {
		return p, nil
	}
	s.logger.Error("AwilixResolutionError: Could not resolve '" + providerID + "'.")
	return nil, errors.New("\n  Unable to retrieve the tax provider with id: " + providerID + "\n  Please make sure that the provider is registered in the container and it is configured correctly in your project configuration file.")
}

// GetProvider mirrors getProvider.
func (s *TaxModuleService) GetProvider(providerID string) (taxtypes.ITaxProvider, error) {
	return s.retrieveProvider(providerID)
}

func (s *TaxModuleService) tx(ctx context.Context, fn func(t *TaxModuleService) error) error {
	return s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		return fn(&TaxModuleService{MedusaService: tx})
	})
}

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func copyWithout(r *dal.Record, keys ...string) *dal.Record {
	out := dal.NewRecord()
	skip := map[string]bool{}
	for _, k := range keys {
		skip[k] = true
	}
	for _, k := range r.Keys() {
		if !skip[k] {
			v, _ := r.Get(k)
			out.Set(k, v)
		}
	}
	return out
}

func records(v any) []*dal.Record {
	switch t := v.(type) {
	case []*dal.Record:
		return t
	case []any:
		var out []*dal.Record
		for _, x := range t {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
}

func rows(res any) []*dal.Record {
	if r, ok := res.(*dal.Record); ok {
		return []*dal.Record{r}
	}
	l, _ := res.([]*dal.Record)
	return l
}

// CreateTaxRates mirrors createTaxRates.
func (s *TaxModuleService) CreateTaxRates(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.tx(ctx, func(t *TaxModuleService) error {
		var err error
		out, err = t.createTaxRates(ctx, data)
		return err
	})
	return out, err
}

func (s *TaxModuleService) createTaxRates(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	rules := make([]any, len(data))
	rateData := make([]*dal.Record, len(data))
	for i, r := range data {
		rules[i], _ = r.Get("rules")
		rateData[i] = copyWithout(r, "rules")
	}
	rates, err := s.Internal("TaxRate").Create(ctx, rateData)
	if err != nil {
		return nil, err
	}
	var toCreate []*dal.Record
	for i, rate := range rates {
		if rules[i] == nil {
			continue
		}
		for _, r := range records(rules[i]) {
			n := copyWithout(r)
			n.Set("created_by", get(rate, "created_by"))
			n.Set("tax_rate_id", get(rate, "id"))
			toCreate = append(toCreate, n)
		}
	}
	if len(toCreate) > 0 {
		if _, err := s.Internal("TaxRateRule").Create(ctx, toCreate); err != nil {
			return nil, err
		}
	}
	return rates, nil
}

// UpdateTaxRates mirrors updateTaxRates(idOrSelector, data).
func (s *TaxModuleService) UpdateTaxRates(ctx context.Context, idOrSelector any, data *dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.tx(ctx, func(t *TaxModuleService) error {
		selector, _ := idOrSelector.(*dal.Record)
		if selector == nil {
			selector = dal.NewRecord()
			selector.Set("id", idOrSelector)
		}
		update := data
		if rules, ok := data.Get("rules"); ok && rules != nil {
			if err := t.setTaxRateRulesForTaxRates(ctx, selector, records(rules), get(data, "updated_by")); err != nil {
				return err
			}
			update = copyWithout(data, "rules")
		}
		u := dal.NewRecord()
		u.Set("selector", selector)
		u.Set("data", update)
		res, err := t.Internal("TaxRate").Update(ctx, u)
		if err != nil {
			return err
		}
		out = rows(res)
		return nil
	})
	return out, err
}

func (s *TaxModuleService) setTaxRateRulesForTaxRates(ctx context.Context, selector *dal.Record, rules []*dal.Record, createdBy any) error {
	rates, err := s.Internal("TaxRate").List(ctx, selector, utilsmodulessdk.FindConfig{Select: []string{"id"}})
	if err != nil {
		return err
	}
	rateIDs := make([]any, len(rates))
	for i, r := range rates {
		rateIDs[i] = get(r, "id")
	}
	if len(rateIDs) > 0 {
		f := dal.NewRecord()
		f.Set("tax_rate_id", rateIDs)
		existing, err := s.Internal("TaxRateRule").List(ctx, f, utilsmodulessdk.FindConfig{Select: []string{"id"}})
		if err != nil {
			return err
		}
		ids := make([]any, len(existing))
		for i, r := range existing {
			ids[i] = get(r, "id")
		}
		if _, err := s.Internal("TaxRateRule").SoftDelete(ctx, ids); err != nil {
			return err
		}
	}
	if len(rules) == 0 {
		return nil
	}
	var toCreate []*dal.Record
	for _, id := range rateIDs {
		for _, r := range rules {
			n := copyWithout(r)
			n.Set("created_by", createdBy)
			n.Set("tax_rate_id", id)
			toCreate = append(toCreate, n)
		}
	}
	_, err = s.Internal("TaxRateRule").Create(ctx, toCreate)
	return err
}

// UpsertTaxRates mirrors upsertTaxRates.
func (s *TaxModuleService) UpsertTaxRates(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	return s.Internal("TaxRate").Upsert(ctx, data)
}

func normalizeCode(v any) any {
	if s, ok := v.(string); ok {
		return strings.ToLower(s)
	}
	return v
}

// CreateTaxRegions mirrors createTaxRegions.
func (s *TaxModuleService) CreateTaxRegions(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.tx(ctx, func(t *TaxModuleService) error {
		defaultRates := make([]*dal.Record, len(data))
		regionData := make([]*dal.Record, len(data))
		for i, region := range data {
			if d, ok := get(region, "default_tax_rate").(*dal.Record); ok && d != nil {
				r := copyWithout(d)
				r.Set("is_default", true)
				r.Set("created_by", get(region, "created_by"))
				defaultRates[i] = r
			}
			rest := copyWithout(region, "default_tax_rate")
			if p, _ := rest.Get("province_code"); p != nil && p != "" {
				rest.Set("province_code", normalizeCode(p))
			} else {
				rest.Set("province_code", nil)
			}
			rest.Set("country_code", normalizeCode(get(rest, "country_code")))
			regionData[i] = rest
		}
		if err := t.verifyProvinceToCountryMatch(ctx, regionData); err != nil {
			return err
		}
		regions, err := t.Internal("TaxRegion").Create(ctx, regionData)
		if err != nil {
			return err
		}
		var rates []*dal.Record
		for i, region := range regions {
			if defaultRates[i] == nil {
				continue
			}
			r := copyWithout(defaultRates[i])
			r.Set("tax_region_id", get(region, "id"))
			rates = append(rates, r)
		}
		if len(rates) > 0 {
			if _, err := t.createTaxRates(ctx, rates); err != nil {
				return err
			}
		}
		out = regions
		return nil
	})
	return out, err
}

func (s *TaxModuleService) verifyProvinceToCountryMatch(ctx context.Context, regions []*dal.Record) error {
	var parentIDs []any
	for _, r := range regions {
		if p := get(r, "parent_id"); p != nil {
			parentIDs = append(parentIDs, p)
		}
	}
	if len(parentIDs) == 0 {
		return nil
	}
	in := dal.NewRecord()
	in.Set("$in", parentIDs)
	f := dal.NewRecord()
	f.Set("id", in)
	parents, err := s.Internal("TaxRegion").List(ctx, f, utilsmodulessdk.FindConfig{Select: []string{"id", "country_code"}})
	if err != nil {
		return err
	}
	for _, r := range regions {
		pid := get(r, "parent_id")
		if pid == nil {
			continue
		}
		var parent *dal.Record
		for _, p := range parents {
			if fmt.Sprint(get(p, "id")) == fmt.Sprint(pid) {
				parent = p
			}
		}
		if parent == nil {
			return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Province region must belong to a parent region. You are trying to create a province region with (country: %s, province: %s) but parent does not exist", jsStr(get(r, "country_code")), jsStr(get(r, "province_code"))))
		}
		if fmt.Sprint(get(parent, "country_code")) != fmt.Sprint(get(r, "country_code")) {
			return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Province region must belong to a parent region with the same country code. You are trying to create a province region with (country: %s, province: %s) but parent expects (country: %s)", jsStr(get(r, "country_code")), jsStr(get(r, "province_code")), jsStr(get(parent, "country_code"))))
		}
	}
	return nil
}

// jsStr mirrors template literal interpolation (null -> "null").
func jsStr(v any) string {
	if v == nil {
		return "null"
	}
	return fmt.Sprint(v)
}

// CreateTaxRateRules mirrors createTaxRateRules.
func (s *TaxModuleService) CreateTaxRateRules(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	if len(data) == 0 {
		return []*dal.Record{}, nil
	}
	return s.Internal("TaxRateRule").Create(ctx, data)
}

// UpdateTaxRegions mirrors the generated updateTaxRegions (array of
// { id, ... }).
func (s *TaxModuleService) UpdateTaxRegions(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	in := make([]any, len(data))
	for i, d := range data {
		in[i] = d
	}
	res, err := s.Internal("TaxRegion").Update(ctx, in)
	if err != nil {
		return nil, err
	}
	return rows(res), nil
}
