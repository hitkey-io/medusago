// Package steps mirrors @medusajs/core-flows/src/cart/steps.
package steps

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	. "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	customersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/customer/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func invalid(format string, args ...any) error {
	return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf(format, args...))
}

func graph(ctx composer.StepExecutionContext, entity string, fields []string, filters *dal.Record) ([]any, error) {
	r, err := commonsteps.QueryOf(ctx.Container).Graph(ctx.Context, query.GraphInput{Entity: entity, Fields: fields, Filters: filters}, joiner.Options{})
	if err != nil {
		return nil, err
	}
	return r.Data, nil
}

func idFilter(id any) *dal.Record {
	f := dal.NewRecord()
	f.Set("id", id)
	return f
}

// FindSalesChannelStep mirrors findSalesChannelStep (SalesChannelID nil =
// undefined; a pointer to "" = null).
var FindSalesChannelStep = composer.CreateStep("find-sales-channel",
	func(ctx composer.StepExecutionContext, salesChannelID any) (composer.StepResponse[*dal.Record], error) {
		svc := commonsteps.ModuleOf(ctx.Container, utilsmodulessdk.SALES_CHANNEL)
		var channel *dal.Record
		if id, ok := salesChannelID.(string); ok && id != "" {
			c, err := svc.Internal("SalesChannel").Retrieve(ctx.Context, id, utilsmodulessdk.FindConfig{})
			if err != nil {
				return composer.StepResponse[*dal.Record]{}, err
			}
			channel = c
		} else if salesChannelID == nil {
			stores, err := commonsteps.ModuleOf(ctx.Container, utilsmodulessdk.STORE).ListBySuffix(ctx.Context, "Stores", dal.NewRecord(), utilsmodulessdk.FindConfig{Select: []string{"id", "default_sales_channel_id"}})
			if err != nil {
				return composer.StepResponse[*dal.Record]{}, err
			}
			if len(stores) > 0 {
				if id, _ := Get(stores[0], "default_sales_channel_id").(string); id != "" {
					c, err := svc.Internal("SalesChannel").Retrieve(ctx.Context, id, utilsmodulessdk.FindConfig{})
					if err != nil {
						return composer.StepResponse[*dal.Record]{}, err
					}
					channel = c
				}
			}
		}
		if channel == nil {
			return composer.NewStepResponse[*dal.Record](nil), nil
		}
		if Truthy(Get(channel, "is_disabled")) {
			return composer.StepResponse[*dal.Record]{}, invalid("Unable to assign cart to disabled Sales Channel: %v", Get(channel, "name"))
		}
		return composer.NewStepResponse(channel), nil
	}, nil)

func firstRecord(rows []any) *dal.Record {
	if len(rows) == 0 {
		return nil
	}
	r, _ := rows[0].(*dal.Record)
	return r
}

// FindOneOrAnyRegionStep mirrors findOneOrAnyRegionStep.
var FindOneOrAnyRegionStep = composer.CreateStep("find-one-or-any-region",
	func(ctx composer.StepExecutionContext, regionID string) (composer.StepResponse[*dal.Record], error) {
		fields := []string{"*", "countries.*"}
		if regionID != "" {
			rows, err := graph(ctx, "region", fields, idFilter(regionID))
			if err != nil {
				return composer.NewStepResponse[*dal.Record](nil), nil
			}
			return composer.NewStepResponse(firstRecord(rows)), nil
		}
		stores, err := graph(ctx, "store", []string{"*"}, nil)
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		store := firstRecord(stores)
		if store == nil {
			return composer.StepResponse[*dal.Record]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, "Store not found")
		}
		rows, err := graph(ctx, "region", fields, idFilter(Get(store, "default_region_id")))
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		return composer.NewStepResponse(firstRecord(rows)), nil
	}, nil)

// FindOrCreateCustomerInput mirrors FindOrCreateCustomerStepInput (nil =
// undefined).
type FindOrCreateCustomerInput struct {
	CustomerID any
	Email      any
	HasAccount *bool
}

// CustomerData mirrors FindOrCreateCustomerOutputStepOutput (Defined is
// false for { customer: undefined, email: undefined }).
type CustomerData struct {
	Customer *dal.Record
	Email    any
}

type customerCompensation struct {
	customer *dal.Record
	created  bool
}

var emailRegex = regexp.MustCompile(`^(([^<>()[\]\\.,;:\s@"]+(\.[^<>()[\]\\.,;:\s@"]+)*)|.(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$`)

// ValidateEmail mirrors validateEmail.
func ValidateEmail(email string) (string, error) {
	if !emailRegex.MatchString(strings.ToLower(email)) {
		return "", invalid("The email is not valid")
	}
	return strings.ToLower(email), nil
}

func listCustomersByEmail(ctx context.Context, svc commonsteps.ModuleService, email string, hasAccount *bool) ([]*dal.Record, error) {
	f := dal.NewRecord()
	f.Set("email", email)
	if hasAccount != nil {
		f.Set("has_account", *hasAccount)
	}
	return svc.ListBySuffix(ctx, "Customers", f, utilsmodulessdk.FindConfig{Relations: []string{"groups"}})
}

// FindOrCreateCustomerStep mirrors findOrCreateCustomerStep.
var FindOrCreateCustomerStep = composer.CreateStep("find-or-create-customer",
	func(ctx composer.StepExecutionContext, in FindOrCreateCustomerInput) (composer.StepResponse[CustomerData], error) {
		if in.CustomerID == nil && in.Email == nil {
			return composer.NewStepResponse(CustomerData{}, any(customerCompensation{})), nil
		}
		svc := commonsteps.ModuleOf(ctx.Container, utilsmodulessdk.CUSTOMER)
		out := CustomerData{}
		var original *dal.Record
		created := false
		if id, ok := in.CustomerID.(string); ok && id != "" {
			c, err := svc.Internal("Customer").Retrieve(ctx.Context, id, utilsmodulessdk.FindConfig{Relations: []string{"groups"}})
			if err != nil {
				return composer.StepResponse[CustomerData]{}, err
			}
			original = c
			out.Customer = c
			out.Email = Get(c, "email")
		}
		if email, ok := in.Email.(string); ok && email != "" {
			validated, err := ValidateEmail(email)
			if err != nil {
				return composer.StepResponse[CustomerData]{}, err
			}
			var customer *dal.Record
			if original != nil {
				customer = original
			} else {
				list, err := listCustomersByEmail(ctx.Context, svc, validated, in.HasAccount)
				if err != nil {
					return composer.StepResponse[CustomerData]{}, err
				}
				if len(list) > 0 {
					customer = list[0]
				}
			}
			if customer != nil && Truthy(Get(customer, "has_account")) {
				out.Customer = customer
				out.Email = Get(customer, "email")
				return composer.NewStepResponse(out, any(customerCompensation{created: false})), nil
			}
			if customer != nil && Get(customer, "email") != validated {
				f := false
				list, err := listCustomersByEmail(ctx.Context, svc, validated, &f)
				if err != nil {
					return composer.StepResponse[CustomerData]{}, err
				}
				customer = nil
				if len(list) > 0 {
					customer = list[0]
				}
			}
			if customer == nil {
				d := dal.NewRecord()
				d.Set("email", validated)
				createdRows, err := customersteps.CreateCustomers(ctx.Context, svc, []*dal.Record{d})
				if err != nil {
					return composer.StepResponse[CustomerData]{}, err
				}
				customer = createdRows[0]
				created = true
			}
			original = customer
			out.Customer = customer
			out.Email = Get(customer, "email")
		}
		return composer.NewStepResponse(out, any(customerCompensation{customer: original, created: created})), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		c, _ := data.(customerCompensation)
		if !c.created || c.customer == nil {
			return nil
		}
		_, err := commonsteps.ModuleOf(ctx.Container, utilsmodulessdk.CUSTOMER).Internal("Customer").Delete(ctx.Context, []any{Get(c.customer, "id")})
		return err
	})

// ValidateSalesChannelStep mirrors validateSalesChannelStep.
var ValidateSalesChannelStep = composer.CreateStep("validate-sales-channel",
	func(ctx composer.StepExecutionContext, channel *dal.Record) (composer.StepResponse[any], error) {
		if !Truthy(Get(channel, "id")) {
			return composer.StepResponse[any]{}, invalid("Sales channel is required when creating a cart. Either provide a sales channel ID or set the default sales channel for the store.")
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// ValidateCartStep mirrors validateCartStep.
var ValidateCartStep = composer.CreateStep("validate-cart",
	func(ctx composer.StepExecutionContext, cart *dal.Record) (composer.StepResponse[any], error) {
		if Has(cart, "completed_at") {
			return composer.StepResponse[any]{}, invalid("Cart %v is already completed.", Get(cart, "id"))
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)
