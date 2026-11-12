package carts

import (
	"strconv"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// gcQueryGraph mirrors useQueryGraphStep({ entity, fields, filters, options:
// { throwIfKeyNotFound } }).config({ name }): the rows.
func gcQueryGraph(e *composer.Execution, name, entity string, fields []string, filters *dal.Record, throwIfKeyNotFound bool) ([]any, error) {
	res, err := commonsteps.UseQueryGraphStep.Config(name, composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
		GraphInput: query.GraphInput{Entity: entity, Fields: fields, Filters: filters},
		Options:    joiner.Options{ThrowIfKeyNotFound: throwIfKeyNotFound},
	})
	if err != nil {
		return nil, err
	}
	rows, _ := res.Data.([]any)
	if rows == nil {
		rows = []any{}
	}
	return rows, nil
}

// gcFirst returns rows[0] (nil when empty).
func gcFirst(rows []any) *dal.Record {
	if len(rows) == 0 {
		return nil
	}
	r, _ := rows[0].(*dal.Record)
	return r
}

// gcRecords converts a list of records.
func gcRecords(v any) []*dal.Record {
	switch t := v.(type) {
	case []*dal.Record:
		return t
	case []any:
		out := make([]*dal.Record, 0, len(t))
		for _, x := range t {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
}

// gcRecord builds an ordered record from key/value pairs.
func gcRecord(kv ...any) *dal.Record {
	r := dal.NewRecord()
	for i := 0; i+1 < len(kv); i += 2 {
		r.Set(kv[i].(string), kv[i+1])
	}
	return r
}

// gcGet returns r[k] (nil when absent).
func gcGet(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

// gcStr returns r[k] as a string.
func gcStr(r *dal.Record, k string) string {
	s, _ := gcGet(r, k).(string)
	return s
}

// gcToNumber mirrors BigNumber.toNumber().
func gcToNumber(bn *totals.BN) float64 {
	f, _ := strconv.ParseFloat(bn.Text('f'), 64)
	return f
}

// gcUndefinedProperty mirrors the TypeError of reading a property of
// undefined.
type gcUndefinedProperty string

func (p gcUndefinedProperty) Error() string {
	return "Cannot read properties of undefined (reading '" + string(p) + "')"
}
