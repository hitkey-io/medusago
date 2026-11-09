// Package invites mirrors api/admin/invites (the RBAC role handling is an
// Enterprise feature and is not ported: invite tokens are never redacted).
package invites

import (
	"context"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/framework/src/http/middlewares"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// DefaultAdminInviteFields mirrors defaultAdminInviteFields.
var DefaultAdminInviteFields = []string{"id", "email", "accepted", "token", "expires_at", "metadata", "created_at", "updated_at", "deleted_at"}

var (
	// RetrieveTransformQueryConfig mirrors retrieveTransformQueryConfig.
	RetrieveTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminInviteFields, Entity: "invite"}
	// ListTransformQueryConfig mirrors listTransformQueryConfig.
	ListTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminInviteFields, IsList: true, Entity: "invite"}
)

// AdminGetInviteParams mirrors AdminGetInviteParams.
var AdminGetInviteParams = utils.CreateSelectParams()

// AdminGetInvitesParamsFields mirrors AdminGetInvitesParamsFields.
var AdminGetInvitesParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("email", utils.StringOrStrings().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
)

// AdminGetInvitesParams mirrors AdminGetInvitesParams.
var AdminGetInvitesParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(50), Offset: utils.Num(0)}).
	Merge(AdminGetInvitesParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetInvitesParamsFields))

// AdminGetInviteAcceptParams mirrors AdminGetInviteAcceptParams.
var AdminGetInviteAcceptParams = utils.CreateSelectParams().Merge(zod.Object(zod.F("token", zod.String())))

// AdminCreateInvite mirrors AdminCreateInvite.
var AdminCreateInvite = zod.Object(
	zod.F("email", zod.String()),
	zod.F("roles", zod.Array(zod.Object(
		zod.F("role_id", zod.String().Min(1)),
		zod.F("scopes", zod.Array(zod.Object(
			zod.F("type", zod.String().Min(1)),
			zod.F("id", zod.String().Min(1)),
		)).Nullish()),
	)).Nullish()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
).Strict()

// AdminInviteAccept mirrors AdminInviteAccept.
var AdminInviteAccept = zod.Object(
	zod.F("email", zod.String().Nullish()),
	zod.F("first_name", zod.String().Nullish()),
	zod.F("last_name", zod.String().Nullish()),
).Strict()

func mw(fns ...medusahttp.MiddlewareFunction) []medusahttp.MiddlewareFunction { return fns }

// AdminInviteRoutesMiddlewares mirrors adminInviteRoutesMiddlewares
// (authorize is RBAC and not ported).
var AdminInviteRoutesMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/invites", Middlewares: mw(httputils.ValidateAndTransformQuery(AdminGetInvitesParams, ListTransformQueryConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/invites", Middlewares: mw(
		httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateInvite)),
		httputils.ValidateAndTransformQuery(AdminGetInviteParams, RetrieveTransformQueryConfig),
	)},
	{Methods: []string{"POST"}, Matcher: "/admin/invites/accept", Middlewares: mw(
		middlewares.Authenticate([]string{"user"}, []string{"session", "bearer"}, medusahttp.AuthenticateOptions{AllowUnregistered: true}),
		httputils.ValidateAndTransformBody(httputils.Schema(AdminInviteAccept)),
		httputils.ValidateAndTransformQuery(AdminGetInviteAcceptParams, RetrieveTransformQueryConfig),
	)},
	{Methods: []string{"GET"}, Matcher: "/admin/invites/:id", Middlewares: mw(httputils.ValidateAndTransformQuery(AdminGetInviteParams, RetrieveTransformQueryConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/invites/:id/resend", Middlewares: mw(httputils.ValidateAndTransformQuery(AdminGetInviteParams, RetrieveTransformQueryConfig))},
}

// RefetchInvite mirrors refetchInvite.
func RefetchInvite(ctx context.Context, req *medusahttp.MedusaRequest, id string) (any, error) {
	return httputils.RefetchEntity(ctx, req.Scope, "invite", id, req.QueryConfig.Fields)
}
