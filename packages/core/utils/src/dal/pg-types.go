package dal

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Querier is satisfied by *pgxpool.Pool, *pgx.Conn and pgx.Tx.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Row is a result row: column name -> value parsed like node-postgres.
type Row map[string]any

// Postgres type OIDs handled like the node-postgres text parsers.
const (
	oidBool        = 16
	oidInt8        = 20
	oidInt2        = 21
	oidInt4        = 23
	oidOID         = 26
	oidJSON        = 114
	oidFloat4      = 700
	oidFloat8      = 701
	oidTextArray   = 1009
	oidVarcharArr  = 1015
	oidTimestamp   = 1114
	oidTimestamptz = 1184
	oidDate        = 1082
	oidJSONB       = 3802
)

// queryRows runs a query with the simple protocol (text results, like
// node-postgres) and parses every column the way node-postgres does.
func queryRows(ctx context.Context, db Querier, sql string, args ...any) ([]Row, error) {
	rows, err := db.Query(ctx, sql, append([]any{pgx.QueryExecModeSimpleProtocol}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fields := rows.FieldDescriptions()
	var out []Row
	for rows.Next() {
		raw := rows.RawValues()
		row := make(Row, len(fields))
		for i, f := range fields {
			if raw[i] == nil {
				row[f.Name] = nil
				continue
			}
			v, err := parseText(f.DataTypeOID, string(raw[i]))
			if err != nil {
				return nil, fmt.Errorf("column %s: %w", f.Name, err)
			}
			row[f.Name] = v
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func parseText(oid uint32, s string) (any, error) {
	switch oid {
	case oidBool:
		return s == "t", nil
	case oidInt2, oidInt4, oidOID:
		n, err := strconv.ParseInt(s, 10, 64)
		return float64(n), err
	case oidFloat4, oidFloat8:
		return strconv.ParseFloat(s, 64)
	case oidTimestamptz, oidTimestamp:
		return parseTimestamp(s, oid == oidTimestamp)
	case oidJSON, oidJSONB:
		return DecodeOrderedJSON([]byte(s))
	case oidTextArray, oidVarcharArr:
		return parseTextArray(s)
	default:
		// text, varchar, numeric, int8, date, uuid... stay strings like in
		// node-postgres.
		return s, nil
	}
}

func parseTimestamp(s string, local bool) (time.Time, error) {
	layouts := []string{"2006-01-02 15:04:05.999999-07", "2006-01-02 15:04:05.999999-07:00", "2006-01-02 15:04:05.999999-07:00:00"}
	if local {
		layouts = []string{"2006-01-02 15:04:05.999999"}
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			// Dates have millisecond precision in JavaScript.
			return t.Truncate(time.Millisecond), nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse timestamp %q", s)
}

// parseTextArray parses a one-dimensional Postgres text array literal.
func parseTextArray(s string) ([]any, error) {
	if len(s) < 2 || s[0] != '{' || s[len(s)-1] != '}' {
		return nil, fmt.Errorf("invalid array %q", s)
	}
	body := s[1 : len(s)-1]
	out := []any{}
	if body == "" {
		return out, nil
	}
	var cur strings.Builder
	inQuotes, escaped, quoted := false, false, false
	flush := func() {
		v := cur.String()
		if !quoted && v == "NULL" {
			out = append(out, nil)
		} else {
			out = append(out, v)
		}
		cur.Reset()
		quoted = false
	}
	for _, r := range body {
		switch {
		case escaped:
			cur.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case r == '"':
			inQuotes = !inQuotes
			quoted = true
		case r == ',' && !inQuotes:
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return out, nil
}

// QueryRows runs a query and parses its rows like node-postgres.
func QueryRows(ctx context.Context, db Querier, sql string, args ...any) ([]Row, error) {
	return queryRows(ctx, db, sql, args...)
}
