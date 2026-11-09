package http

import (
	"io"
	"mime"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// MulterArray mirrors multer({ storage: memoryStorage() }).array(field):
// multipart requests have their files in req.Files and their text fields
// in req.Body; other requests are left untouched.
func MulterArray(field string) MiddlewareFunction {
	return func(req *MedusaRequest, res *MedusaResponse, next NextFunction) error {
		ct, _, _ := mime.ParseMediaType(req.Header.Get("Content-Type"))
		if ct != "multipart/form-data" {
			next(nil)
			return nil
		}
		reader, err := req.MultipartReader()
		if err != nil {
			next(err)
			return nil
		}
		body := dal.NewRecord()
		files := []UploadedFile{}
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				next(err)
				return nil
			}
			data, err := io.ReadAll(part)
			if err != nil {
				next(err)
				return nil
			}
			if part.FileName() == "" {
				body.Set(part.FormName(), string(data))
				continue
			}
			if part.FormName() != field {
				next(&MulterError{Code: "LIMIT_UNEXPECTED_FILE", Field: part.FormName()})
				return nil
			}
			mt := part.Header.Get("Content-Type")
			if mt == "" {
				mt = "application/octet-stream"
			}
			files = append(files, UploadedFile{FieldName: part.FormName(), OriginalName: latin1(part.FileName()), MimeType: mt, Buffer: data})
		}
		req.Files = files
		req.Body = body
		next(nil)
		return nil
	}
}

// latin1 mirrors busboy's default filename decoding (latin1): each UTF-8
// byte becomes one character.
func latin1(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		b.WriteRune(rune(s[i]))
	}
	return b.String()
}

// MulterError mirrors MulterError.
type MulterError struct {
	Code  string
	Field string
}

func (e *MulterError) Error() string {
	if e.Code == "LIMIT_UNEXPECTED_FILE" {
		return "Unexpected field"
	}
	return e.Code
}
