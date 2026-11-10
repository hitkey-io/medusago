package dal

import (
	"strings"
)

// isFieldPathResolvable mirrors isFieldPathResolvable from
// prune-find-options-against-metadata.ts.
func (r *Repository) isFieldPathResolvable(path string) bool {
	if path == "*" {
		return true
	}
	segments := strings.Split(path, ".")
	current := r.Entity
	for i, raw := range segments {
		segment := strings.Split(raw, ":")[0]
		isLast := i == len(segments)-1
		if segment == "*" {
			return isLast && current != nil
		}
		if current == nil {
			return false
		}
		prop, ok := current.MetaProperty(segment)
		if !ok {
			if _, isFormula := formulaOf(current.Meta.ClassName, segment); isFormula && isLast {
				return true
			}
			return false
		}
		if !isLast {
			if prop.Kind == "scalar" || prop.Entity == "" {
				return false
			}
			target, err := r.Schema.Entity(prop.Entity)
			if err != nil {
				return false
			}
			current = target
		}
	}
	return true
}

// pruneFindOptions mirrors pruneFindOptionsAgainstMetadata: fields and
// relations that do not resolve against the entity metadata are dropped.
func (r *Repository) pruneFindOptions(opts FindOptions) FindOptions {
	if opts.Select != nil {
		kept := make([]string, 0, len(opts.Select))
		for _, f := range opts.Select {
			if r.isFieldPathResolvable(f) {
				kept = append(kept, f)
			}
		}
		opts.Select = kept
	}
	if opts.Relations != nil {
		kept := make([]string, 0, len(opts.Relations))
		for _, p := range opts.Relations {
			if r.isFieldPathResolvable(p) {
				// Like MikroORM's populate, a path ending with a scalar
				// property ("captures.raw_amount") populates its relation.
				if trimmed := r.trimScalarTail(p); trimmed != "" {
					kept = append(kept, trimmed)
				}
			}
		}
		opts.Relations = kept
	}
	return opts
}

// trimScalarTail drops the trailing scalar segment of a relation path.
func (r *Repository) trimScalarTail(path string) string {
	segments := strings.Split(path, ".")
	current := r.Entity
	for i, seg := range segments {
		prop, ok := current.MetaProperty(strings.Split(seg, ":")[0])
		if !ok || prop.Kind == "scalar" || prop.Entity == "" {
			return strings.Join(segments[:i], ".")
		}
		target, err := r.Schema.Entity(prop.Entity)
		if err != nil {
			return strings.Join(segments[:i+1], ".")
		}
		current = target
	}
	return path
}
