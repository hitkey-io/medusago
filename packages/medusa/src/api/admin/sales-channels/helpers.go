package saleschannels

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// RefetchSalesChannel mirrors refetchSalesChannel (nil when missing).
func RefetchSalesChannel(req *medusahttp.MedusaRequest, id any, fields []string) (any, error) {
	f := dal.NewRecord()
	f.Set("id", id)
	return httputils.RemoteQueryFirst(req.Ctx(), req.Scope, "sales_channel", f, fields)
}

// Respond writes { sales_channel } with status 200 (undefined omitted).
func Respond(res *medusahttp.MedusaResponse, v any) error {
	out := dal.NewRecord()
	if v != nil {
		out.Set("sales_channel", v)
	}
	return res.Status(200).JSON(out)
}
