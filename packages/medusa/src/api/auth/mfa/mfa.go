// Package mfa mirrors api/auth/mfa (shared helpers of the MFA routes).
package mfa

import (
	"context"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	authservices "github.com/hitkey-io/medusago/packages/modules/auth/src/services"
)

// AuthService resolves the auth module.
func AuthService(req *medusahttp.MedusaRequest) *authservices.AuthModuleService {
	s, _ := req.Scope.Resolve(utilsmodulessdk.AUTH).(*authservices.AuthModuleService)
	return s
}

// AuthIdentityID is req.auth_context.auth_identity_id.
func AuthIdentityID(req *medusahttp.MedusaRequest) any {
	if req.AuthContext == nil {
		return nil
	}
	v, _ := req.AuthContext.Get("auth_identity_id")
	return v
}

// Emit mirrors eventBus.emit({ name, data }).
func Emit(ctx context.Context, req *medusahttp.MedusaRequest, name string, data *dal.Record) error {
	bus, _ := req.Scope.Resolve(utilsmodulessdk.EVENT_BUS).(composer.EventBus)
	if bus == nil {
		return nil
	}
	return bus.Emit(ctx, []composer.Message{{Name: name, Data: data}})
}

// Body returns the validated body record.
func Body(req *medusahttp.MedusaRequest) *dal.Record {
	b, _ := req.ValidatedBody.(*dal.Record)
	if b == nil {
		b = dal.NewRecord()
	}
	return b
}

// Str returns a body string field.
func Str(r *dal.Record, k string) string {
	v, _ := r.Get(k)
	s, _ := v.(string)
	return s
}
