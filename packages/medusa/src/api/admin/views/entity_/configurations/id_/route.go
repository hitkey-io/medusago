// Package id_ mirrors api/admin/views/entity_/configurations/id_ (behind view_configurations, not ported).
package id_

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/views/entity_/configurations"
)

// Route is the route file.
var Route = medusahttp.Route{GET: configurations.NotPorted, POST: configurations.NotPorted, DELETE: configurations.NotPorted}
