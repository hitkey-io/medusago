// Package active mirrors api/admin/views/[entity]/configurations/active.
package active

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	settingsservices "github.com/hitkey-io/medusago/packages/modules/settings/src/services"
)

func settings(req *medusahttp.MedusaRequest) *settingsservices.SettingsModuleService {
	return settingsservices.Of(req.Scope.Resolve(utilsmodulessdk.SETTINGS))
}

// GET mirrors GET: the active configuration (the service DTO) or null with
// the code default.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	vc, err := settings(req).GetActiveViewConfiguration(req.Ctx(), req.Params["entity"], req.AuthString("actor_id"))
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if vc == nil {
		out.Set("view_configuration", nil)
		out.Set("is_default_active", true)
		out.Set("default_type", "code")
		return res.JSON(out)
	}
	isDefault, _ := vc.Get("is_system_default")
	out.Set("view_configuration", vc)
	out.Set("is_default_active", isDefault)
	if settingsservices.Truthy(isDefault) {
		out.Set("default_type", "system")
	}
	return res.JSON(out)
}

// POST mirrors POST: null clears the active configuration.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	var id any
	if body != nil {
		id, _ = body.Get("view_configuration_id")
	}
	svc := settings(req)
	entity, actor := req.Params["entity"], req.AuthString("actor_id")
	if id == nil {
		if err := svc.ClearActiveViewConfiguration(req.Ctx(), entity, actor); err != nil {
			return err
		}
	} else {
		s, _ := id.(string)
		if err := svc.SetActiveViewConfiguration(req.Ctx(), entity, actor, s); err != nil {
			return err
		}
	}
	out := dal.NewRecord()
	out.Set("success", true)
	return res.JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST}
