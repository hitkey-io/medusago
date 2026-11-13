// Package workflows mirrors @medusajs/core-flows/src/price-list/workflows.
package workflows

import (
	"fmt"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/price-list/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func idEvents(ids []any) []any {
	out := make([]any, len(ids))
	for i, id := range ids {
		ev := dal.NewRecord()
		ev.Set("id", id)
		out[i] = ev
	}
	return out
}

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func emit(e *composer.Execution, name string, data any) error {
	_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: name, Data: data})
	return err
}

// CreatePriceListsWorkflow mirrors createPriceListsWorkflow.
var CreatePriceListsWorkflow = composer.CreateWorkflow("create-price-lists",
	func(e *composer.Execution, data []*dal.Record) ([]*dal.Record, error) {
		m, err := steps.ValidateVariantPriceLinksStep.Run(e, data)
		if err != nil {
			return nil, err
		}
		lists, err := steps.CreatePriceListsStep.Run(e, steps.PriceListsStepInput{Data: data, VariantPriceMap: m})
		if err != nil {
			return nil, err
		}
		return lists, emit(e, "price-list.created", idEvents(commonsteps.RecordIDs(lists)))
	})

// UpdatePriceListsWorkflow mirrors updatePriceListsWorkflow.
var UpdatePriceListsWorkflow = composer.CreateWorkflow("update-price-lists",
	func(e *composer.Execution, data []*dal.Record) (any, error) {
		if _, err := steps.ValidatePriceListsStep.Run(e, data); err != nil {
			return nil, err
		}
		if _, err := steps.UpdatePriceListsStep.Run(e, data); err != nil {
			return nil, err
		}
		return nil, emit(e, "price-list.updated", idEvents(commonsteps.RecordIDs(data)))
	})

// DeletePriceListsWorkflow mirrors deletePriceListsWorkflow.
var DeletePriceListsWorkflow = composer.CreateWorkflow("delete-price-lists",
	func(e *composer.Execution, ids []any) (any, error) {
		if _, err := steps.DeletePriceListsStep.Run(e, ids); err != nil {
			return nil, err
		}
		keys := dal.NewRecord()
		keys.Set("price_list_id", ids)
		entry := dal.NewRecord()
		entry.Set(utilsmodulessdk.PRICING, keys)
		if _, err := commonsteps.RemoveRemoteLinkStep.Run(e, []*dal.Record{entry}); err != nil {
			return nil, err
		}
		return nil, emit(e, "price-list.deleted", idEvents(ids))
	})

// pricesEventData builds [{ id, price_set_ids }] of the lists with prices.
func pricesEventData(data []*dal.Record, m steps.VariantPriceMap) []any {
	out := []any{}
	for _, pl := range data {
		prices := records(get(pl, "prices"))
		if len(prices) == 0 {
			continue
		}
		var ids []any
		seen := map[string]bool{}
		for _, p := range prices {
			v := m[fmt.Sprint(get(p, "variant_id"))]
			k := fmt.Sprint(v)
			if !seen[k] {
				seen[k] = true
				ids = append(ids, v)
			}
		}
		ev := dal.NewRecord()
		ev.Set("id", get(pl, "id"))
		ev.Set("price_set_ids", ids)
		out = append(out, ev)
	}
	return out
}

func records(v any) []*dal.Record {
	switch t := v.(type) {
	case []*dal.Record:
		return t
	case []any:
		out := make([]*dal.Record, 0, len(t))
		for _, x := range t {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
}

// CreatePriceListPricesWorkflow mirrors createPriceListPricesWorkflow.
var CreatePriceListPricesWorkflow = composer.CreateWorkflow("create-price-list-prices",
	func(e *composer.Execution, data []*dal.Record) ([]*dal.Record, error) {
		if _, err := steps.ValidatePriceListsStep.Run(e, data); err != nil {
			return nil, err
		}
		m, err := steps.ValidateVariantPriceLinksStep.Run(e, data)
		if err != nil {
			return nil, err
		}
		prices, err := steps.CreatePriceListPricesStep.Run(e, steps.PriceListsStepInput{Data: data, VariantPriceMap: m})
		if err != nil {
			return nil, err
		}
		return prices, emit(e, "price-list.prices_added", pricesEventData(data, m))
	})

// UpdatePriceListPricesWorkflow mirrors updatePriceListPricesWorkflow.
var UpdatePriceListPricesWorkflow = composer.CreateWorkflow("update-price-list-prices",
	func(e *composer.Execution, data []*dal.Record) ([]*dal.Record, error) {
		if _, err := steps.ValidatePriceListsStep.Run(e, data); err != nil {
			return nil, err
		}
		m, err := steps.ValidateVariantPriceLinksStep.Run(e, data)
		if err != nil {
			return nil, err
		}
		prices, err := steps.UpdatePriceListPricesStep.Run(e, steps.PriceListsStepInput{Data: data, VariantPriceMap: m})
		if err != nil {
			return nil, err
		}
		return prices, emit(e, "price-list.prices_updated", pricesEventData(data, m))
	})

// RemovePriceListPricesWorkflow mirrors removePriceListPricesWorkflow.
var RemovePriceListPricesWorkflow = composer.CreateWorkflow("remove-price-list-prices",
	func(e *composer.Execution, ids []any) ([]any, error) {
		f := dal.NewRecord()
		f.Set("id", ids)
		q, err := commonsteps.UseQueryGraphStep.Config("get-price-list-prices-to-remove", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "price", Fields: []string{"id", "price_list_id", "price_set_id"}, Filters: f},
		})
		if err != nil {
			return nil, err
		}
		removed, err := steps.RemovePriceListPricesStep.Run(e, ids)
		if err != nil {
			return nil, err
		}
		var order []string
		byList := map[string][]any{}
		seen := map[string]bool{}
		rows, _ := q.Data.([]any)
		for _, r := range rows {
			p, _ := r.(*dal.Record)
			pl, ps := get(p, "price_list_id"), get(p, "price_set_id")
			if pl == nil || pl == "" || ps == nil || ps == "" {
				continue
			}
			k := fmt.Sprint(pl)
			if _, ok := byList[k]; !ok {
				order = append(order, k)
			}
			if !seen[k+"|"+fmt.Sprint(ps)] {
				seen[k+"|"+fmt.Sprint(ps)] = true
				byList[k] = append(byList[k], ps)
			}
		}
		events := []any{}
		for _, k := range order {
			ev := dal.NewRecord()
			ev.Set("id", k)
			ev.Set("price_set_ids", byList[k])
			events = append(events, ev)
		}
		return removed, emit(e, "price-list.prices_removed", events)
	})

// BatchPriceListPricesInput mirrors BatchPriceListPricesWorkflowDTO.
type BatchPriceListPricesInput struct {
	ID     any           `json:"id"`
	Create []*dal.Record `json:"create"`
	Update []*dal.Record `json:"update"`
	Delete []any         `json:"delete"`
}

// BatchPriceListPricesResult mirrors BatchPriceListPricesWorkflowResult.
type BatchPriceListPricesResult struct {
	Created []*dal.Record `json:"created"`
	Updated []*dal.Record `json:"updated"`
	Deleted []any         `json:"deleted"`
}

func listWithPrices(id any, prices []*dal.Record) []*dal.Record {
	r := dal.NewRecord()
	r.Set("id", id)
	ps := make([]any, len(prices))
	for i, p := range prices {
		ps[i] = p
	}
	r.Set("prices", ps)
	return []*dal.Record{r}
}

// BatchPriceListPricesWorkflow mirrors batchPriceListPricesWorkflow.
var BatchPriceListPricesWorkflow = composer.CreateWorkflow("batch-price-list-prices",
	func(e *composer.Execution, in BatchPriceListPricesInput) (BatchPriceListPricesResult, error) {
		var created, updated []*dal.Record
		var deleted []any
		err := composer.Parallelize(
			func() (err error) {
				created, err = CreatePriceListPricesWorkflow.RunAsStep(e, listWithPrices(in.ID, in.Create))
				return err
			},
			func() (err error) {
				updated, err = UpdatePriceListPricesWorkflow.RunAsStep(e, listWithPrices(in.ID, in.Update))
				return err
			},
			func() (err error) {
				deleted, err = RemovePriceListPricesWorkflow.RunAsStep(e, in.Delete)
				return err
			},
		)
		if err != nil {
			return BatchPriceListPricesResult{}, err
		}
		return BatchPriceListPricesResult{Created: created, Updated: updated, Deleted: deleted}, nil
	})

// The Go bodies take one property of the upstream input object.
func init() {
	CreatePriceListsWorkflow.InputFrom("price_lists_data")
	UpdatePriceListsWorkflow.InputFrom("price_lists_data")
	DeletePriceListsWorkflow.InputFrom("ids")
	CreatePriceListPricesWorkflow.InputFrom("data")
	UpdatePriceListPricesWorkflow.InputFrom("data")
	RemovePriceListPricesWorkflow.InputFrom("ids")
	BatchPriceListPricesWorkflow.InputFrom("data")
}
