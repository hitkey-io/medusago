// Package services mirrors @medusajs/region/src/services.
package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/region/src/models"
)

// RegionModuleService mirrors RegionModuleService.
type RegionModuleService struct {
	*utilsmodulessdk.MedusaService
}

// NewRegionModuleService builds the service.
func NewRegionModuleService(opts modulessdk.ServiceOptions, joinerConfig *joiner.ModuleJoinerConfig) (*RegionModuleService, error) {
	return &RegionModuleService{MedusaService: utilsmodulessdk.NewMedusaService(opts.DB, models.All, joinerConfig)}, nil
}

func invalid(format string, args ...any) error {
	return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf(format, args...))
}

func strOf(r *dal.Record, k string) string {
	v, _ := r.Get(k)
	s, _ := v.(string)
	return s
}

// normalizeInput mirrors normalizeInput: currency code lower case, name
// trimmed, countries lower case; undefined values removed.
func normalizeInput(regions []*dal.Record) []*dal.Record {
	out := make([]*dal.Record, len(regions))
	for i, r := range regions {
		n := dal.NewRecord()
		for _, k := range r.Keys() {
			v, _ := r.Get(k)
			switch k {
			case "currency_code":
				if s, ok := v.(string); ok && s != "" {
					v = strings.ToLower(s)
				} else {
					continue
				}
			case "name":
				s, ok := v.(string)
				if !ok {
					continue
				}
				v = strings.TrimSpace(s)
			case "countries":
				l, ok := v.([]any)
				if !ok {
					continue
				}
				cs := make([]any, len(l))
				for j, c := range l {
					cs[j] = strings.ToLower(fmt.Sprint(c))
				}
				v = cs
			}
			n.Set(k, v)
		}
		out[i] = n
	}
	return out
}

func withoutCountries(r *dal.Record) *dal.Record {
	n := dal.NewRecord()
	for _, k := range r.Keys() {
		if k == "countries" {
			continue
		}
		v, _ := r.Get(k)
		n.Set(k, v)
	}
	return n
}

func countriesOf(r *dal.Record) ([]any, bool) {
	v, ok := r.Get("countries")
	if !ok {
		return nil, false
	}
	l, _ := v.([]any)
	return l, true
}

// validateCountries mirrors validateCountries.
func (s *RegionModuleService) validateCountries(ctx context.Context, countries []any) error {
	if len(countries) == 0 {
		return nil
	}
	seen := map[string]int{}
	var unique []any
	for _, c := range countries {
		k := fmt.Sprint(c)
		if seen[k] == 0 {
			unique = append(unique, c)
		}
		seen[k]++
	}
	if len(unique) != len(countries) {
		var dups []string
		added := map[string]bool{}
		for _, c := range countries {
			k := fmt.Sprint(c)
			if seen[k] > 1 && !added[k] {
				added[k] = true
				dups = append(dups, k)
			}
		}
		return invalid(`Countries with codes: "%s" are already assigned to a region`, strings.Join(dups, ", "))
	}
	f := dal.NewRecord()
	f.Set("iso_2", unique)
	inDB, err := s.Internal("Country").List(ctx, f, utilsmodulessdk.FindConfig{Select: []string{"iso_2", "region_id"}})
	if err != nil {
		return err
	}
	if len(inDB) != len(unique) {
		have := map[string]bool{}
		for _, c := range inDB {
			have[strings.ToLower(strOf(c, "iso_2"))] = true
		}
		var missing []string
		for _, c := range unique {
			if !have[fmt.Sprint(c)] {
				missing = append(missing, fmt.Sprint(c))
			}
		}
		return invalid(`Countries with codes: "%s" do not exist`, strings.Join(missing, ", "))
	}
	var assigned []string
	for _, c := range inDB {
		if v, _ := c.Get("region_id"); v != nil && v != "" {
			assigned = append(assigned, strOf(c, "iso_2"))
		}
	}
	if len(assigned) > 0 {
		return invalid(`Countries with codes: "%s" are already assigned to a region`, strings.Join(assigned, ", "))
	}
	return nil
}

func countryUpdate(iso []any, regionID any) *dal.Record {
	in := dal.NewRecord()
	in.Set("iso_2", iso)
	data := dal.NewRecord()
	data.Set("region_id", regionID)
	u := dal.NewRecord()
	u.Set("selector", in)
	u.Set("data", data)
	return u
}

// CreateRegions mirrors createRegions.
func (s *RegionModuleService) CreateRegions(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := &RegionModuleService{MedusaService: tx}
		var err error
		out, err = t.createRegions(ctx, data)
		return err
	})
	return out, err
}

func (s *RegionModuleService) createRegions(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	normalized := normalizeInput(data)
	db := make([]*dal.Record, len(normalized))
	for i, r := range normalized {
		db[i] = withoutCountries(r)
	}
	created, err := s.Internal("Region").Create(ctx, db)
	if err != nil {
		return nil, err
	}
	anyCountries := false
	var all []any
	for _, r := range normalized {
		cs, _ := countriesOf(r)
		if len(cs) > 0 {
			anyCountries = true
		}
		all = append(all, cs...)
	}
	if anyCountries {
		if err := s.validateCountries(ctx, all); err != nil {
			return nil, err
		}
		var updates []any
		for i, r := range normalized {
			cs, _ := countriesOf(r)
			if cs == nil {
				cs = []any{}
			}
			id, _ := created[i].Get("id")
			updates = append(updates, countryUpdate(cs, id))
		}
		if _, err := s.Internal("Country").Update(ctx, updates); err != nil {
			return nil, err
		}
	}
	return created, nil
}

// UpdateRegions mirrors updateRegions(idOrSelector, data).
func (s *RegionModuleService) UpdateRegions(ctx context.Context, idOrSelector any, data *dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := &RegionModuleService{MedusaService: tx}
		var input []*dal.Record
		withID := func(id any) *dal.Record {
			r := dal.NewRecord()
			r.Set("id", id)
			for _, k := range data.Keys() {
				v, _ := data.Get(k)
				r.Set(k, v)
			}
			return r
		}
		if id, ok := idOrSelector.(string); ok {
			input = []*dal.Record{withID(id)}
		} else {
			sel, _ := idOrSelector.(*dal.Record)
			rows, err := t.Internal("Region").List(ctx, sel, utilsmodulessdk.FindConfig{})
			if err != nil {
				return err
			}
			for _, r := range rows {
				id, _ := r.Get("id")
				input = append(input, withID(id))
			}
		}
		var err error
		out, err = t.updateRegions(ctx, input)
		return err
	})
	return out, err
}

// UpsertRegions mirrors upsertRegions.
func (s *RegionModuleService) UpsertRegions(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := &RegionModuleService{MedusaService: tx}
		var forCreate, forUpdate []*dal.Record
		for _, r := range data {
			if strOf(r, "id") != "" {
				forUpdate = append(forUpdate, r)
			} else {
				forCreate = append(forCreate, r)
			}
		}
		if len(forCreate) > 0 {
			c, err := t.createRegions(ctx, forCreate)
			if err != nil {
				return err
			}
			out = append(out, c...)
		}
		if len(forUpdate) > 0 {
			u, err := t.updateRegions(ctx, forUpdate)
			if err != nil {
				return err
			}
			out = append(out, u...)
		}
		return nil
	})
	return out, err
}

func (s *RegionModuleService) updateRegions(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	normalized := normalizeInput(data)
	var withCountryUpdate []any
	for _, r := range normalized {
		if _, ok := countriesOf(r); ok {
			id, _ := r.Get("id")
			withCountryUpdate = append(withCountryUpdate, id)
		}
	}
	if len(withCountryUpdate) > 0 {
		sel := dal.NewRecord()
		sel.Set("region_id", withCountryUpdate)
		nulls := dal.NewRecord()
		nulls.Set("region_id", nil)
		u := dal.NewRecord()
		u.Set("selector", sel)
		u.Set("data", nulls)
		if _, err := s.Internal("Country").Update(ctx, u); err != nil {
			return nil, err
		}
		var all []any
		for _, r := range normalized {
			cs, _ := countriesOf(r)
			all = append(all, cs...)
		}
		if err := s.validateCountries(ctx, all); err != nil {
			return nil, err
		}
		var updates []any
		for _, r := range normalized {
			cs, _ := countriesOf(r)
			if cs == nil {
				cs = []any{}
			}
			id, _ := r.Get("id")
			updates = append(updates, countryUpdate(cs, id))
		}
		if _, err := s.Internal("Country").Update(ctx, updates); err != nil {
			return nil, err
		}
	}
	db := make([]any, len(normalized))
	for i, r := range normalized {
		db[i] = withoutCountries(r)
	}
	res, err := s.Internal("Region").Update(ctx, db)
	if err != nil {
		return nil, err
	}
	if r, ok := res.(*dal.Record); ok {
		return []*dal.Record{r}, nil
	}
	rows, _ := res.([]*dal.Record)
	return rows, nil
}

// SoftDeleteRegions mirrors softDeleteRegions: the countries lose their
// region.
func (s *RegionModuleService) SoftDeleteRegions(ctx context.Context, ids []any) error {
	return s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := &RegionModuleService{MedusaService: tx}
		if _, err := t.Internal("Region").SoftDelete(ctx, ids); err != nil {
			return err
		}
		sel := dal.NewRecord()
		sel.Set("region_id", ids)
		nulls := dal.NewRecord()
		nulls.Set("region_id", nil)
		u := dal.NewRecord()
		u.Set("selector", sel)
		u.Set("data", nulls)
		_, err := t.Internal("Country").Update(ctx, u)
		return err
	})
}
