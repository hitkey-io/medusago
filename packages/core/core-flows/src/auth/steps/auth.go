// Package steps mirrors @medusajs/core-flows/src/auth/steps.
package steps

import (
	"fmt"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// SetAuthAppMetadataStepInput mirrors SetAuthAppMetadataStepInput (a nil
// Value deletes the association with the actor).
type SetAuthAppMetadataStepInput struct {
	AuthIdentityID any
	ActorType      string
	Value          any
}

type appMetadataCompensation struct {
	id       any
	key      string
	value    any
	oldValue any
}

func authService(c composer.Container) commonsteps.ModuleService {
	return commonsteps.ModuleOf(c, utilsmodulessdk.AUTH)
}

func retrieveIdentity(ctx composer.StepExecutionContext, id any) (*dal.Record, *dal.Record, error) {
	identity, err := authService(ctx.Container).Internal("AuthIdentity").Retrieve(ctx.Context, id, utilsmodulessdk.FindConfig{})
	if err != nil {
		return nil, nil, err
	}
	meta, _ := identity.Get("app_metadata")
	m, _ := meta.(*dal.Record)
	if m == nil {
		m = dal.NewRecord()
	}
	return identity, m, nil
}

func saveAppMetadata(ctx composer.StepExecutionContext, id any, m *dal.Record) error {
	u := dal.NewRecord()
	u.Set("id", id)
	u.Set("app_metadata", m)
	_, err := commonsteps.UpdateRecords(ctx.Context, authService(ctx.Container), "AuthIdentity", []any{u})
	return err
}

// SetAuthAppMetadataStep mirrors setAuthAppMetadataStep.
var SetAuthAppMetadataStep = composer.CreateStep("set-auth-app-metadata",
	func(ctx composer.StepExecutionContext, in SetAuthAppMetadataStepInput) (composer.StepResponse[*dal.Record], error) {
		key := in.ActorType + "_id"
		identity, m, err := retrieveIdentity(ctx, in.AuthIdentityID)
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		old, defined := m.Get(key)
		if defined && in.Value != nil {
			return composer.StepResponse[*dal.Record]{}, fmt.Errorf("Key %s already exists in app metadata", key)
		}
		m.Set(key, in.Value)
		id, _ := identity.Get("id")
		if err := saveAppMetadata(ctx, id, m); err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		return composer.NewStepResponse(identity, any(appMetadataCompensation{id: id, key: key, value: in.Value, oldValue: old})), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		c, ok := data.(appMetadataCompensation)
		if !ok {
			return nil
		}
		identity, m, err := retrieveIdentity(ctx, c.id)
		if err != nil {
			return err
		}
		if c.value == nil {
			m.Set(c.key, c.oldValue)
		} else {
			m.Delete(c.key)
		}
		id, _ := identity.Get("id")
		return saveAppMetadata(ctx, id, m)
	})

// DeleteAuthIdentityStep mirrors deleteAuthIdentityStep (hard delete; the
// compensation recreates the identities with their provider identities).
var DeleteAuthIdentityStep = composer.CreateStep("delete-auth-identity",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[any], error) {
		svc := authService(ctx.Container)
		f := dal.NewRecord()
		f.Set("id", ids)
		prev, err := svc.ListBySuffix(ctx.Context, "AuthIdentities", f, utilsmodulessdk.FindConfig{Relations: []string{"provider_identities"}})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		if _, err := svc.Internal("AuthIdentity").Delete(ctx.Context, ids); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(prev)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		prev, _ := data.([]*dal.Record)
		if len(prev) == 0 {
			return nil
		}
		_, err := authService(ctx.Container).Internal("AuthIdentity").Create(ctx.Context, prev)
		return err
	})
