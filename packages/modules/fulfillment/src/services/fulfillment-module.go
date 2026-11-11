// Package services mirrors @medusajs/fulfillment/src/services.
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
	"github.com/hitkey-io/medusago/packages/modules/fulfillment/src/models"
	"github.com/hitkey-io/medusago/packages/modules/fulfillment/src/utils"
)

// FulfillmentModuleService mirrors FulfillmentModuleService (writes).
type FulfillmentModuleService struct {
	*utilsmodulessdk.MedusaService
	provider *FulfillmentProviderService
}

// NewFulfillmentModuleService builds the service.
func NewFulfillmentModuleService(opts modulessdk.ServiceOptions, joinerConfig *joiner.ModuleJoinerConfig) (*FulfillmentModuleService, error) {
	return &FulfillmentModuleService{
		MedusaService: utilsmodulessdk.NewMedusaService(opts.DB, models.All, joinerConfig),
		provider:      NewFulfillmentProviderService(opts.Providers, opts.Logger),
	}, nil
}

func (s *FulfillmentModuleService) tx(ctx context.Context, fn func(t *FulfillmentModuleService) error) error {
	return s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		return fn(&FulfillmentModuleService{MedusaService: tx, provider: s.provider})
	})
}

func invalid(msg string) error {
	return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, msg)
}

func notFound(msg string) error {
	return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, msg)
}

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
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

func anyList(rs []*dal.Record) []any {
	out := make([]any, len(rs))
	for i, r := range rs {
		out[i] = r
	}
	return out
}

func ids(rs []*dal.Record) []any {
	out := make([]any, len(rs))
	for i, r := range rs {
		out[i] = get(r, "id")
	}
	return out
}

func clone(r *dal.Record) *dal.Record {
	n := dal.NewRecord()
	for _, k := range r.Keys() {
		v, _ := r.Get(k)
		if sub, ok := v.(*dal.Record); ok {
			v = clone(sub)
		} else if l, ok := v.([]any); ok {
			cl := make([]any, len(l))
			for i, x := range l {
				if xr, ok := x.(*dal.Record); ok {
					cl[i] = clone(xr)
				} else {
					cl[i] = x
				}
			}
			v = cl
		}
		n.Set(k, v)
	}
	return n
}

func rows(res any) []*dal.Record {
	if r, ok := res.(*dal.Record); ok {
		return []*dal.Record{r}
	}
	l, _ := res.([]*dal.Record)
	return l
}

func hasID(r *dal.Record) bool {
	id, ok := r.Get("id")
	return ok && id != nil && id != ""
}

var requirePropForType = map[string][]string{
	"country":  {"country_code"},
	"province": {"country_code", "province_code"},
	"city":     {"country_code", "province_code", "city"},
	"zip":      {"country_code", "province_code", "city", "postal_expression"},
}

func truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	case bool:
		return t
	}
	return true
}

// ValidateGeoZones mirrors validateGeoZones.
func ValidateGeoZones(zones []*dal.Record) error {
	for _, z := range zones {
		typ := fmt.Sprint(get(z, "type"))
		if get(z, "type") == nil {
			typ = "undefined"
		}
		props, ok := requirePropForType[typ]
		if !ok {
			return invalid("Invalid geo zone type: " + typ)
		}
		for _, p := range props {
			if !truthy(get(z, p)) {
				return invalid(fmt.Sprintf("Missing required property %s for geo zone type %s", p, typ))
			}
		}
	}
	return nil
}

// CreateFulfillmentSets mirrors createFulfillmentSets.
func (s *FulfillmentModuleService) CreateFulfillmentSets(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	if len(data) == 0 {
		return []*dal.Record{}, nil
	}
	for _, fs := range data {
		for _, sz := range records(get(fs, "service_zones")) {
			if zones := records(get(sz, "geo_zones")); len(zones) > 0 {
				if err := ValidateGeoZones(zones); err != nil {
					return nil, err
				}
			}
		}
	}
	return s.Internal("FulfillmentSet").Create(ctx, data)
}

// CreateServiceZones mirrors createServiceZones.
func (s *FulfillmentModuleService) CreateServiceZones(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	if len(data) == 0 {
		return []*dal.Record{}, nil
	}
	for _, sz := range data {
		if zones := records(get(sz, "geo_zones")); len(zones) > 0 {
			if err := ValidateGeoZones(zones); err != nil {
				return nil, err
			}
		}
	}
	return s.Internal("ServiceZone").Create(ctx, data)
}

// CreateShippingOptions mirrors createShippingOptions.
func (s *FulfillmentModuleService) CreateShippingOptions(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	if len(data) == 0 {
		return []*dal.Record{}, nil
	}
	var rules []*dal.Record
	for _, d := range data {
		rules = append(rules, records(get(d, "rules"))...)
	}
	if len(rules) > 0 {
		if err := utils.ValidateAndNormalizeRules(rules); err != nil {
			return nil, err
		}
	}
	return s.Internal("ShippingOption").Create(ctx, data)
}

// CreateShippingOptionRules mirrors createShippingOptionRules.
func (s *FulfillmentModuleService) CreateShippingOptionRules(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	if len(data) == 0 {
		return []*dal.Record{}, nil
	}
	if err := utils.ValidateAndNormalizeRules(data); err != nil {
		return nil, err
	}
	return s.Internal("ShippingOptionRule").Create(ctx, data)
}

// UpdateShippingOptionRules mirrors updateShippingOptionRules.
func (s *FulfillmentModuleService) UpdateShippingOptionRules(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	if len(data) == 0 {
		return []*dal.Record{}, nil
	}
	if err := utils.ValidateAndNormalizeRules(data); err != nil {
		return nil, err
	}
	in := make([]any, len(data))
	for i, d := range data {
		in[i] = d
	}
	res, err := s.Internal("ShippingOptionRule").Update(ctx, in)
	if err != nil {
		return nil, err
	}
	switch t := res.(type) {
	case []*dal.Record:
		return t, nil
	case *dal.Record:
		return []*dal.Record{t}, nil
	}
	return []*dal.Record{}, nil
}

// CreateGeoZones mirrors createGeoZones.
func (s *FulfillmentModuleService) CreateGeoZones(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	if err := ValidateGeoZones(data); err != nil {
		return nil, err
	}
	return s.Internal("GeoZone").Create(ctx, data)
}

// UpdateGeoZones mirrors updateGeoZones.
func (s *FulfillmentModuleService) UpdateGeoZones(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	if len(data) == 0 {
		return []*dal.Record{}, nil
	}
	if err := ValidateGeoZones(data); err != nil {
		return nil, err
	}
	res, err := s.Internal("GeoZone").Update(ctx, anyList(data))
	return rows(res), err
}

func difference(expected []any, existing map[string]bool) []string {
	var out []string
	seen := map[string]bool{}
	for _, e := range expected {
		k := fmt.Sprint(e)
		if !existing[k] && !seen[k] {
			out = append(out, k)
			seen[k] = true
		}
	}
	return out
}

func idSet(rs []*dal.Record) map[string]bool {
	out := map[string]bool{}
	for _, r := range rs {
		out[fmt.Sprint(get(r, "id"))] = true
	}
	return out
}

// UpdateFulfillmentSets mirrors updateFulfillmentSets.
func (s *FulfillmentModuleService) UpdateFulfillmentSets(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	// one transaction manager: collections loaded in the method are the
	// initialized ones its updates replace
	ctx = dal.WithSharedIdentityMap(ctx)
	var out []*dal.Record
	err := s.tx(ctx, func(t *FulfillmentModuleService) error {
		if len(data) == 0 {
			out = []*dal.Record{}
			return nil
		}
		f := dal.NewRecord()
		f.Set("id", ids(data))
		sets, err := t.Internal("FulfillmentSet").List(ctx, f, utilsmodulessdk.FindConfig{Relations: []string{"service_zones", "service_zones.geo_zones"}})
		if err != nil {
			return err
		}
		if missing := difference(ids(data), idSet(sets)); len(missing) > 0 {
			return notFound("The following fulfillment sets does not exists: " + strings.Join(missing, ", "))
		}
		setMap := map[string]*dal.Record{}
		for _, fs := range sets {
			setMap[fmt.Sprint(get(fs, "id"))] = fs
		}
		var szToDelete, gzToDelete []any
		in := make([]*dal.Record, len(data))
		for i, d := range data {
			d = clone(d)
			in[i] = d
			updated := records(get(d, "service_zones"))
			if _, has := d.Get("service_zones"); !has || get(d, "service_zones") == nil {
				continue
			}
			existingSet := setMap[fmt.Sprint(get(d, "id"))]
			existingZones := records(get(existingSet, "service_zones"))
			given := map[string]bool{}
			var givenIDs []any
			for _, sz := range updated {
				if hasID(sz) {
					given[fmt.Sprint(get(sz, "id"))] = true
					givenIDs = append(givenIDs, get(sz, "id"))
				}
			}
			zoneMap := map[string]*dal.Record{}
			for _, z := range existingZones {
				id := fmt.Sprint(get(z, "id"))
				zoneMap[id] = z
				if !given[id] {
					szToDelete = append(szToDelete, get(z, "id"))
					for _, g := range records(get(z, "geo_zones")) {
						gzToDelete = append(gzToDelete, get(g, "id"))
					}
				}
			}
			if missing := difference(givenIDs, idSet(existingZones)); len(missing) > 0 {
				return notFound("The following service zones does not exists: " + strings.Join(missing, ", "))
			}
			mapped := make([]any, len(updated))
			for j, sz := range updated {
				if !hasID(sz) {
					if zones := records(get(sz, "geo_zones")); len(zones) > 0 {
						if err := ValidateGeoZones(zones); err != nil {
							return err
						}
					}
					mapped[j] = sz
					continue
				}
				existing := clone(zoneMap[fmt.Sprint(get(sz, "id"))])
				existing.Delete("geo_zones")
				mapped[j] = existing
			}
			d.Set("service_zones", mapped)
		}
		if len(szToDelete) > 0 {
			if len(gzToDelete) > 0 {
				if _, err := t.Internal("GeoZone").Delete(ctx, gzToDelete); err != nil {
					return err
				}
			}
			if _, err := t.Internal("ServiceZone").Delete(ctx, szToDelete); err != nil {
				return err
			}
		}
		res, err := t.Internal("FulfillmentSet").Update(ctx, anyList(in))
		out = rows(res)
		return err
	})
	return out, err
}

// UpdateServiceZones mirrors updateServiceZones(idOrSelector, data).
func (s *FulfillmentModuleService) UpdateServiceZones(ctx context.Context, idOrSelector any, data *dal.Record) ([]*dal.Record, error) {
	// one transaction manager: collections loaded in the method are the
	// initialized ones its updates replace
	ctx = dal.WithSharedIdentityMap(ctx)
	var out []*dal.Record
	err := s.tx(ctx, func(t *FulfillmentModuleService) error {
		var in []*dal.Record
		withID := func(id any) *dal.Record {
			n := clone(data)
			r := dal.NewRecord()
			r.Set("id", id)
			for _, k := range n.Keys() {
				v, _ := n.Get(k)
				r.Set(k, v)
			}
			return r
		}
		if id, ok := idOrSelector.(string); ok {
			in = []*dal.Record{withID(id)}
		} else {
			sel, _ := idOrSelector.(*dal.Record)
			zones, err := t.Internal("ServiceZone").List(ctx, sel, utilsmodulessdk.FindConfig{})
			if err != nil {
				return err
			}
			if len(zones) == 0 {
				out = []*dal.Record{}
				return nil
			}
			for _, z := range zones {
				in = append(in, withID(get(z, "id")))
			}
		}
		var err error
		out, err = t.updateServiceZones(ctx, in)
		return err
	})
	return out, err
}

func (s *FulfillmentModuleService) updateServiceZones(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	if len(data) == 0 {
		return []*dal.Record{}, nil
	}
	f := dal.NewRecord()
	f.Set("id", ids(data))
	zones, err := s.Internal("ServiceZone").List(ctx, f, utilsmodulessdk.FindConfig{Relations: []string{"geo_zones"}})
	if err != nil {
		return nil, err
	}
	if missing := difference(ids(data), idSet(zones)); len(missing) > 0 {
		return nil, notFound("The following service zones does not exists: " + strings.Join(missing, ", "))
	}
	zoneMap := map[string]*dal.Record{}
	for _, z := range zones {
		zoneMap[fmt.Sprint(get(z, "id"))] = z
	}
	var gzToDelete []any
	for _, sz := range data {
		if v, has := sz.Get("geo_zones"); !has || v == nil {
			continue
		}
		existing := zoneMap[fmt.Sprint(get(sz, "id"))]
		existingGeo := records(get(existing, "geo_zones"))
		given := map[string]bool{}
		var givenIDs []any
		for _, g := range records(get(sz, "geo_zones")) {
			if hasID(g) {
				given[fmt.Sprint(get(g, "id"))] = true
				givenIDs = append(givenIDs, get(g, "id"))
			}
		}
		geoMap := map[string]*dal.Record{}
		for _, g := range existingGeo {
			id := fmt.Sprint(get(g, "id"))
			geoMap[id] = g
			if !given[id] {
				gzToDelete = append(gzToDelete, get(g, "id"))
			}
		}
		if missing := difference(givenIDs, idSet(existingGeo)); len(missing) > 0 {
			return nil, notFound("The following geo zones does not exists: " + strings.Join(missing, ", "))
		}
		updated := records(get(sz, "geo_zones"))
		mapped := make([]any, len(updated))
		for j, g := range updated {
			if !hasID(g) {
				if err := ValidateGeoZones([]*dal.Record{g}); err != nil {
					return nil, err
				}
				mapped[j] = g
				continue
			}
			merged := clone(geoMap[fmt.Sprint(get(g, "id"))])
			for _, k := range g.Keys() {
				v, _ := g.Get(k)
				merged.Set(k, v)
			}
			mapped[j] = merged
		}
		sz.Set("geo_zones", mapped)
	}
	if len(gzToDelete) > 0 {
		if _, err := s.Internal("GeoZone").Delete(ctx, gzToDelete); err != nil {
			return nil, err
		}
	}
	res, err := s.Internal("ServiceZone").Update(ctx, anyList(data))
	return rows(res), err
}

// UpsertServiceZones mirrors upsertServiceZones.
func (s *FulfillmentModuleService) UpsertServiceZones(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	// one transaction manager: collections loaded in the method are the
	// initialized ones its updates replace
	ctx = dal.WithSharedIdentityMap(ctx)
	var out []*dal.Record
	err := s.tx(ctx, func(t *FulfillmentModuleService) error {
		var forCreate, forUpdate []*dal.Record
		for _, d := range data {
			if hasID(d) {
				forUpdate = append(forUpdate, clone(d))
			} else {
				forCreate = append(forCreate, d)
			}
		}
		if len(forCreate) > 0 {
			c, err := t.CreateServiceZones(ctx, forCreate)
			if err != nil {
				return err
			}
			out = append(out, c...)
		}
		if len(forUpdate) > 0 {
			u, err := t.updateServiceZones(ctx, forUpdate)
			if err != nil {
				return err
			}
			out = append(out, u...)
		}
		return nil
	})
	return out, err
}

// UpdateShippingOptions mirrors updateShippingOptions(idOrSelector, data).
func (s *FulfillmentModuleService) UpdateShippingOptions(ctx context.Context, idOrSelector any, data *dal.Record) ([]*dal.Record, error) {
	// one transaction manager: collections loaded in the method are the
	// initialized ones its updates replace
	ctx = dal.WithSharedIdentityMap(ctx)
	var out []*dal.Record
	err := s.tx(ctx, func(t *FulfillmentModuleService) error {
		var in []*dal.Record
		withID := func(id any) *dal.Record {
			n := clone(data)
			r := dal.NewRecord()
			r.Set("id", id)
			for _, k := range n.Keys() {
				v, _ := n.Get(k)
				r.Set(k, v)
			}
			return r
		}
		if id, ok := idOrSelector.(string); ok {
			in = []*dal.Record{withID(id)}
		} else {
			sel, _ := idOrSelector.(*dal.Record)
			opts, err := t.Internal("ShippingOption").List(ctx, sel, utilsmodulessdk.FindConfig{})
			if err != nil {
				return err
			}
			for _, o := range opts {
				in = append(in, withID(get(o, "id")))
			}
		}
		var err error
		out, err = t.updateShippingOptions(ctx, in)
		return err
	})
	return out, err
}

func (s *FulfillmentModuleService) updateShippingOptions(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	if len(data) == 0 {
		return []*dal.Record{}, nil
	}
	f := dal.NewRecord()
	f.Set("id", ids(data))
	options, err := s.Internal("ShippingOption").List(ctx, f, utilsmodulessdk.FindConfig{Relations: []string{"rules", "type"}})
	if err != nil {
		return nil, err
	}
	if missing := difference(ids(data), idSet(options)); len(missing) > 0 {
		return nil, notFound("The following shipping options do not exist: " + strings.Join(missing, ", "))
	}
	optMap := map[string]*dal.Record{}
	for _, o := range options {
		optMap[fmt.Sprint(get(o, "id"))] = o
	}
	for _, so := range data {
		existing := optMap[fmt.Sprint(get(so, "id"))]
		if typeID := get(so, "type_id"); truthy(typeID) {
			so.Set("shipping_option_type_id", typeID)
			so.Delete("type_id")
		}
		if v, has := so.Get("rules"); !has || v == nil {
			continue
		}
		existingRules := records(get(existing, "rules"))
		ruleMap := map[string]*dal.Record{}
		for _, r := range existingRules {
			ruleMap[fmt.Sprint(get(r, "id"))] = r
		}
		var missing []string
		for _, r := range records(get(so, "rules")) {
			if hasID(r) {
				if _, ok := ruleMap[fmt.Sprint(get(r, "id"))]; !ok {
					missing = append(missing, fmt.Sprint(get(r, "id")))
				}
			}
		}
		if len(missing) > 0 {
			return nil, notFound(fmt.Sprintf("The following rules does not exists: %s on shipping option %v", strings.Join(missing, ", "), get(so, "id")))
		}
		var updatedRules []*dal.Record
		for _, r := range records(get(so, "rules")) {
			if !hasID(r) {
				continue
			}
			merged := dal.NewRecord()
			if ex, ok := ruleMap[fmt.Sprint(get(r, "id"))]; ok {
				for _, k := range ex.Keys() {
					v, _ := ex.Get(k)
					merged.Set(k, v)
				}
			}
			for _, k := range r.Keys() {
				if k == "created_at" || k == "updated_at" || k == "deleted_at" {
					continue
				}
				v, _ := r.Get(k)
				merged.Set(k, v)
			}
			ruleMap[fmt.Sprint(get(r, "id"))] = merged
			updatedRules = append(updatedRules, merged)
		}
		if err := utils.ValidateAndNormalizeRules(updatedRules); err != nil {
			return nil, err
		}
		given := records(get(so, "rules"))
		mapped := make([]any, len(given))
		for j, r := range given {
			if !hasID(r) {
				if err := utils.ValidateAndNormalizeRules([]*dal.Record{r}); err != nil {
					return nil, err
				}
				mapped[j] = r
				continue
			}
			mapped[j] = ruleMap[fmt.Sprint(get(r, "id"))]
		}
		so.Set("rules", mapped)
	}
	res, err := s.Internal("ShippingOption").Update(ctx, anyList(data))
	return rows(res), err
}

// UpsertShippingOptions mirrors upsertShippingOptions.
func (s *FulfillmentModuleService) UpsertShippingOptions(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	// one transaction manager: collections loaded in the method are the
	// initialized ones its updates replace
	ctx = dal.WithSharedIdentityMap(ctx)
	var out []*dal.Record
	err := s.tx(ctx, func(t *FulfillmentModuleService) error {
		var forCreate, forUpdate []*dal.Record
		for _, d := range data {
			if hasID(d) {
				forUpdate = append(forUpdate, clone(d))
			} else {
				forCreate = append(forCreate, d)
			}
		}
		if len(forCreate) > 0 {
			c, err := t.CreateShippingOptions(ctx, forCreate)
			if err != nil {
				return err
			}
			out = append(out, c...)
		}
		if len(forUpdate) > 0 {
			u, err := t.updateShippingOptions(ctx, forUpdate)
			if err != nil {
				return err
			}
			out = append(out, u...)
		}
		return nil
	})
	return out, err
}

// updateByIDOrSelector mirrors the update<X>(idOrSelector, data) overrides
// that retrieve a single id (throwing when missing) or list a selector.
func (s *FulfillmentModuleService) updateByIDOrSelector(ctx context.Context, className string, idOrSelector any, data *dal.Record) ([]*dal.Record, error) {
	var in []any
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
		if _, err := s.Internal(className).Retrieve(ctx, id, utilsmodulessdk.FindConfig{}); err != nil {
			return nil, err
		}
		in = []any{withID(id)}
	} else {
		sel, _ := idOrSelector.(*dal.Record)
		list, err := s.Internal(className).List(ctx, sel, utilsmodulessdk.FindConfig{})
		if err != nil {
			return nil, err
		}
		for _, r := range list {
			in = append(in, withID(get(r, "id")))
		}
	}
	if len(in) == 0 {
		return []*dal.Record{}, nil
	}
	res, err := s.Internal(className).Update(ctx, in)
	return rows(res), err
}

// UpdateShippingProfiles mirrors updateShippingProfiles.
func (s *FulfillmentModuleService) UpdateShippingProfiles(ctx context.Context, idOrSelector any, data *dal.Record) ([]*dal.Record, error) {
	return s.updateByIDOrSelector(ctx, "ShippingProfile", idOrSelector, data)
}

// UpdateShippingOptionTypes mirrors updateShippingOptionTypes.
func (s *FulfillmentModuleService) UpdateShippingOptionTypes(ctx context.Context, idOrSelector any, data *dal.Record) ([]*dal.Record, error) {
	return s.updateByIDOrSelector(ctx, "ShippingOptionType", idOrSelector, data)
}

// UpsertShippingProfiles mirrors upsertShippingProfiles.
func (s *FulfillmentModuleService) UpsertShippingProfiles(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var forCreate []*dal.Record
	var forUpdate []any
	for _, d := range data {
		if hasID(d) {
			forUpdate = append(forUpdate, d)
		} else {
			forCreate = append(forCreate, d)
		}
	}
	var out []*dal.Record
	if len(forCreate) > 0 {
		c, err := s.Internal("ShippingProfile").Create(ctx, forCreate)
		if err != nil {
			return nil, err
		}
		out = append(out, c...)
	}
	if len(forUpdate) > 0 {
		res, err := s.Internal("ShippingProfile").Update(ctx, forUpdate)
		if err != nil {
			return nil, err
		}
		out = append(out, rows(res)...)
	}
	return out, nil
}

// ValidateShippingProfileDeletion mirrors validateShippingProfileDeletion.
func (s *FulfillmentModuleService) ValidateShippingProfileDeletion(ctx context.Context, profileIDs []any) error {
	f := dal.NewRecord()
	f.Set("id", profileIDs)
	profiles, err := s.Internal("ShippingProfile").List(ctx, f, utilsmodulessdk.FindConfig{Relations: []string{"shipping_options"}})
	if err != nil {
		return err
	}
	var undeletable []string
	for _, p := range profiles {
		if len(records(get(p, "shipping_options"))) > 0 {
			undeletable = append(undeletable, fmt.Sprint(get(p, "id")))
		}
	}
	if len(undeletable) > 0 {
		return invalid(fmt.Sprintf("Cannot delete Shipping Profiles %s with associated Shipping Options. Delete Shipping Options first and try again.", strings.Join(undeletable, ",")))
	}
	return nil
}

// SoftDeleteShippingProfiles mirrors softDeleteShippingProfiles.
func (s *FulfillmentModuleService) SoftDeleteShippingProfiles(ctx context.Context, profileIDs []any) (*dal.CascadeResult, error) {
	if err := s.ValidateShippingProfileDeletion(ctx, profileIDs); err != nil {
		return nil, err
	}
	return s.Internal("ShippingProfile").SoftDelete(ctx, profileIDs)
}

// RetrieveFulfillmentOptions mirrors retrieveFulfillmentOptions: the
// registered provider's options.
func (s *FulfillmentModuleService) RetrieveFulfillmentOptions(ctx context.Context, providerID string) ([]any, error) {
	return s.provider.GetFulfillmentOptions(ctx, providerID)
}
