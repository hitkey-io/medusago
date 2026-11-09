package users

import (
	"context"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// RefetchUser mirrors refetchUser.
func RefetchUser(ctx context.Context, userID string, scope medusahttp.MedusaContainer, fields []string) (any, error) {
	f := dal.NewRecord()
	f.Set("id", userID)
	vars := dal.NewRecord()
	vars.Set("filters", f)
	r, err := httputils.RemoteQueryList(ctx, scope, "user", vars, fields)
	if err != nil || len(r.Data) == 0 {
		return nil, err
	}
	return r.Data[0], nil
}
