package middlewares

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
)

// PostgresErrorCode mirrors the PostgresError enum from exception-formatter.ts.
const (
	PostgresDuplicateError       = "23505"
	PostgresForeignKeyError      = "23503"
	PostgresSerializationFailure = "40001"
	PostgresNullViolation        = "23502"
)

// PostgresError is the driver-agnostic shape of a Postgres error. The database
// layer converts driver errors into this type so the formatter stays
// independent of the driver in use.
type PostgresError struct {
	Code    string
	Message string
	Detail  string
	Table   string
	Column  string
}

func (e *PostgresError) Error() string { return e.Message }

var foreignKeyDetailRe = regexp.MustCompile(`Key \(([\w-\d]+)\)=\(([\w-\d]+)\) is not present in table "(\w+)"`)

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// FormatException mirrors formatException: known Postgres errors are turned
// into MedusaErrors, everything else is returned unchanged.
func FormatException(err error) error {
	var pgErr *PostgresError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case PostgresDuplicateError:
		detail := pgErr.Detail
		if len(detail) > 4 {
			detail = detail[4:]
		} else {
			detail = ""
		}
		detail = strings.NewReplacer("(", "", ")", "", "=", " ").Replace(detail)
		return common.NewMedusaError(
			common.MedusaErrorTypes.DUPLICATE_ERROR,
			fmt.Sprintf("%s with %s", capitalize(pgErr.Table), detail),
		)
	case PostgresForeignKeyError:
		matches := foreignKeyDetailRe.FindStringSubmatch(pgErr.Detail)
		if len(matches) != 4 {
			// Upstream serializes the (null) match result with JSON.stringify.
			raw, _ := json.Marshal(matches)
			return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, string(raw))
		}
		return common.NewMedusaError(
			common.MedusaErrorTypes.NOT_FOUND,
			fmt.Sprintf("%s with %s %s does not exist.", capitalize(matches[3]), matches[1], matches[2]),
		)
	case PostgresSerializationFailure:
		msg := pgErr.Detail
		if msg == "" {
			msg = pgErr.Message
		}
		return common.NewMedusaError(common.MedusaErrorTypes.CONFLICT, msg)
	case PostgresNullViolation:
		return common.NewMedusaError(
			common.MedusaErrorTypes.INVALID_DATA,
			fmt.Sprintf("Can't insert null value in field %s on insert in table %s", pgErr.Column, pgErr.Table),
		)
	default:
		return err
	}
}
