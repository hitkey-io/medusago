package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

func graphInput(entity string, fields []string, filters *dal.Record) query.GraphInput {
	return query.GraphInput{Entity: entity, Fields: fields, Filters: filters}
}
