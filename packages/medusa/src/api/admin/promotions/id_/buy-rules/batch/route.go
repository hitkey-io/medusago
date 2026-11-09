// Package batch mirrors api/admin/promotions/[id]/buy-rules/batch/route.ts.
package batch

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/promotion/steps"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/promotions"
)

// Route is the route file.
var Route = medusahttp.Route{POST: promotions.BatchRulesRoute(steps.RuleTypeBuyRules)}
