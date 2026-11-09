// Package uploads mirrors api/admin/uploads.
package uploads

import (
	"encoding/base64"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/file/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	localfile "github.com/hitkey-io/medusago/packages/modules/providers/file-local/src/services"
)

// RetrieveUploadConfig mirrors retrieveUploadConfig.
var RetrieveUploadConfig = httputils.QueryConfig{Defaults: []string{"id", "url"}, Entity: "file"}

// AdminUploadPreSignedUrl mirrors AdminUploadPreSignedUrl.
var AdminUploadPreSignedUrl = zod.Object(
	zod.F("originalname", zod.String()),
	zod.F("mime_type", zod.String()),
	zod.F("size", zod.Number()),
	zod.F("access", zod.Enum("public", "private").Optional()),
)

func mw(fns ...medusahttp.MiddlewareFunction) []medusahttp.MiddlewareFunction { return fns }

// Middlewares mirrors adminUploadRoutesMiddlewares (authorize is RBAC and
// not ported).
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"POST"}, Matcher: "/admin/uploads", Middlewares: mw(
		medusahttp.MulterArray("files"),
		httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveUploadConfig),
	)},
	{Methods: []string{"GET"}, Matcher: "/admin/uploads/:id", Middlewares: mw(
		httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveUploadConfig),
	)},
	{Methods: []string{"POST"}, Matcher: "/admin/uploads/presigned-urls", Middlewares: mw(
		httputils.ValidateAndTransformBody(httputils.Schema(AdminUploadPreSignedUrl)),
	)},
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	if len(req.Files) == 0 {
		return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "No files were uploaded")
	}
	files := make([]*localfile.UploadFile, len(req.Files))
	for i, f := range req.Files {
		files[i] = &localfile.UploadFile{Filename: f.OriginalName, MimeType: f.MimeType, Content: base64.StdEncoding.EncodeToString(f.Buffer), Access: "public"}
	}
	result, err := workflows.UploadFilesWorkflow.Run(req.Ctx(), req.Scope, files)
	if err != nil {
		return err
	}
	list := make([]any, len(result))
	for i, r := range result {
		list[i] = r
	}
	out := dal.NewRecord()
	out.Set("files", list)
	return res.Status(200).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
