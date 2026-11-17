package products

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	featureflags "github.com/hitkey-io/medusago/packages/medusa/src/feature-flags"
)

// indexEngineEnabled mirrors FeatureFlag.isFeatureEnabled(IndexEngineFeatureFlag.key).
func indexEngineEnabled(req *medusahttp.MedusaRequest) bool {
	return featureflags.IsFeatureEnabled(req.Scope, featureflags.IndexEngine.Key)
}

// withEstimateCount inserts estimate_count (= count) right after count, like
// `{ products, count, ...(flag && { estimate_count: count }), offset, limit }`.
func withEstimateCount(out *dal.Record) {
	count, _ := out.Get("count")
	reordered := dal.NewRecord()
	for _, k := range out.Keys() {
		v, _ := out.Get(k)
		reordered.Set(k, v)
		if k == "count" {
			reordered.Set("estimate_count", count)
		}
	}
	for _, k := range out.Keys() {
		out.Delete(k)
	}
	for _, k := range reordered.Keys() {
		v, _ := reordered.Get(k)
		out.Set(k, v)
	}
}
