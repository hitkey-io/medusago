// Package auth mirrors api/auth/middlewares.ts.
package auth

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/framework/src/http/middlewares"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	authutils "github.com/hitkey-io/medusago/packages/medusa/src/api/auth/utils"
)

// ResetPasswordRequest mirrors ResetPasswordRequest.
var ResetPasswordRequest = zod.Object(
	zod.F("identifier", zod.String()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Optional().Default(dal.NewRecord())),
)

func boolPtr(b bool) *bool { return &b }

// AuthRoutesMiddlewares mirrors authRoutesMiddlewares (the entries of the
// ported routes).
var AuthRoutesMiddlewares = []medusahttp.MiddlewareRoute{
	{
		Methods: []string{"POST"},
		Matcher: "/auth/session",
		Middlewares: []medusahttp.MiddlewareFunction{
			middlewares.Authenticate([]string{"*"}, []string{"bearer"}, medusahttp.AuthenticateOptions{AllowUnregistered: true, RequireMfa: boolPtr(false)}),
		},
	},
	{
		Methods: []string{"DELETE"},
		Matcher: "/auth/session",
		Middlewares: []medusahttp.MiddlewareFunction{
			middlewares.Authenticate([]string{"*"}, []string{"session"}, medusahttp.AuthenticateOptions{AllowUnregistered: true, RequireMfa: boolPtr(false)}),
		},
	},
	{
		Methods: []string{"POST"},
		Matcher: "/auth/token/refresh",
		Middlewares: []medusahttp.MiddlewareFunction{
			middlewares.Authenticate([]string{"*"}, []string{"session", "bearer"}, medusahttp.AuthenticateOptions{AllowUnregistered: true, RequireMfa: boolPtr(false)}),
		},
	},
	{
		Methods:     []string{"POST"},
		Matcher:     "/auth/mfa/challenges/:id/verify",
		Middlewares: []medusahttp.MiddlewareFunction{middlewares.Authenticate([]string{"*"}, []string{"session", "bearer"}, medusahttp.AuthenticateOptions{AllowUnregistered: true, RequireMfa: boolPtr(false)}), httputils.ValidateAndTransformBody(httputils.Schema(AuthMfaVerifyChallengeRequest))},
	},
	{
		Methods:     []string{"GET"},
		Matcher:     "/auth/mfa/factors",
		Middlewares: []medusahttp.MiddlewareFunction{middlewares.Authenticate([]string{"*"}, []string{"session", "bearer"}, medusahttp.AuthenticateOptions{AllowUnregistered: true})},
	},
	{
		Methods:     []string{"POST"},
		Matcher:     "/auth/mfa/factors",
		Middlewares: []medusahttp.MiddlewareFunction{middlewares.Authenticate([]string{"*"}, []string{"session", "bearer"}, medusahttp.AuthenticateOptions{AllowUnregistered: true}), httputils.ValidateAndTransformBody(httputils.Schema(AuthMfaCreateFactorRequest))},
	},
	{
		Methods:     []string{"POST"},
		Matcher:     "/auth/mfa/factors/:id/verify",
		Middlewares: []medusahttp.MiddlewareFunction{middlewares.Authenticate([]string{"*"}, []string{"session", "bearer"}, medusahttp.AuthenticateOptions{AllowUnregistered: true}), httputils.ValidateAndTransformBody(httputils.Schema(AuthMfaVerifyFactorRequest))},
	},
	{
		Methods:     []string{"DELETE"},
		Matcher:     "/auth/mfa/factors/:id",
		Middlewares: []medusahttp.MiddlewareFunction{middlewares.Authenticate([]string{"*"}, []string{"session", "bearer"}, medusahttp.AuthenticateOptions{AllowUnregistered: true}), httputils.ValidateAndTransformBody(httputils.Schema(AuthMfaDisableFactorRequest))},
	},
	{
		Methods:     []string{"POST"},
		Matcher:     "/auth/mfa/recovery-codes",
		Middlewares: []medusahttp.MiddlewareFunction{middlewares.Authenticate([]string{"*"}, []string{"session", "bearer"}, medusahttp.AuthenticateOptions{AllowUnregistered: true}), httputils.ValidateAndTransformBody(httputils.Schema(AuthMfaGenerateRecoveryCodesRequest))},
	},
	{
		Methods:     []string{"GET", "POST"},
		Matcher:     "/auth/:actor_type/:auth_provider/callback",
		Middlewares: []medusahttp.MiddlewareFunction{authutils.ValidateScopeProviderAssociation("")},
	},
	{
		Methods:     []string{"POST"},
		Matcher:     "/auth/:actor_type/:auth_provider/register",
		Middlewares: []medusahttp.MiddlewareFunction{authutils.ValidateScopeProviderAssociation("")},
	},
	{
		Methods:     []string{"POST"},
		Matcher:     "/auth/:actor_type/:auth_provider",
		Middlewares: []medusahttp.MiddlewareFunction{authutils.ValidateScopeProviderAssociation("")},
	},
	{
		Methods:     []string{"GET"},
		Matcher:     "/auth/:actor_type/:auth_provider",
		Middlewares: []medusahttp.MiddlewareFunction{authutils.ValidateScopeProviderAssociation("")},
	},
	{
		Methods:     []string{"POST"},
		Matcher:     "/auth/:actor_type/:auth_provider/reset-password",
		Middlewares: []medusahttp.MiddlewareFunction{authutils.ValidateScopeProviderAssociation(""), httputils.ValidateAndTransformBody(httputils.Schema(ResetPasswordRequest))},
	},
	{
		Methods:     []string{"POST"},
		Matcher:     "/auth/:actor_type/:auth_provider/update",
		Middlewares: []medusahttp.MiddlewareFunction{authutils.ValidateScopeProviderAssociation(""), authutils.ValidateToken()},
	},
	{
		Methods:     []string{"POST"},
		Matcher:     "/auth/verification/request",
		Middlewares: []medusahttp.MiddlewareFunction{middlewares.Authenticate([]string{"*"}, []string{"session", "bearer"}, medusahttp.AuthenticateOptions{AllowUnregistered: true, RequireMfa: boolPtr(false)}), httputils.ValidateAndTransformBody(httputils.Schema(VerificationRequest))},
	},
	{
		Methods:     []string{"POST"},
		Matcher:     "/auth/verification/confirm",
		Middlewares: []medusahttp.MiddlewareFunction{middlewares.Authenticate([]string{"*"}, []string{"session", "bearer"}, medusahttp.AuthenticateOptions{AllowUnregistered: true, AllowUnauthenticated: true}), httputils.ValidateAndTransformBody(httputils.Schema(VerificationConfirmRequest))},
	},
}

// VerificationRequest mirrors VerificationRequest.
var VerificationRequest = zod.Object(
	zod.F("entity_id", zod.String().Min(1)),
	zod.F("entity_type", zod.String().Min(1)),
	zod.F("code_provider", zod.String().Min(1).Default("token")),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Optional().Default(dal.NewRecord())),
)

// VerificationConfirmRequest mirrors VerificationConfirmRequest.
var VerificationConfirmRequest = zod.Object(
	zod.F("code", zod.String().Min(1)),
	zod.F("code_provider", zod.String().Min(1).Optional()),
)

// AuthMfaVerifyChallengeRequest mirrors AuthMfaVerifyChallengeRequest.
var AuthMfaVerifyChallengeRequest = zod.Object(
	zod.F("method", zod.String().Min(1)),
	zod.F("code", zod.String().Min(1)),
)

// AuthMfaCreateFactorRequest mirrors AuthMfaCreateFactorRequest.
var AuthMfaCreateFactorRequest = zod.Object(
	zod.F("provider", zod.String().Min(1)),
	zod.F("label", zod.String().Nullable().Optional()),
	zod.F("issuer", zod.String().Optional()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullable().Optional()),
)

// AuthMfaVerifyFactorRequest mirrors AuthMfaVerifyFactorRequest.
var AuthMfaVerifyFactorRequest = zod.Object(zod.F("code", zod.String().Min(1)))

// AuthMfaDisableFactorRequest mirrors AuthMfaDisableFactorRequest.
var AuthMfaDisableFactorRequest = zod.Object(
	zod.F("method", zod.String().Min(1).Optional()),
	zod.F("code", zod.String().Min(1).Optional()),
)

// AuthMfaGenerateRecoveryCodesRequest mirrors AuthMfaGenerateRecoveryCodesRequest.
var AuthMfaGenerateRecoveryCodesRequest = zod.Object(zod.F("count", zod.Number().Int().Min(1).Max(50).Optional()))
