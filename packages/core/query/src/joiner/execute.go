package joiner

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// Response is the result of a RemoteJoiner query: rows, plus pagination
// metadata when the root query was paginated.
type Response struct {
	Data     []any
	Metadata *dal.Record
	Paged    bool
}

// filterFields mirrors filterFields: only the requested fields (and the
// nested expands) are kept, in the order of the fields.
func filterFields(data any, fields []string, expands *NestedExpands) any {
	rec, ok := data.(*dal.Record)
	if fields == nil || !ok || rec == nil {
		return data
	}
	var filtered *dal.Record
	if slices.Contains(fields, "*") {
		filtered = rec
	} else {
		filtered = dal.NewRecord()
		for _, f := range fields {
			if v, ok := rec.Get(f); ok {
				filtered.Set(f, v)
			}
		}
	}
	if expands != nil {
		for _, key := range expands.Keys {
			exp := expands.Values[key]
			if exp == nil {
				continue
			}
			v, present := rec.Get(key)
			if list, isList := v.([]any); isList {
				out := make([]any, len(list))
				for i, item := range list {
					out[i] = filterFields(item, exp.Fields, exp.Expands)
				}
				filtered.Set(key, out)
				continue
			}
			if v == nil {
				// filterFields(null) returns null, which is kept; an absent
				// key (undefined) is not
				if present {
					filtered.Set(key, nil)
				}
				continue
			}
			if f := filterFields(v, exp.Fields, exp.Expands); f != nil {
				filtered.Set(key, f)
			}
		}
	}
	if filtered.Len() == 0 {
		return nil
	}
	return filtered
}

func dedupeIDs(ids []any) []any {
	out := make([]any, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		if id == nil {
			continue
		}
		if list, ok := id.([]any); ok {
			for _, x := range list {
				k := fmt.Sprint(x)
				if x != nil && !seen[k] {
					seen[k] = true
					out = append(out, x)
				}
			}
			continue
		}
		k := fmt.Sprint(id)
		if !seen[k] {
			seen[k] = true
			out = append(out, id)
		}
	}
	return out
}

// fetchData mirrors fetchData.
func fetchData(ctx context.Context, fetcher DataFetcher, expand *ExpandProperty, pkField string, ids []any, hasIDs bool, relationship *Relationship, opts Options) (*FetchResult, error) {
	if hasIDs {
		ids = dedupeIDs(ids)
	}
	pk := pkField
	if relationship != nil {
		if relationship.Inverse {
			parts := strings.Split(relationship.ForeignKey, ".")
			pk = parts[len(parts)-1]
		} else {
			pk = relationship.PrimaryKey
		}
	}
	res, err := fetcher.Fetch(ctx, expand, pk, ids, hasIDs, relationship)
	if err != nil {
		return nil, err
	}

	if hasIDs && ((opts.ThrowIfKeyNotFound && relationship == nil) || (opts.ThrowIfRelationNotFound && relationship != nil)) {
		found := map[string]bool{}
		for _, row := range res.Rows {
			if r, ok := row.(*dal.Record); ok {
				v, _ := r.Get(pk)
				found[fmt.Sprint(v)] = true
			}
		}
		var missing []string
		for _, id := range ids {
			if !found[fmt.Sprint(id)] {
				missing = append(missing, fmt.Sprint(id))
			}
		}
		if len(missing) > 0 {
			name := expand.ServiceConfig.Entity
			if name == "" {
				if s, ok := expand.ServiceConfig.Args["methodSuffix"].(string); ok {
					name = s
				} else {
					name = expand.ServiceConfig.ServiceName
				}
			}
			return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND,
				fmt.Sprintf("%s %s not found: %s", name, pk, strings.Join(missing, ", ")))
		}
	}

	rows := make([]any, len(res.Rows))
	for i, row := range res.Rows {
		rows[i] = filterFields(row, expand.Fields, expand.Expands)
	}
	res.Rows = rows
	return res, nil
}

// ExecutePlan mirrors executePlan (without the residual cross module
// filters, applied separately).
func ExecutePlan(ctx context.Context, plan *QueryPlan, fetcher DataFetcher, catalog *GraphCatalog, opts Options) (*Response, error) {
	var ids []any
	hasIDs := false
	if plan.PrimaryKeyArg != nil && plan.PrimaryKeyArg.Value != nil {
		hasIDs = true
		if list, ok := plan.PrimaryKeyArg.Value.([]any); ok {
			ids = list
		} else {
			ids = []any{plan.PrimaryKeyArg.Value}
		}
	}
	res, err := fetchData(ctx, fetcher, plan.Root, plan.PkName, ids, hasIDs, nil, opts)
	if err != nil {
		return nil, err
	}
	if err := handleExpands(ctx, res.Rows, plan, fetcher, catalog, opts); err != nil {
		return nil, err
	}
	return &Response{Data: res.Rows, Metadata: res.Metadata, Paged: res.Paged}, nil
}

type pathCtx struct {
	path         string
	expand       *ExpandProperty
	relationship *Relationship
	nestedItems  []any
	field        string
	fieldsArray  []string
	ids          []any
}

func getItemsForPath(rootItems []any, fullPath string) []any {
	nested := rootItems
	levels := strings.Split(fullPath, ".")
	for i := 1; i < len(levels)-1; i++ {
		nested = getNestedItems(nested, levels[i])
	}
	return nested
}

func computeIdsForRelationship(items []any, rel *Relationship) (string, []string, []any) {
	var field string
	var fieldsArray []string
	if rel.Inverse {
		field = rel.PrimaryKey
		fieldsArray = rel.PrimaryKeyArr
	} else {
		parts := strings.Split(rel.ForeignKey, ".")
		field = parts[len(parts)-1]
		fieldsArray = rel.ForeignKeyArr
	}
	var ids []any
	seen := map[string]bool{}
	add := func(v any) {
		k := fmt.Sprint(v)
		if !seen[k] {
			seen[k] = true
			ids = append(ids, v)
		}
	}
	for _, item := range items {
		rec, ok := item.(*dal.Record)
		if !ok || rec == nil {
			continue
		}
		if len(fieldsArray) == 1 {
			v, _ := rec.Get(fieldsArray[0])
			if list, isList := v.([]any); isList {
				for _, x := range list {
					add(x)
				}
			} else {
				add(v)
			}
			continue
		}
		values := make([]any, len(fieldsArray))
		for i, f := range fieldsArray {
			values[i], _ = rec.Get(f)
		}
		add(values)
	}
	return field, fieldsArray, ids
}

func handleExpands(ctx context.Context, items []any, plan *QueryPlan, fetcher DataFetcher, catalog *GraphCatalog, opts Options) error {
	root := plan.Expands.Get(BasePath)
	stages := root.ExecutionStages
	if len(stages) > 0 {
		stages = stages[1:]
	}
	for _, stage := range stages {
		type group struct {
			ctxs         []*pathCtx
			relationship *Relationship
			result       *FetchResult
		}
		var groups []*group
		for _, st := range stage {
			var ctxs []*pathCtx
			for _, path := range st.Paths {
				expand := plan.Expands.Get(path)
				nested := getItemsForPath(items, path)
				if len(nested) == 0 || expand == nil {
					continue
				}
				rel := catalog.GetEntityRelationship(expand.ParentConfig, expand.Property, expand.Entity)
				if rel == nil {
					continue
				}
				field, fieldsArray, ids := computeIdsForRelationship(nested, rel)
				ctxs = append(ctxs, &pathCtx{path: path, expand: expand, relationship: rel, nestedItems: nested, field: field, fieldsArray: fieldsArray, ids: ids})
			}
			if len(ctxs) == 0 {
				continue
			}
			var order []string
			byPk := map[string][]*pathCtx{}
			for _, c := range ctxs {
				if _, ok := byPk[c.field]; !ok {
					order = append(order, c.field)
				}
				byPk[c.field] = append(byPk[c.field], c)
			}
			for _, pk := range order {
				cs := byPk[pk]
				var unionIDs []any
				var unionFields []string
				var unionArgs []Arg
				for _, c := range cs {
					unionIDs = append(unionIDs, c.ids...)
					unionFields = append(unionFields, c.expand.Fields...)
					unionArgs = append(unionArgs, c.expand.Args...)
				}
				agg := *cs[0].expand
				agg.Fields = dedupeStrings(unionFields)
				if len(unionArgs) > 0 {
					agg.Args = unionArgs
				}
				res, err := fetchData(ctx, fetcher, &agg, pk, unionIDs, true, cs[0].relationship, opts)
				if err != nil {
					return err
				}
				groups = append(groups, &group{ctxs: cs, relationship: cs[0].relationship, result: res})
			}
		}

		for _, g := range groups {
			joinFields := g.relationship.PrimaryKeyArr
			if g.relationship.Inverse {
				joinFields = g.relationship.ForeignKeyArr
			}
			related := createRelatedDataMap(g.result.Rows, joinFields)
			for _, c := range g.ctxs {
				assignRelatedToItems(c.nestedItems, c.relationship, related, c.field, c.fieldsArray)
			}
		}
	}
	if len(plan.Shortcuts) > 0 {
		applyShortcuts(items, plan.Expands, plan.Shortcuts)
	}
	return nil
}

type relatedValue struct {
	single any
	list   []any
	isList bool
}

func createRelatedDataMap(rows []any, joinFields []string) map[string]*relatedValue {
	out := map[string]*relatedValue{}
	for _, row := range rows {
		rec, ok := row.(*dal.Record)
		if !ok || rec == nil {
			continue
		}
		parts := make([]string, len(joinFields))
		for i, f := range joinFields {
			v, _ := rec.Get(f)
			parts[i] = fmt.Sprint(v)
		}
		key := strings.Join(parts, ",")
		existing, ok := out[key]
		switch {
		case !ok:
			out[key] = &relatedValue{single: row}
		case existing.isList:
			existing.list = append(existing.list, row)
		default:
			out[key] = &relatedValue{list: []any{existing.single, row}, isList: true}
		}
	}
	return out
}

func (v *relatedValue) value(asList bool) any {
	if v == nil {
		if asList {
			return []any{}
		}
		return nil
	}
	if v.isList {
		return v.list
	}
	if asList {
		return []any{v.single}
	}
	return v.single
}

func assignRelatedToItems(items []any, rel *Relationship, related map[string]*relatedValue, field string, fieldsArray []string) {
	for _, item := range items {
		rec, ok := item.(*dal.Record)
		if !ok || rec == nil {
			continue
		}
		parts := make([]string, len(fieldsArray))
		for i, f := range fieldsArray {
			v, _ := rec.Get(f)
			parts[i] = fmt.Sprint(v)
		}
		itemKey := strings.Join(parts, ",")

		if existing, ok := rec.Get(rel.Alias); ok && existing != nil {
			if fieldValues, isList := mustGet(rec, field).([]any); isList {
				if list, ok := existing.([]any); ok {
					for i, it := range list {
						if itRec, ok := it.(*dal.Record); ok {
							k, _ := itRec.Get(rel.PrimaryKey)
							mergeInto(itRec, related[fmt.Sprint(k)].value(false))
							list[i] = itRec
						}
					}
				}
				_ = fieldValues
				continue
			}
			if existingRec, ok := existing.(*dal.Record); ok {
				mergeInto(existingRec, related[itemKey].value(false))
			}
			continue
		}

		if ids, isList := mustGet(rec, field).([]any); isList {
			out := make([]any, len(ids))
			for i, id := range ids {
				out[i] = related[fmt.Sprint(id)].value(rel.IsList)
			}
			rec.Set(rel.Alias, out)
			continue
		}
		if related[itemKey] == nil && !rel.IsList {
			// relatedDataMap[itemKey] is undefined: JSON omits the key.
			rec.Delete(rel.Alias)
			continue
		}
		rec.Set(rel.Alias, related[itemKey].value(rel.IsList))
	}
}

func mustGet(r *dal.Record, key string) any {
	v, _ := r.Get(key)
	return v
}

func mergeInto(dst *dal.Record, src any) {
	srcRec, ok := src.(*dal.Record)
	if !ok || srcRec == nil {
		return
	}
	for _, k := range srcRec.Keys() {
		v, _ := srcRec.Get(k)
		dst.Set(k, v)
	}
}

// applyShortcuts mirrors applyShortcuts: field aliases point to the data
// fetched through their path, and the intermediate alias mappings are
// removed.
func applyShortcuts(items []any, parsed *ExpandMap, shortcuts []ShortcutSpec) {
	getChildren := func(item any, prop string) any {
		if list, ok := item.([]any); ok {
			out := []any{}
			for _, it := range list {
				if rec, ok := it.(*dal.Record); ok {
					v, _ := rec.Get(prop)
					if sub, isList := v.([]any); isList {
						out = append(out, sub...)
					} else {
						out = append(out, v)
					}
				} else {
					out = append(out, nil)
				}
			}
			return out
		}
		if rec, ok := item.(*dal.Record); ok {
			v, _ := rec.Get(prop)
			return v
		}
		return nil
	}
	type removal struct {
		target any
		prop   string
	}
	var cleanup []removal

	for _, alias := range shortcuts {
		propPath := alias.Path[len(alias.Location):]
		location := items
		for _, prop := range alias.Location {
			location = getNestedItems(location, prop)
		}
		for _, loc := range location {
			locRec, ok := loc.(*dal.Record)
			if !ok || locRec == nil {
				continue
			}
			var current any = locRec
			var parentRemove *removal
			curPath := append([]string{BasePath}, alias.Location...)
			for _, prop := range propPath {
				if current == nil {
					break
				}
				curPath = append(curPath, prop)
				if cfg := parsed.Get(strings.Join(curPath, ".")); cfg != nil && cfg.IsAliasMapping && parentRemove == nil {
					parentRemove = &removal{target: current, prop: prop}
				}
				current = getChildren(current, prop)
			}
			if list, isList := current.([]any); isList {
				if len(list) < 2 && !alias.IsList {
					if len(list) == 1 {
						locRec.Set(alias.Property, list[0])
					} else {
						locRec.Set(alias.Property, nil)
					}
				} else {
					locRec.Set(alias.Property, list)
				}
			} else if alias.IsList {
				if current != nil {
					locRec.Set(alias.Property, []any{current})
				} else {
					locRec.Set(alias.Property, []any{})
				}
			} else {
				locRec.Set(alias.Property, current)
			}
			if parentRemove != nil {
				cleanup = append(cleanup, *parentRemove)
			}
		}
	}
	for _, c := range cleanup {
		switch t := c.target.(type) {
		case []any:
			for _, it := range t {
				if rec, ok := it.(*dal.Record); ok {
					rec.Delete(c.prop)
				}
			}
		case *dal.Record:
			t.Delete(c.prop)
		}
	}
}
