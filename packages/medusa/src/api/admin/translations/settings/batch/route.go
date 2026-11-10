// Package batch mirrors api/admin/translations/settings/batch/route.ts.
package batch

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/translation/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	translationsbatch "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/translations/batch"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	in := translationsbatch.BatchBody(body)
	result, err := workflows.BatchTranslationSettingsWorkflow.Run(req.Ctx(), req.Scope, in)
	if err != nil {
		return err
	}
	deleted := dal.NewRecord()
	deleted.Set("ids", in.Delete)
	deleted.Set("object", "translation_settings")
	deleted.Set("deleted", true)
	out := dal.NewRecord()
	out.Set("created", result.Created)
	out.Set("updated", result.Updated)
	out.Set("deleted", deleted)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
