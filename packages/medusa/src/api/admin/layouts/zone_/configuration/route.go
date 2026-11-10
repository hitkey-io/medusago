// Package configuration mirrors api/admin/layouts/[zone]/configuration.
package configuration

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/settings/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/settings/workflows"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	settingsservices "github.com/hitkey-io/medusago/packages/modules/settings/src/services"
)

func first(ctx context.Context, svc *utilsmodulessdk.MedusaService, model string, filters *dal.Record) (*dal.Record, error) {
	take := 1
	rows, err := svc.Internal(model).List(ctx, filters, utilsmodulessdk.FindConfig{Take: &take})
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return rows[0], nil
}

// getConfigurations mirrors getConfigurations.
func getConfigurations(ctx context.Context, svc *utilsmodulessdk.MedusaService, zone, userID string) (*dal.Record, error) {
	f := dal.NewRecord()
	f.Set("zone", zone)
	f.Set("user_id", userID)
	personal, err := first(ctx, svc, "LayoutConfiguration", f)
	if err != nil {
		return nil, err
	}
	// getSystemDefaultLayoutConfiguration
	d := dal.NewRecord()
	d.Set("zone", zone)
	d.Set("is_system_default", true)
	def, err := first(ctx, svc, "LayoutConfiguration", d)
	if err != nil {
		return nil, err
	}
	// getActiveLayoutScope: user preference "active_layout.<zone>"
	p := dal.NewRecord()
	p.Set("user_id", userID)
	p.Set("key", "active_layout."+zone)
	pref, err := first(ctx, svc, "UserPreference", p)
	if err != nil {
		return nil, err
	}
	stored := ""
	if pref != nil {
		if v, ok := pref.Get("value"); ok {
			if r, ok := v.(*dal.Record); ok {
				s, _ := r.Get("scope")
				if str, ok := s.(string); ok && (str == "personal" || str == "default") {
					stored = str
				}
			}
		}
	}
	active := "default"
	switch {
	case stored == "personal" && personal != nil:
		active = "personal"
	case stored == "default":
		active = "default"
	case personal != nil:
		active = "personal"
	}
	out := dal.NewRecord()
	if personal != nil {
		out.Set("personal_configuration", personal)
	} else {
		out.Set("personal_configuration", nil)
	}
	if def != nil {
		out.Set("default_configuration", def)
	} else {
		out.Set("default_configuration", nil)
	}
	out.Set("active_scope", active)
	return out, nil
}

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	svc := settingsservices.Of(req.Scope.Resolve(utilsmodulessdk.SETTINGS)).MedusaService
	out, err := getConfigurations(req.Ctx(), svc, req.Params["zone"], req.AuthString("actor_id"))
	if err != nil {
		return err
	}
	return res.JSON(out)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	isDefault, _ := body.Get("is_default")
	cfg, _ := body.Get("configuration")
	in := steps.SetLayoutConfigurationStepInput{Zone: req.Params["zone"], UserID: req.AuthString("actor_id"), Configuration: recordOrNil(cfg)}
	in.IsDefault, _ = isDefault.(bool)
	if _, err := workflows.SetLayoutConfigurationWorkflow.Run(req.Ctx(), req.Scope, in); err != nil {
		return err
	}
	return GET(req, res)
}

func recordOrNil(v any) *dal.Record {
	r, _ := v.(*dal.Record)
	return r
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	if _, err := workflows.ClearLayoutConfigurationWorkflow.Run(req.Ctx(), req.Scope, steps.ClearLayoutConfigurationStepInput{Zone: req.Params["zone"], UserID: req.AuthString("actor_id")}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("success", true)
	return res.JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST, DELETE: DELETE}

var layoutWidgetPreference = zod.Object(
	zod.F("hidden", zod.Boolean().Optional()),
	zod.F("section", zod.String().Optional()),
	zod.F("order", zod.Number().Optional()),
)

// AdminSetLayoutConfiguration mirrors AdminSetLayoutConfiguration.
var AdminSetLayoutConfiguration = zod.Object(
	zod.F("is_default", zod.Boolean().Optional().Default(false)),
	zod.F("configuration", zod.Object(zod.F("widgets", zod.Record(zod.String(), layoutWidgetPreference)))),
)

// Middlewares mirrors layoutConfigurationRoutesMiddlewares (requireSuperAdmin
// is a no-op without RBAC).
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"POST"}, Matcher: "/admin/layouts/:zone/configuration", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminSetLayoutConfiguration))}},
}
