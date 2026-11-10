// Package id_ mirrors api/admin/views/[entity]/configurations/[id].
package id_

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/settings/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/views/entity_/configurations"
	settingsservices "github.com/hitkey-io/medusago/packages/modules/settings/src/services"
)

func byIDAndEntity(req *medusahttp.MedusaRequest) *dal.Record {
	f := dal.NewRecord()
	f.Set("id", req.Params["id"])
	f.Set("entity", req.Params["entity"])
	return f
}

func notFound(req *medusahttp.MedusaRequest) error {
	return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("View configuration with id: %s not found", req.Params["id"]))
}

// ownedByOther reports whether the configuration has an owner other than
// the actor.
func ownedByOther(req *medusahttp.MedusaRequest, vc *dal.Record) bool {
	owner, _ := vc.Get("user_id")
	return settingsservices.Truthy(owner) && owner != req.AuthString("actor_id")
}

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	vc, err := configurations.First(req, req.QueryConfig.Fields, byIDAndEntity(req))
	if err != nil {
		return err
	}
	if vc == nil {
		return notFound(req)
	}
	if ownedByOther(req, vc) {
		isAdmin := false
		if req.AuthContext != nil {
			if md, _ := req.AuthContext.Get("app_metadata"); md != nil {
				if r, ok := md.(*dal.Record); ok {
					a, _ := r.Get("admin")
					isAdmin = settingsservices.Truthy(a)
				}
			}
		}
		if !isAdmin {
			return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "You don't have access to this view configuration")
		}
	}
	out := dal.NewRecord()
	out.Set("view_configuration", vc)
	return res.JSON(out)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	existing, err := configurations.First(req, []string{"id", "user_id", "is_system_default"}, byIDAndEntity(req))
	if err != nil {
		return err
	}
	if existing == nil {
		return notFound(req)
	}
	if ownedByOther(req, existing) {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "You can only update your own view configurations")
	}
	input := dal.NewRecord()
	input.Set("id", req.Params["id"])
	if body, ok := req.ValidatedBody.(*dal.Record); ok && body != nil {
		for _, k := range body.Keys() {
			v, _ := body.Get(k)
			input.Set(k, v)
		}
	}
	if _, err := workflows.UpdateViewConfigurationWorkflow.Run(req.Ctx(), req.Scope, input); err != nil {
		return err
	}
	vc, err := configurations.First(req, req.QueryConfig.Fields, byIDAndEntity(req))
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if vc != nil {
		out.Set("view_configuration", vc)
	}
	return res.JSON(out)
}

// DELETE mirrors DELETE (the configuration is hard deleted).
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	existing, err := configurations.First(req, []string{"id", "user_id", "is_system_default", "entity", "name"}, byIDAndEntity(req))
	if err != nil {
		return err
	}
	if existing == nil {
		return notFound(req)
	}
	if ownedByOther(req, existing) {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "You can only delete your own view configurations")
	}
	svc := settingsservices.Of(req.Scope.Resolve(utilsmodulessdk.SETTINGS))
	if err := svc.DeleteViewConfigurations(req.Ctx(), []any{req.Params["id"]}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", req.Params["id"])
	out.Set("object", "view_configuration")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST, DELETE: DELETE}
