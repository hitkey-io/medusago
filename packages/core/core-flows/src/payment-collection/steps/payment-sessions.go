package steps

import (
	"context"
	"fmt"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// PaymentService is the payment module surface used by the steps.
type PaymentService interface {
	commonsteps.ModuleService
	CreatePaymentCollections(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
	CreatePaymentSession(ctx context.Context, collectionID string, input *dal.Record) (*dal.Record, error)
	DeletePaymentSession(ctx context.Context, id any) error
	CreateAccountHolder(ctx context.Context, input *dal.Record) (*dal.Record, error)
	UpdatePaymentSession(ctx context.Context, data *dal.Record) (*dal.Record, error)
}

// PaymentOf resolves the payment module service.
func PaymentOf(c composer.Container) PaymentService {
	s, _ := c.Resolve(utilsmodulessdk.PAYMENT).(PaymentService)
	return s
}

func g(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

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

// ValidatePaymentProviderInRegionStep mirrors validatePaymentProviderInRegionStep.
var ValidatePaymentProviderInRegionStep = composer.CreateStep("validate-payment-provider-in-region",
	func(ctx composer.StepExecutionContext, in struct {
		CollectionID string
		ProviderID   string
	}) (composer.StepResponse[any], error) {
		q := commonsteps.QueryOf(ctx.Container)
		f := dal.NewRecord()
		f.Set("id", in.CollectionID)
		res, err := q.Graph(ctx.Context, query.GraphInput{Entity: "payment_collection", Fields: []string{"cart.id", "cart.region_id", "cart.completed_at"}, Filters: f}, joiner.Options{})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		var cart *dal.Record
		if list := recs(res.Data); len(list) > 0 {
			cart, _ = g(list[0], "cart").(*dal.Record)
		}
		regionID, _ := g(cart, "region_id").(string)
		if regionID == "" || g(cart, "completed_at") != nil {
			return composer.NewStepResponse[any](nil), nil
		}
		rf := dal.NewRecord()
		rf.Set("id", regionID)
		rr, err := q.Graph(ctx.Context, query.GraphInput{Entity: "region", Fields: []string{"payment_providers.id"}, Filters: rf}, joiner.Options{})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		enabled := map[any]bool{}
		if list := recs(rr.Data); len(list) > 0 {
			for _, p := range recs(g(list[0], "payment_providers")) {
				enabled[g(p, "id")] = true
			}
		}
		if !enabled[in.ProviderID] {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Payment provider %s is not enabled in the cart's region %s.", in.ProviderID, regionID))
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// ValidatePaymentSessionsDeletableStep mirrors validatePaymentSessionsDeletableStep.
var ValidatePaymentSessionsDeletableStep = composer.CreateStep("validate-payment-sessions-deletable",
	func(ctx composer.StepExecutionContext, sessions []*dal.Record) (composer.StepResponse[any], error) {
		for _, s := range sessions {
			status, _ := g(s, "status").(string)
			if status == "pending_authorization" {
				return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, fmt.Sprintf("Payment session with id: %v can't be replaced because its status is %s", g(s, "id"), status))
			}
			if status != "authorized" && status != "captured" {
				continue
			}
			payment, _ := g(s, "payment").(*dal.Record)
			if g(payment, "canceled_at") != nil {
				continue
			}
			refunded := totals.Convert(0)
			for _, r := range recs(g(payment, "refunds")) {
				refunded = totals.Add(refunded, g(r, "raw_amount"))
			}
			amount := g(payment, "raw_amount")
			if amount == nil {
				amount = g(payment, "amount")
			}
			if g(payment, "captured_at") == nil || totals.Lt(refunded, amount) {
				return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, fmt.Sprintf("Payment session with id: %v can't be replaced because it has an authorized or captured payment", g(s, "id")))
			}
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// DeletePaymentSessionsStep mirrors deletePaymentSessionsStep.
var DeletePaymentSessionsStep = composer.CreateStep("delete-payment-sessions",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[[]any], error) {
		if len(ids) == 0 {
			return composer.NewStepResponse([]any{}, any(nil)), nil
		}
		svc := PaymentOf(ctx.Container)
		f := dal.NewRecord()
		f.Set("id", ids)
		sessions, err := svc.ListBySuffix(ctx.Context, "PaymentSessions", f, utilsmodulessdk.FindConfig{Select: []string{"provider_id", "currency_code", "amount", "data", "context", "payment_collection.id"}})
		if err != nil {
			return composer.StepResponse[[]any]{}, err
		}
		byID := map[any]*dal.Record{}
		for _, s := range sessions {
			byID[g(s, "id")] = s
		}
		var deleted []*dal.Record
		var deletedIDs []any
		for _, id := range ids {
			if err := svc.DeletePaymentSession(ctx.Context, id); err == nil {
				deleted = append(deleted, byID[id])
				deletedIDs = append(deletedIDs, id)
			}
		}
		if deletedIDs == nil {
			deletedIDs = []any{}
		}
		return composer.NewStepResponse(deletedIDs, any(deleted)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		deleted, _ := data.([]*dal.Record)
		svc := PaymentOf(ctx.Container)
		for _, s := range deleted {
			pc, _ := g(s, "payment_collection").(*dal.Record)
			id, _ := g(pc, "id").(string)
			if id == "" {
				continue
			}
			p := dal.NewRecord()
			p.Set("provider_id", g(s, "provider_id"))
			p.Set("currency_code", g(s, "currency_code"))
			p.Set("amount", g(s, "amount"))
			d := g(s, "data")
			if d == nil {
				d = dal.NewRecord()
			}
			p.Set("data", d)
			p.Set("context", g(s, "context"))
			_, _ = svc.CreatePaymentSession(ctx.Context, id, p)
		}
		return nil
	})

// CreatePaymentSessionStep mirrors createPaymentSessionStep.
var CreatePaymentSessionStep = composer.CreateStep("create-payment-session",
	func(ctx composer.StepExecutionContext, in *dal.Record) (composer.StepResponse[*dal.Record], error) {
		p := dal.NewRecord()
		p.Set("provider_id", g(in, "provider_id"))
		p.Set("currency_code", g(in, "currency_code"))
		p.Set("amount", g(in, "amount"))
		data := g(in, "data")
		if data == nil {
			data = dal.NewRecord()
		}
		p.Set("data", data)
		p.Set("context", g(in, "context"))
		meta := g(in, "metadata")
		if meta == nil {
			meta = dal.NewRecord()
		}
		p.Set("metadata", meta)
		id, _ := g(in, "payment_collection_id").(string)
		session, err := PaymentOf(ctx.Container).CreatePaymentSession(ctx.Context, id, p)
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		return composer.NewStepResponse(session, g(session, "id")), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		if data == nil {
			return nil
		}
		return PaymentOf(ctx.Container).DeletePaymentSession(ctx.Context, data)
	})

// CreatePaymentAccountHolderStep mirrors createPaymentAccountHolderStep.
var CreatePaymentAccountHolderStep = composer.CreateStep("create-payment-account-holder",
	func(ctx composer.StepExecutionContext, in *dal.Record) (composer.StepResponse[*dal.Record], error) {
		holder, err := PaymentOf(ctx.Container).CreateAccountHolder(ctx.Context, in)
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		return composer.NewStepResponse(holder), nil
	}, nil)

// CreatePaymentCollectionsStep mirrors createPaymentCollectionsStep (cart).
var CreatePaymentCollectionsStep = composer.CreateStep("create-payment-collections",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		if len(data) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, any([]any{})), nil
		}
		created, err := PaymentOf(ctx.Container).CreatePaymentCollections(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(created, any(commonsteps.RecordIDs(created))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		_, err := PaymentOf(ctx.Container).Internal("PaymentCollection").Delete(ctx.Context, ids)
		return err
	})

// ValidatePaymentSessionUpdateInput mirrors ValidatePaymentSessionUpdateStepInput.
type ValidatePaymentSessionUpdateInput struct {
	PaymentCollectionID any
	PaymentSessionID    any
	PaymentSession      *dal.Record
}

// ValidatePaymentSessionUpdateStep mirrors validatePaymentSessionUpdateStep.
var ValidatePaymentSessionUpdateStep = composer.CreateStep("validate-payment-session-update",
	func(ctx composer.StepExecutionContext, in ValidatePaymentSessionUpdateInput) (composer.StepResponse[any], error) {
		session := in.PaymentSession
		if session == nil || g(session, "payment_collection_id") != in.PaymentCollectionID {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND,
				fmt.Sprintf("Payment session with id: %v was not found in payment collection %v", in.PaymentSessionID, in.PaymentCollectionID))
		}
		switch g(session, "status") {
		case "pending", "requires_more", "error":
		default:
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED,
				fmt.Sprintf("Payment session with id: %v can't be updated because its status is %v", g(session, "id"), g(session, "status")))
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// UpdatePaymentSessionStep mirrors updatePaymentSessionStep.
var UpdatePaymentSessionStep = composer.CreateStep("update-payment-session",
	func(ctx composer.StepExecutionContext, in *dal.Record) (composer.StepResponse[*dal.Record], error) {
		svc := PaymentOf(ctx.Container)
		prev, err := svc.Internal("PaymentSession").Retrieve(ctx.Context, g(in, "id"), utilsmodulessdk.FindConfig{Select: []string{"id", "amount", "currency_code", "data"}})
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		u := dal.NewRecord()
		for _, k := range []string{"id", "amount", "currency_code", "data"} {
			u.Set(k, g(in, k))
		}
		session, err := svc.UpdatePaymentSession(ctx.Context, u)
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		return composer.NewStepResponse(session, any(prev)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		prev, _ := data.(*dal.Record)
		if prev == nil {
			return nil
		}
		_, err := PaymentOf(ctx.Container).UpdatePaymentSession(ctx.Context, prev)
		return err
	})
