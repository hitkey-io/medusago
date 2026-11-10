// Package statistics mirrors api/admin/translations/statistics/route.ts.
package statistics

import (
	"fmt"
	"strings"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/translation/src/services"
)

func strings_(v any) []string {
	list, _ := v.([]any)
	out := make([]string, 0, len(list))
	for _, x := range list {
		out = append(out, fmt.Sprint(x))
	}
	return out
}

// IsServiceNotFound mirrors the catch of the entity queries: a missing
// service alias counts as no entities.
func IsServiceNotFound(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "service with alias") && strings.Contains(msg, "was not found")
}

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	svc, _ := req.Scope.Resolve(utilsmodulessdk.TRANSLATION).(*services.TranslationModuleService)
	q := httputils.QueryOf(req.Scope)
	locales, _ := req.ValidatedQuery.Get("locales")
	entityTypes, _ := req.ValidatedQuery.Get("entity_types")

	var order []string
	counts := map[string]int{}
	for _, entityType := range strings_(entityTypes) {
		pagination := dal.NewRecord()
		pagination.Set("take", float64(1))
		pagination.Set("skip", float64(0))
		count := 0
		r, err := q.Graph(req.Ctx(), query.GraphInput{Entity: entityType, Fields: []string{"id"}, Pagination: pagination}, joiner.Options{ThrowIfKeyNotFound: false})
		if err != nil {
			if !IsServiceNotFound(err) {
				return err
			}
		} else if r.Metadata != nil {
			if c, ok := r.Metadata.Get("count"); ok {
				if f, isNum := c.(float64); isNum {
					count = int(f)
				}
			}
		}
		if _, seen := counts[entityType]; !seen {
			order = append(order, entityType)
		}
		counts[entityType] = count
	}
	stats, err := svc.GetStatistics(req.Ctx(), strings_(locales), order, counts)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("statistics", stats)
	return res.JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
