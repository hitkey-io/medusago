// Package entities mirrors api/admin/views/entities (behind the
// view_configurations feature flag).
package entities

import (
	"sort"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/settings/src/services"
	"github.com/hitkey-io/medusago/packages/modules/settings/src/utils"
)

// GET mirrors GET: the entities the settings module discovers in the
// schemas of the loaded modules, sorted by name.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	settings := services.Of(req.Scope.Resolve(utilsmodulessdk.SETTINGS))
	entities := []utils.EntityInfo{}
	if settings != nil {
		list, err := settings.ListDiscoverableEntities(req.Ctx())
		if err != nil {
			return err
		}
		entities = list
	}
	sort.SliceStable(entities, func(i, j int) bool {
		return utils.LocaleCompare(entities[i].Name, entities[j].Name) < 0
	})
	out := make([]any, len(entities))
	for i, e := range entities {
		r := dal.NewRecord()
		r.Set("name", e.Name)
		r.Set("pluralName", e.PluralName)
		r.Set("module", e.Module)
		r.Set("propertyCount", e.PropertyCount)
		r.Set("hasOverrides", e.HasOverrides)
		out[i] = r
	}
	body := dal.NewRecord()
	body.Set("entities", out)
	return res.JSON(body)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
