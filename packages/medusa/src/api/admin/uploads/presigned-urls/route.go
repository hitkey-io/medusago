// Package presignedurls mirrors api/admin/uploads/presigned-urls/route.ts.
package presignedurls

import (
	"fmt"
	"mime"
	"strings"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	filetypes "github.com/hitkey-io/medusago/packages/core/types/src/file"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	fileservices "github.com/hitkey-io/medusago/packages/modules/file/src/services"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	mt, _ := body.Get("mime_type")
	mimeType, _ := mt.(string)
	media, params, err := mime.ParseMediaType(mimeType)
	slash := strings.Index(media, "/")
	if err != nil || slash <= 0 || slash == len(media)-1 {
		return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Invalid file type \"%s\"", mimeType), common.MedusaErrorTypes.INVALID_DATA)
	}
	extension := media[slash+1:]
	svc := req.Scope.Resolve(utilsmodulessdk.FILE).(*fileservices.FileModuleService)
	access := "private"
	if a, ok := body.Get("access"); ok && a != nil {
		access = fmt.Sprint(a)
	}
	result, err := svc.GetUploadFileURL(req.Ctx(), filetypes.ProviderGetPresignedUploadUrlDTO{
		Filename: fmt.Sprintf("%s.%s", common.ULID(), extension),
		MimeType: mimeType,
		Access:   access,
	})
	if err != nil {
		return err
	}
	typeString := media
	for k, v := range params {
		typeString += ";" + k + "=" + v
	}
	out := dal.NewRecord()
	out.Set("url", result.URL)
	out.Set("filename", result.Key)
	out.Set("mime_type", typeString)
	size, _ := body.Get("size")
	out.Set("size", size)
	out.Set("extension", extension)
	orig, _ := body.Get("originalname")
	out.Set("originalname", orig)
	return res.JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
