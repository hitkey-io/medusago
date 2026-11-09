package stores

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// RefetchStore mirrors refetchStore (nil when missing).
func RefetchStore(req *medusahttp.MedusaRequest, id any, fields []string) (any, error) {
	f := dal.NewRecord()
	f.Set("id", id)
	return httputils.RemoteQueryFirst(req.Ctx(), req.Scope, "store", f, fields)
}

// Respond writes { store } (undefined omitted).
func Respond(res *medusahttp.MedusaResponse, v any) error {
	out := dal.NewRecord()
	if v != nil {
		out.Set("store", v)
	}
	return res.Status(200).JSON(out)
}
