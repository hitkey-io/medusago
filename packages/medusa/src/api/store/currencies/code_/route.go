// Package code_ mirrors api/store/currencies/[code]/route.ts.
package code_

import (
	"fmt"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	f := dal.NewRecord()
	f.Set("code", req.Params["code"])
	c, err := httputils.RemoteQueryFirst(req.Ctx(), req.Scope, "currency", f, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	if c == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Currency with code: %s was not found", req.Params["code"]))
	}
	out := dal.NewRecord()
	out.Set("currency", c)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
