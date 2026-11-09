package middlewares

import (
	"errors"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
)

func TestGetHttpResponseFromError(t *testing.T) {
	types := common.MedusaErrorTypes
	cases := []struct {
		name   string
		err    error
		status int
		body   ErrorBody
	}{
		{"not found", common.NewMedusaError(types.NOT_FOUND, "Product with id: x was not found"), 404,
			ErrorBody{Type: "not_found", Message: "Product with id: x was not found"}},
		{"invalid data", common.NewMedusaError(types.INVALID_DATA, "bad"), 400,
			ErrorBody{Type: "invalid_data", Message: "bad"}},
		{"not allowed", common.NewMedusaError(types.NOT_ALLOWED, "nope"), 400,
			ErrorBody{Type: "not_allowed", Message: "nope"}},
		{"unauthorized", common.NewMedusaError(types.UNAUTHORIZED, "Unauthorized"), 401,
			ErrorBody{Type: "unauthorized", Message: "Unauthorized"}},
		{"forbidden", common.NewMedusaError(types.FORBIDDEN, "Forbidden"), 403,
			ErrorBody{Type: "forbidden", Message: "Forbidden"}},
		{"duplicate", common.NewMedusaError(types.DUPLICATE_ERROR, "dup"), 422,
			ErrorBody{Code: "invalid_request_error", Type: "duplicate_error", Message: "dup"}},
		{"payment authorization", common.NewMedusaError(types.PAYMENT_AUTHORIZATION_ERROR, "declined"), 422,
			ErrorBody{Type: "payment_authorization_error", Message: "declined"}},
		{"conflict", common.NewMedusaError(types.CONFLICT, "whatever"), 409,
			ErrorBody{Code: "invalid_state_error", Type: "conflict", Message: "The request conflicted with another request. You may retry the request with the provided Idempotency-Key."}},
		{"db error", common.NewMedusaError(types.DB_ERROR, "db"), 500,
			ErrorBody{Code: "api_error", Type: "database_error", Message: "db"}},
		{"unexpected state keeps body", common.NewMedusaError(types.UNEXPECTED_STATE, "odd", "cart_incompatible_state"), 500,
			ErrorBody{Code: "cart_incompatible_state", Type: "unexpected_state", Message: "odd"}},
		{"plain error", errors.New("boom"), 500,
			ErrorBody{Code: "unknown_error", Type: "unknown_error", Message: "An unknown error occurred."}},
		{"http 4xx", &HttpError{StatusCode: 400, Message: "invalid json"}, 400,
			ErrorBody{Type: "invalid_data", Message: "invalid json"}},
		{"http 5xx", &HttpError{StatusCode: 503, Message: "internal"}, 503,
			ErrorBody{Type: "unknown_error", Message: "An unknown error occurred."}},
		{"pg duplicate", &PostgresError{Code: PostgresDuplicateError, Table: "product", Detail: "Key (handle)=(shirt) already exists."}, 422,
			ErrorBody{Code: "invalid_request_error", Type: "duplicate_error", Message: "Product with handle shirt already exists."}},
		{"pg foreign key", &PostgresError{Code: PostgresForeignKeyError, Detail: `Key (region_id)=(reg_1) is not present in table "region".`}, 404,
			ErrorBody{Type: "not_found", Message: "Region with region_id reg_1 does not exist."}},
		{"pg foreign key unparsable", &PostgresError{Code: PostgresForeignKeyError, Detail: "???"}, 404,
			ErrorBody{Type: "not_found", Message: "null"}},
		{"pg null violation", &PostgresError{Code: PostgresNullViolation, Table: "cart", Column: "currency_code"}, 400,
			ErrorBody{Type: "invalid_data", Message: "Can't insert null value in field currency_code on insert in table cart"}},
		{"pg serialization", &PostgresError{Code: PostgresSerializationFailure, Message: "could not serialize"}, 409,
			ErrorBody{Code: "invalid_state_error", Type: "conflict", Message: "The request conflicted with another request. You may retry the request with the provided Idempotency-Key."}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := GetHttpResponseFromError(tc.err)
			if got.StatusCode != tc.status {
				t.Errorf("status: got %d, want %d", got.StatusCode, tc.status)
			}
			if got.Body != tc.body {
				t.Errorf("body: got %+v, want %+v", got.Body, tc.body)
			}
		})
	}

}
