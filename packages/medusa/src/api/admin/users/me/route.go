// Package me mirrors api/admin/users/me/route.ts.
package me

import (
	"fmt"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.AuthString("actor_id")
	if id == "" {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, "User ID not found")
	}
	vars := dal.NewRecord()
	vars.Set("id", id)
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "user", vars, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	if len(r.Data) == 0 || r.Data[0] == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("User with id: %s was not found", id))
	}
	out := dal.NewRecord()
	out.Set("user", r.Data[0])
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
