// Package common mirrors @medusajs/utils/src/common.
package common

import (
	"errors"
	"time"
)

// MedusaErrorTypes mirrors MedusaErrorTypes from errors.ts.
var MedusaErrorTypes = struct {
	DB_ERROR                    string
	DUPLICATE_ERROR             string
	INVALID_ARGUMENT            string
	INVALID_DATA                string
	UNAUTHORIZED                string
	FORBIDDEN                   string
	NOT_FOUND                   string
	NOT_ALLOWED                 string
	UNEXPECTED_STATE            string
	CONFLICT                    string
	UNKNOWN_MODULES             string
	PAYMENT_AUTHORIZATION_ERROR string
	PAYMENT_REQUIRES_MORE_ERROR string
}{
	DB_ERROR:                    "database_error",
	DUPLICATE_ERROR:             "duplicate_error",
	INVALID_ARGUMENT:            "invalid_argument",
	INVALID_DATA:                "invalid_data",
	UNAUTHORIZED:                "unauthorized",
	FORBIDDEN:                   "forbidden",
	NOT_FOUND:                   "not_found",
	NOT_ALLOWED:                 "not_allowed",
	UNEXPECTED_STATE:            "unexpected_state",
	CONFLICT:                    "conflict",
	UNKNOWN_MODULES:             "unknown_modules",
	PAYMENT_AUTHORIZATION_ERROR: "payment_authorization_error",
	PAYMENT_REQUIRES_MORE_ERROR: "payment_requires_more_error",
}

// MedusaErrorCodes mirrors MedusaErrorCodes from errors.ts.
var MedusaErrorCodes = struct {
	INSUFFICIENT_INVENTORY  string
	CART_INCOMPATIBLE_STATE string
	UNKNOWN_MODULES         string
}{
	INSUFFICIENT_INVENTORY:  "insufficient_inventory",
	CART_INCOMPATIBLE_STATE: "cart_incompatible_state",
	UNKNOWN_MODULES:         "unknown_modules",
}

// MedusaError is the standardized error used across the project.
type MedusaError struct {
	Type    string
	Message string
	Code    string
	Date    time.Time
}

// NewMedusaError creates a MedusaError. code is optional.
func NewMedusaError(errType, message string, code ...string) *MedusaError {
	e := &MedusaError{Type: errType, Message: message, Date: time.Now()}
	if len(code) > 0 {
		e.Code = code[0]
	}
	return e
}

func (e *MedusaError) Error() string { return e.Message }

// IsMedusaError reports whether err is (or wraps) a MedusaError.
func IsMedusaError(err error) bool {
	var me *MedusaError
	return errors.As(err, &me)
}

// AsMedusaError extracts a MedusaError from err.
func AsMedusaError(err error) (*MedusaError, bool) {
	var me *MedusaError
	ok := errors.As(err, &me)
	return me, ok
}
