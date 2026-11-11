package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/fulfillment/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func recs(v any) []*dal.Record {
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

func copyRecords(in []*dal.Record) []*dal.Record {
	out := make([]*dal.Record, len(in))
	for i, r := range in {
		n := dal.NewRecord()
		for _, k := range r.Keys() {
			v, _ := r.Get(k)
			n.Set(k, v)
		}
		out[i] = n
	}
	return out
}

// CreateShippingOptionsWorkflow mirrors createShippingOptionsWorkflow.
var CreateShippingOptionsWorkflow = composer.CreateWorkflow("create-shipping-options-workflow",
	func(e *composer.Execution, input []*dal.Record) ([]*dal.Record, error) {
		input = copyRecords(input)
		err := composer.Parallelize(
			func() error { _, err := steps.ValidateFulfillmentProvidersStep.Run(e, input); return err },
			func() error { _, err := steps.ValidateShippingOptionPricesStep.Run(e, input); return err },
		)
		if err != nil {
			return nil, err
		}
		created, err := steps.UpsertShippingOptionsStep.Run(e, input)
		if err != nil {
			return nil, err
		}
		prices := make([]steps.OptionPrices, len(input))
		for i, in := range input {
			id, _ := created[i].Get("id")
			v, _ := in.Get("prices")
			prices[i] = steps.OptionPrices{ID: id, Prices: recs(v), Defined: true}
		}
		links, err := steps.CreateShippingOptionsPriceSetsStep.Run(e, prices)
		if err != nil {
			return nil, err
		}
		events := make([]any, len(created))
		for i, c := range created {
			ev := dal.NewRecord()
			id, _ := c.Get("id")
			ev.Set("id", id)
			events[i] = ev
		}
		// parallel branches: the event step (no I/O) completes first upstream
		err = composer.Parallelize(
			func() error {
				_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "shipping-option.created", Data: events})
				return err
			},
			func() error { _, err := steps.SetShippingOptionsPriceSetsStep.Run(e, links); return err },
		)
		if err != nil {
			return nil, err
		}
		return created, nil
	})

// UpdateShippingOptionsWorkflow mirrors updateShippingOptionsWorkflow.
var UpdateShippingOptionsWorkflow = composer.CreateWorkflow("update-shipping-options-workflow",
	func(e *composer.Execution, input []*dal.Record) ([]*dal.Record, error) {
		input = copyRecords(input)
		err := composer.Parallelize(
			func() error { _, err := steps.ValidateFulfillmentProvidersStep.Run(e, input); return err },
			func() error { _, err := steps.ValidateShippingOptionPricesStep.Run(e, input); return err },
		)
		if err != nil {
			return nil, err
		}
		prices := make([]steps.OptionPrices, len(input))
		for i, in := range input {
			v, defined := in.Get("prices")
			in.Delete("prices")
			if t, _ := in.Get("price_type"); t == "calculated" {
				prices[i] = steps.OptionPrices{Prices: []*dal.Record{}, Defined: true}
			} else {
				prices[i] = steps.OptionPrices{Prices: recs(v), Defined: defined && v != nil}
			}
		}
		updated, err := steps.UpsertShippingOptionsStep.Run(e, input)
		if err != nil {
			return nil, err
		}
		ids := make([]any, len(updated))
		for i, u := range updated {
			ids[i], _ = u.Get("id")
		}
		for i := range prices {
			if i < len(updated) {
				prices[i].ID, _ = updated[i].Get("id")
			}
		}
		// parallel branches: the event step (no I/O) completes first upstream
		err = composer.Parallelize(
			func() error {
				_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "shipping-option.updated", Data: ids})
				return err
			},
			func() error { _, err := steps.SetShippingOptionsPricesStep.Run(e, prices); return err },
		)
		if err != nil {
			return nil, err
		}
		return updated, nil
	})

// DeleteShippingOptionsWorkflow mirrors deleteShippingOptionsWorkflow.
var DeleteShippingOptionsWorkflow = composer.CreateWorkflow("delete-shipping-options-workflow",
	func(e *composer.Execution, ids []any) (any, error) {
		deleted, err := steps.DeleteShippingOptionsStep.Run(e, ids)
		if err != nil {
			return nil, err
		}
		err = composer.Parallelize(
			func() error {
				_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "shipping-option.deleted", Data: ids})
				return err
			},
			func() error {
				var entries []*dal.Record
				if deleted != nil {
					entries = []*dal.Record{deleted}
				}
				_, err := commonsteps.RemoveRemoteLinkStep.Run(e, entries)
				return err
			},
		)
		return nil, err
	})

// BatchShippingOptionRulesInput mirrors BatchShippingOptionRulesInput.
type BatchShippingOptionRulesInput struct {
	Create []*dal.Record
	Update []*dal.Record
	Delete []any
}

// BatchShippingOptionRulesResult mirrors BatchShippingOptionRulesOutput.
type BatchShippingOptionRulesResult struct {
	Created []*dal.Record
	Updated []*dal.Record
	Deleted []any
}

// BatchShippingOptionRulesWorkflow mirrors batchShippingOptionRulesWorkflow.
var BatchShippingOptionRulesWorkflow = composer.CreateWorkflow("batch-shipping-option-rules",
	func(e *composer.Execution, in BatchShippingOptionRulesInput) (BatchShippingOptionRulesResult, error) {
		var out BatchShippingOptionRulesResult
		create, update, del := in.Create, in.Update, in.Delete
		if create == nil {
			create = []*dal.Record{}
		}
		if update == nil {
			update = []*dal.Record{}
		}
		if del == nil {
			del = []any{}
		}
		err := composer.Parallelize(
			func() (err error) {
				out.Created, err = steps.CreateShippingOptionRulesStep.Run(e, create)
				return err
			},
			func() (err error) {
				out.Updated, err = steps.UpdateShippingOptionRulesStep.Run(e, update)
				return err
			},
			func() (err error) {
				out.Deleted, err = steps.DeleteShippingOptionRulesStep.Run(e, del)
				return err
			},
		)
		return out, err
	})
