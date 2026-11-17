// Package sync mirrors api/admin/index/sync.
package sync

import (
	"context"
	"fmt"
	"log/slog"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

type syncer interface {
	Sync(ctx context.Context, strategy string) error
}

// POST mirrors POST: indexService.sync({ strategy }), answered with
// res.send(200) ("OK").
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	warnIndexDeprecation(req)
	index, _ := req.Scope.Resolve(utilsmodulessdk.INDEX).(syncer)
	strategy := ""
	if body, ok := req.ValidatedBody.(*dal.Record); ok {
		if v, ok := body.Get("strategy"); ok && v != nil {
			strategy = fmt.Sprint(v)
		}
	}
	if err := index.Sync(req.Ctx(), strategy); err != nil {
		return err
	}
	res.SendStatus(200)
	return nil
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}

// warnIndexDeprecation mirrors
// req.scope.resolve(LOGGER).warn(INDEX_MODULE_DEPRECATION_MESSAGE).
func warnIndexDeprecation(req *medusahttp.MedusaRequest) {
	logger, ok := req.Scope.Resolve(common.ContainerRegistrationKeys.LOGGER).(*slog.Logger)
	if !ok || logger == nil {
		logger = slog.Default()
	}
	logger.Warn(utilsmodulessdk.IndexModuleDeprecationMessage)
}
