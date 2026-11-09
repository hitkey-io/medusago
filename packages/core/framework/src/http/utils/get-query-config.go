// Package utils mirrors @medusajs/framework/src/http/utils.
package utils

import (
	"fmt"
	"strings"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// QueryConfig mirrors QueryConfig (the second argument of
// validateAndTransformQuery).
type QueryConfig struct {
	Defaults     []string
	DefaultLimit float64
	IsList       bool
	Entity       string
	Allowed      []string
	Disallowed   []string
	// StoreRelationsLimit overrides req.storeRelationsLimit (0 = unset).
	StoreRelationsLimit int
}

// orderedSet is a JS Set of strings.
type orderedSet struct {
	items []string
	index map[string]bool
}

func newSet(items []string) *orderedSet {
	s := &orderedSet{index: map[string]bool{}}
	for _, i := range items {
		s.add(i)
	}
	return s
}

func (s *orderedSet) add(v string) {
	if !s.index[v] {
		s.index[v] = true
		s.items = append(s.items, v)
	}
}

func (s *orderedSet) delete(v string) {
	if !s.index[v] {
		return
	}
	delete(s.index, v)
	for i, x := range s.items {
		if x == v {
			s.items = append(s.items[:i], s.items[i+1:]...)
			return
		}
	}
}

func (s *orderedSet) has(v string) bool { return s.index[v] }

func (s *orderedSet) list() []string { return append([]string{}, s.items...) }

// ParsedFields mirrors ParsedFields.
type ParsedFields struct {
	Fields     *orderedSet
	StarFields *orderedSet
}

func stripModifier(f string) string {
	return strings.TrimSpace(strings.TrimLeft(f, "+ \t-"))
}

// ParseFields mirrors FieldParser.parse (fields nil = undefined).
func ParseFields(fields *string, defaults []string) ParsedFields {
	star := newSet(nil)
	all := newSet(defaults)
	if fields != nil {
		var custom []string
		for _, f := range strings.Split(*fields, ",") {
			if f != "" {
				custom = append(custom, f)
			}
		}
		replace := len(custom) == 0
		for _, f := range custom {
			if !(strings.HasPrefix(f, "-") || strings.HasPrefix(f, "+") || strings.HasPrefix(f, " ") || strings.HasPrefix(f, "*") || strings.HasSuffix(f, ".*")) {
				replace = true
				break
			}
		}
		if replace {
			all = newSet(nil)
			for _, f := range custom {
				all.add(stripModifier(f))
			}
		} else {
			for _, f := range custom {
				switch {
				case strings.HasPrefix(f, "+") || strings.HasPrefix(f, " "):
					all.add(stripModifier(f))
				case strings.HasPrefix(f, "-"):
					name := stripModifier(f)
					for _, req := range all.list() {
						n := strings.TrimPrefix(req, "*")
						if n == name || strings.HasPrefix(n, name+".") {
							all.delete(req)
						}
					}
				default:
					all.add(f)
				}
			}
		}
		all.add("id")
	}
	for _, f := range all.list() {
		if strings.HasPrefix(f, "*") || strings.HasSuffix(f, ".*") {
			if strings.HasPrefix(f, "*") {
				star.add(f[1:])
			} else {
				star.add(strings.TrimSuffix(f, ".*"))
			}
			all.delete(f)
		}
	}
	return ParsedFields{Fields: all, StarFields: star}
}

func (p ParsedFields) toCheck() []string {
	return append(p.Fields.list(), p.StarFields.list()...)
}

func allowedFilter(allowed []string, p ParsedFields) []string {
	set := newSet(allowed)
	var out []string
	for _, f := range p.toCheck() {
		if !set.has(f) {
			out = append(out, f)
		}
	}
	return out
}

func restrictedFilter(restricted []string, p ParsedFields) []string {
	var out []string
	for _, f := range p.toCheck() {
		segs := strings.Split(f, ".")
	outer:
		for _, r := range restricted {
			for _, s := range segs {
				if s == r {
					out = append(out, f)
					break outer
				}
			}
		}
	}
	return out
}

func disallowedFilter(disallowed []string, p ParsedFields) []string {
	set := map[string]bool{}
	for _, d := range disallowed {
		set[strings.ToLower(strings.TrimSpace(d))] = true
	}
	var out []string
	for _, f := range p.toCheck() {
		for _, seg := range strings.Split(strings.ToLower(strings.TrimSpace(f)), ".") {
			if set[strings.ToLower(strings.TrimSpace(seg))] {
				out = append(out, f)
				break
			}
		}
	}
	return out
}

// StringToSelectRelationObject mirrors stringToSelectRelationObject.
func StringToSelectRelationObject(fields []string) (selectFields, relations []string) {
	sel := newSet(nil)
	rel := newSet(nil)
	for _, f := range fields {
		sel.add(f)
		if !strings.Contains(f, ".") {
			continue
		}
		segs := strings.Split(f, ".")
		rel.add(strings.Join(segs[:len(segs)-1], "."))
	}
	return sel.list(), rel.list()
}

// BuildOrder mirrors buildOrder for a single field.
func BuildOrder(field, direction string) *dal.Record {
	out := dal.NewRecord()
	segs := strings.Split(field, ".")
	for _, s := range segs {
		if s == "__proto__" || s == "constructor" || s == "prototype" {
			return out
		}
	}
	parent := out
	for len(segs) > 1 {
		k := segs[0]
		segs = segs[1:]
		next, ok := parent.Get(k)
		r, isRec := next.(*dal.Record)
		if !ok || !isRec {
			r = dal.NewRecord()
			parent.Set(k, r)
		}
		parent = r
	}
	parent.Set(segs[0], direction)
	return out
}

// PreparedQuery is the result of PrepareListQuery / PrepareRetrieveQuery.
type PreparedQuery struct {
	ListConfig        *medusahttp.ListConfig
	RetrieveConfig    *medusahttp.ListConfig
	RemoteQueryConfig medusahttp.QueryConfig
}

func numberOf(v *dal.Record, key string) (float64, bool) {
	x, ok := v.Get(key)
	if !ok {
		return 0, false
	}
	f, ok := x.(float64)
	return f, ok
}

// PrepareListQuery mirrors prepareListQuery.
func PrepareListQuery(validated *dal.Record, qc QueryConfig, restricted []string) (*PreparedQuery, error) {
	defaultLimit := qc.DefaultLimit
	if defaultLimit == 0 {
		defaultLimit = 50
	}
	limit, ok := numberOf(validated, "limit")
	if !ok {
		limit = defaultLimit
	}
	offset, ok := numberOf(validated, "offset")
	if !ok {
		offset = 0
	}
	var fieldsPtr *string
	if f, ok := validated.Get("fields"); ok {
		if s, ok := f.(string); ok {
			fieldsPtr = &s
		}
	}
	var withDeleted *bool
	if w, ok := validated.Get("with_deleted"); ok {
		if b, ok := w.(bool); ok {
			withDeleted = &b
		}
	}
	parsed := ParseFields(fieldsPtr, qc.Defaults)

	var notAllowed []string
	if len(qc.Allowed) > 0 {
		notAllowed = allowedFilter(qc.Allowed, parsed)
	} else if len(restricted) > 0 {
		notAllowed = restrictedFilter(restricted, parsed)
	}
	for _, f := range notAllowed {
		parsed.Fields.delete(f)
		parsed.StarFields.delete(f)
	}
	if len(qc.Disallowed) > 0 {
		for _, f := range disallowedFilter(qc.Disallowed, parsed) {
			parsed.Fields.delete(f)
			parsed.StarFields.delete(f)
		}
	}

	sel, relations := StringToSelectRelationObject(parsed.Fields.list())
	allRelations := newSet(relations)
	for _, s := range parsed.StarFields.list() {
		allRelations.add(s)
	}

	var finalOrder *dal.Record
	if o, ok := validated.Get("order"); ok {
		if order, ok := o.(string); ok {
			field, dir := order, "ASC"
			if strings.HasPrefix(order, "-") {
				field, dir = order[1:], "DESC"
			}
			if len(qc.Allowed) > 0 && !newSet(qc.Allowed).has(field) {
				return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Order field %s is not valid", field))
			}
			if built := BuildOrder(field, dir); built.Len() > 0 {
				finalOrder = built
			}
		}
	}

	lc := &medusahttp.ListConfig{Relations: allRelations.list(), Skip: &offset, Take: &limit, Order: finalOrder, WithDeleted: withDeleted}
	if len(sel) > 0 {
		lc.Select = sel
	}
	fields := parsed.Fields.list()
	for _, s := range parsed.StarFields.list() {
		fields = append(fields, s+".*")
	}
	pagination := dal.NewRecord()
	if qc.IsList {
		pagination.Set("skip", offset)
		pagination.Set("take", limit)
		if finalOrder != nil {
			pagination.Set("order", finalOrder)
		}
	}
	return &PreparedQuery{
		ListConfig:        lc,
		RemoteQueryConfig: medusahttp.QueryConfig{Entity: qc.Entity, Fields: fields, Pagination: pagination, WithDeleted: withDeleted},
	}, nil
}

// PrepareRetrieveQuery mirrors prepareRetrieveQuery.
func PrepareRetrieveQuery(validated *dal.Record, qc QueryConfig, restricted []string) (*PreparedQuery, error) {
	q, err := PrepareListQuery(validated, qc, restricted)
	if err != nil {
		return nil, err
	}
	return &PreparedQuery{
		RetrieveConfig:    &medusahttp.ListConfig{Select: q.ListConfig.Select, Relations: q.ListConfig.Relations},
		RemoteQueryConfig: medusahttp.QueryConfig{Fields: q.RemoteQueryConfig.Fields, Pagination: dal.NewRecord(), WithDeleted: q.RemoteQueryConfig.WithDeleted},
	}, nil
}
