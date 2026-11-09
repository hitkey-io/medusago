// Package configuration mirrors api/admin/layouts/[zone]/configuration.
package configuration

import (
	"context"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
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
	svc := req.Scope.Resolve(utilsmodulessdk.SETTINGS).(*utilsmodulessdk.MedusaService)
	out, err := getConfigurations(req.Ctx(), svc, req.Params["zone"], req.AuthString("actor_id"))
	if err != nil {
		return err
	}
	return res.JSON(out)
}

// Route is the route file (POST and DELETE need workflows, not ported yet).
var Route = medusahttp.Route{GET: GET}
