package dal

import "fmt"

// Formula mirrors a lazy MikroORM @Formula property: an SQL expression over
// the entity alias, selected only when the property is requested
// explicitly, serialized as a number (serializer: Number).
type Formula struct {
	Property string
	// Before is the metadata property the formula is serialized before
	// (its position among the entity properties).
	Before string
	SQL    func(alias string) string
}

var formulas = map[string][]Formula{}

// RegisterFormula registers a formula property of an entity class (module
// entity hooks call it, like applyFormulas upstream).
func RegisterFormula(className string, f Formula) {
	formulas[className] = append(formulas[className], f)
}

func formulaOf(className, prop string) (Formula, bool) {
	for _, f := range formulas[className] {
		if f.Property == prop {
			return f, true
		}
	}
	return Formula{}, false
}

// formulaColumns returns the select expressions of the requested formulas.
func formulaColumns(className, alias string, tree *fieldTree) []string {
	if tree == nil || tree.all {
		return nil
	}
	var out []string
	for _, f := range formulas[className] {
		if tree.fields[f.Property] {
			out = append(out, fmt.Sprintf("(%s) as %s", f.SQL(quote(alias)), quote(f.Property)))
		}
	}
	return out
}

// formulaValue mirrors serializer: Number over the selected value.
func formulaValue(v any) any {
	switch t := v.(type) {
	case nil:
		return float64(0)
	case string:
		var f float64
		if _, err := fmt.Sscan(t, &f); err == nil {
			return f
		}
		return t
	}
	return v
}
