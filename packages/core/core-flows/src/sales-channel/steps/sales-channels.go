package steps

import (
	"context"
	"fmt"
	"strings"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	modulessdk "github.com/hitkey-io/medusago/packages/core/modules-sdk/src"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// moduleService is the generated module service surface used here.
type moduleService interface {
	Internal(className string) *utilsmodulessdk.InternalService
	ListBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error)
}

func salesChannels(c composer.Container) moduleService {
	s, _ := c.Resolve(utilsmodulessdk.SALES_CHANNEL).(moduleService)
	return s
}

func ids(rs []*dal.Record) []any {
	out := make([]any, len(rs))
	for i, r := range rs {
		out[i], _ = r.Get("id")
	}
	return out
}

// CreateSalesChannelsStep mirrors createSalesChannelsStep.
var CreateSalesChannelsStep = composer.CreateStep("create-sales-channels",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		created, err := salesChannels(ctx.Container).Internal("SalesChannel").Create(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(created, any(ids(created))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		list, _ := data.([]any)
		if list == nil {
			return nil
		}
		_, err := salesChannels(ctx.Container).Internal("SalesChannel").Delete(ctx.Context, list)
		return err
	})

// UpdateSalesChannelsStepInput mirrors UpdateSalesChannelsStepInput.
type UpdateSalesChannelsStepInput struct {
	Selector *dal.Record
	Update   *dal.Record
}

// UpdateSalesChannels mirrors updateSalesChannels(selector, data).
func UpdateSalesChannels(ctx context.Context, svc moduleService, selector, data *dal.Record) ([]*dal.Record, error) {
	rows, err := svc.Internal("SalesChannel").List(ctx, selector, utilsmodulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	in := make([]any, len(rows))
	for i, r := range rows {
		n := dal.NewRecord()
		id, _ := r.Get("id")
		n.Set("id", id)
		for _, k := range data.Keys() {
			v, _ := data.Get(k)
			n.Set(k, v)
		}
		in[i] = n
	}
	if len(in) == 0 {
		return []*dal.Record{}, nil
	}
	res, err := svc.Internal("SalesChannel").Update(ctx, in)
	if err != nil {
		return nil, err
	}
	if r, ok := res.(*dal.Record); ok {
		return []*dal.Record{r}, nil
	}
	out, _ := res.([]*dal.Record)
	return out, nil
}

// UpdateSalesChannelsStep mirrors updateSalesChannelsStep.
var UpdateSalesChannelsStep = composer.CreateStep("update-sales-channels",
	func(ctx composer.StepExecutionContext, in UpdateSalesChannelsStepInput) (composer.StepResponse[[]*dal.Record], error) {
		svc := salesChannels(ctx.Container)
		selects, relations := common.GetSelectsAndRelationsFromObjectArray([]common.ObjectLike{in.Update}, common.SelectsAndRelationsOptions{}, "")
		prev, err := svc.ListBySuffix(ctx.Context, "SalesChannels", in.Selector, utilsmodulessdk.FindConfig{Select: selects, Relations: relations})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		out, err := UpdateSalesChannels(ctx.Context, svc, in.Selector, in.Update)
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
		in := make([]*dal.Record, len(prev))
		for i, r := range prev {
			n := dal.NewRecord()
			for _, k := range []string{"id", "name", "description", "is_disabled", "metadata"} {
				if v, ok := r.Get(k); ok {
					n.Set(k, v)
				}
			}
			in[i] = n
		}
		_, err := salesChannels(ctx.Container).Internal("SalesChannel").Upsert(ctx.Context, in)
		return err
	})

// CanDeleteSalesChannelsOrThrowStep mirrors canDeleteSalesChannelsOrThrowStep.
var CanDeleteSalesChannelsOrThrowStep = composer.CreateStep("can-delete-sales-channels-or-throw-step",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[bool], error) {
		store, _ := ctx.Container.Resolve(utilsmodulessdk.STORE).(moduleService)
		f := dal.NewRecord()
		f.Set("default_sales_channel_id", ids)
		stores, err := store.ListBySuffix(ctx.Context, "Stores", f, utilsmodulessdk.FindConfig{Select: []string{"default_sales_channel_id"}})
		if err != nil {
			return composer.StepResponse[bool]{}, err
		}
		if len(stores) > 0 {
			defaults := make([]string, len(stores))
			for i, s := range stores {
				v, _ := s.Get("default_sales_channel_id")
				defaults[i] = fmt.Sprint(v)
			}
			return composer.StepResponse[bool]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Cannot delete default sales channels: "+strings.Join(defaults, ", "))
		}
		return composer.NewStepResponse(true), nil
	}, nil)

// DeleteSalesChannelsStep mirrors deleteSalesChannelsStep.
var DeleteSalesChannelsStep = composer.CreateStep("delete-sales-channels",
	func(ctx composer.StepExecutionContext, list []any) (composer.StepResponse[any], error) {
		if _, err := salesChannels(ctx.Container).Internal("SalesChannel").SoftDelete(ctx.Context, list); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(list)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		list, _ := data.([]any)
		if len(list) == 0 {
			return nil
		}
		_, err := salesChannels(ctx.Container).Internal("SalesChannel").Restore(ctx.Context, list)
		return err
	})

// DetachProductsFromSalesChannelsStep mirrors detachProductsFromSalesChannelsStep.
var DetachProductsFromSalesChannelsStep = composer.CreateStep("detach-products-from-sales-channels-step",
	func(ctx composer.StepExecutionContext, links []ProductChannelLink) (composer.StepResponse[any], error) {
		if len(links) == 0 {
			return composer.NewStepResponse[any](nil, any([]*dal.Record{})), nil
		}
		defs := make([]*dal.Record, len(links))
		for i, l := range links {
			defs[i] = modulessdk.NewLinkDefinition("product", "product_id", l.ProductID, "sales_channel", "sales_channel_id", l.SalesChannelID)
		}
		if _, err := commonsteps.LinkOf(ctx.Container).Dismiss(ctx.Context, defs); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(defs)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		links, _ := data.([]*dal.Record)
		if len(links) == 0 {
			return nil
		}
		_, err := commonsteps.LinkOf(ctx.Container).Create(ctx.Context, links)
		return err
	})

// LocationChannelLink is { sales_channel_id, location_id }.
type LocationChannelLink struct {
	SalesChannelID string
	LocationID     string
}

func locationLinks(links []LocationChannelLink) []*dal.Record {
	defs := make([]*dal.Record, len(links))
	for i, l := range links {
		defs[i] = modulessdk.NewLinkDefinition("sales_channel", "sales_channel_id", l.SalesChannelID, "stock_location", "stock_location_id", l.LocationID)
	}
	return defs
}

// AssociateLocationsWithSalesChannelsStep mirrors associateLocationsWithSalesChannelsStep.
var AssociateLocationsWithSalesChannelsStep = composer.CreateStep("associate-locations-with-sales-channels-step",
	func(ctx composer.StepExecutionContext, links []LocationChannelLink) (composer.StepResponse[[]*dal.Record], error) {
		if len(links) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, any([]*dal.Record{})), nil
		}
		defs := locationLinks(links)
		created, err := commonsteps.LinkOf(ctx.Container).Create(ctx.Context, defs)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(created, any(defs)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		links, _ := data.([]*dal.Record)
		if len(links) == 0 {
			return nil
		}
		_, err := commonsteps.LinkOf(ctx.Container).Dismiss(ctx.Context, links)
		return err
	})

// DetachLocationsFromSalesChannelsStep mirrors detachLocationsFromSalesChannelsStep.
var DetachLocationsFromSalesChannelsStep = composer.CreateStep("detach-locations-from-sales-channels",
	func(ctx composer.StepExecutionContext, links []LocationChannelLink) (composer.StepResponse[[]*dal.Record], error) {
		if len(links) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, any([]*dal.Record{})), nil
		}
		defs := locationLinks(links)
		if _, err := commonsteps.LinkOf(ctx.Container).Dismiss(ctx.Context, defs); err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse[[]*dal.Record](nil, any(defs)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		links, _ := data.([]*dal.Record)
		if len(links) == 0 {
			return nil
		}
		_, err := commonsteps.LinkOf(ctx.Container).Create(ctx.Context, links)
		return err
	})
