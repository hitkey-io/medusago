package middlewares

import (
	"io"
	"mime"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// This file mirrors bodyparser.ts and the body-parser 1.x parsers it uses
// (json strict, text, urlencoded extended) with their error behaviour.

// BodyParser returns the json/text/urlencoded parser for a config (nil =
// defaults), for ApiLoaderOptions.BodyParser.
func BodyParser(kind string, cfg *medusahttp.BodyParserConfig) medusahttp.MiddlewareFunction {
	limit := int64(100 * 1024)
	preserve := false
	if cfg != nil {
		if cfg.SizeLimit != "" {
			if n, ok := parseBytes(cfg.SizeLimit); ok {
				limit = n
			}
		}
		preserve = cfg.PreserveRawBody
	}
	return func(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
		if _, done := req.Locals["_body"]; done {
			next(nil)
			return nil
		}
		if req.Body == nil {
			req.Body = dal.NewRecord()
		}
		if !hasBody(req) {
			next(nil)
			return nil
		}
		if !typeIs(req.Header.Get("Content-Type"), kind) {
			next(nil)
			return nil
		}
		req.Locals["_body"] = true

		charset := contentCharset(req.Header.Get("Content-Type"))
		if kind == "json" && charset != "" && !strings.HasPrefix(charset, "utf-") {
			return &HttpError{StatusCode: 415, Message: `unsupported charset "` + strings.ToUpper(charset) + `"`}
		}
		if kind == "urlencoded" && charset != "" && charset != "utf-8" {
			return &HttpError{StatusCode: 415, Message: `unsupported charset "` + strings.ToUpper(charset) + `"`}
		}
		if cl := req.Header.Get("Content-Length"); cl != "" {
			if n, err := strconv.ParseInt(cl, 10, 64); err == nil && n > limit {
				return &HttpError{StatusCode: 413, Message: "request entity too large"}
			}
		}
		raw, err := io.ReadAll(io.LimitReader(req.Request.Body, limit+1))
		if err != nil {
			return &HttpError{StatusCode: 400, Message: "request aborted"}
		}
		if int64(len(raw)) > limit {
			return &HttpError{StatusCode: 413, Message: "request entity too large"}
		}
		if preserve {
			req.RawBody = raw
		}
		str := string(raw)
		if !utf8.Valid(raw) {
			str = strings.ToValidUTF8(str, "�")
		}
		switch kind {
		case "json":
			if len(str) == 0 {
				req.Body = dal.NewRecord()
				break
			}
			if m := firstCharRe.FindStringSubmatch(str); m != nil && m[1] != "{" && m[1] != "[" {
				return &HttpError{StatusCode: 400, Message: strictSyntaxError(str, m[1]).Error()}
			}
			if err := ValidateJSONLikeV8(str); err != nil {
				return &HttpError{StatusCode: 400, Message: err.Error()}
			}
			v, err := dal.DecodeOrderedJSON([]byte(str))
			if err != nil {
				return &HttpError{StatusCode: 400, Message: err.Error()}
			}
			req.Body = v
		case "text":
			req.Body = str
		case "urlencoded":
			if strings.Count(str, "&")+1 > 1000 {
				return &HttpError{StatusCode: 413, Message: "too many parameters"}
			}
			req.Body = medusahttp.ParseQSBody(str)
		}
		next(nil)
		return nil
	}
}

var firstCharRe = regexp.MustCompile(`^[\x20\x09\x0a\x0d]*([^\x20\x09\x0a\x0d])`)

func hasBody(req *medusahttp.MedusaRequest) bool {
	if len(req.TransferEncoding) > 0 || req.Header.Get("Transfer-Encoding") != "" {
		return true
	}
	cl := req.Header.Get("Content-Length")
	if cl == "" {
		return req.ContentLength > 0
	}
	_, err := strconv.ParseFloat(cl, 64)
	return err == nil
}

func typeIs(contentType, kind string) bool {
	if contentType == "" {
		return false
	}
	mt, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	switch kind {
	case "json":
		return mt == "application/json" || (strings.HasPrefix(mt, "application/") && strings.HasSuffix(mt, "+json"))
	case "text":
		return mt == "text/plain"
	case "urlencoded":
		return mt == "application/x-www-form-urlencoded"
	}
	return false
}

func contentCharset(contentType string) string {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return ""
	}
	return strings.ToLower(params["charset"])
}

var bytesRe = regexp.MustCompile(`(?i)^((-|\+)?(\d+(?:\.\d+)?)) *(kb|mb|gb|tb|pb|b)?$`)

// parseBytes mirrors the bytes package parse.
func parseBytes(s string) (int64, bool) {
	m := bytesRe.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n, true
		}
		return 0, false
	}
	f, _ := strconv.ParseFloat(m[1], 64)
	mult := map[string]float64{"": 1, "b": 1, "kb": 1 << 10, "mb": 1 << 20, "gb": 1 << 30, "tb": 1 << 40, "pb": 1 << 50}[strings.ToLower(m[4])]
	return int64(f * mult), true
}
