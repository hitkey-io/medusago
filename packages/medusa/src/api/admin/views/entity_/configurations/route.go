package configurations

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/settings/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	settingsservices "github.com/hitkey-io/medusago/packages/modules/settings/src/services"
)

// Graph runs query.graph({ entity: "view_configuration", fields, filters,
// pagination }).
func Graph(ctx context.Context, req *medusahttp.MedusaRequest, fields []string, filters, pagination *dal.Record) (*query.GraphResult, error) {
	res, err := httputils.QueryOf(req.Scope).Graph(ctx, query.GraphInput{Entity: "view_configuration", Fields: fields, Filters: filters, Pagination: pagination}, joiner.Options{})
	if err != nil {
		return nil, err
	}
	if res.Metadata == nil {
		res.Metadata = dal.NewRecord()
	}
	return res, nil
}

// First returns the first view configuration matching filters (nil when
// none).
func First(req *medusahttp.MedusaRequest, fields []string, filters *dal.Record) (*dal.Record, error) {
	res, err := Graph(req.Ctx(), req, fields, filters, nil)
	if err != nil || len(res.Data) == 0 {
		return nil, err
	}
	r, _ := res.Data[0].(*dal.Record)
	return r, nil
}

func orDefault(v any, def any) any {
	if v == nil {
		return def
	}
	return v
}

// GET mirrors GET: the user's and the system default configurations of
// the entity.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	filters := dal.NewRecord()
	if req.FilterableFields != nil {
		for _, k := range req.FilterableFields.Keys() {
			v, _ := req.FilterableFields.Get(k)
			filters.Set(k, v)
		}
	}
	filters.Set("entity", req.Params["entity"])
	mine := dal.NewRecord()
	mine.Set("user_id", req.AuthString("actor_id"))
	defaults := dal.NewRecord()
	defaults.Set("is_system_default", true)
	filters.Set("$or", []any{mine, defaults})

	r, err := Graph(req.Ctx(), req, req.QueryConfig.Fields, filters, req.QueryConfig.Pagination)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("view_configurations", r.Data)
	out.Set("count", orDefault(httputils.MetaValue(r.Metadata, "count"), 0.0))
	out.Set("offset", orDefault(httputils.MetaValue(r.Metadata, "skip"), 0.0))
	out.Set("limit", orDefault(httputils.MetaValue(r.Metadata, "take"), 20.0))
	return res.JSON(out)
}

// POST mirrors POST: the input is the raw request body (not the validated
// one) with the entity and the owner (null for a system default).
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.Body.(*dal.Record)
	if body == nil {
		body = dal.NewRecord()
	}
	isDefault, _ := body.Get("is_system_default")
	name, _ := body.Get("name")
	if !settingsservices.Truthy(isDefault) && !settingsservices.Truthy(name) {
		return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Name is required unless creating a system default view")
	}
	input := dal.NewRecord()
	for _, k := range body.Keys() {
		v, _ := body.Get(k)
		input.Set(k, v)
	}
	input.Set("entity", req.Params["entity"])
	if settingsservices.Truthy(isDefault) {
		input.Set("user_id", nil)
	} else {
		input.Set("user_id", req.AuthString("actor_id"))
	}
	result, err := workflows.CreateViewConfigurationWorkflow.Run(req.Ctx(), req.Scope, input)
	if err != nil {
		return err
	}
	var id any
	if result != nil {
		id, _ = result.Get("id")
	}
	f := dal.NewRecord()
	f.Set("id", id)
	vc, err := First(req, req.QueryConfig.Fields, f)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if vc != nil {
		out.Set("view_configuration", vc)
	}
	return res.JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST}
