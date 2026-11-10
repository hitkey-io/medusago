package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	pcsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/payment-collection/steps"
	pcworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/payment-collection/workflows"
	paymentsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/payment/steps"
	paymentworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/payment/workflows"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func queryOne(e *composer.Execution, name, entity string, id any, fields []string) (*dal.Record, error) {
	f := dal.NewRecord()
	f.Set("id", id)
	isList := false
	res, err := commonsteps.UseQueryGraphStep.Config(name, composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
		GraphInput: query.GraphInput{Entity: entity, Fields: fields, Filters: f},
		Options:    joiner.Options{ThrowIfKeyNotFound: true},
		IsList:     &isList,
	})
	if err != nil {
		return nil, err
	}
	return rec(res.Data), nil
}

// CreateOrderPaymentCollectionWorkflow mirrors createOrderPaymentCollectionWorkflow.
var CreateOrderPaymentCollectionWorkflow = composer.CreateWorkflow("create-order-payment-collection",
	func(e *composer.Execution, in *dal.Record) ([]*dal.Record, error) {
		order, err := queryOne(e, "get-order", "order", g(in, "order_id"), []string{"id", "summary", "total", "currency_code", "region_id"})
		if err != nil {
			return nil, err
		}
		data := dal.NewRecord()
		data.Set("currency_code", g(order, "currency_code"))
		data.Set("amount", g(in, "amount"))
		created, err := pcsteps.CreatePaymentCollectionsStep.Run(e, []*dal.Record{data})
		if err != nil {
			return nil, err
		}
		o := dal.NewRecord()
		o.Set("order_id", g(order, "id"))
		p := dal.NewRecord()
		p.Set("payment_collection_id", g(created[0], "id"))
		link := dal.NewRecord()
		link.Set(utilsmodulessdk.ORDER, o)
		link.Set(utilsmodulessdk.PAYMENT, p)
		if _, err := commonsteps.CreateRemoteLinkStep.Config("order-payment-collection-link", composer.StepOptions{}).Run(e, []*dal.Record{link}); err != nil {
			return nil, err
		}
		return created, nil
	})

var throwUnlessPaymentCollectionNotPaid = composer.CreateStep("validate-existing-payment-collection",
	func(ctx composer.StepExecutionContext, pc *dal.Record) (composer.StepResponse[any], error) {
		if g(pc, "status") != "not_paid" {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Can only mark 'not_paid' payment collection as paid")
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// MarkPaymentCollectionAsPaid mirrors markPaymentCollectionAsPaid.
var MarkPaymentCollectionAsPaid = composer.CreateWorkflow("mark-payment-collection-as-paid",
	func(e *composer.Execution, in *dal.Record) (*dal.Record, error) {
		pc, err := queryOne(e, "get-payment-collection", "payment_collection", g(in, "payment_collection_id"), []string{"id", "status", "amount"})
		if err != nil {
			return nil, err
		}
		if _, err := throwUnlessPaymentCollectionNotPaid.Run(e, pc); err != nil {
			return nil, err
		}
		provider, _ := g(in, "provider_id").(string)
		if provider == "" {
			provider = "pp_system_default"
		}
		id, _ := g(pc, "id").(string)
		session, err := pcworkflows.CreatePaymentSessionsWorkflow.RunAsStep(e, pcworkflows.CreatePaymentSessionsInput{PaymentCollectionID: id, ProviderID: provider, Data: dal.NewRecord(), Context: dal.NewRecord()})
		if err != nil {
			return nil, err
		}
		sessionID, _ := g(session, "id").(string)
		payment, err := paymentsteps.AuthorizePaymentSessionStep.Run(e, paymentsteps.AuthorizePaymentSessionStepInput{ID: sessionID})
		if err != nil {
			return nil, err
		}
		cap := dal.NewRecord()
		cap.Set("payment_id", g(payment, "id"))
		cap.Set("captured_by", g(in, "captured_by"))
		cap.Set("amount", g(pc, "amount"))
		if _, err := paymentworkflows.CapturePaymentWorkflow.RunAsStep(e, cap); err != nil {
			return nil, err
		}
		return payment, nil
	})

var throwUnlessStatusIsNotPaid = composer.CreateStep("validate-payment-collection",
	func(ctx composer.StepExecutionContext, pc *dal.Record) (composer.StepResponse[any], error) {
		if g(pc, "status") != "not_paid" {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Can only delete payment collections where status is not_paid")
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// DeleteOrderPaymentCollections mirrors deleteOrderPaymentCollections.
var DeleteOrderPaymentCollections = composer.CreateWorkflow("delete-order-payment-collectionworkflow",
	func(e *composer.Execution, id string) (any, error) {
		pc, err := queryOne(e, "payment-collection-query", "payment_collection", id, []string{"id", "status"})
		if err != nil {
			return nil, err
		}
		if _, err := throwUnlessStatusIsNotPaid.Run(e, pc); err != nil {
			return nil, err
		}
		p := dal.NewRecord()
		p.Set("payment_collection_id", id)
		entry := dal.NewRecord()
		entry.Set(utilsmodulessdk.PAYMENT, p)
		_, err = commonsteps.RemoveRemoteLinkStep.Run(e, []*dal.Record{entry})
		return nil, err
	})
