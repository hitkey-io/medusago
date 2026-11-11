package utils

import (
	"fmt"
	"math"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	searchutils "github.com/hitkey-io/medusago/packages/core/utils/src/search"
)

// WeightedPart is one weighted text part of a document's search vector.
type WeightedPart struct {
	Text   string
	Weight string
}

// Projection mirrors the result of projectIndexedDocument.
type Projection struct {
	ID            string
	Indexed       *dal.Record
	SearchText    string
	WeightedParts []WeightedPart
	Vectors       map[string][]float64
}

// coerce mirrors coerce (nil: undefined).
func coerce(value any, planned *PlannedField) any {
	switch planned.Kind {
	case "number":
		n := searchutils.SearchValueToNumber(value, planned.IsDate)
		if n == nil {
			return nil
		}
		return *n
	case "boolean":
		return searchutils.SearchValueToBoolean(value)
	case "text", "keyword":
		if s, ok := value.(string); ok {
			return s
		}
		return searchutils.JSString(value)
	}
	return value
}

func coerceVector(value any, planned *PlannedField) ([]float64, error) {
	arr, ok := value.([]any)
	if !ok {
		dims := "undefined"
		if planned.Dimensions != nil {
			dims = fmt.Sprint(*planned.Dimensions)
		}
		return nil, fail(fmt.Sprintf(`Vector field "%s" must be a numeric array of length %s`, planned.Path, dims))
	}
	nums := make([]float64, len(arr))
	for i, e := range arr {
		nums[i] = searchutils.JSNumber(e)
		if math.IsNaN(nums[i]) {
			return nil, fail(fmt.Sprintf(`Vector field "%s" contains non-numeric components`, planned.Path))
		}
	}
	if planned.Dimensions != nil && len(nums) != *planned.Dimensions {
		return nil, fail(fmt.Sprintf(`Vector field "%s" expected %d dimensions, got %d`, planned.Path, *planned.Dimensions, len(nums)))
	}
	return nums, nil
}

// ProjectIndexedDocument mirrors projectIndexedDocument: the indexed shape
// used for filters and full text, the primary key, the searchable text
// and the vectors.
func ProjectIndexedDocument(document *dal.Record, plan *IndexPlan) (*Projection, error) {
	p := &Projection{Indexed: dal.NewRecord(), Vectors: map[string][]float64{}}
	var textParts []string
	for _, planned := range plan.List() {
		values := searchutils.ReadDocumentPath(document, strings.Split(planned.Path, "."))
		if len(values) == 0 {
			continue
		}
		if planned.Kind == "vector" {
			if planned.Field.Embed != nil && *planned.Field.Embed {
				continue
			}
			embedding, err := coerceVector(values[0], planned)
			if err != nil {
				return nil, err
			}
			p.Vectors[planned.Path] = embedding
			continue
		}
		var coerced []any
		for _, v := range values {
			if c := coerce(v, planned); c != nil {
				coerced = append(coerced, c)
			}
		}
		if len(coerced) == 0 {
			continue
		}
		var value any = coerced[0]
		if planned.IsArray {
			value = coerced
		}
		searchutils.SetDocumentPath(p.Indexed, planned.Path, value)

		if planned.Kind == "text" || planned.Kind == "keyword" || (planned.Kind == "number" && !planned.IsDate) {
			entries := []any{value}
			if planned.IsArray {
				entries = coerced
			}
			parts := make([]string, len(entries))
			for i, e := range entries {
				parts[i] = searchutils.JSString(e)
			}
			asText := strings.Join(parts, " ")
			if IsSearchable(planned.Field) {
				textParts = append(textParts, asText)
				p.WeightedParts = append(p.WeightedParts, WeightedPart{Text: asText, Weight: WeightLabel(planned.Field.Searchable)})
			}
		}
	}
	primaryKey, _ := document.Get(plan.PrimaryKey)
	if primaryKey == nil || primaryKey == "" {
		return nil, fail(fmt.Sprintf(`A document written to the postgres search provider is missing its primary key "%s"`, plan.PrimaryKey))
	}
	p.ID = searchutils.JSString(primaryKey)
	p.SearchText = strings.Join(textParts, " ")
	return p, nil
}
