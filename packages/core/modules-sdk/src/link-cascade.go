package modulessdk

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// ModuleCaller is a module service whose generated methods can be called by
// name (MedusaService.Call): Link calls softDelete<Suffix> / restore<Suffix>
// on modules when cascading.
type ModuleCaller interface {
	Call(ctx context.Context, method string, args []any) (any, error)
}

// cascadeRelationship mirrors a RemoteRelationship stored in Link.relations.
type cascadeRelationship struct {
	joiner.JoinerRelationship
	isPrimary bool
}

// CascadeError mirrors CascadeError.
type CascadeError struct {
	ServiceName string
	Method      string
	Args        *dal.Record
	Error       error
}

func (l *Link) addRelationship(serviceName string, rel cascadeRelationship) {
	if l.relations == nil {
		l.relations = map[string]map[string][]cascadeRelationship{}
	}
	key := rel.ForeignKey
	if rel.isPrimary {
		key = rel.PrimaryKey
	}
	m := l.relations[serviceName]
	if m == nil {
		m = map[string][]cascadeRelationship{}
		l.relations[serviceName] = m
		l.relationOrder = append(l.relationOrder, serviceName)
	}
	m[key] = append(m[key], rel)
}

// addModule mirrors Link.addModule for the relationship part.
func (l *Link) addModule(cfg *joiner.ModuleJoinerConfig, serviceName string) {
	for _, rel := range cfg.Relationships {
		if cfg.IsLink && !rel.DeleteCascade {
			continue
		}
		l.addRelationship(serviceName, cascadeRelationship{JoinerRelationship: rel})
	}
	for _, ext := range cfg.Extends {
		rel := ext.Relationship
		rel.ServiceName = serviceName
		l.addRelationship(ext.ServiceName, cascadeRelationship{JoinerRelationship: rel, isPrimary: true})
	}
}

// linkableKeysOf mirrors getLinkableKeys.
func linkableKeysOf(cfg *joiner.ModuleJoinerConfig) []string {
	if cfg == nil {
		return nil
	}
	if len(cfg.LinkableKeys) > 0 {
		out := make([]string, len(cfg.LinkableKeys))
		for i, kv := range cfg.LinkableKeys {
			out[i] = kv.Key
		}
		return out
	}
	return cfg.PrimaryKeys
}

func toPascalCase(s string) string {
	var b strings.Builder
	up := true
	for _, r := range s {
		if r == '_' || r == '-' || r == ' ' {
			up = true
			continue
		}
		if up {
			b.WriteRune(unicode.ToUpper(r))
			up = false
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func anyList(v any) []any {
	if l, ok := v.([]any); ok {
		return l
	}
	if l, ok := v.([]string); ok {
		out := make([]any, len(l))
		for i, s := range l {
			out[i] = s
		}
		return out
	}
	if v == nil {
		return nil
	}
	return []any{v}
}

// callService runs <method> on a link or module service with the delete
// keys and returns the removed linkable keys.
func (l *Link) callService(ctx context.Context, serviceName, method string, keys *dal.Record, returnFields []string) (*dal.Record, error) {
	if svc, ok := l.services[serviceName]; ok {
		filters := []*dal.Record{keys}
		if strings.HasPrefix(method, "restore") {
			return svc.Restore(ctx, filters, returnFields)
		}
		return svc.SoftDelete(ctx, filters, returnFields)
	}
	mod, ok := l.modules[serviceName].(ModuleCaller)
	if !ok {
		return nil, fmt.Errorf("service %s does not support %s", serviceName, method)
	}
	res, err := mod.Call(ctx, method, []any{keys})
	if err != nil {
		return nil, err
	}
	rec, _ := res.(*dal.Record)
	out := dal.NewRecord()
	if rec == nil {
		return out, nil
	}
	pick := map[string]bool{}
	for _, f := range returnFields {
		pick[f] = true
	}
	for _, k := range rec.Keys() {
		if len(pick) > 0 && !pick[k] {
			continue
		}
		v, _ := rec.Get(k)
		vals := []any{}
		for _, x := range anyList(v) {
			if x != nil && x != "" {
				vals = append(vals, x)
			}
		}
		out.Set(k, vals)
	}
	return out, nil
}

func (l *Link) returnFieldsOf(serviceName string) []string {
	if svc, ok := l.services[serviceName]; ok {
		return linkableKeysOf(svc.Config)
	}
	return linkableKeysOf(l.moduleConfigs[serviceName])
}

// executeCascade mirrors Link.executeCascade (sequentially; upstream runs
// the branches concurrently).
func (l *Link) executeCascade(ctx context.Context, removed *dal.Record, executionMethod string) ([]CascadeError, *dal.Record) {
	returnIds := dal.NewRecord()
	processed := map[string]map[string]bool{}
	var errs []CascadeError

	type serviceKeys struct {
		name string
		keys *dal.Record
	}
	var cascade func(services []serviceKeys, isCascading bool)
	cascade = func(services []serviceKeys, isCascading bool) {
		if len(errs) > 0 {
			return
		}
		for _, si := range services {
			rels := l.relations[si.name]
			if rels == nil {
				continue
			}
			for _, field := range si.keys.Keys() {
				fv, _ := si.keys.Get(field)
				values := anyList(fv)
				related := rels[field]
				if len(related) == 0 || len(values) == 0 {
					continue
				}
				for _, rel := range related {
					hash := rel.ServiceName + "-" + rel.PrimaryKey
					if processed[hash] == nil {
						processed[hash] = map[string]bool{}
					}
					var unprocessed []any
					for _, id := range values {
						k := fmt.Sprint(id)
						if !processed[hash][k] {
							unprocessed = append(unprocessed, id)
						}
					}
					if len(unprocessed) == 0 {
						continue
					}
					for _, id := range unprocessed {
						processed[hash][fmt.Sprint(id)] = true
					}
					delKeys := dal.NewRecord()
					delKeys.Set(rel.PrimaryKey, unprocessed)
					returnFields := l.returnFieldsOf(rel.ServiceName)
					method := executionMethod
					if s, ok := rel.Args["methodSuffix"].(string); ok && s != "" {
						method += toPascalCase(s)
					}
					deleted, err := l.callService(ctx, rel.ServiceName, method, delKeys, returnFields)
					if err != nil {
						errs = append(errs, CascadeError{ServiceName: rel.ServiceName, Method: method, Args: delKeys, Error: err})
						continue
					}
					if deleted == nil || deleted.Len() == 0 {
						continue
					}
					if !isCascading {
						returnIds.Set(rel.ServiceName, deleted)
					} else if len(returnFields) > 0 {
						mainKey := returnFields[0]
						cur, _ := returnIds.Get(rel.ServiceName)
						entry, _ := cur.(*dal.Record)
						if entry == nil {
							entry = dal.NewRecord()
							returnIds.Set(rel.ServiceName, entry)
						}
						prev, _ := entry.Get(mainKey)
						seen := map[string]bool{}
						merged := []any{}
						dv, _ := deleted.Get(mainKey)
						for _, x := range append(anyList(prev), anyList(dv)...) {
							k := fmt.Sprint(x)
							if !seen[k] {
								seen[k] = true
								merged = append(merged, x)
							}
						}
						entry.Set(mainKey, merged)
					}
					for _, k := range deleted.Keys() {
						h := rel.ServiceName + "-" + k
						if processed[h] == nil {
							processed[h] = map[string]bool{}
						}
						v, _ := deleted.Get(k)
						for _, id := range anyList(v) {
							processed[h][fmt.Sprint(id)] = true
						}
					}
					cascade([]serviceKeys{{name: rel.ServiceName, keys: deleted}}, true)
				}
			}
		}
	}

	var start []serviceKeys
	for _, name := range removed.Keys() {
		v, _ := removed.Get(name)
		r, _ := v.(*dal.Record)
		keys := dal.NewRecord()
		if r != nil {
			for _, f := range r.Keys() {
				fv, _ := r.Get(f)
				keys.Set(f, anyList(fv))
			}
		}
		start = append(start, serviceKeys{name: name, keys: keys})
	}
	cascade(start, false)
	return errs, returnIds
}

// Delete mirrors link.delete: links (and cascaded entities) of the given
// module keys are soft deleted. Errors are returned, not thrown.
func (l *Link) Delete(ctx context.Context, removed *dal.Record) ([]CascadeError, *dal.Record) {
	return l.executeCascade(ctx, removed, "softDelete")
}

// Restore mirrors link.restore.
func (l *Link) Restore(ctx context.Context, removed *dal.Record) ([]CascadeError, *dal.Record) {
	return l.executeCascade(ctx, removed, "restore")
}

// DeleteEntityInput builds { [module]: { [key]: values } }.
func DeleteEntityInput(module, key string, values any) *dal.Record {
	inner := dal.NewRecord()
	inner.Set(key, anyList(values))
	out := dal.NewRecord()
	out.Set(module, inner)
	return out
}

var _ = json.Marshal
