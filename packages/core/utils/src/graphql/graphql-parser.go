// Package graphql mirrors the GraphQL helpers of @medusajs/utils/src/graphql
// for what the module runtime needs from the module schemas: a parser of the
// type system definition language, cleanGraphQLSchema, the merge of the type
// definitions (mergeTypeDefs) and the relations of the merged schema
// (extractRelationsFromGQL).
package graphql

import (
	"fmt"
	"strings"
)

// TypeRef is a type reference: a named type, a list or a non-null wrapper.
type TypeRef struct {
	Name    string
	OfType  *TypeRef
	List    bool
	NonNull bool
}

// NamedType returns the innermost named type.
func (t *TypeRef) NamedType() string {
	for t.OfType != nil {
		t = t.OfType
	}
	return t.Name
}

// IsList reports whether the type is a list (possibly non-null).
func (t *TypeRef) IsList() bool {
	if t.NonNull && t.OfType != nil {
		return t.OfType.IsList()
	}
	return t.List
}

// String prints the type like graphql-js.
func (t *TypeRef) String() string {
	switch {
	case t.NonNull:
		return t.OfType.String() + "!"
	case t.List:
		return "[" + t.OfType.String() + "]"
	}
	return t.Name
}

// FieldDefinition is a field of an object, interface or input type.
type FieldDefinition struct {
	Name string
	Type *TypeRef
}

// Definition kinds.
const (
	KindObject    = "ObjectTypeDefinition"
	KindInterface = "InterfaceTypeDefinition"
	KindEnum      = "EnumTypeDefinition"
	KindScalar    = "ScalarTypeDefinition"
	KindInput     = "InputObjectTypeDefinition"
	KindUnion     = "UnionTypeDefinition"
	KindDirective = "DirectiveDefinition"
	KindSchema    = "SchemaDefinition"
)

// Definition is a type system definition or extension.
type Definition struct {
	Kind      string
	Extension bool
	Name      string
	Fields    []*FieldDefinition
	// Values are the enum values; Types the union members.
	Values []string
	Types  []string
	// Interfaces are the implemented interfaces.
	Interfaces []string
	// Directives are the directives applied to the type (object types,
	// interfaces and input types).
	Directives []*Directive
}

// Directive is an applied directive: @name(arg: value, ...).
type Directive struct {
	Name      string
	Arguments []*Argument
}

// Argument is an argument of an applied directive.
type Argument struct {
	Name  string
	Value *Value
}

// Value kinds.
const (
	ValueVariable = "Variable"
	ValueInt      = "IntValue"
	ValueFloat    = "FloatValue"
	ValueString   = "StringValue"
	ValueBoolean  = "BooleanValue"
	ValueNull     = "NullValue"
	ValueEnum     = "EnumValue"
	ValueList     = "ListValue"
	ValueObject   = "ObjectValue"
)

// Value is a constant value of the language. Value holds the scalar value
// (as written, a string's content), Values the items of a list.
type Value struct {
	Kind   string
	Value  string
	Values []*Value
}

// Document is a parsed schema.
type Document struct {
	Definitions []*Definition
}

type token struct {
	kind  byte // 'n' name, 'p' punctuator, 's' string, 'v' number, 0 EOF
	value string
}

type lexer struct {
	src string
	pos int
}

func (l *lexer) next() (token, error) {
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == ',':
			l.pos++
		case c == '#':
			for l.pos < len(l.src) && l.src[l.pos] != '\n' && l.src[l.pos] != '\r' {
				l.pos++
			}
		default:
			goto tok
		}
	}
	return token{}, nil
tok:
	c := l.src[l.pos]
	switch {
	case c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z'):
		start := l.pos
		for l.pos < len(l.src) {
			c := l.src[l.pos]
			if c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
				l.pos++
				continue
			}
			break
		}
		return token{'n', l.src[start:l.pos]}, nil
	case c == '-' || (c >= '0' && c <= '9'):
		start := l.pos
		l.pos++
		for l.pos < len(l.src) && strings.IndexByte("0123456789.eE+-", l.src[l.pos]) >= 0 {
			l.pos++
		}
		return token{'v', l.src[start:l.pos]}, nil
	case c == '"':
		if strings.HasPrefix(l.src[l.pos:], `"""`) {
			end := strings.Index(l.src[l.pos+3:], `"""`)
			for end >= 0 && l.src[l.pos+3+end-1] == '\\' {
				next := strings.Index(l.src[l.pos+3+end+3:], `"""`)
				if next < 0 {
					end = -1
					break
				}
				end += 3 + next
			}
			if end < 0 {
				return token{}, fmt.Errorf("Syntax Error: Unterminated string.")
			}
			v := l.src[l.pos+3 : l.pos+3+end]
			l.pos += 6 + end
			return token{'s', v}, nil
		}
		start := l.pos
		l.pos++
		for l.pos < len(l.src) && l.src[l.pos] != '"' {
			if l.src[l.pos] == '\\' {
				l.pos++
			}
			if l.src[l.pos] == '\n' {
				return token{}, fmt.Errorf("Syntax Error: Unterminated string.")
			}
			l.pos++
		}
		if l.pos >= len(l.src) {
			return token{}, fmt.Errorf("Syntax Error: Unterminated string.")
		}
		l.pos++
		return token{'s', l.src[start+1 : l.pos-1]}, nil
	case c == '.':
		if strings.HasPrefix(l.src[l.pos:], "...") {
			l.pos += 3
			return token{'p', "..."}, nil
		}
	case strings.IndexByte("!$&()=:@[]{}|", c) >= 0:
		l.pos++
		return token{'p', string(c)}, nil
	}
	return token{}, fmt.Errorf("Syntax Error: Unexpected character: %q.", c)
}

type parser struct {
	toks []token
	i    int
}

func (p *parser) peek() token {
	if p.i < len(p.toks) {
		return p.toks[p.i]
	}
	return token{}
}

func (p *parser) take() token {
	t := p.peek()
	if p.i < len(p.toks) {
		p.i++
	}
	return t
}

func (p *parser) is(kind byte, value string) bool {
	t := p.peek()
	return t.kind == kind && t.value == value
}

func (p *parser) expect(kind byte, value string) error {
	t := p.take()
	if t.kind != kind || (value != "" && t.value != value) {
		return fmt.Errorf("Syntax Error: Expected %q, found %q.", value, t.value)
	}
	return nil
}

func (p *parser) name() (string, error) {
	t := p.take()
	if t.kind != 'n' {
		return "", fmt.Errorf("Syntax Error: Expected Name, found %q.", t.value)
	}
	return t.value, nil
}

// Parse parses a schema in the type system definition language.
func Parse(src string) (*Document, error) {
	l := &lexer{src: src}
	var toks []token
	for {
		t, err := l.next()
		if err != nil {
			return nil, err
		}
		if t.kind == 0 {
			break
		}
		toks = append(toks, t)
	}
	p := &parser{toks: toks}
	doc := &Document{}
	for p.i < len(p.toks) {
		def, err := p.definition()
		if err != nil {
			return nil, err
		}
		if def != nil {
			doc.Definitions = append(doc.Definitions, def)
		}
	}
	return doc, nil
}

func (p *parser) skipDescription() {
	if p.peek().kind == 's' {
		p.take()
	}
}

// skipValue skips a value (or a balanced group).
func (p *parser) skipValue() error {
	t := p.take()
	switch {
	case t.kind == 'p' && (t.value == "[" || t.value == "{" || t.value == "("):
		closing := map[string]string{"[": "]", "{": "}", "(": ")"}[t.value]
		for !p.is('p', closing) {
			if p.peek().kind == 0 {
				return fmt.Errorf("Syntax Error: Expected %q.", closing)
			}
			if err := p.skipValue(); err != nil {
				return err
			}
		}
		p.take()
	case t.kind == 'p' && t.value == "$":
		_, err := p.name()
		return err
	case t.kind == 0:
		return fmt.Errorf("Syntax Error: Unexpected <EOF>.")
	}
	return nil
}

func (p *parser) directives() error {
	_, err := p.parseDirectives()
	return err
}

// parseDirectives parses the applied directives at the current position.
func (p *parser) parseDirectives() ([]*Directive, error) {
	var out []*Directive
	for p.is('p', "@") {
		p.take()
		n, err := p.name()
		if err != nil {
			return nil, err
		}
		d := &Directive{Name: n}
		if p.is('p', "(") {
			p.take()
			for !p.is('p', ")") {
				if p.peek().kind == 0 {
					return nil, fmt.Errorf("Syntax Error: Expected Name, found <EOF>.")
				}
				an, err := p.name()
				if err != nil {
					return nil, err
				}
				if err := p.expect('p', ":"); err != nil {
					return nil, err
				}
				v, err := p.value()
				if err != nil {
					return nil, err
				}
				d.Arguments = append(d.Arguments, &Argument{Name: an, Value: v})
			}
			p.take()
		}
		out = append(out, d)
	}
	return out, nil
}

// value parses a value.
func (p *parser) value() (*Value, error) {
	t := p.take()
	switch t.kind {
	case 's':
		return &Value{Kind: ValueString, Value: t.value}, nil
	case 'v':
		if strings.ContainsAny(t.value, ".eE") {
			return &Value{Kind: ValueFloat, Value: t.value}, nil
		}
		return &Value{Kind: ValueInt, Value: t.value}, nil
	case 'n':
		switch t.value {
		case "true", "false":
			return &Value{Kind: ValueBoolean, Value: t.value}, nil
		case "null":
			return &Value{Kind: ValueNull}, nil
		}
		return &Value{Kind: ValueEnum, Value: t.value}, nil
	case 'p':
		switch t.value {
		case "$":
			n, err := p.name()
			return &Value{Kind: ValueVariable, Value: n}, err
		case "[":
			v := &Value{Kind: ValueList}
			for !p.is('p', "]") {
				if p.peek().kind == 0 {
					return nil, fmt.Errorf("Syntax Error: Expected \"]\".")
				}
				item, err := p.value()
				if err != nil {
					return nil, err
				}
				v.Values = append(v.Values, item)
			}
			p.take()
			return v, nil
		case "{":
			// object values are kept as a whole (their fields are not
			// needed)
			p.i--
			if err := p.skipValue(); err != nil {
				return nil, err
			}
			return &Value{Kind: ValueObject}, nil
		}
	case 0:
		return nil, fmt.Errorf("Syntax Error: Unexpected <EOF>.")
	}
	return nil, fmt.Errorf("Syntax Error: Unexpected %q.", t.value)
}

func (p *parser) typeRef() (*TypeRef, error) {
	var t *TypeRef
	if p.is('p', "[") {
		p.take()
		of, err := p.typeRef()
		if err != nil {
			return nil, err
		}
		if err := p.expect('p', "]"); err != nil {
			return nil, err
		}
		t = &TypeRef{List: true, OfType: of}
	} else {
		n, err := p.name()
		if err != nil {
			return nil, err
		}
		t = &TypeRef{Name: n}
	}
	if p.is('p', "!") {
		p.take()
		t = &TypeRef{NonNull: true, OfType: t}
	}
	return t, nil
}

func (p *parser) fields(input bool) ([]*FieldDefinition, error) {
	if !p.is('p', "{") {
		return nil, nil
	}
	p.take()
	var out []*FieldDefinition
	for !p.is('p', "}") {
		if p.peek().kind == 0 {
			return nil, fmt.Errorf("Syntax Error: Expected Name, found <EOF>.")
		}
		p.skipDescription()
		n, err := p.name()
		if err != nil {
			return nil, err
		}
		if p.is('p', "(") {
			if err := p.skipValue(); err != nil {
				return nil, err
			}
		}
		if err := p.expect('p', ":"); err != nil {
			return nil, err
		}
		t, err := p.typeRef()
		if err != nil {
			return nil, err
		}
		if input && p.is('p', "=") {
			p.take()
			if err := p.skipValue(); err != nil {
				return nil, err
			}
		}
		if err := p.directives(); err != nil {
			return nil, err
		}
		out = append(out, &FieldDefinition{Name: n, Type: t})
	}
	p.take()
	return out, nil
}

func (p *parser) definition() (*Definition, error) {
	p.skipDescription()
	extension := false
	if p.is('n', "extend") {
		p.take()
		extension = true
	}
	kw, err := p.name()
	if err != nil {
		return nil, err
	}
	def := &Definition{Extension: extension}
	switch kw {
	case "schema":
		def.Kind = KindSchema
		if err := p.directives(); err != nil {
			return nil, err
		}
		if p.is('p', "{") {
			if err := p.skipValue(); err != nil {
				return nil, err
			}
		}
		return def, nil
	case "scalar":
		def.Kind = KindScalar
		if def.Name, err = p.name(); err != nil {
			return nil, err
		}
		return def, p.directives()
	case "type", "interface", "input":
		def.Kind = map[string]string{"type": KindObject, "interface": KindInterface, "input": KindInput}[kw]
		if def.Name, err = p.name(); err != nil {
			return nil, err
		}
		if p.is('n', "implements") {
			p.take()
			if p.is('p', "&") {
				p.take()
			}
			for p.peek().kind == 'n' {
				def.Interfaces = append(def.Interfaces, p.take().value)
				if !p.is('p', "&") {
					break
				}
				p.take()
			}
		}
		if def.Directives, err = p.parseDirectives(); err != nil {
			return nil, err
		}
		def.Fields, err = p.fields(kw == "input")
		return def, err
	case "union":
		def.Kind = KindUnion
		if def.Name, err = p.name(); err != nil {
			return nil, err
		}
		if err := p.directives(); err != nil {
			return nil, err
		}
		if p.is('p', "=") {
			p.take()
			if p.is('p', "|") {
				p.take()
			}
			for p.peek().kind == 'n' {
				def.Types = append(def.Types, p.take().value)
				if !p.is('p', "|") {
					break
				}
				p.take()
			}
		}
		return def, nil
	case "enum":
		def.Kind = KindEnum
		if def.Name, err = p.name(); err != nil {
			return nil, err
		}
		if err := p.directives(); err != nil {
			return nil, err
		}
		if p.is('p', "{") {
			p.take()
			for !p.is('p', "}") {
				if p.peek().kind == 0 {
					return nil, fmt.Errorf("Syntax Error: Expected Name, found <EOF>.")
				}
				p.skipDescription()
				v, err := p.name()
				if err != nil {
					return nil, err
				}
				def.Values = append(def.Values, v)
				if err := p.directives(); err != nil {
					return nil, err
				}
			}
			p.take()
		}
		return def, nil
	case "directive":
		def.Kind = KindDirective
		if err := p.expect('p', "@"); err != nil {
			return nil, err
		}
		if def.Name, err = p.name(); err != nil {
			return nil, err
		}
		if p.is('p', "(") {
			if err := p.skipValue(); err != nil {
				return nil, err
			}
		}
		if p.is('n', "repeatable") {
			p.take()
		}
		if err := p.expect('n', "on"); err != nil {
			return nil, err
		}
		if p.is('p', "|") {
			p.take()
		}
		for p.peek().kind == 'n' {
			p.take()
			if !p.is('p', "|") {
				break
			}
			p.take()
		}
		return def, nil
	}
	return nil, fmt.Errorf("Syntax Error: Unexpected Name %q.", kw)
}
