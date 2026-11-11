package middlewares

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// StoreSearchConfigLocal is the request local holding the store search
// configuration (req.storeSearchConfig).
const StoreSearchConfigLocal = "storeSearchConfig"

// StoreSearchIndexConfig mirrors StoreSearchIndexConfig.
type StoreSearchIndexConfig struct {
	// Filters are ANDed onto every query for the index (nil: none).
	Filters func(req *medusahttp.MedusaRequest) (*dal.Record, error)
}

// StoreSearchConfig mirrors StoreSearchConfig. AllowedIndexes values are
// a bool or a *StoreSearchIndexConfig, like `boolean | StoreSearchIndexConfig`.
type StoreSearchConfig struct {
	AllowedIndexes map[string]any
}

// GetStoreSearchConfig returns req.storeSearchConfig (nil when unset).
func GetStoreSearchConfig(req *medusahttp.MedusaRequest) *StoreSearchConfig {
	if req.Locals == nil {
		return nil
	}
	c, _ := req.Locals[StoreSearchConfigLocal].(*StoreSearchConfig)
	return c
}

// ConfigureStoreSearch mirrors configureStoreSearch(config): the indexes
// POST /store/search may query, merged over what an earlier call allowed.
func ConfigureStoreSearch(config StoreSearchConfig) medusahttp.MiddlewareFunction {
	return func(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
		merged := &StoreSearchConfig{AllowedIndexes: map[string]any{}}
		if existing := GetStoreSearchConfig(req); existing != nil {
			for k, v := range existing.AllowedIndexes {
				merged.AllowedIndexes[k] = v
			}
		}
		for k, v := range config.AllowedIndexes {
			merged.AllowedIndexes[k] = v
		}
		if req.Locals == nil {
			req.Locals = map[string]any{}
		}
		req.Locals[StoreSearchConfigLocal] = merged
		next(nil)
		return nil
	}
}
