// Package settings mirrors api/admin/translations/settings/route.ts.
package settings

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/translation/src/services"
)

func stringsToAny(list []string) []any {
	out := make([]any, len(list))
	for i, s := range list {
		out[i] = s
	}
	return out
}

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	svc, _ := req.Scope.Resolve(utilsmodulessdk.TRANSLATION).(*services.TranslationModuleService)
	entityType := ""
	if req.ValidatedQuery != nil {
		if v, ok := req.ValidatedQuery.Get("entity_type"); ok {
			entityType, _ = v.(string)
		}
	}
	fields, err := svc.GetTranslatableFields(req.Ctx(), entityType)
	if err != nil {
		return err
	}
	inactive, err := svc.GetInactiveTranslatableFields(req.Ctx(), entityType)
	if err != nil {
		return err
	}
	settings, err := svc.ListTranslationSettings(req.Ctx(), req.FilterableFields, utilsmodulessdk.FindConfig{})
	if err != nil {
		return err
	}
	byType := map[string]*dal.Record{}
	for _, s := range settings {
		et, _ := s.Get("entity_type")
		if str, ok := et.(string); ok {
			byType[str] = s
		}
	}
	out := dal.NewRecord()
	for _, et := range fields.Keys {
		setting, ok := byType[et]
		if !ok {
			continue
		}
		r := dal.NewRecord()
		id, _ := setting.Get("id")
		r.Set("id", id)
		r.Set("fields", stringsToAny(fields.Fields[et]))
		if f, ok := inactive.Fields[et]; ok {
			r.Set("inactive_fields", stringsToAny(f))
		}
		active, _ := setting.Get("is_active")
		r.Set("is_active", active)
		out.Set(et, r)
	}
	body := dal.NewRecord()
	body.Set("translation_settings", out)
	return res.JSON(body)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
