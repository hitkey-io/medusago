// Package api holds the API routes and middlewares used by the API route
// chapters of the Learn documentation. In a project these files live under
// src/api.
package api

import (
	"strconv"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// region: query-string
// queryString returns a query parameter when it is a single string.
func queryString(q *dal.Record, key string) (string, bool) {
	if q == nil {
		return "", false
	}
	v, ok := q.Get(key)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// endregion: query-string

// region: body
// body returns the parsed request body as an object (empty when the request
// has no JSON object body).
func body(raw any) *dal.Record {
	if r, ok := raw.(*dal.Record); ok && r != nil {
		return r
	}
	return dal.NewRecord()
}

// endregion: body

// numberOf reads a number from a validated object.
func numberOf(r *dal.Record, key string) float64 {
	v, _ := r.Get(key)
	f, _ := v.(float64)
	return f
}

// region: parse-number
// parseNumber converts a query string value into a number, leaving other
// values untouched so that validation reports them.
func parseNumber(v any) any {
	s, ok := v.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return v
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return v
	}
	return f
}

// endregion: parse-number
