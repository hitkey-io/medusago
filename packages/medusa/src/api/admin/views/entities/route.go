// Package entities mirrors api/admin/views/entities (behind view_configurations, not ported).
package entities

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/views/entity_/configurations"
)

// Route is the route file.
var Route = medusahttp.Route{GET: configurations.NotPorted}
