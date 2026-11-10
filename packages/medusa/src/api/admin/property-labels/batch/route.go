// Package batch mirrors api/admin/property-labels/batch (behind view_configurations, not ported).
package batch

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/views/entity_/configurations"
)

// Route is the route file.
var Route = medusahttp.Route{POST: configurations.NotPorted}
