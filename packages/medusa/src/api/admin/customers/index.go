// Package customers mirrors api/admin/customers.
package customers

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// DefaultAdminCustomerFields mirrors defaultAdminCustomerFields.
var DefaultAdminCustomerFields = []string{"id", "company_name", "first_name", "last_name", "email", "phone", "metadata", "has_account", "created_by", "created_at", "updated_at", "deleted_at"}

// Query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: DefaultAdminCustomerFields, Entity: "customer"}
	ListConfig     = httputils.QueryConfig{Defaults: DefaultAdminCustomerFields, IsList: true, Entity: "customer"}
)

// AdminCustomerGroupInCustomerParams mirrors AdminCustomerGroupInCustomerParams.
var AdminCustomerGroupInCustomerParams = zod.Object(
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("name", utils.StringOrStrings().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
)

// AdminCustomersParamsFields mirrors AdminCustomersParamsFields.
var AdminCustomersParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", zod.Union(zod.String(), zod.Array(zod.String()), utils.CreateOperatorMap(nil, nil)).Optional()),
	zod.F("email", utils.StringOrStrings().Optional()),
	zod.F("groups", zod.Union(AdminCustomerGroupInCustomerParams, zod.String(), zod.Array(zod.String())).Optional()),
	zod.F("company_name", utils.StringOrStrings().Optional()),
	zod.F("first_name", utils.StringOrStrings().Optional()),
	zod.F("last_name", utils.StringOrStrings().Optional()),
	zod.F("has_account", commonvalidators.BooleanString().Optional()),
	zod.F("created_by", utils.StringOrStrings().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
)

// AdminCustomersParams mirrors AdminCustomersParams.
var AdminCustomersParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(50), Offset: utils.Num(0)}).
	Merge(AdminCustomersParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(AdminCustomersParamsFields))

// Middlewares mirrors adminCustomerRoutesMiddlewares (reads).
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/customers", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminCustomersParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/customers/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
}

func init() { Middlewares = append(Middlewares, WriteMiddlewares...) }

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RemoteQueryListRoute("customers", "customers"), POST: POST}
