// Package analytics mirrors @medusajs/types/src/analytics.
package analytics

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// IAnalyticsProvider mirrors IAnalyticsProvider
// (AbstractAnalyticsProviderService). The data records are
// ProviderTrackAnalyticsEventDTO ({ event, actor_id?, group?: { type?, id? },
// properties? }) and ProviderIdentifyAnalyticsEventDTO ({ actor_id?,
// group?, properties? }).
type IAnalyticsProvider interface {
	Track(ctx context.Context, data *dal.Record) error
	Identify(ctx context.Context, data *dal.Record) error
}

// ShutdownProvider is implemented by providers with a shutdown (flushing
// queued events).
type ShutdownProvider interface {
	Shutdown(ctx context.Context) error
}
