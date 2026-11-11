// Package code_ mirrors api/admin/locales/[code]/route.ts.
package code_

import (
	"fmt"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/caching"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	filters := dal.NewRecord()
	filters.Set("code", req.Params["code"])
	r, err := httputils.QueryOf(req.Scope).Graph(req.Ctx(), query.GraphInput{Entity: "locale", Fields: req.QueryConfig.Fields, Filters: filters}, joiner.Options{Cache: caching.Enabled()})
	if err != nil {
		return err
	}
	if len(r.Data) == 0 {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Locale with code: %s was not found", req.Params["code"]))
	}
	out := dal.NewRecord()
	out.Set("locale", r.Data[0])
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
