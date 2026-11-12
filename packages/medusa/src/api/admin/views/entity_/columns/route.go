// Package columns mirrors api/admin/views/[entity]/columns (behind the
// view_configurations feature flag).
package columns

import (
	"fmt"
	"strings"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/settings/src/services"
)

const notInitializedMessage = "Entity discovery has not been initialized. Please ensure the application has fully started."

// GET mirrors GET: the available columns of an entity. Upstream answers
// filterReadableColumns(req, entity, columns), which only narrows the
// columns when RBAC is enabled (not ported): the columns are returned as
// generated.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	entity := req.Params["entity"]

	settings := services.Of(req.Scope.Resolve(utilsmodulessdk.SETTINGS))
	if settings == nil || !settings.IsEntityDiscoveryInitialized() {
		return common.NewMedusaError(common.MedusaErrorTypes.UNEXPECTED_STATE, notInitializedMessage)
	}

	if !settings.HasEntity(entity) {
		available, err := settings.ListDiscoverableEntities(req.Ctx())
		if err != nil {
			return err
		}
		names := []string{}
		for i, e := range available {
			if i >= 10 {
				break
			}
			names = append(names, e.PluralName)
		}
		more := ""
		if len(available) > 10 {
			more = "..."
		}
		return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA,
			fmt.Sprintf("Unsupported entity: %s. Available entities include: %s%s", entity, strings.Join(names, ", "), more))
	}

	columns, err := settings.GenerateEntityColumns(req.Ctx(), entity)
	if err != nil {
		return err
	}
	if columns == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.UNEXPECTED_STATE,
			fmt.Sprintf("Failed to generate columns for entity: %s", entity))
	}

	out := make([]any, len(columns))
	for i, c := range columns {
		out[i] = c
	}
	body := dal.NewRecord()
	body.Set("columns", out)
	return res.JSON(body)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
