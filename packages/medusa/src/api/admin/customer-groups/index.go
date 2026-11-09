// Package customergroups mirrors api/admin/customer-groups.
package customergroups

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// DefaultAdminCustomerGroupFields mirrors defaultAdminCustomerGroupFields.
var DefaultAdminCustomerGroupFields = []string{"id", "name", "created_by", "created_at", "updated_at", "deleted_at", "metadata"}

// Query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: DefaultAdminCustomerGroupFields, Entity: "customer_group"}
	ListConfig     = httputils.QueryConfig{Defaults: DefaultAdminCustomerGroupFields, IsList: true, Entity: "customer_group"}
)

// AdminCustomerInGroupFilters mirrors AdminCustomerInGroupFilters.
var AdminCustomerInGroupFilters = zod.Object(
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("email", zod.Union(zod.String(), zod.Array(zod.String()), utils.CreateOperatorMap(nil, nil)).Optional()),
	zod.F("default_billing_address_id", utils.StringOrStrings().Optional()),
	zod.F("default_shipping_address_id", utils.StringOrStrings().Optional()),
	zod.F("company_name", utils.StringOrStrings().Optional()),
	zod.F("first_name", utils.StringOrStrings().Optional()),
	zod.F("last_name", utils.StringOrStrings().Optional()),
	zod.F("created_by", utils.StringOrStrings().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
)

// AdminGetCustomerGroupsParamsFields mirrors AdminGetCustomerGroupsParamsFields.
var AdminGetCustomerGroupsParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("name", utils.StringOrStrings().Optional()),
	zod.F("customers", zod.Union(zod.String(), zod.Array(zod.String()), AdminCustomerInGroupFilters).Optional()),
	zod.F("created_by", utils.StringOrStrings().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
)

// AdminGetCustomerGroupsParams mirrors AdminGetCustomerGroupsParams.
var AdminGetCustomerGroupsParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(50), Offset: utils.Num(0)}).
	Merge(AdminGetCustomerGroupsParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetCustomerGroupsParamsFields))

// Middlewares mirrors adminCustomerGroupRoutesMiddlewares (reads).
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/customer-groups", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetCustomerGroupsParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/customer-groups/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RemoteQueryListRoute("customer_group", "customer_groups")}
