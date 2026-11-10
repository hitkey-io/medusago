// Package middlewares mirrors @medusajs/framework/src/http/middlewares.
package middlewares

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

const (
	apiError            = "api_error"
	invalidRequestError = "invalid_request_error"
	invalidStateError   = "invalid_state_error"
)

// ErrorBody is the JSON error payload returned by the API.
type ErrorBody struct {
	Code    string `json:"code,omitempty"`
	Type    string `json:"type"`
	Message string `json:"message"`
	// MessageFirst renders {message, type}: the body of http-errors errors.
	MessageFirst bool `json:"-"`
}

// MarshalJSON keeps the key order of the upstream body objects.
func (b ErrorBody) MarshalJSON() ([]byte, error) {
	r := dal.NewRecord()
	if b.MessageFirst {
		r.Set("message", b.Message)
		r.Set("type", b.Type)
		return r.MarshalJSON()
	}
	if b.Code != "" {
		r.Set("code", b.Code)
	}
	r.Set("type", b.Type)
	r.Set("message", b.Message)
	return r.MarshalJSON()
}

// ErrorHttpResponse mirrors ErrorHttpResponse from error-handler.ts.
type ErrorHttpResponse struct {
	StatusCode int
	Body       ErrorBody
}

// HttpError is the equivalent of the http-errors objects raised by express
// internals (body parsing, etc.): a status code plus a client-safe message.
type HttpError struct {
	StatusCode int
	Message    string
}

func (e *HttpError) Error() string { return e.Message }

// GetHttpResponseFromError resolves the status code and body an error is
// converted into, mirroring getHttpResponseFromError.
func GetHttpResponseFromError(err error) ErrorHttpResponse {
	var httpErr *HttpError
	if errors.As(err, &httpErr) {
		msg := httpErr.Message
		if httpErr.StatusCode >= 500 {
			msg = "An unknown error occurred."
		}
		return ErrorHttpResponse{
			StatusCode: httpErr.StatusCode,
			Body:       ErrorBody{Message: msg, Type: mapStatusCodeToErrorType(httpErr.StatusCode), MessageFirst: true},
		}
	}

	formatted := FormatException(err)
	me, ok := common.AsMedusaError(formatted)
	if !ok {
		return unknownError()
	}

	body := ErrorBody{Code: me.Code, Type: me.Type, Message: me.Message}
	statusCode := http.StatusInternalServerError

	switch me.Type {
	case common.MedusaErrorTypes.CONFLICT:
		statusCode = http.StatusConflict
		body.Code = invalidStateError
		body.Message = "The request conflicted with another request. You may retry the request with the provided Idempotency-Key."
	case common.MedusaErrorTypes.UNAUTHORIZED:
		statusCode = http.StatusUnauthorized
	case common.MedusaErrorTypes.FORBIDDEN:
		statusCode = http.StatusForbidden
	case common.MedusaErrorTypes.PAYMENT_AUTHORIZATION_ERROR:
		statusCode = http.StatusUnprocessableEntity
	case common.MedusaErrorTypes.DUPLICATE_ERROR:
		statusCode = http.StatusUnprocessableEntity
		body.Code = invalidRequestError
	case common.MedusaErrorTypes.NOT_ALLOWED, common.MedusaErrorTypes.INVALID_DATA:
		statusCode = http.StatusBadRequest
	case common.MedusaErrorTypes.NOT_FOUND:
		statusCode = http.StatusNotFound
	case common.MedusaErrorTypes.DB_ERROR:
		body.Code = apiError
	case common.MedusaErrorTypes.UNEXPECTED_STATE, common.MedusaErrorTypes.INVALID_ARGUMENT:
		// 500 with the original body
	default:
		return unknownError()
	}

	return ErrorHttpResponse{StatusCode: statusCode, Body: body}
}

func unknownError() ErrorHttpResponse {
	return ErrorHttpResponse{
		StatusCode: http.StatusInternalServerError,
		Body: ErrorBody{
			Code:    "unknown_error",
			Type:    "unknown_error",
			Message: "An unknown error occurred.",
		},
	}
}

// HandleError writes err to w the way the upstream coreErrorHandler does.
func HandleError(w http.ResponseWriter, r *http.Request, err error) {
	res := GetHttpResponseFromError(err)

	if res.StatusCode >= 500 {
		slog.Error("request failed", "path", r.URL.Path, "status", res.StatusCode, "error", err)
	} else {
		slog.Info(err.Error(), "path", r.URL.Path, "status", res.StatusCode)
	}

	WriteJSON(w, res.StatusCode, res.Body)
}

// WriteJSON writes v as a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	raw, err := dal.MarshalValue(v)
	if err != nil {
		raw, _ = json.Marshal(v)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(raw)
}

func mapStatusCodeToErrorType(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest:
		return common.MedusaErrorTypes.INVALID_DATA
	case http.StatusUnauthorized:
		return common.MedusaErrorTypes.UNAUTHORIZED
	case http.StatusForbidden:
		return common.MedusaErrorTypes.FORBIDDEN
	case http.StatusNotFound:
		return common.MedusaErrorTypes.NOT_FOUND
	case http.StatusConflict:
		return common.MedusaErrorTypes.CONFLICT
	default:
		return "unknown_error"
	}
}

// ErrorHandler mirrors errorHandler(): the core Express error middleware.
func ErrorHandler() medusahttp.ErrorHandlerFunction {
	return func(err error, req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, _ medusahttp.NextFunction) {
		r := GetHttpResponseFromError(err)
		if r.StatusCode >= 500 {
			slog.Error("request failed", "path", req.Path(), "status", r.StatusCode, "error", err)
		} else {
			slog.Info(err.Error())
		}
		// errors carrying zod issues (validators run outside the request
		// validation middlewares) answer with the issue messages
		var ze *zod.ZodError
		if errors.As(err, &ze) {
			msgs := make([]string, len(ze.Issues))
			for i, issue := range ze.Issues {
				msgs[i] = zod.FromZodIssue(issue)
			}
			body := dal.NewRecord()
			body.Set("type", common.MedusaErrorTypes.INVALID_DATA)
			body.Set("message", strings.Join(msgs, "\n"))
			_ = res.Status(r.StatusCode).JSON(body)
			return
		}
		_ = res.Status(r.StatusCode).JSON(r.Body)
	}
}
