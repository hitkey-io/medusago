// Package zod mirrors @medusajs/framework/zod: a Go port of the parts of
// zod 4 the API validators use, with the same parsing semantics (key order,
// defaults, unions, strictness) and issue structures, so that
// zod-helpers.go can produce the exact upstream error messages.
//
// Values are JSON-like: *dal.Record objects, []any arrays, string, float64,
// bool, nil (null) and Undefined (a missing value).
package zod

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// undefinedType is the type of Undefined.
type undefinedType struct{}

// Undefined represents a JS undefined value (a missing key).
var Undefined = undefinedType{}

// IsUndefined reports whether v is Undefined.
func IsUndefined(v any) bool { _, ok := v.(undefinedType); return ok }

// Issue mirrors a zod 4 issue (finalized, without input).
type Issue struct {
	Code     string
	Path     []any
	Message  string
	Expected string
	// Values of invalid_value issues.
	Values []any
	// Keys of unrecognized_keys issues.
	Keys []string
	// Errors of invalid_union issues: the issues of each option.
	Errors  [][]*Issue
	Minimum any
	Maximum any
	Origin  string
	Divisor any
	// Continue marks non-aborting issues (refinements).
	Continue bool
}

// ZodError mirrors ZodError.
type ZodError struct{ Issues []*Issue }

func (e *ZodError) Error() string {
	var msgs []string
	for _, i := range e.Issues {
		msgs = append(msgs, i.Message)
	}
	return strings.Join(msgs, "; ")
}

type payload struct {
	value  any
	issues []*Issue
}

func aborted(p *payload, start int) bool {
	for _, i := range p.issues[start:] {
		if !i.Continue {
			return true
		}
	}
	return false
}

// Schema is a zod schema.
type Schema struct {
	kind string
	run  func(p *payload)

	// optin/optout mirror _zod.optin / _zod.optout ("optional" or "").
	optin, optout string

	// object
	shape    *dal.Record // key -> *Schema, ordered
	catchall string      // "strip" | "strict" | "passthrough"

	inner   *Schema
	options []*Schema
	checks  []func(p *payload)

	// refiners re-apply the refinements of an object when it is rebuilt by
	// strict/passthrough/strip (zod 4 keeps an object's checks there).
	refiners []func(*Schema) *Schema
}

// Parse mirrors schema.parse: the output or a *ZodError.
func (s *Schema) Parse(v any) (any, error) {
	p := &payload{value: v}
	s.run(p)
	if len(p.issues) > 0 {
		return nil, &ZodError{Issues: p.issues}
	}
	return p.value, nil
}

func (s *Schema) withChecks(p *payload) {
	for _, c := range s.checks {
		c(p)
	}
}

func clone(s *Schema) *Schema {
	c := *s
	c.checks = append([]func(*payload){}, s.checks...)
	return &c
}

// ParsedType mirrors util.parsedType (the "received" of messages).
func ParsedType(v any) string {
	switch t := v.(type) {
	case undefinedType:
		return "undefined"
	case nil:
		return "null"
	case string:
		return "string"
	case float64:
		if math.IsNaN(t) {
			return "NaN"
		}
		return "number"
	case int:
		return "number"
	case bool:
		return "boolean"
	case []any:
		return "array"
	case *dal.Record:
		return "object"
	}
	return "object"
}

func invalidType(p *payload, expected string) {
	p.issues = append(p.issues, &Issue{
		Code:     "invalid_type",
		Expected: expected,
		Message:  fmt.Sprintf("Invalid input: expected %s, received %s", expected, ParsedType(p.value)),
	})
}

// String mirrors z.string().
func String() *Schema {
	s := &Schema{kind: "string"}
	s.run = func(p *payload) {
		if _, ok := p.value.(string); !ok {
			invalidType(p, "string")
			return
		}
		s.withChecks(p)
	}
	return s
}

// Number mirrors z.number() (finite numbers only, like zod 4).
func Number() *Schema {
	s := &Schema{kind: "number"}
	s.run = func(p *payload) {
		switch t := p.value.(type) {
		case float64:
			if math.IsNaN(t) || math.IsInf(t, 0) {
				invalidType(p, "number")
				return
			}
		case int:
			p.value = float64(t)
		default:
			invalidType(p, "number")
			return
		}
		s.withChecks(p)
	}
	return s
}

// Boolean mirrors z.boolean().
func Boolean() *Schema {
	s := &Schema{kind: "boolean"}
	s.run = func(p *payload) {
		if _, ok := p.value.(bool); !ok {
			invalidType(p, "boolean")
			return
		}
		s.withChecks(p)
	}
	return s
}

// Unknown mirrors z.unknown().
func Unknown() *Schema {
	s := &Schema{kind: "unknown", optin: "optional", optout: "optional"}
	s.run = func(p *payload) { s.withChecks(p) }
	return s
}

// Any mirrors z.any().
func Any() *Schema {
	s := Unknown()
	s.kind = "any"
	return s
}

// Null mirrors z.null().
func Null() *Schema {
	s := &Schema{kind: "null"}
	s.run = func(p *payload) {
		if p.value != nil || IsUndefined(p.value) {
			invalidType(p, "null")
		}
	}
	return s
}

func stringifyPrimitive(v any) string {
	if s, ok := v.(string); ok {
		return `"` + s + `"`
	}
	return JSString(v)
}

// Literal mirrors z.literal(value).
func Literal(values ...any) *Schema {
	s := &Schema{kind: "literal"}
	s.run = func(p *payload) {
		for _, v := range values {
			if v == p.value {
				return
			}
		}
		msg := ""
		if len(values) == 1 {
			msg = "Invalid input: expected " + stringifyPrimitive(values[0])
		} else {
			var parts []string
			for _, v := range values {
				parts = append(parts, stringifyPrimitive(v))
			}
			msg = "Invalid option: expected one of " + strings.Join(parts, "|")
		}
		p.issues = append(p.issues, &Issue{Code: "invalid_value", Values: values, Message: msg})
	}
	return s
}

// Enum mirrors z.enum([...]).
func Enum(values ...string) *Schema {
	vs := make([]any, len(values))
	for i, v := range values {
		vs[i] = v
	}
	s := Literal(vs...)
	s.kind = "enum"
	return s
}

// Array mirrors z.array(element).
func Array(el *Schema) *Schema {
	s := &Schema{kind: "array", inner: el}
	s.run = func(p *payload) {
		arr, ok := p.value.([]any)
		if !ok {
			invalidType(p, "array")
			return
		}
		out := make([]any, len(arr))
		for i, x := range arr {
			ep := &payload{value: x}
			el.run(ep)
			for _, iss := range ep.issues {
				iss.Path = append([]any{float64(i)}, iss.Path...)
			}
			p.issues = append(p.issues, ep.issues...)
			if IsUndefined(ep.value) {
				out[i] = nil
			} else {
				out[i] = ep.value
			}
		}
		p.value = out
		s.withChecks(p)
	}
	return s
}

// Array mirrors schema.array().
func (s *Schema) Array() *Schema { return Array(s) }

// Shape is an ordered object shape.
type Shape []Field

// Field is one key of a shape.
type Field struct {
	Key    string
	Schema *Schema
}

// F builds a Field.
func F(key string, s *Schema) Field { return Field{Key: key, Schema: s} }

// Object mirrors z.object(shape) (unknown keys stripped).
func Object(fields ...Field) *Schema {
	shape := dal.NewRecord()
	for _, f := range fields {
		shape.Set(f.Key, f.Schema)
	}
	return newObject(shape, "strip")
}

func newObject(shape *dal.Record, catchall string) *Schema {
	s := &Schema{kind: "object", shape: shape, catchall: catchall}
	s.run = func(p *payload) {
		in, ok := p.value.(*dal.Record)
		if !ok || in == nil {
			invalidType(p, "object")
			return
		}
		out := dal.NewRecord()
		for _, key := range shape.Keys() {
			v, _ := shape.Get(key)
			el := v.(*Schema)
			raw, present := in.Get(key)
			if !present {
				raw = Undefined
			}
			ep := &payload{value: raw}
			el.run(ep)
			if len(ep.issues) > 0 {
				if !(el.optout == "optional" && !present) {
					for _, iss := range ep.issues {
						iss.Path = append([]any{key}, iss.Path...)
					}
					p.issues = append(p.issues, ep.issues...)
				}
			}
			if !IsUndefined(ep.value) {
				out.Set(key, ep.value)
			}
		}
		var unrecognized []string
		for _, key := range in.Keys() {
			if shape.Has(key) {
				continue
			}
			switch catchall {
			case "passthrough":
				v, _ := in.Get(key)
				out.Set(key, v)
			case "strict":
				unrecognized = append(unrecognized, key)
			}
		}
		if len(unrecognized) > 0 {
			label := "key"
			if len(unrecognized) > 1 {
				label = "keys"
			}
			quoted := make([]string, len(unrecognized))
			for i, k := range unrecognized {
				quoted[i] = `"` + k + `"`
			}
			p.issues = append(p.issues, &Issue{Code: "unrecognized_keys", Keys: unrecognized, Message: "Unrecognized " + label + ": " + strings.Join(quoted, ", ")})
		}
		p.value = out
		s.withChecks(p)
	}
	return s
}

func (s *Schema) objectShape() *dal.Record {
	if s.kind != "object" {
		panic("zod: not an object schema")
	}
	return s.shape
}

// Strict mirrors object.strict().
func (s *Schema) Strict() *Schema { return s.reapply(newObject(s.objectShape(), "strict")) }

func (s *Schema) reapply(o *Schema) *Schema {
	for _, r := range s.refiners {
		o = r(o)
	}
	return o
}

// Passthrough mirrors object.passthrough().
func (s *Schema) Passthrough() *Schema {
	return s.reapply(newObject(s.objectShape(), "passthrough"))
}

// Strip mirrors object.strip().
func (s *Schema) Strip() *Schema { return s.reapply(newObject(s.objectShape(), "strip")) }

// IsObject reports whether the schema is an object (has .strict()).
func (s *Schema) IsObject() bool { return s.kind == "object" }

// Extend mirrors object.extend(fields).
func (s *Schema) Extend(fields ...Field) *Schema {
	shape := dal.NewRecord()
	for _, k := range s.objectShape().Keys() {
		v, _ := s.shape.Get(k)
		shape.Set(k, v)
	}
	for _, f := range fields {
		shape.Set(f.Key, f.Schema)
	}
	return newObject(shape, s.catchall)
}

// Merge mirrors object.merge(other): the shapes are merged, the catchall of
// other wins.
func (s *Schema) Merge(other *Schema) *Schema {
	shape := dal.NewRecord()
	for _, k := range s.objectShape().Keys() {
		v, _ := s.shape.Get(k)
		shape.Set(k, v)
	}
	for _, k := range other.objectShape().Keys() {
		v, _ := other.shape.Get(k)
		shape.Set(k, v)
	}
	return newObject(shape, other.catchall)
}

// Pick mirrors object.pick({...}).
func (s *Schema) Pick(keys ...string) *Schema {
	shape := dal.NewRecord()
	for _, k := range s.objectShape().Keys() {
		for _, x := range keys {
			if x == k {
				v, _ := s.shape.Get(k)
				shape.Set(k, v)
			}
		}
	}
	return newObject(shape, s.catchall)
}

// Omit mirrors object.omit({...}).
func (s *Schema) Omit(keys ...string) *Schema {
	shape := dal.NewRecord()
	for _, k := range s.objectShape().Keys() {
		skip := false
		for _, x := range keys {
			if x == k {
				skip = true
			}
		}
		if !skip {
			v, _ := s.shape.Get(k)
			shape.Set(k, v)
		}
	}
	return newObject(shape, s.catchall)
}

// Partial mirrors object.partial().
func (s *Schema) Partial() *Schema {
	shape := dal.NewRecord()
	for _, k := range s.objectShape().Keys() {
		v, _ := s.shape.Get(k)
		shape.Set(k, v.(*Schema).Optional())
	}
	return newObject(shape, s.catchall)
}

// Record mirrors z.record(key, value) with string keys.
func Record(key, value *Schema) *Schema {
	s := &Schema{kind: "record"}
	s.run = func(p *payload) {
		in, ok := p.value.(*dal.Record)
		if !ok || in == nil {
			invalidType(p, "record")
			return
		}
		out := dal.NewRecord()
		for _, k := range in.Keys() {
			kp := &payload{value: k}
			key.run(kp)
			if len(kp.issues) > 0 {
				p.issues = append(p.issues, &Issue{Code: "invalid_key", Origin: "record", Path: []any{k}, Message: "Invalid key in record"})
				continue
			}
			v, _ := in.Get(k)
			vp := &payload{value: v}
			value.run(vp)
			for _, iss := range vp.issues {
				iss.Path = append([]any{k}, iss.Path...)
			}
			p.issues = append(p.issues, vp.issues...)
			if !IsUndefined(vp.value) {
				out.Set(kp.value.(string), vp.value)
			}
		}
		p.value = out
	}
	return s
}

// Union mirrors z.union([...]).
func Union(options ...*Schema) *Schema {
	s := &Schema{kind: "union", options: options}
	for _, o := range options {
		if o.optin == "optional" {
			s.optin = "optional"
		}
		if o.optout == "optional" {
			s.optout = "optional"
		}
	}
	s.run = func(p *payload) {
		var results []*payload
		for _, o := range options {
			r := &payload{value: p.value}
			o.run(r)
			if len(r.issues) == 0 {
				p.value = r.value
				s.withChecks(p)
				return
			}
			results = append(results, r)
		}
		var nonAborted []*payload
		for _, r := range results {
			if !aborted(r, 0) {
				nonAborted = append(nonAborted, r)
			}
		}
		if len(nonAborted) == 1 {
			p.value = nonAborted[0].value
			p.issues = append(p.issues, nonAborted[0].issues...)
			return
		}
		errs := make([][]*Issue, len(results))
		for i, r := range results {
			errs[i] = r.issues
		}
		p.issues = append(p.issues, &Issue{Code: "invalid_union", Errors: errs, Message: "Invalid input"})
	}
	return s
}

// Optional mirrors schema.optional().
func (s *Schema) Optional() *Schema {
	w := &Schema{kind: "optional", inner: s, optin: "optional", optout: "optional"}
	w.run = func(p *payload) {
		if IsUndefined(p.value) {
			return
		}
		s.run(p)
	}
	return w
}

// Nullable mirrors schema.nullable().
func (s *Schema) Nullable() *Schema {
	w := &Schema{kind: "nullable", inner: s, optin: s.optin, optout: s.optout}
	w.run = func(p *payload) {
		if p.value == nil {
			return
		}
		s.run(p)
	}
	return w
}

// Nullish mirrors schema.nullish().
func (s *Schema) Nullish() *Schema { return s.Nullable().Optional() }

// Default mirrors schema.default(value).
func (s *Schema) Default(v any) *Schema {
	w := &Schema{kind: "default", inner: s, optin: "optional"}
	w.run = func(p *payload) {
		if IsUndefined(p.value) {
			p.value = v
			return
		}
		s.run(p)
	}
	return w
}

// Preprocess mirrors z.preprocess(fn, schema).
func Preprocess(fn func(v any) any, s *Schema) *Schema {
	w := &Schema{kind: "pipe", inner: s, optin: s.optin, optout: s.optout}
	w.run = func(p *payload) {
		p.value = fn(p.value)
		s.run(p)
	}
	return w
}

// Transform mirrors schema.transform(fn).
func (s *Schema) Transform(fn func(v any) (any, error)) *Schema {
	w := &Schema{kind: "pipe", inner: s, optin: s.optin}
	w.run = func(p *payload) {
		s.run(p)
		if len(p.issues) > 0 && aborted(p, 0) {
			return
		}
		v, err := fn(p.value)
		if err != nil {
			p.issues = append(p.issues, &Issue{Code: "custom", Message: err.Error()})
			return
		}
		p.value = v
	}
	return w
}

// Refine mirrors schema.refine(fn, { message }).
func (s *Schema) Refine(fn func(v any) bool, message string) *Schema {
	if message == "" {
		message = "Invalid input"
	}
	c := clone(s)
	inner := s.run
	c.run = func(p *payload) {
		inner(p)
		if len(p.issues) > 0 && aborted(p, 0) {
			return
		}
		if !fn(p.value) {
			p.issues = append(p.issues, &Issue{Code: "custom", Message: message, Continue: true})
		}
	}
	c.refiners = append(append([]func(*Schema) *Schema{}, s.refiners...), func(o *Schema) *Schema { return o.Refine(fn, message) })
	return c
}

// SuperRefine mirrors schema.superRefine: fn adds issues through add.
func (s *Schema) SuperRefine(fn func(v any, add func(*Issue))) *Schema {
	c := clone(s)
	inner := s.run
	c.run = func(p *payload) {
		inner(p)
		if len(p.issues) > 0 && aborted(p, 0) {
			return
		}
		fn(p.value, func(i *Issue) {
			if i.Code == "" {
				i.Code = "custom"
			}
			i.Continue = true
			p.issues = append(p.issues, i)
		})
	}
	c.refiners = append(append([]func(*Schema) *Schema{}, s.refiners...), func(o *Schema) *Schema { return o.SuperRefine(fn) })
	return c
}

// Lazy mirrors z.lazy(fn).
func Lazy(fn func() *Schema) *Schema {
	w := &Schema{kind: "lazy"}
	var resolved *Schema
	w.run = func(p *payload) {
		if resolved == nil {
			resolved = fn()
		}
		resolved.run(p)
	}
	return w
}

func addCheck(s *Schema, check func(p *payload)) *Schema {
	c := clone(s)
	c.checks = append(c.checks, check)
	base := c
	switch c.kind {
	case "string":
		base.run = func(p *payload) {
			if _, ok := p.value.(string); !ok {
				invalidType(p, "string")
				return
			}
			base.withChecks(p)
		}
	case "number":
		base.run = func(p *payload) {
			switch t := p.value.(type) {
			case float64:
				if math.IsNaN(t) || math.IsInf(t, 0) {
					invalidType(p, "number")
					return
				}
			case int:
				p.value = float64(t)
			default:
				invalidType(p, "number")
				return
			}
			base.withChecks(p)
		}
	case "array":
		el := c.inner
		arr := Array(el)
		base.run = func(p *payload) {
			arr.run(p)
			if len(p.issues) > 0 {
				return
			}
			base.withChecks(p)
		}
	}
	return c
}

func lengthOf(v any) int {
	switch t := v.(type) {
	case string:
		return len([]rune(t))
	case []any:
		return len(t)
	}
	return 0
}

// Min mirrors .min(n) for strings, arrays and numbers (gte).
func (s *Schema) Min(n float64) *Schema {
	return addCheck(s, func(p *payload) {
		origin := s.kind
		if f, ok := p.value.(float64); ok {
			if f < n {
				p.issues = append(p.issues, &Issue{Code: "too_small", Minimum: n, Origin: origin, Continue: true, Message: fmt.Sprintf("Too small: expected number to be >=%s", JSString(n))})
			}
			return
		}
		if float64(lengthOf(p.value)) < n {
			unit := "characters"
			if origin == "array" {
				unit = "items"
			}
			p.issues = append(p.issues, &Issue{Code: "too_small", Minimum: n, Origin: origin, Continue: true, Message: fmt.Sprintf("Too small: expected %s to have >=%s %s", origin, JSString(n), unit)})
		}
	})
}

// Max mirrors .max(n).
func (s *Schema) Max(n float64) *Schema {
	return addCheck(s, func(p *payload) {
		origin := s.kind
		if f, ok := p.value.(float64); ok {
			if f > n {
				p.issues = append(p.issues, &Issue{Code: "too_big", Maximum: n, Origin: origin, Continue: true, Message: fmt.Sprintf("Too big: expected number to be <=%s", JSString(n))})
			}
			return
		}
		if float64(lengthOf(p.value)) > n {
			unit := "characters"
			if origin == "array" {
				unit = "items"
			}
			p.issues = append(p.issues, &Issue{Code: "too_big", Maximum: n, Origin: origin, Continue: true, Message: fmt.Sprintf("Too big: expected %s to have <=%s %s", origin, JSString(n), unit)})
		}
	})
}

// Nonnegative mirrors .nonnegative().
func (s *Schema) Nonnegative() *Schema { return s.Min(0) }

// Int mirrors z.number().int().
func (s *Schema) Int() *Schema {
	return addCheck(s, func(p *payload) {
		if f, ok := p.value.(float64); ok && f != math.Trunc(f) {
			p.issues = append(p.issues, &Issue{Code: "invalid_type", Expected: "int", Message: "Invalid input: expected int, received number"})
		}
	})
}

var emailRe = regexp.MustCompile(`^(?i)[A-Za-z0-9_'+\-\.]*[A-Za-z0-9_+\-]@([A-Za-z0-9][A-Za-z0-9\-]*\.)+[A-Za-z]{2,}$`)

// Email mirrors z.string().email().
func (s *Schema) Email() *Schema {
	return addCheck(s, func(p *payload) {
		if str, ok := p.value.(string); ok && (!emailRe.MatchString(str) || strings.Contains(str, "..")) {
			p.issues = append(p.issues, &Issue{Code: "invalid_format", Origin: "string", Continue: true, Message: "Invalid email address"})
		}
	})
}
