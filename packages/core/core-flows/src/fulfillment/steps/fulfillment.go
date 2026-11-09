// Package steps mirrors @medusajs/core-flows/src/fulfillment/steps.
package steps

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/modules/fulfillment/src/services"
)

// Fulfillment resolves the fulfillment module service.
func Fulfillment(c composer.Container) *services.FulfillmentModuleService {
	s, _ := c.Resolve(utilsmodulessdk.FULFILLMENT).(*services.FulfillmentModuleService)
	return s
}

func createStep(name, className string, create func(ctx composer.StepExecutionContext, data []*dal.Record) ([]*dal.Record, error)) *composer.Step[[]*dal.Record, []*dal.Record] {
	return composer.CreateStep(name,
		func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
			created, err := create(ctx, data)
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
			_, err := Fulfillment(ctx.Container).Internal(className).Delete(ctx.Context, ids)
			return err
		})
}

// CreateFulfillmentSets mirrors createFulfillmentSets (step).
var CreateFulfillmentSets = createStep("create-fulfillment-sets", "FulfillmentSet", func(ctx composer.StepExecutionContext, data []*dal.Record) ([]*dal.Record, error) {
	return Fulfillment(ctx.Container).CreateFulfillmentSets(ctx.Context, data)
})

// CreateServiceZonesStep mirrors createServiceZonesStep.
var CreateServiceZonesStep = createStep("create-service-zones", "ServiceZone", func(ctx composer.StepExecutionContext, data []*dal.Record) ([]*dal.Record, error) {
	return Fulfillment(ctx.Container).CreateServiceZones(ctx.Context, data)
})

// CreateShippingProfilesStep mirrors createShippingProfilesStep.
var CreateShippingProfilesStep = createStep("create-shipping-profiles", "ShippingProfile", func(ctx composer.StepExecutionContext, data []*dal.Record) ([]*dal.Record, error) {
	if len(data) == 0 {
		return []*dal.Record{}, nil
	}
	return Fulfillment(ctx.Container).Internal("ShippingProfile").Create(ctx.Context, data)
})

// DeleteFulfillmentSetsStep mirrors deleteFulfillmentSetsStep.
var DeleteFulfillmentSetsStep = commonsteps.SoftDeleteRecordsStep("delete-fulfillment-sets", utilsmodulessdk.FULFILLMENT, "FulfillmentSet")

// DeleteServiceZonesStep mirrors deleteServiceZonesStep.
var DeleteServiceZonesStep = commonsteps.SoftDeleteRecordsStep("delete-service-zones", utilsmodulessdk.FULFILLMENT, "ServiceZone")

// DeleteShippingProfilesStep mirrors deleteShippingProfilesStep (with the
// module's deletion validation).
var DeleteShippingProfilesStep = composer.CreateStep("delete-shipping-profile",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[any], error) {
		if _, err := Fulfillment(ctx.Container).SoftDeleteShippingProfiles(ctx.Context, ids); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(ids)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		_, err := Fulfillment(ctx.Container).Internal("ShippingProfile").Restore(ctx.Context, ids)
		return err
	})

// SelectorUpdateInput is { selector, update }.
type SelectorUpdateInput struct {
	Selector *dal.Record
	Update   *dal.Record
}

// UpdateServiceZonesStep mirrors updateServiceZonesStep.
var UpdateServiceZonesStep = composer.CreateStep("update-service-zones",
	func(ctx composer.StepExecutionContext, in SelectorUpdateInput) (composer.StepResponse[[]*dal.Record], error) {
		svc := Fulfillment(ctx.Container)
		prev, err := commonsteps.PrevDataFor(ctx.Context, svc, "ServiceZones", in.Selector, in.Update)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		out, err := svc.UpdateServiceZones(ctx.Context, in.Selector, in.Update)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(out, any(prev)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		prev, _ := data.([]*dal.Record)
		if len(prev) == 0 {
			return nil
		}
		_, err := Fulfillment(ctx.Container).UpsertServiceZones(ctx.Context, prev)
		return err
	})

// UpdateShippingProfilesStep mirrors updateShippingProfilesStep.
var UpdateShippingProfilesStep = composer.CreateStep("update-shipping-profiles",
	func(ctx composer.StepExecutionContext, in SelectorUpdateInput) (composer.StepResponse[[]*dal.Record], error) {
		svc := Fulfillment(ctx.Container)
		prev, err := commonsteps.PrevDataFor(ctx.Context, svc, "ShippingProfiles", in.Selector, in.Update)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		out, err := svc.UpdateShippingProfiles(ctx.Context, in.Selector, in.Update)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(out, any(prev)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		prev, _ := data.([]*dal.Record)
		if len(prev) == 0 {
			return nil
		}
		_, err := Fulfillment(ctx.Container).UpsertShippingProfiles(ctx.Context, prev)
		return err
	})
