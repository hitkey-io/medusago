package workflows

import (
	"context"

	authsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/auth/steps"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// RequestVerificationInput mirrors RequestAuthVerificationDTO.
type RequestVerificationInput struct {
	AuthIdentityID any
	EntityID       any
	EntityType     any
	CodeProvider   any
	Metadata       any
}

type verificationRequester interface {
	RequestAuthVerification(ctx context.Context, authIdentityID, entityID, entityType, codeProvider any, metadata any) (*dal.Record, error)
}

// RequestVerificationStep mirrors requestVerificationStep.
var RequestVerificationStep = composer.CreateStep("request-verification",
	func(ctx composer.StepExecutionContext, in RequestVerificationInput) (composer.StepResponse[*dal.Record], error) {
		svc, _ := ctx.Container.Resolve(utilsmodulessdk.AUTH).(verificationRequester)
		r, err := svc.RequestAuthVerification(ctx.Context, in.AuthIdentityID, in.EntityID, in.EntityType, in.CodeProvider, in.Metadata)
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		return composer.NewStepResponse(r), nil
	}, nil)

// RequestVerificationWorkflow mirrors requestVerificationWorkflow: the code
// goes to the auth.verification_requested event, not to the response.
var RequestVerificationWorkflow = composer.CreateWorkflow("request-verification-workflow",
	func(e *composer.Execution, in RequestVerificationInput) (*dal.Record, error) {
		result, err := RequestVerificationStep.Run(e, in)
		if err != nil {
			return nil, err
		}
		get := func(k string) any { v, _ := result.Get(k); return v }
		metadata := get("metadata")
		if metadata == nil {
			metadata = dal.NewRecord()
		}
		data := dal.NewRecord()
		data.Set("entity_id", get("entity_id"))
		data.Set("entity_type", get("entity_type"))
		data.Set("code_provider", get("code_provider"))
		data.Set("auth_identity_id", get("auth_identity_id"))
		data.Set("code", get("code"))
		data.Set("expires_at", get("expires_at"))
		data.Set("metadata", metadata)
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "auth.verification_requested", Data: data}); err != nil {
			return nil, err
		}
		out := dal.NewRecord()
		for _, k := range result.Keys() {
			if k == "code" || k == "expires_at" {
				continue
			}
			v, _ := result.Get(k)
			out.Set(k, v)
		}
		return out, nil
	})

// SetAuthAppMetadataWorkflow mirrors setAuthAppMetadataWorkflow.
var SetAuthAppMetadataWorkflow = composer.CreateWorkflow("set-auth-app-metadata-workflow",
	func(e *composer.Execution, in authsteps.SetAuthAppMetadataStepInput) (any, error) {
		return authsteps.SetAuthAppMetadataStep.Run(e, in)
	})
