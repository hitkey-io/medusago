package customers

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/customer/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// DefaultAdminCustomerAddressFields mirrors defaultAdminCustomerAddressFields.
var DefaultAdminCustomerAddressFields = []string{"id", "address_name", "company", "customer_id", "first_name", "last_name", "address_1", "address_2", "city", "province", "postal_code", "country_code", "phone", "metadata", "created_at", "updated_at"}

// Address query configs.
var (
	RetrieveAddressConfig = httputils.QueryConfig{Defaults: DefaultAdminCustomerAddressFields, Entity: "customer_address"}
	ListAddressesConfig   = httputils.QueryConfig{Defaults: DefaultAdminCustomerAddressFields, IsList: true, Entity: "customer_address"}
)

func customerFields() []zod.Field {
	return []zod.Field{
		zod.F("email", zod.String().Email().Nullish()),
		zod.F("company_name", zod.String().Nullish()),
		zod.F("first_name", zod.String().Nullish()),
		zod.F("last_name", zod.String().Nullish()),
		zod.F("phone", zod.String().Nullish()),
		zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
	}
}

// AdminCreateCustomer mirrors AdminCreateCustomer.
var AdminCreateCustomer = utils.WithAdditionalData(zod.Object(customerFields()...))

// AdminUpdateCustomer mirrors AdminUpdateCustomer.
var AdminUpdateCustomer = utils.WithAdditionalData(zod.Object(customerFields()...))

// CreateCustomerAddress mirrors CreateCustomerAddress.
var CreateCustomerAddress = zod.Object(
	zod.F("address_name", zod.String().Nullish()),
	zod.F("is_default_shipping", zod.Boolean().Optional()),
	zod.F("is_default_billing", zod.Boolean().Optional()),
	zod.F("company", zod.String().Nullish()),
	zod.F("first_name", zod.String().Nullish()),
	zod.F("last_name", zod.String().Nullish()),
	zod.F("address_1", zod.String().Nullish()),
	zod.F("address_2", zod.String().Nullish()),
	zod.F("city", zod.String().Nullish()),
	zod.F("country_code", zod.String().Nullish()),
	zod.F("province", zod.String().Nullish()),
	zod.F("postal_code", zod.String().Nullish()),
	zod.F("phone", zod.String().Nullish()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
)

// AdminCreateCustomerAddress mirrors AdminCreateCustomerAddress.
var AdminCreateCustomerAddress = utils.WithAdditionalData(CreateCustomerAddress)

// AdminCustomerAddressesParams mirrors AdminCustomerAddressesParams.
var AdminCustomerAddressesParams = utils.CreateFindParams(utils.FindParamsOptions{Offset: utils.Num(0), Limit: utils.Num(50)}).Merge(zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("company", utils.StringOrStrings().Optional()),
	zod.F("city", utils.StringOrStrings().Optional()),
	zod.F("country_code", utils.StringOrStrings().Optional()),
	zod.F("province", utils.StringOrStrings().Optional()),
	zod.F("postal_code", utils.StringOrStrings().Optional()),
))

var _ = commonvalidators.ApplyAndAndOrOperators

func mw(fns ...medusahttp.MiddlewareFunction) []medusahttp.MiddlewareFunction { return fns }

// WriteMiddlewares mirrors the write routes of adminCustomerRoutesMiddlewares.
var WriteMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"POST"}, Matcher: "/admin/customers", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateCustomer)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/customers/:id", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateCustomer)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/customers/:id/addresses", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateCustomerAddress)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
	{Methods: []string{"GET"}, Matcher: "/admin/customers/:id/addresses/:address_id", Middlewares: mw(httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveAddressConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/customers/:id/addresses/:address_id", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateCustomerAddress)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
	{Methods: []string{"DELETE"}, Matcher: "/admin/customers/:id/addresses/:address_id", Middlewares: mw(httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
	{Methods: []string{"GET"}, Matcher: "/admin/customers/:id/addresses", Middlewares: mw(httputils.ValidateAndTransformQuery(AdminCustomerAddressesParams, ListAddressesConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/customers/:id/customer-groups", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(utils.CreateLinkBody())), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
}

// RefetchCustomer mirrors refetchCustomer (nil when missing).
func RefetchCustomer(req *medusahttp.MedusaRequest, id any, fields []string) (any, error) {
	f := dal.NewRecord()
	f.Set("id", id)
	return httputils.RemoteQueryFirst(req.Ctx(), req.Scope, "customer", f, fields)
}

// Respond refetches the customer and writes { customer }.
func Respond(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, id any) error {
	v, err := RefetchCustomer(req, id, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if v != nil {
		out.Set("customer", v)
	}
	return res.Status(200).JSON(out)
}

// SplitAdditionalData returns the body without additional_data, and
// additional_data.
func SplitAdditionalData(body *dal.Record) (*dal.Record, *dal.Record) {
	rest := dal.NewRecord()
	var additional *dal.Record
	if body == nil {
		return rest, nil
	}
	for _, k := range body.Keys() {
		v, _ := body.Get(k)
		if k == "additional_data" {
			additional, _ = v.(*dal.Record)
			continue
		}
		rest.Set(k, v)
	}
	return rest, additional
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	rest, additional := SplitAdditionalData(body)
	rest.Set("created_by", req.AuthString("actor_id"))
	result, err := workflows.CreateCustomersWorkflow.Run(req.Ctx(), req.Scope, workflows.CreateCustomersInput{CustomersData: []*dal.Record{rest}, AdditionalData: additional})
	if err != nil {
		return err
	}
	id, _ := result[0].Get("id")
	return Respond(req, res, id)
}
