package featureflags

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	ff "github.com/hitkey-io/medusago/packages/core/utils/src/feature-flags"
)

// Resolver resolves container registrations.
type Resolver interface {
	Resolve(key string) any
}

// IsFeatureEnabled mirrors FeatureFlag.isFeatureEnabled(key) over the flag
// router registered in the container.
func IsFeatureEnabled(c Resolver, key string) bool {
	if c == nil {
		return false
	}
	router, _ := c.Resolve(common.ContainerRegistrationKeys.FEATURE_FLAG_ROUTER).(*ff.FlagRouter)
	return router != nil && router.IsFeatureEnabled(key)
}

// IsPresent mirrors isPresent.
func IsPresent(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	case []any:
		return len(t) > 0
	case interface{ Len() int }:
		return t.Len() > 0
	}
	return true
}
