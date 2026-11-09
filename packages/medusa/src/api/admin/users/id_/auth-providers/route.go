// Package authproviders mirrors api/admin/users/[id]/auth-providers/route.ts.
package authproviders

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	meta := dal.NewRecord()
	meta.Set("user_id", req.Params["id"])
	f := dal.NewRecord()
	f.Set("app_metadata", meta)
	r, err := httputils.QueryOf(req.Scope).Graph(req.Ctx(), query.GraphInput{Entity: "auth_identity", Fields: []string{"provider_identities.provider"}, Filters: f}, joiner.Options{})
	if err != nil {
		return err
	}
	providers := []any{}
	for _, row := range r.Data {
		rec, _ := row.(*dal.Record)
		if rec == nil {
			continue
		}
		pv, _ := rec.Get("provider_identities")
		var list []any
		switch t := pv.(type) {
		case []any:
			list = t
		case []*dal.Record:
			for _, x := range t {
				list = append(list, x)
			}
		}
		for _, p := range list {
			if pr, ok := p.(*dal.Record); ok {
				v, _ := pr.Get("provider")
				providers = append(providers, v)
			}
		}
	}
	out := dal.NewRecord()
	out.Set("providers", providers)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
