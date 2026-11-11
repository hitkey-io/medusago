package modulesdiff

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	typesmodulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	modulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/medusa/src/modules"
)

// moduleServiceInstance builds the module's own service (with its
// overridden methods), for the cases marked "service".
func moduleServiceInstance(pool *pgxpool.Pool, key string) (any, error) {
	exp, ok := modules.Registry[modulessdk.MODULE_PACKAGE_NAMES[key]]
	if !ok || exp.Service == nil {
		return nil, fmt.Errorf("module %s has no service", key)
	}
	return exp.Service(typesmodulessdk.ServiceOptions{DB: pool})
}

var (
	contextType    = reflect.TypeOf((*context.Context)(nil)).Elem()
	errorType      = reflect.TypeOf((*error)(nil)).Elem()
	recordType     = reflect.TypeOf((*dal.Record)(nil))
	recordListType = reflect.TypeOf([]*dal.Record(nil))
)

// callService calls the exported method of a module service matching the
// upstream method name (createProducts -> CreateProducts). Upstream accepts
// a single object or an array where the Go methods take a list, and returns
// a single object for a single object (or an id) input.
func callService(ctx context.Context, svc any, method string, args []any) (any, error) {
	name := strings.ToUpper(method[:1]) + method[1:]
	m := reflect.ValueOf(svc).MethodByName(name)
	if !m.IsValid() {
		return nil, fmt.Errorf("%T has no method %s", svc, name)
	}
	t := m.Type()
	in := []reflect.Value{reflect.ValueOf(ctx)}
	if t.NumIn() == 0 || t.In(0) != contextType {
		return nil, fmt.Errorf("%s: unsupported signature %s", name, t)
	}
	single := false
	for i := 1; i < t.NumIn(); i++ {
		var arg any
		if i-1 < len(args) {
			arg = args[i-1]
		}
		pt := t.In(i)
		switch {
		case pt == recordListType:
			var list []*dal.Record
			switch a := arg.(type) {
			case []any:
				for _, x := range a {
					r, _ := x.(*dal.Record)
					list = append(list, r)
				}
			case *dal.Record:
				single = true
				list = []*dal.Record{a}
			}
			in = append(in, reflect.ValueOf(list))
		case pt == recordType:
			r, _ := arg.(*dal.Record)
			in = append(in, reflect.ValueOf(r))
		case pt.Kind() == reflect.Interface:
			if i == 1 {
				if _, ok := arg.(string); ok {
					single = true
				}
			}
			if arg == nil {
				in = append(in, reflect.Zero(pt))
			} else {
				in = append(in, reflect.ValueOf(arg))
			}
		case pt.Kind() == reflect.String:
			s, _ := arg.(string)
			in = append(in, reflect.ValueOf(s))
		case pt == reflect.TypeOf([]any(nil)):
			a, _ := arg.([]any)
			in = append(in, reflect.ValueOf(a))
		case pt == reflect.TypeOf([]string(nil)):
			list := []string{}
			a, _ := arg.([]any)
			for _, x := range a {
				list = append(list, fmt.Sprint(x))
			}
			in = append(in, reflect.ValueOf(list))
		case pt.Kind() == reflect.Struct:
			// options objects: snake_case keys -> exported fields
			v := reflect.New(pt).Elem()
			if r, ok := arg.(*dal.Record); ok {
				for _, k := range r.Keys() {
					x, _ := r.Get(k)
					name := ""
					for _, part := range strings.Split(k, "_") {
						if part != "" {
							name += strings.ToUpper(part[:1]) + part[1:]
						}
					}
					f := v.FieldByName(name)
					if f.IsValid() && f.CanSet() && x != nil && reflect.TypeOf(x).AssignableTo(f.Type()) {
						f.Set(reflect.ValueOf(x))
					}
				}
			}
			in = append(in, v)
		default:
			return nil, fmt.Errorf("%s: unsupported parameter %s", name, pt)
		}
	}
	out := m.Call(in)
	last := out[len(out)-1]
	if last.Type() == errorType && !last.IsNil() {
		return nil, last.Interface().(error)
	}
	if len(out) < 2 {
		return nil, nil
	}
	result := out[0].Interface()
	if list, ok := result.([]*dal.Record); ok && single {
		if len(list) == 0 {
			return nil, nil
		}
		return list[0], nil
	}
	return result, nil
}

// sortAt sorts the arrays at path (a "*" segment walks every array item)
// by the string form of an item field.
func sortAt(v any, path []string, field string) any {
	if len(path) == 0 {
		arr, ok := v.([]any)
		if !ok {
			return v
		}
		key := func(item any) string {
			if m, ok := item.(map[string]any); ok {
				return fmt.Sprint(m[field])
			}
			return ""
		}
		sorted := append([]any{}, arr...)
		sort.SliceStable(sorted, func(i, j int) bool { return key(sorted[i]) < key(sorted[j]) })
		return sorted
	}
	switch t := v.(type) {
	case map[string]any:
		if child, ok := t[path[0]]; ok {
			t[path[0]] = sortAt(child, path[1:], field)
		}
	case []any:
		if path[0] == "*" {
			for i := range t {
				t[i] = sortAt(t[i], path[1:], field)
			}
		} else if i, err := strconv.Atoi(path[0]); err == nil && i < len(t) {
			t[i] = sortAt(t[i], path[1:], field)
		}
	}
	return v
}
