// Package services mirrors @medusajs/promotion/src/services.
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
	"github.com/hitkey-io/medusago/packages/modules/promotion/src/models"
)

// PromotionModuleService mirrors PromotionModuleService.
type PromotionModuleService struct {
	*utilsmodulessdk.MedusaService
}

// NewPromotionModuleService builds the service.
func NewPromotionModuleService(opts modulessdk.ServiceOptions, joinerConfig *joiner.ModuleJoinerConfig) (*PromotionModuleService, error) {
	return &PromotionModuleService{MedusaService: utilsmodulessdk.NewMedusaService(opts.DB, models.All, joinerConfig)}, nil
}

func (s *PromotionModuleService) with(tx *utilsmodulessdk.MedusaService) *PromotionModuleService {
	return &PromotionModuleService{MedusaService: tx}
}

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func str(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func copyRecord(r *dal.Record, skip ...string) *dal.Record {
	out := dal.NewRecord()
	if r == nil {
		return out
	}
outer:
	for _, k := range r.Keys() {
		for _, s := range skip {
			if s == k {
				continue outer
			}
		}
		v, _ := r.Get(k)
		out.Set(k, v)
	}
	return out
}

// isPresent mirrors isPresent.
func isPresent(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	case []any:
		return len(t) > 0
	case *dal.Record:
		return len(t.Keys()) > 0
	}
	return true
}

func ids(rows []*dal.Record) []any {
	out := make([]any, len(rows))
	for i, r := range rows {
		out[i] = get(r, "id")
	}
	return out
}

func (s *PromotionModuleService) listCampaigns(ctx context.Context, campaignIDs []any) ([]*dal.Record, error) {
	f := dal.NewRecord()
	f.Set("id", campaignIDs)
	return s.ListBySuffix(ctx, "Campaigns", f, utilsmodulessdk.FindConfig{Relations: []string{"budget", "promotions"}})
}

// validateCampaignBudgetData mirrors validateCampaignBudgetData.
func validateCampaignBudgetData(b *dal.Record) error {
	t := str(get(b, "type"))
	if t == "" {
		return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Campaign Budget type is a required field")
	}
	if t == "spend" && !isPresent(get(b, "currency_code")) {
		return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Campaign Budget type is a required field")
	}
	return nil
}

// CreateCampaigns mirrors createCampaigns.
func (s *PromotionModuleService) CreateCampaigns(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var created []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		var err error
		created, err = createCampaigns(ctx, tx, data)
		return err
	})
	if err != nil {
		return nil, err
	}
	return s.listCampaigns(ctx, ids(created))
}

// createCampaigns mirrors createCampaigns_: the campaigns and their budgets
// (the created campaign rows, nothing is read back).
func createCampaigns(ctx context.Context, tx *utilsmodulessdk.MedusaService, data []*dal.Record) ([]*dal.Record, error) {
	var created []*dal.Record
	err := func() error {
		budgets := map[string]*dal.Record{}
		campaigns := make([]*dal.Record, len(data))
		for i, d := range data {
			if b, _ := get(d, "budget").(*dal.Record); b != nil {
				budgets[str(get(d, "campaign_identifier"))] = b
			}
			campaigns[i] = copyRecord(d, "budget")
		}
		var err error
		created, err = tx.Internal("Campaign").Create(ctx, campaigns)
		if err != nil {
			return err
		}
		var budgetData []*dal.Record
		for _, c := range created {
			b := budgets[str(get(c, "campaign_identifier"))]
			if b == nil {
				continue
			}
			if err := validateCampaignBudgetData(b); err != nil {
				return err
			}
			n := copyRecord(b)
			n.Set("campaign", get(c, "id"))
			budgetData = append(budgetData, n)
		}
		if len(budgetData) > 0 {
			if _, err := tx.Internal("CampaignBudget").Create(ctx, budgetData); err != nil {
				return err
			}
		}
		return nil
	}()
	return created, err
}

// UpdateCampaigns mirrors updateCampaigns.
func (s *PromotionModuleService) UpdateCampaigns(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var updated []any
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		campaignIDs := make([]any, len(data))
		for i, d := range data {
			campaignIDs[i] = get(d, "id")
		}
		f := dal.NewRecord()
		f.Set("id", campaignIDs)
		existing, err := tx.Internal("Campaign").List(ctx, f, utilsmodulessdk.FindConfig{Relations: []string{"budget"}})
		if err != nil {
			return err
		}
		byID := map[string]*dal.Record{}
		for _, e := range existing {
			byID[str(get(e, "id"))] = e
		}
		var campaigns []any
		var updateBudgets []any
		var createBudgets []*dal.Record
		for _, d := range data {
			budget, _ := get(d, "budget").(*dal.Record)
			c := copyRecord(d, "budget")
			ex := byID[str(get(c, "id"))]
			campaigns = append(campaigns, c)
			exBudget, _ := get(ex, "budget").(*dal.Record)
			if (exBudget != nil && budget != nil && jsTruthy(get(budget, "type"))) || (budget != nil && jsTruthy(get(budget, "currency_code"))) {
				return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Campaign budget attributes (type, currency_code) are immutable")
			}
			if budget != nil {
				if exBudget != nil {
					u := dal.NewRecord()
					u.Set("id", get(exBudget, "id"))
					for _, k := range budget.Keys() {
						v, _ := budget.Get(k)
						u.Set(k, v)
					}
					updateBudgets = append(updateBudgets, u)
				} else {
					n := copyRecord(budget)
					n.Set("campaign", get(ex, "id"))
					createBudgets = append(createBudgets, n)
				}
			}
		}
		res, err := tx.Internal("Campaign").Update(ctx, campaigns)
		if err != nil {
			return err
		}
		switch t := res.(type) {
		case []*dal.Record:
			updated = ids(t)
		case *dal.Record:
			updated = []any{get(t, "id")}
		}
		if len(updateBudgets) > 0 {
			if _, err := tx.Internal("CampaignBudget").Update(ctx, updateBudgets); err != nil {
				return err
			}
		}
		if len(createBudgets) > 0 {
			if _, err := tx.Internal("CampaignBudget").Create(ctx, createBudgets); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.listCampaigns(ctx, updated)
}

func jsTruthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	case bool:
		return t
	case float64:
		return t != 0
	}
	return true
}

func difference(a []any, b []any) []string {
	have := map[string]bool{}
	for _, x := range b {
		have[str(x)] = true
	}
	var out []string
	for _, x := range a {
		if !have[str(x)] {
			out = append(out, str(x))
		}
	}
	return out
}

// AddPromotionsToCampaign mirrors addPromotionsToCampaign.
func (s *PromotionModuleService) AddPromotionsToCampaign(ctx context.Context, campaignID any, promotionIDs []any) ([]any, error) {
	var out []any
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		campaign, err := tx.Internal("Campaign").Retrieve(ctx, campaignID, utilsmodulessdk.FindConfig{})
		if err != nil {
			return err
		}
		f := dal.NewRecord()
		f.Set("id", promotionIDs)
		f.Set("campaign_id", nil)
		promotions, err := tx.Internal("Promotion").List(ctx, f, utilsmodulessdk.FindConfig{Relations: []string{"application_method"}})
		if err != nil {
			return err
		}
		// arrayDifference(found, requested): the found ids missing from the
		// request (always empty).
		if diff := difference(ids(promotions), promotionIDs); len(diff) > 0 {
			return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Cannot add promotions (%s) to campaign. These promotions are either already part of a campaign or not found.", strings.Join(diff, ",")))
		}
		budget, _ := get(campaign, "budget").(*dal.Record)
		if budget != nil && get(budget, "type") == "spend" {
			for _, p := range promotions {
				am, _ := get(p, "application_method").(*dal.Record)
				if get(am, "currency_code") != get(budget, "currency_code") {
					return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Cannot add promotions to campaign where currency_code don't match.")
				}
			}
		}
		updates := make([]any, len(promotions))
		for i, p := range promotions {
			u := dal.NewRecord()
			u.Set("id", get(p, "id"))
			u.Set("campaign_id", get(campaign, "id"))
			updates[i] = u
		}
		if len(updates) > 0 {
			if _, err := tx.Internal("Promotion").Update(ctx, updates); err != nil {
				return err
			}
		}
		out = ids(promotions)
		return nil
	})
	return out, err
}

// RemovePromotionsFromCampaign mirrors removePromotionsFromCampaign.
func (s *PromotionModuleService) RemovePromotionsFromCampaign(ctx context.Context, campaignID any, promotionIDs []any) ([]any, error) {
	var out []any
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		if _, err := tx.Internal("Campaign").Retrieve(ctx, campaignID, utilsmodulessdk.FindConfig{}); err != nil {
			return err
		}
		f := dal.NewRecord()
		f.Set("id", promotionIDs)
		promotions, err := tx.Internal("Promotion").List(ctx, f, utilsmodulessdk.FindConfig{})
		if err != nil {
			return err
		}
		if diff := difference(ids(promotions), promotionIDs); len(diff) > 0 {
			return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Promotions with ids (%s) not found.", strings.Join(diff, ",")))
		}
		updates := make([]any, len(promotions))
		for i, p := range promotions {
			u := dal.NewRecord()
			u.Set("id", get(p, "id"))
			u.Set("campaign_id", nil)
			updates[i] = u
		}
		if len(updates) > 0 {
			if _, err := tx.Internal("Promotion").Update(ctx, updates); err != nil {
				return err
			}
		}
		out = ids(promotions)
		return nil
	})
	return out, err
}
