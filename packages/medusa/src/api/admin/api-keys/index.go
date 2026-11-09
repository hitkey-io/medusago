// Package apikeys mirrors api/admin/api-keys.
package apikeys

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// DefaultAdminApiKeyFields mirrors defaultAdminApiKeyFields.
var DefaultAdminApiKeyFields = []string{"id", "title", "token", "redacted", "type", "last_used_at", "updated_at", "created_at", "created_by", "revoked_at", "revoked_by", "sales_channels.id", "sales_channels.name"}

// Query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: DefaultAdminApiKeyFields, Entity: "api_key"}
	ListConfig     = httputils.QueryConfig{Defaults: DefaultAdminApiKeyFields, DefaultLimit: 20, IsList: true, Entity: "api_key"}
)

// AdminGetApiKeysParamsFields mirrors AdminGetApiKeysParamsFields.
var AdminGetApiKeysParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("title", utils.StringOrStrings().Optional()),
	zod.F("token", utils.StringOrStrings().Optional()),
	zod.F("type", zod.Enum("publishable", "secret").Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("revoked_at", utils.CreateOperatorMap(nil, nil).Optional()),
)

// AdminGetApiKeysParams mirrors AdminGetApiKeysParams.
var AdminGetApiKeysParams = utils.CreateFindParams(utils.FindParamsOptions{Offset: utils.Num(0), Limit: utils.Num(20)}).
	Merge(AdminGetApiKeysParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetApiKeysParamsFields))

// Middlewares mirrors adminApiKeyRoutesMiddlewares (reads).
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/api-keys", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetApiKeysParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/api-keys/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RemoteQueryListRoute("api_key", "api_keys")}
