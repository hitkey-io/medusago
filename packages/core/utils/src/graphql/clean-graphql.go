package graphql

var builtinScalars = []string{"String", "Int", "Float", "Boolean", "ID"}

// CleanGraphQLSchema mirrors cleanGraphQLSchema: fields whose type is not
// defined are removed, and extensions of types that are not defined are
// removed (and reported with an "__extended" key). It returns the cleaned
// document and the map of what was not found.
func CleanGraphQLSchema(doc *Document) (*Document, map[string]map[string]string) {
	typeNames := map[string]bool{}
	for _, s := range builtinScalars {
		typeNames[s] = true
	}
	for _, d := range doc.Definitions {
		if d.Extension {
			continue
		}
		switch d.Kind {
		case KindObject, KindInterface, KindEnum, KindScalar, KindInput, KindUnion:
			typeNames[d.Name] = true
		}
	}
	notFound := map[string]map[string]string{}
	mark := func(parent, key, value string) {
		if notFound[parent] == nil {
			notFound[parent] = map[string]string{}
		}
		notFound[parent][key] = value
	}
	out := &Document{}
	for _, d := range doc.Definitions {
		if d.Extension && d.Kind == KindObject && !typeNames[d.Name] {
			mark(d.Name, "__extended", "")
			continue
		}
		c := *d
		if d.Kind == KindObject || d.Kind == KindInterface {
			c.Fields = nil
			for _, f := range d.Fields {
				if name := f.Type.NamedType(); !typeNames[name] {
					mark(d.Name, f.Name, name)
					continue
				}
				c.Fields = append(c.Fields, f)
			}
		}
		out.Definitions = append(out.Definitions, &c)
	}
	return out, notFound
}
