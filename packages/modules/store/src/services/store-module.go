// Package services mirrors @medusajs/store/src/services.
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
	"github.com/hitkey-io/medusago/packages/modules/store/src/models"
)

// StoreModuleService mirrors StoreModuleService.
type StoreModuleService struct {
	*utilsmodulessdk.MedusaService
}

// NewStoreModuleService builds the service.
func NewStoreModuleService(opts modulessdk.ServiceOptions, joinerConfig *joiner.ModuleJoinerConfig) (*StoreModuleService, error) {
	return &StoreModuleService{MedusaService: utilsmodulessdk.NewMedusaService(opts.DB, models.All, joinerConfig)}, nil
}

var storeRelations = []string{"supported_currencies", "supported_locales"}

func invalid(msg string) error {
	return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, msg)
}

func listOf(v any) []*dal.Record {
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

// normalizeInput mirrors normalizeInput: currency codes lower cased, name
// trimmed, undefined removed.
func normalizeInput(stores []*dal.Record) []*dal.Record {
	out := make([]*dal.Record, len(stores))
	for i, s := range stores {
		n := dal.NewRecord()
		for _, k := range s.Keys() {
			v, _ := s.Get(k)
			switch k {
			case "supported_currencies":
				items := listOf(v)
				if items == nil {
					continue
				}
				cs := make([]any, len(items))
				for j, c := range items {
					cc := dal.NewRecord()
					for _, ck := range c.Keys() {
						cv, _ := c.Get(ck)
						cc.Set(ck, cv)
					}
					if code, ok := cc.Get("currency_code"); ok {
						cc.Set("currency_code", strings.ToLower(fmt.Sprint(code)))
					}
					cs[j] = cc
				}
				v = cs
			case "name":
				str, ok := v.(string)
				if !ok {
					continue
				}
				v = strings.TrimSpace(str)
			}
			n.Set(k, v)
		}
		out[i] = n
	}
	return out
}

func duplicates(items []string) []string {
	seen := map[string]bool{}
	added := map[string]bool{}
	var out []string
	for _, i := range items {
		if seen[i] && !added[i] {
			out = append(out, i)
			added[i] = true
		}
		seen[i] = true
	}
	return out
}

func validateUnique(items []string, field string) error {
	if d := duplicates(items); len(d) > 0 {
		return invalid(fmt.Sprintf("Duplicate %s codes: %s", field, strings.Join(d, ", ")))
	}
	return nil
}

func validateCreateRequest(stores []*dal.Record) error {
	for _, s := range stores {
		v, _ := s.Get("supported_currencies")
		if currencies := listOf(v); len(currencies) > 0 {
			codes := make([]string, len(currencies))
			for i, c := range currencies {
				cv, _ := c.Get("currency_code")
				codes[i] = fmt.Sprint(cv)
			}
			if err := validateUnique(codes, "currency"); err != nil {
				return err
			}
			seenDefault := false
			for _, c := range currencies {
				if d, _ := c.Get("is_default"); d == true {
					if seenDefault {
						return invalid("Only one default currency is allowed")
					}
					seenDefault = true
				}
			}
			if !seenDefault {
				return invalid("There should be a default currency set for the store")
			}
		}
		lv, _ := s.Get("supported_locales")
		if locales := listOf(lv); len(locales) > 0 {
			codes := make([]string, len(locales))
			for i, l := range locales {
				c, _ := l.Get("locale_code")
				codes[i] = fmt.Sprint(c)
			}
			if err := validateUnique(codes, "locale"); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *StoreModuleService) write(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	normalized := normalizeInput(data)
	if err := validateCreateRequest(normalized); err != nil {
		return nil, err
	}
	return s.Internal("Store").UpsertWithReplace(ctx, normalized, storeRelations)
}

// CreateStores mirrors createStores.
func (s *StoreModuleService) CreateStores(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		var err error
		out, err = (&StoreModuleService{MedusaService: tx}).write(ctx, data)
		return err
	})
	return out, err
}

// UpsertStores mirrors upsertStores.
func (s *StoreModuleService) UpsertStores(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := &StoreModuleService{MedusaService: tx}
		var forCreate, forUpdate []*dal.Record
		for _, d := range data {
			if id, _ := d.Get("id"); id != nil && id != "" {
				forUpdate = append(forUpdate, d)
			} else {
				forCreate = append(forCreate, d)
			}
		}
		for _, part := range [][]*dal.Record{forCreate, forUpdate} {
			if len(part) == 0 {
				continue
			}
			r, err := t.write(ctx, part)
			if err != nil {
				return err
			}
			out = append(out, r...)
		}
		return nil
	})
	return out, err
}

// UpdateStores mirrors updateStores(idOrSelector, data).
func (s *StoreModuleService) UpdateStores(ctx context.Context, idOrSelector any, data *dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := &StoreModuleService{MedusaService: tx}
		withID := func(id any) *dal.Record {
			r := dal.NewRecord()
			r.Set("id", id)
			for _, k := range data.Keys() {
				v, _ := data.Get(k)
				r.Set(k, v)
			}
			return r
		}
		var input []*dal.Record
		if id, ok := idOrSelector.(string); ok {
			input = []*dal.Record{withID(id)}
		} else {
			sel, _ := idOrSelector.(*dal.Record)
			rows, err := t.Internal("Store").List(ctx, sel, utilsmodulessdk.FindConfig{})
			if err != nil {
				return err
			}
			for _, r := range rows {
				id, _ := r.Get("id")
				input = append(input, withID(id))
			}
		}
		var err error
		out, err = t.write(ctx, input)
		return err
	})
	return out, err
}
