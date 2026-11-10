package joiner

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// This file mirrors cross-module-joins/{residual-filters,in-memory-filter}.ts:
// filters on other modules that cannot be pushed down are stripped from the
// query, their data is loaded through injected expands and the root items
// are filtered in memory after the fetch (injected properties are hidden).

// ResidualHiddenProperty mirrors ResidualHiddenProperty.
type ResidualHiddenProperty struct {
	Location []string
	Property string
}

type consumeContext struct {
	query         *RemoteJoinerQuery
	serviceConfig *ServiceConfig
	catalog       *GraphCatalog
	visiblePaths  map[string]bool
	hidden        []ResidualHiddenProperty
	hiddenKeys    map[string]bool
}

// consumeResiduals mirrors consumeResiduals.
func consumeResiduals(query *RemoteJoinerQuery, serviceConfig *ServiceConfig, catalog *GraphCatalog, residuals []ResidualCrossModuleFilter, orderBy []ResidualOrderBy) []ResidualHiddenProperty {
	ctx := &consumeContext{query: query, serviceConfig: serviceConfig, catalog: catalog, visiblePaths: collectVisiblePaths(query), hiddenKeys: map[string]bool{}}
	for _, r := range residuals {
		stripResidualFilters(query, r)
		ensureResidualData(ctx, strings.Split(r.Path, "."), r.Filters)
	}
	ctx.consumeResidualOrderBy(orderBy)
	return ctx.hidden
}

func collectVisiblePaths(query *RemoteJoinerQuery) map[string]bool {
	visible := map[string]bool{}
	for _, e := range query.Expands {
		hasContent := len(e.Fields) > 0
		for _, a := range e.Args {
			if a.Name != "filters" {
				hasContent = true
			}
		}
		if !hasContent {
			continue
		}
		segments := strings.Split(e.Property, ".")
		for i := 1; i <= len(segments); i++ {
			visible[strings.Join(segments[:i], ".")] = true
		}
	}
	return visible
}

func stripResidualFilters(query *RemoteJoinerQuery, r ResidualCrossModuleFilter) {
	for i := range query.Expands {
		e := &query.Expands[i]
		if e.Property != r.Path {
			continue
		}
		if a := getFiltersArg(e.Args); a != nil && a.Value == any(r.Filters) {
			var remaining []Arg
			for _, x := range e.Args {
				if x.Name != "filters" {
					remaining = append(remaining, x)
				}
			}
			e.Args = remaining
			return
		}
	}
	if a := getFiltersArg(query.Args); a != nil {
		if root, ok := a.Value.(*dal.Record); ok {
			if v, _ := root.Get(r.Path); v == any(r.Filters) {
				root.Delete(r.Path)
			}
		}
	}
}

func isOperatorMap(v any) bool {
	r, ok := v.(*dal.Record)
	if !ok || r.Len() == 0 {
		return false
	}
	for _, k := range r.Keys() {
		if !strings.HasPrefix(k, "$") {
			return false
		}
	}
	return true
}

func ensureResidualData(ctx *consumeContext, segments []string, filters *dal.Record) {
	var selectFields []string
	var childValueNodes []string
	childSeen := map[string]bool{}
	type nested struct {
		key   string
		value *dal.Record
	}
	var nestedRelations []nested

	entity := ""
	if hops := resolvePathHops(ctx.serviceConfig, segments, ctx.catalog); len(hops) > 0 {
		entity = hops[len(hops)-1].entity
	}
	var crossjoinable map[string]bool
	if md := ctx.catalog.GetAliasMetadata(entity); md != nil && md.Crossjoinable != nil {
		crossjoinable = map[string]bool{}
		for _, f := range md.Crossjoinable {
			crossjoinable[f] = true
		}
	}

	var collect func(level *dal.Record)
	collect = func(level *dal.Record) {
		for _, key := range level.Keys() {
			value, _ := level.Get(key)
			switch {
			case key == "$and" || key == "$or":
				if list, ok := value.([]any); ok {
					for _, c := range list {
						if r, ok := c.(*dal.Record); ok {
							collect(r)
						}
					}
				}
				continue
			case key == "$not":
				if r, ok := value.(*dal.Record); ok {
					collect(r)
				}
				continue
			case strings.HasPrefix(key, "$"):
				continue
			}
			if r, ok := value.(*dal.Record); ok && !isOperatorMap(value) {
				if resolvePathHops(ctx.serviceConfig, append(append([]string{}, segments...), key), ctx.catalog) != nil {
					nestedRelations = append(nestedRelations, nested{key, r})
					continue
				}
				if crossjoinable != nil && !crossjoinable[key] {
					if !childSeen[key] {
						childSeen[key] = true
						childValueNodes = append(childValueNodes, key)
					}
					continue
				}
			}
			selectFields = append(selectFields, key)
		}
	}
	collect(filters)

	ensureExpandPath(ctx, segments, selectFields)
	for _, key := range childValueNodes {
		ensureExpandPath(ctx, append(append([]string{}, segments...), key), []string{"*"})
	}
	for _, n := range nestedRelations {
		ensureResidualData(ctx, append(append([]string{}, segments...), n.key), n.value)
	}
}

func ensureExpandPath(ctx *consumeContext, segments []string, fields []string) {
	for i := 1; i < len(segments); i++ {
		ensureExpandEntry(ctx, segments[:i], nil)
	}
	ensureExpandEntry(ctx, segments, fields)
}

func ensureExpandEntry(ctx *consumeContext, segments []string, fields []string) {
	subtreeHidden := false
	for i := 1; i <= len(segments); i++ {
		if !ctx.visiblePaths[strings.Join(segments[:i], ".")] {
			addHidden(ctx, segments[:i-1], segments[i-1])
			subtreeHidden = true
			break
		}
	}
	property := strings.Join(segments, ".")
	idx := -1
	for i, e := range ctx.query.Expands {
		if e.Property == property {
			idx = i
			break
		}
	}
	if idx < 0 {
		ctx.query.Expands = append(ctx.query.Expands, ExpandSpec{Property: property, Fields: []string{}})
		idx = len(ctx.query.Expands) - 1
	}
	e := &ctx.query.Expands[idx]
	if e.Fields == nil {
		e.Fields = []string{}
	}
	for _, f := range e.Fields {
		if f == "*" {
			return
		}
	}
	for _, f := range fields {
		if f == "*" {
			e.Fields = []string{"*"}
			return
		}
		exists := false
		for _, x := range e.Fields {
			if x == f {
				exists = true
				break
			}
		}
		if exists {
			continue
		}
		e.Fields = append(e.Fields, f)
		if !subtreeHidden {
			addHidden(ctx, segments, f)
		}
	}
}

func addHidden(ctx *consumeContext, location []string, property string) {
	key := strings.Join(append(append([]string{}, location...), property), ".")
	if ctx.hiddenKeys[key] {
		return
	}
	ctx.hiddenKeys[key] = true
	ctx.hidden = append(ctx.hidden, ResidualHiddenProperty{Location: append([]string{}, location...), Property: property})
}

// applyResidualFilters mirrors applyResidualFilters.
func applyResidualFilters(items []any, residuals []ResidualCrossModuleFilter) ([]any, error) {
	out := []any{}
	for _, item := range items {
		if item == nil {
			continue
		}
		keep := true
		for _, r := range residuals {
			ok, err := matchesRelatedPath(item, strings.Split(r.Path, "."), r.Filters)
			if err != nil {
				return nil, err
			}
			if !ok {
				keep = false
				break
			}
		}
		if keep {
			out = append(out, item)
		}
	}
	return out, nil
}

func matchesRelatedPath(value any, segments []string, filters *dal.Record) (bool, error) {
	if value == nil {
		return false, nil
	}
	if list, ok := value.([]any); ok {
		for _, e := range list {
			ok, err := matchesRelatedPath(e, segments, filters)
			if err != nil || ok {
				return ok, err
			}
		}
		return false, nil
	}
	if len(segments) > 0 {
		r, ok := value.(*dal.Record)
		if !ok {
			return false, nil
		}
		v, _ := r.Get(segments[0])
		return matchesRelatedPath(v, segments[1:], filters)
	}
	return matchesFilters(value, filters)
}

// hideResidualProperties mirrors hideResidualProperties (the injected
// properties become non enumerable: they are left out of the output).
func hideResidualProperties(items []any, hidden []ResidualHiddenProperty) {
	for _, h := range hidden {
		targets := items
		for _, s := range h.Location {
			targets = getNestedItems(targets, s)
		}
		for _, t := range targets {
			if r, ok := t.(*dal.Record); ok && r.Has(h.Property) {
				r.Delete(h.Property)
			}
		}
	}
}

func toArray(v any) []any {
	if list, ok := v.([]any); ok {
		return list
	}
	return []any{v}
}

// matchesFilters mirrors matchesFilters.
func matchesFilters(record any, filters *dal.Record) (bool, error) {
	r, ok := record.(*dal.Record)
	if !ok {
		return false, nil
	}
	for _, key := range filters.Keys() {
		cond, _ := filters.Get(key)
		var ok bool
		var err error
		switch key {
		case "$and":
			ok = true
			for _, e := range toArray(cond) {
				er, isRec := e.(*dal.Record)
				if !isRec {
					ok = false
					break
				}
				if m, err2 := matchesFilters(r, er); err2 != nil {
					return false, err2
				} else if !m {
					ok = false
					break
				}
			}
		case "$or":
			for _, e := range toArray(cond) {
				if er, isRec := e.(*dal.Record); isRec {
					m, err2 := matchesFilters(r, er)
					if err2 != nil {
						return false, err2
					}
					if m {
						ok = true
						break
					}
				}
			}
		case "$not":
			if cr, isRec := cond.(*dal.Record); isRec {
				m, err2 := matchesFilters(r, cr)
				if err2 != nil {
					return false, err2
				}
				ok = !m
			}
		default:
			v, _ := r.Get(key)
			ok, err = matchesCondition(v, cond)
		}
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

func matchesCondition(value, cond any) (bool, error) {
	if isOperatorMap(cond) {
		cr := cond.(*dal.Record)
		for _, op := range cr.Keys() {
			arg, _ := cr.Get(op)
			ok, err := applyOperator(value, op, arg)
			if err != nil || !ok {
				return false, err
			}
		}
		return true, nil
	}
	if cr, ok := cond.(*dal.Record); ok {
		if list, isList := value.([]any); isList {
			for _, e := range list {
				if _, isRec := e.(*dal.Record); isRec {
					m, err := matchesFilters(e, cr)
					if err != nil || m {
						return m, err
					}
				}
			}
			return false, nil
		}
		return matchesFilters(value, cr)
	}
	if list, ok := cond.([]any); ok {
		for _, e := range list {
			if equalsValue(value, e) {
				return true, nil
			}
		}
		return false, nil
	}
	return equalsValue(value, cond), nil
}

func applyOperator(value any, op string, arg any) (bool, error) {
	switch op {
	case "$and":
		for _, e := range toArray(arg) {
			if ok, err := matchesCondition(value, e); err != nil || !ok {
				return false, err
			}
		}
		return true, nil
	case "$or":
		for _, e := range toArray(arg) {
			if ok, err := matchesCondition(value, e); err != nil || ok {
				return ok, err
			}
		}
		return false, nil
	case "$not":
		ok, err := matchesCondition(value, arg)
		return !ok, err
	case "$eq":
		return equalsValue(value, arg), nil
	case "$ne":
		return value != nil && !equalsValue(value, arg), nil
	case "$in":
		if value == nil {
			return false, nil
		}
		for _, e := range toArray(arg) {
			if equalsValue(value, e) {
				return true, nil
			}
		}
		return false, nil
	case "$nin":
		if value == nil {
			return false, nil
		}
		for _, e := range toArray(arg) {
			if equalsValue(value, e) {
				return false, nil
			}
		}
		return true, nil
	case "$gt", "$gte", "$lt", "$lte":
		c, ok := compareValues(value, arg)
		if !ok {
			return false, nil
		}
		switch op {
		case "$gt":
			return c > 0, nil
		case "$gte":
			return c >= 0, nil
		case "$lt":
			return c < 0, nil
		}
		return c <= 0, nil
	case "$like", "$ilike":
		return likeMatches(value, arg, op == "$ilike"), nil
	case "$re":
		if value == nil {
			return false, nil
		}
		re, err := regexp.Compile(fmt.Sprint(arg))
		if err != nil {
			return false, nil
		}
		return re.MatchString(jsString(value)), nil
	case "$fulltext":
		if value == nil {
			return false, nil
		}
		haystack := strings.ToLower(jsString(value))
		for _, t := range strings.Fields(strings.ToLower(jsString(arg))) {
			if !strings.Contains(haystack, t) {
				return false, nil
			}
		}
		return true, nil
	case "$exists":
		return (value != nil) == (arg != false), nil
	case "$is":
		if arg == nil {
			return value == nil, nil
		}
		return equalsValue(value, arg), nil
	case "$overlap", "$contains", "$contained":
		list, ok := value.([]any)
		if !ok {
			return false, nil
		}
		args := toArray(arg)
		in := func(x any, set []any) bool {
			for _, s := range set {
				if equalsValue(x, s) {
					return true
				}
			}
			return false
		}
		switch op {
		case "$overlap":
			for _, a := range args {
				if in(a, list) {
					return true, nil
				}
			}
			return false, nil
		case "$contains":
			for _, a := range args {
				if !in(a, list) {
					return false, nil
				}
			}
			return true, nil
		}
		for _, v := range list {
			if !in(v, args) {
				return false, nil
			}
		}
		return true, nil
	}
	return false, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf(`Operator "%s" is not supported for in-memory cross-module filtering.`, op))
}

func jsString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	}
	return fmt.Sprint(v)
}

func normalizeScalar(v any) any {
	if r, ok := v.(*dal.Record); ok {
		if n, ok := r.Get("numeric"); ok {
			if f, isNum := n.(float64); isNum {
				return f
			}
		}
	}
	return v
}

func toTime(v any) (int64, bool) {
	switch t := v.(type) {
	case time.Time:
		return t.UnixMilli(), true
	case string:
		if d, ok := common.ParseJSDate(t); ok {
			return d.UnixMilli(), true
		}
	case float64:
		return int64(t), true
	}
	return 0, false
}

func equalsValue(value, expected any) bool {
	_, vt := value.(time.Time)
	_, et := expected.(time.Time)
	if vt || et {
		a, ok1 := toTime(value)
		b, ok2 := toTime(expected)
		return ok1 && ok2 && a == b
	}
	a, b := normalizeScalar(value), normalizeScalar(expected)
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	al, aIsList := a.([]any)
	bl, bIsList := b.([]any)
	if aIsList && bIsList {
		if len(al) != len(bl) {
			return false
		}
		for i := range al {
			if !equalsValue(al[i], bl[i]) {
				return false
			}
		}
		return true
	}
	if aIsList || bIsList {
		return false
	}
	if _, ok := a.(*dal.Record); ok {
		return false
	}
	return a == b
}

func compareValues(value, expected any) (float64, bool) {
	if value == nil || expected == nil {
		return 0, false
	}
	_, vt := value.(time.Time)
	_, et := expected.(time.Time)
	if vt || et {
		a, ok1 := toTime(value)
		b, ok2 := toTime(expected)
		if !ok1 || !ok2 {
			return 0, false
		}
		return float64(a - b), true
	}
	a, b := normalizeScalar(value), normalizeScalar(expected)
	_, aNum := a.(float64)
	_, bNum := b.(float64)
	toNum := func(x any) (float64, bool) {
		switch t := x.(type) {
		case float64:
			return t, true
		case string:
			if strings.TrimSpace(t) == "" {
				return 0, true
			}
			f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
			return f, err == nil
		case bool:
			if t {
				return 1, true
			}
			return 0, true
		}
		return 0, false
	}
	if aNum || bNum {
		x, ok1 := toNum(a)
		y, ok2 := toNum(b)
		if !ok1 || !ok2 {
			return 0, false
		}
		return x - y, true
	}
	as, aStr := a.(string)
	bs, bStr := b.(string)
	if aStr && bStr {
		x, err1 := strconv.ParseFloat(strings.TrimSpace(as), 64)
		y, err2 := strconv.ParseFloat(strings.TrimSpace(bs), 64)
		if strings.TrimSpace(as) != "" && strings.TrimSpace(bs) != "" && err1 == nil && err2 == nil {
			return x - y, true
		}
		switch {
		case as < bs:
			return -1, true
		case as > bs:
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

var likeEscape = regexp.MustCompile(`[.*+?^${}()|\[\]\\]`)

func likeMatches(value, pattern any, caseInsensitive bool) bool {
	p, ok := pattern.(string)
	if value == nil || !ok {
		return false
	}
	src := likeEscape.ReplaceAllString(p, `\$0`)
	src = strings.ReplaceAll(src, "%", `[\s\S]*`)
	src = strings.ReplaceAll(src, "_", ".")
	src = "^" + src + "$"
	if caseInsensitive {
		src = "(?i)" + src
	}
	re, err := regexp.Compile(src)
	if err != nil {
		return false
	}
	return re.MatchString(jsString(value))
}

type rootPagination struct {
	skip, take   *int
	takeIsNull   bool
	withMetadata bool
}

// extractRootPagination mirrors extractRootPagination.
func extractRootPagination(root *ExpandProperty) *rootPagination {
	if len(root.Args) == 0 {
		return nil
	}
	p := &rootPagination{}
	found := false
	var kept []Arg
	for _, a := range root.Args {
		switch a.Name {
		case "skip", "offset":
			found, p.withMetadata = true, true
			if n, ok := toInt(a.Value); ok {
				p.skip = &n
			}
		case "cursor":
			found, p.withMetadata = true, true
		case "take", "limit":
			found = true
			if n, ok := toInt(a.Value); ok {
				p.take = &n
			} else if a.Value == nil {
				p.takeIsNull = true
			}
		default:
			kept = append(kept, a)
		}
	}
	root.Args = kept
	if !found {
		return nil
	}
	return p
}

func toInt(v any) (int, bool) {
	switch t := v.(type) {
	case int:
		return t, true
	case float64:
		return int(t), true
	case *int:
		if t != nil {
			return *t, true
		}
	}
	return 0, false
}

// consumeResidualOrderBy mirrors consumeResidualOrderBy (shares the
// consume context with the residual filters).
func (ctx *consumeContext) consumeResidualOrderBy(orderBy []ResidualOrderBy) {
	for _, o := range orderBy {
		segments := o.Segments
		prefixLength := 0
		for end := len(segments); end >= 1; end-- {
			if resolvePathHops(ctx.serviceConfig, segments[:end], ctx.catalog) != nil {
				prefixLength = end
				break
			}
		}
		fieldPath := segments[prefixLength:]
		if prefixLength == 0 {
			ensureRootField(ctx, fieldPath[0])
			continue
		}
		if len(fieldPath) == 0 {
			ensureExpandPath(ctx, segments, nil)
			continue
		}
		var shape any = true
		for i := len(fieldPath) - 1; i >= 0; i-- {
			r := dal.NewRecord()
			r.Set(fieldPath[i], shape)
			shape = r
		}
		ensureResidualData(ctx, segments[:prefixLength], shape.(*dal.Record))
	}
}

func ensureRootField(ctx *consumeContext, field string) {
	for _, f := range ctx.query.Fields {
		if f == "*" || f == field {
			return
		}
	}
	ctx.query.Fields = append(ctx.query.Fields, field)
	addHidden(ctx, nil, field)
}

// sortByResidualOrder mirrors sortByResidualOrder (stable, like
// Array.prototype.sort).
func sortByResidualOrder(items []any, orderBy []ResidualOrderBy) {
	sort.SliceStable(items, func(i, j int) bool {
		for _, o := range orderBy {
			if c := compareSortValues(getSortValue(items[i], o.Segments), getSortValue(items[j], o.Segments), o.Direction); c != 0 {
				return c < 0
			}
		}
		return false
	})
}

func getSortValue(item any, segments []string) any {
	current := item
	for _, s := range segments {
		if current == nil {
			return nil
		}
		if list, ok := current.([]any); ok {
			if len(list) == 0 {
				return nil
			}
			current = list[0]
		}
		r, ok := current.(*dal.Record)
		if !ok {
			return nil
		}
		current, _ = r.Get(s)
	}
	if list, ok := current.([]any); ok {
		if len(list) == 0 {
			return nil
		}
		return list[0]
	}
	return current
}

func compareSortValues(left, right any, direction string) float64 {
	var result float64
	if left == nil || right == nil {
		if left == nil && right == nil {
			return 0
		}
		if left == nil {
			result = 1
		} else {
			result = -1
		}
	} else if c, ok := compareValues(left, right); ok {
		result = c
	}
	if direction == "DESC" {
		return -result
	}
	return result
}
