// Package subscribers mirrors src/subscribers.
package subscribers

import "github.com/hitkey-io/medusago/packages/core/framework/src/plugins"

// All are the subscribers of the plugin, in directory order.
func All() []plugins.Subscriber {
	var out []plugins.Subscriber
	if createGiftCard.Handler != nil {
		out = append(out, createGiftCard)
	}
	return out
}
