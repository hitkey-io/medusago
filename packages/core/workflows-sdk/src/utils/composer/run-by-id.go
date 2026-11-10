package composer

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// runner runs a registered workflow with an untyped (JSON-like) input, the
// way the workflow engine runs MedusaWorkflow.getWorkflow(id).
type runner func(ctx context.Context, c Container, input any, opts RunOptions) (any, RunInfo, error)

var runners = map[string]runner{}

func registerRunner[I, O any](w *Workflow[I, O]) {
	registryMu.Lock()
	runners[w.ID] = func(ctx context.Context, c Container, input any, opts RunOptions) (any, RunInfo, error) {
		var in I
		if err := DecodeInput(input, &in); err != nil {
			return nil, RunInfo{}, err
		}
		opts.Notify = true
		return w.RunWithInfo(ctx, c, in, opts)
	}
	registryMu.Unlock()
}

// RunByID mirrors workflowEngine.run(workflowId, { input, transactionId }):
// the registered workflow runs with the input decoded into its input type.
func RunByID(ctx context.Context, c Container, workflowID string, input any, opts RunOptions) (any, RunInfo, error) {
	registryMu.RLock()
	run := runners[workflowID]
	registryMu.RUnlock()
	if run == nil {
		return nil, RunInfo{}, notFoundWorkflow(workflowID)
	}
	return run(ctx, c, input, opts)
}

// DecodeInput converts a JSON-like value (*dal.Record, []any, primitives)
// into out. Struct fields match by json tag, else by name ignoring case
// and underscores (cart_id -> CartID).
func DecodeInput(in any, out any) error {
	rv := reflect.ValueOf(out)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("DecodeInput: out must be a non-nil pointer")
	}
	return decodeInto(in, rv.Elem())
}

var recordPtrType = reflect.TypeOf((*dal.Record)(nil))

func normalizeName(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, "_", ""))
}

func decodeInto(in any, dst reflect.Value) error {
	if in == nil {
		return nil
	}
	if dst.Type() == recordPtrType {
		if r, ok := in.(*dal.Record); ok {
			dst.Set(reflect.ValueOf(r))
			return nil
		}
		return fmt.Errorf("expected an object, got %T", in)
	}
	src := reflect.ValueOf(in)
	if src.Type().AssignableTo(dst.Type()) {
		dst.Set(src)
		return nil
	}
	switch dst.Kind() {
	case reflect.Pointer:
		v := reflect.New(dst.Type().Elem())
		if err := decodeInto(in, v.Elem()); err != nil {
			return err
		}
		dst.Set(v)
		return nil
	case reflect.Interface:
		if src.Type().Implements(dst.Type()) {
			dst.Set(src)
			return nil
		}
	case reflect.Slice:
		items, ok := in.([]any)
		if !ok {
			return fmt.Errorf("expected an array, got %T", in)
		}
		s := reflect.MakeSlice(dst.Type(), len(items), len(items))
		for i, item := range items {
			if err := decodeInto(item, s.Index(i)); err != nil {
				return err
			}
		}
		dst.Set(s)
		return nil
	case reflect.Map:
		r, ok := in.(*dal.Record)
		if !ok || dst.Type().Key().Kind() != reflect.String {
			return fmt.Errorf("expected an object, got %T", in)
		}
		m := reflect.MakeMapWithSize(dst.Type(), len(r.Keys()))
		for _, k := range r.Keys() {
			v, _ := r.Get(k)
			ev := reflect.New(dst.Type().Elem()).Elem()
			if err := decodeInto(v, ev); err != nil {
				return err
			}
			m.SetMapIndex(reflect.ValueOf(k).Convert(dst.Type().Key()), ev)
		}
		dst.Set(m)
		return nil
	case reflect.Struct:
		r, ok := in.(*dal.Record)
		if !ok {
			return fmt.Errorf("expected an object, got %T", in)
		}
		return decodeStruct(r, dst)
	case reflect.String:
		if s, ok := in.(string); ok {
			dst.SetString(s)
			return nil
		}
	case reflect.Bool:
		if b, ok := in.(bool); ok {
			dst.SetBool(b)
			return nil
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if f, ok := toFloat(in); ok {
			dst.SetInt(int64(f))
			return nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if f, ok := toFloat(in); ok && f >= 0 {
			dst.SetUint(uint64(f))
			return nil
		}
	case reflect.Float32, reflect.Float64:
		if f, ok := toFloat(in); ok {
			dst.SetFloat(f)
			return nil
		}
	}
	if src.Type().ConvertibleTo(dst.Type()) && src.Kind() == dst.Kind() {
		dst.Set(src.Convert(dst.Type()))
		return nil
	}
	return fmt.Errorf("cannot use %T as %s", in, dst.Type())
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}

func decodeStruct(r *dal.Record, dst reflect.Value) error {
	t := dst.Type()
	byName := map[string]int{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name := f.Name
		if tag, ok := f.Tag.Lookup("json"); ok {
			if n := strings.Split(tag, ",")[0]; n == "-" {
				continue
			} else if n != "" {
				name = n
			}
		}
		byName[normalizeName(name)] = i
	}
	for _, k := range r.Keys() {
		i, ok := byName[normalizeName(k)]
		if !ok {
			continue
		}
		v, _ := r.Get(k)
		if err := decodeInto(v, dst.Field(i)); err != nil {
			return fmt.Errorf("%s: %w", k, err)
		}
	}
	return nil
}
