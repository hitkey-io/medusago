package workflows

import (
	"regexp"
	"strings"

	authsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/auth/steps"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CreateCustomerAccountInput mirrors CreateCustomerAccountWorkflowInput.
type CreateCustomerAccountInput struct {
	AuthIdentityID any
	CustomerData   *dal.Record
}

var emailRegex = regexp.MustCompile(`^(([^<>()[\]\\.,;:\s@"]+(\.[^<>()[\]\\.,;:\s@"]+)*)|.(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$`)

func isEmail(s string) bool { return emailRegex.MatchString(strings.ToLower(s)) }

func normalizeString(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func rget(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func present(v any) bool {
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

// ValidateCustomerAccountCreation mirrors validateCustomerAccountCreation.
var ValidateCustomerAccountCreation = composer.CreateStep("validate-customer-account-creation",
	func(ctx composer.StepExecutionContext, in CreateCustomerAccountInput) (composer.StepResponse[any], error) {
		email, _ := rget(in.CustomerData, "email").(string)
		if email == "" {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Email is required to create a customer")
		}
		if present(in.AuthIdentityID) {
			auth := commonsteps.ModuleOf(ctx.Container, utilsmodulessdk.AUTH)
			identity, err := auth.Internal("AuthIdentity").Retrieve(ctx.Context, in.AuthIdentityID, utilsmodulessdk.FindConfig{Relations: []string{"provider_identities"}})
			if err != nil {
				return composer.StepResponse[any]{}, err
			}
			var emails []string
			var list []*dal.Record
			switch t := rget(identity, "provider_identities").(type) {
			case []*dal.Record:
				list = t
			case []any:
				for _, x := range t {
					if r, ok := x.(*dal.Record); ok {
						list = append(list, r)
					}
				}
			}
			for _, pi := range list {
				entity, _ := rget(pi, "entity_id").(string)
				entityID := normalizeString(entity)
				if (rget(pi, "provider") == "emailpass" || isEmail(entityID)) && entityID != "" {
					emails = append(emails, entityID)
				}
				meta, _ := rget(pi, "user_metadata").(*dal.Record)
				if s, ok := rget(meta, "email").(string); ok && normalizeString(s) != "" {
					emails = append(emails, normalizeString(s))
				}
			}
			if len(emails) > 0 {
				found := false
				for _, e := range emails {
					if e == normalizeString(email) {
						found = true
					}
				}
				if !found {
					return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.UNAUTHORIZED, "The customer's email must match the email of the authenticated identity")
				}
			}
		}
		f := dal.NewRecord()
		f.Set("email", email)
		existing, err := commonsteps.ModuleOf(ctx.Container, utilsmodulessdk.CUSTOMER).ListBySuffix(ctx.Context, "Customers", f, utilsmodulessdk.FindConfig{})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		if len(existing) > 0 {
			hasAccount := false
			for _, c := range existing {
				if b, _ := rget(c, "has_account").(bool); b {
					hasAccount = true
				}
			}
			if hasAccount && present(in.AuthIdentityID) {
				return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.DUPLICATE_ERROR, "Customer with this email already has an account")
			}
			if !hasAccount && !present(in.AuthIdentityID) {
				return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.DUPLICATE_ERROR, "Guest customer with this email already exists")
			}
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// CreateCustomerAccountWorkflow mirrors createCustomerAccountWorkflow.
var CreateCustomerAccountWorkflow = composer.CreateWorkflow("create-customer-account",
	func(e *composer.Execution, input CreateCustomerAccountInput) (*dal.Record, error) {
		if _, err := ValidateCustomerAccountCreation.Run(e, input); err != nil {
			return nil, err
		}
		if err := hook(e, "validate", "input", input); err != nil {
			return nil, err
		}
		data := dal.NewRecord()
		if input.CustomerData != nil {
			for _, k := range input.CustomerData.Keys() {
				v, _ := input.CustomerData.Get(k)
				data.Set(k, v)
			}
		}
		data.Set("has_account", present(input.AuthIdentityID))
		created, err := CreateCustomersWorkflow.RunAsStep(e, CreateCustomersInput{CustomersData: []*dal.Record{data}})
		if err != nil {
			return nil, err
		}
		customer := created[0]
		if _, err := authsteps.SetAuthAppMetadataStep.Run(e, authsteps.SetAuthAppMetadataStepInput{AuthIdentityID: input.AuthIdentityID, ActorType: "customer", Value: rget(customer, "id")}); err != nil {
			return nil, err
		}
		return customer, nil
	})
