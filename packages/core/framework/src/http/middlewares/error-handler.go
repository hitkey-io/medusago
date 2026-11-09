// Package middlewares mirrors @medusajs/framework/src/http/middlewares.
package middlewares

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
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
			Body:       ErrorBody{Message: msg, Type: mapStatusCodeToErrorType(httpErr.StatusCode)},
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
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
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
