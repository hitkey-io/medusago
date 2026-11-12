// Package hooks mirrors src/workflows/hooks: the handlers of the core
// workflow hooks, registered for the applications using the plugin.
package hooks

import (
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/modules/loyalty"
)

// pluginEnabled reports an application using the plugin (its loyalty
// module is loaded).
func pluginEnabled(c composer.Container) bool {
	return c != nil && c.Resolve(loyalty.ServiceName) != nil
}

// Register registers the hook handlers (importing src/workflows/hooks).
func Register() {
	registerAfterOrderCreated()
	registerAfterOrderCreditLinesCreated()
	registerBeforePaymentCollectionRefresh()
	registerCompleteCartBeforePaymentAuthorization()
}
