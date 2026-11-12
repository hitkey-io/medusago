// Package workflows mirrors src/workflows/gift-cards/workflows of
// @medusajs/loyalty-plugin.
package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// queryGraph mirrors useQueryGraphStep({ entity, fields, filters, options:
// { throwIfKeyNotFound } }).config({ name }): the rows.
func queryGraph(e *composer.Execution, name, entity string, fields []string, filters *dal.Record, throwIfKeyNotFound bool) ([]any, error) {
	res, err := commonsteps.UseQueryGraphStep.Config(name, composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
		GraphInput: query.GraphInput{Entity: entity, Fields: fields, Filters: filters},
		Options:    joiner.Options{ThrowIfKeyNotFound: throwIfKeyNotFound},
	})
	if err != nil {
		return nil, err
	}
	rows, _ := res.Data.([]any)
	return rows, nil
}

// first returns rows[0] (nil when empty).
func first(rows []any) *dal.Record {
	if len(rows) == 0 {
		return nil
	}
	r, _ := rows[0].(*dal.Record)
	return r
}

// filter builds a { key: value } filter.
func filter(key string, value any) *dal.Record {
	f := dal.NewRecord()
	f.Set(key, value)
	return f
}

// get returns r[k] (nil when absent).
func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

// str returns r[k] as a string.
func str(r *dal.Record, k string) string {
	s, _ := get(r, k).(string)
	return s
}

// record builds an ordered record from key/value pairs.
func record(kv ...any) *dal.Record {
	r := dal.NewRecord()
	for i := 0; i+1 < len(kv); i += 2 {
		r.Set(kv[i].(string), kv[i+1])
	}
	return r
}
