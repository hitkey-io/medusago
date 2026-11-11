package graphql

import (
	"reflect"
	"testing"
)

// The merged type keeps the fields in order of appearance, a non-null
// redefinition makes a field non-null and same-name directives merge their
// list arguments. Expected values from makeExecutableSchema (graphql 17,
// @graphql-tools/schema 11) over the same document.
func TestMergeTypeDefsDirectivesAndFields(t *testing.T) {
	doc, err := Parse(`
directive @Listeners (values: [String!]) on OBJECT
scalar DateTime
type B @Listeners(values: ["b"]) { id: ID x: [C] when: DateTime f: Float }
type A { id: ID b: B n: Int }
extend type A @Listeners(values: ["a"]) { extra: String id: ID! }
type C @Listeners(values: ["c1", "c2"]) { id: ID }
extend type C @Listeners(values: ["c3", "c1"]) { y: String }
extend type B { y: String }
`)
	if err != nil {
		t.Fatal(err)
	}
	s := MergeTypeDefs(doc)
	var names []string
	for _, ty := range s.Types {
		names = append(names, ty.Name)
	}
	if want := []string{"DateTime", "B", "A", "C"}; !reflect.DeepEqual(names, want) {
		t.Errorf("types: want %v, got %v", want, names)
	}
	fields := func(name string) []string {
		var out []string
		for _, f := range s.Type(name).Fields {
			out = append(out, f.Name+":"+f.Type.String())
		}
		return out
	}
	if want := []string{"id:ID!", "b:B", "n:Int", "extra:String"}; !reflect.DeepEqual(fields("A"), want) {
		t.Errorf("A: want %v, got %v", want, fields("A"))
	}
	if want := []string{"id:ID", "x:[C]", "when:DateTime", "f:Float", "y:String"}; !reflect.DeepEqual(fields("B"), want) {
		t.Errorf("B: want %v, got %v", want, fields("B"))
	}
	listeners := func(name string) []string {
		var out []string
		for _, d := range s.Type(name).Directives {
			if d.Name != "Listeners" {
				continue
			}
			for _, v := range d.Arguments[0].Value.Values {
				out = append(out, v.Value)
			}
		}
		return out
	}
	for name, want := range map[string][]string{"A": {"a"}, "B": {"b"}, "C": {"c1", "c2", "c3"}} {
		if got := listeners(name); !reflect.DeepEqual(got, want) {
			t.Errorf("%s listeners: want %v, got %v", name, want, got)
		}
	}
}
