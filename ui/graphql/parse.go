package graphql

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Schema is a GraphQL schema: its types in the order they were written,
// and the names of its root types.
type Schema struct {
	Query, Mutation, Subscription string
	Types                         []*Type
	byName                        map[string]*Type
}

// Type is a named type. Kind is "type", "interface", "input", "enum",
// "union" or "scalar".
type Type struct {
	Name, Kind, Description string
	Interfaces              []string
	Fields                  []Field
	Values                  []Value
	Members                 []string
}

// Field is a field of an object, an interface or an input. Type is as
// written, such as "[User!]!".
type Field struct {
	Name, Type, Description string
	Args                    []Arg
	Default                 string
	// Deprecated is why the field is deprecated, empty if it isn't.
	Deprecated string
}

// Arg is an argument of a field.
type Arg struct {
	Name, Type, Default, Description string
}

// Value is a value of an enum.
type Value struct {
	Name, Description, Deprecated string
}

// Lookup returns the type named name, or nil.
func (s *Schema) Lookup(name string) *Type { return s.byName[name] }

// Named returns the name of the type of a field or an argument without
// its lists and non-nulls: "User" of "[User!]!".
func Named(typ string) string { return strings.Trim(typ, "[]! ") }

// Parse reads a schema written in GraphQL's schema definition language.
// Directives other than @deprecated are left out.
func Parse(sdl string) (*Schema, error) {
	toks, err := lex(sdl)
	if err != nil {
		return nil, err
	}
	p := &parser{toks: toks, s: &Schema{byName: map[string]*Type{}}}
	for !p.done() {
		if err := p.definition(); err != nil {
			return nil, err
		}
	}
	for _, root := range []struct {
		name string
		to   *string
	}{{"Query", &p.s.Query}, {"Mutation", &p.s.Mutation}, {"Subscription", &p.s.Subscription}} {
		if *root.to == "" && p.s.byName[root.name] != nil {
			*root.to = root.name
		}
	}
	return p.s, nil
}

type token struct {
	kind byte // 'n' name, 's' string, 'p' punctuation, 'v' number
	text string
	line int
}

func lex(src string) ([]token, error) {
	var out []token
	line := 1
	src = strings.TrimPrefix(src, "\ufeff")
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case c == '\n':
			line++
			i++
		case c == ' ' || c == '\t' || c == '\r' || c == ',':
			i++
		case c == '#':
			for i < len(src) && src[i] != '\n' {
				i++
			}
		case strings.HasPrefix(src[i:], `"""`):
			end := strings.Index(src[i+3:], `"""`)
			for end >= 0 && src[i+3+end-1] == '\\' {
				next := strings.Index(src[i+3+end+3:], `"""`)
				if next < 0 {
					end = -1
					break
				}
				end += 3 + next
			}
			if end < 0 {
				return nil, fmt.Errorf("graphql: line %d: block string not closed", line)
			}
			raw := src[i+3 : i+3+end]
			out = append(out, token{'s', block(strings.ReplaceAll(raw, `\"""`, `"""`)), line})
			line += strings.Count(raw, "\n")
			i += 3 + end + 3
		case c == '"':
			j := i + 1
			for j < len(src) && src[j] != '"' && src[j] != '\n' {
				if src[j] == '\\' {
					j++
				}
				j++
			}
			if j >= len(src) || src[j] != '"' {
				return nil, fmt.Errorf("graphql: line %d: string not closed", line)
			}
			s, err := strconv.Unquote(strings.ReplaceAll(src[i:j+1], `\/`, "/"))
			if err != nil {
				return nil, fmt.Errorf("graphql: line %d: %w", line, err)
			}
			out = append(out, token{'s', s, line})
			i = j + 1
		case strings.ContainsRune("!$&()=:@[]{}|", rune(c)):
			out = append(out, token{'p', string(c), line})
			i++
		case c == '_' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z':
			j := i
			for j < len(src) && (src[j] == '_' || src[j] >= 'A' && src[j] <= 'Z' || src[j] >= 'a' && src[j] <= 'z' || src[j] >= '0' && src[j] <= '9') {
				j++
			}
			out = append(out, token{'n', src[i:j], line})
			i = j
		case c == '-' || c >= '0' && c <= '9':
			j := i + 1
			for j < len(src) && strings.IndexByte("0123456789.eE+-", src[j]) >= 0 {
				j++
			}
			out = append(out, token{'v', src[i:j], line})
			i = j
		default:
			return nil, fmt.Errorf("graphql: line %d: unexpected %q", line, c)
		}
	}
	return out, nil
}

// block returns a block string's value: its lines without the indent
// they share past the first, and without blank lines around them.
func block(raw string) string {
	lines := strings.Split(raw, "\n")
	indent := -1
	for _, l := range lines[1:] {
		if strings.TrimSpace(l) == "" {
			continue
		}
		n := len(l) - len(strings.TrimLeft(l, " \t"))
		if indent < 0 || n < indent {
			indent = n
		}
	}
	for i := 1; i < len(lines) && indent > 0; i++ {
		if len(lines[i]) >= indent {
			lines[i] = lines[i][indent:]
		}
	}
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

type parser struct {
	toks []token
	at   int
	s    *Schema
}

func (p *parser) done() bool { return p.at >= len(p.toks) }

func (p *parser) peek() token {
	if p.done() {
		return token{}
	}
	return p.toks[p.at]
}

func (p *parser) is(text string) bool {
	t := p.peek()
	return t.kind != 's' && t.text == text
}

func (p *parser) errorf(format string, args ...any) error {
	line := 0
	if p.done() {
		if len(p.toks) > 0 {
			line = p.toks[len(p.toks)-1].line
		}
	} else {
		line = p.peek().line
	}
	return fmt.Errorf("graphql: line %d: %s", line, fmt.Sprintf(format, args...))
}

func (p *parser) expect(text string) error {
	if !p.is(text) {
		return p.errorf("want %q, got %q", text, p.peek().text)
	}
	p.at++
	return nil
}

func (p *parser) name() (string, error) {
	t := p.peek()
	if t.kind != 'n' {
		return "", p.errorf("want a name, got %q", t.text)
	}
	p.at++
	return t.text, nil
}

func (p *parser) description() string {
	if t := p.peek(); t.kind == 's' {
		p.at++
		return t.text
	}
	return ""
}

// keywords start definitions, which end a list of interfaces written
// without braces after it.
var keywords = map[string]bool{
	"type": true, "interface": true, "input": true, "enum": true, "union": true,
	"scalar": true, "schema": true, "extend": true, "directive": true,
}

var errEOF = errors.New("graphql: schema cut short")

func (p *parser) typeRef() (string, error) {
	var b strings.Builder
	if p.is("[") {
		p.at++
		inner, err := p.typeRef()
		if err != nil {
			return "", err
		}
		if err := p.expect("]"); err != nil {
			return "", err
		}
		b.WriteString("[" + inner + "]")
	} else {
		n, err := p.name()
		if err != nil {
			return "", err
		}
		b.WriteString(n)
	}
	if p.is("!") {
		p.at++
		b.WriteString("!")
	}
	return b.String(), nil
}

// value returns a value as written.
func (p *parser) value() (string, error) {
	if p.done() {
		return "", errEOF
	}
	t := p.toks[p.at]
	switch {
	case t.kind == 's':
		p.at++
		return strconv.Quote(t.text), nil
	case t.kind == 'n' || t.kind == 'v':
		p.at++
		return t.text, nil
	case t.text == "$":
		p.at++
		n, err := p.name()
		return "$" + n, err
	case t.text == "[" || t.text == "{":
		end := map[string]string{"[": "]", "{": "}"}[t.text]
		p.at++
		var parts []string
		for !p.is(end) {
			if p.done() {
				return "", errEOF
			}
			prefix := ""
			if t.text == "{" {
				n, err := p.name()
				if err != nil {
					return "", err
				}
				if err := p.expect(":"); err != nil {
					return "", err
				}
				prefix = n + ": "
			}
			v, err := p.value()
			if err != nil {
				return "", err
			}
			parts = append(parts, prefix+v)
		}
		p.at++
		return t.text + strings.Join(parts, ", ") + end, nil
	}
	return "", p.errorf("unexpected %q", t.text)
}

// directives skips directives, returning the reason of @deprecated.
func (p *parser) directives() (deprecated string, err error) {
	for p.is("@") {
		p.at++
		name, err := p.name()
		if err != nil {
			return "", err
		}
		if name == "deprecated" {
			deprecated = "No longer supported"
		}
		if !p.is("(") {
			continue
		}
		p.at++
		for !p.is(")") {
			arg, err := p.name()
			if err != nil {
				return "", err
			}
			if err := p.expect(":"); err != nil {
				return "", err
			}
			v, err := p.value()
			if err != nil {
				return "", err
			}
			if name == "deprecated" && arg == "reason" {
				if s, err := strconv.Unquote(v); err == nil {
					deprecated = s
				}
			}
		}
		p.at++
	}
	return deprecated, nil
}

func (p *parser) args() ([]Arg, error) {
	if !p.is("(") {
		return nil, nil
	}
	p.at++
	var out []Arg
	for !p.is(")") {
		if p.done() {
			return nil, errEOF
		}
		a := Arg{Description: p.description()}
		var err error
		if a.Name, err = p.name(); err != nil {
			return nil, err
		}
		if err := p.expect(":"); err != nil {
			return nil, err
		}
		if a.Type, err = p.typeRef(); err != nil {
			return nil, err
		}
		if p.is("=") {
			p.at++
			if a.Default, err = p.value(); err != nil {
				return nil, err
			}
		}
		if _, err := p.directives(); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	p.at++
	return out, nil
}

func (p *parser) fields() ([]Field, error) {
	if !p.is("{") {
		return nil, nil
	}
	p.at++
	var out []Field
	for !p.is("}") {
		if p.done() {
			return nil, errEOF
		}
		f := Field{Description: p.description()}
		var err error
		if f.Name, err = p.name(); err != nil {
			return nil, err
		}
		if f.Args, err = p.args(); err != nil {
			return nil, err
		}
		if err := p.expect(":"); err != nil {
			return nil, err
		}
		if f.Type, err = p.typeRef(); err != nil {
			return nil, err
		}
		if p.is("=") {
			p.at++
			if f.Default, err = p.value(); err != nil {
				return nil, err
			}
		}
		if f.Deprecated, err = p.directives(); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	p.at++
	return out, nil
}

func (p *parser) definition() error {
	desc := p.description()
	kw, err := p.name()
	if err != nil {
		return err
	}
	extend := kw == "extend"
	if extend {
		if kw, err = p.name(); err != nil {
			return err
		}
	}
	switch kw {
	case "schema":
		if _, err := p.directives(); err != nil {
			return err
		}
		if !p.is("{") {
			return nil
		}
		p.at++
		for !p.is("}") {
			op, err := p.name()
			if err != nil {
				return err
			}
			if err := p.expect(":"); err != nil {
				return err
			}
			n, err := p.name()
			if err != nil {
				return err
			}
			switch op {
			case "query":
				p.s.Query = n
			case "mutation":
				p.s.Mutation = n
			case "subscription":
				p.s.Subscription = n
			}
		}
		p.at++
		return nil
	case "directive":
		if err := p.expect("@"); err != nil {
			return err
		}
		if _, err := p.name(); err != nil {
			return err
		}
		if _, err := p.args(); err != nil {
			return err
		}
		if p.is("repeatable") {
			p.at++
		}
		if err := p.expect("on"); err != nil {
			return err
		}
		if p.is("|") {
			p.at++
		}
		for {
			if _, err := p.name(); err != nil {
				return err
			}
			if !p.is("|") {
				return nil
			}
			p.at++
		}
	case "scalar", "type", "interface", "input", "enum", "union":
	default:
		return p.errorf("unexpected %q", kw)
	}
	name, err := p.name()
	if err != nil {
		return err
	}
	t := p.s.byName[name]
	if t == nil {
		t = &Type{Name: name, Kind: kw}
		p.s.byName[name] = t
		p.s.Types = append(p.s.Types, t)
	}
	if desc != "" && !extend {
		t.Description = desc
	}
	if p.is("implements") {
		p.at++
		if p.is("&") {
			p.at++
		}
		for p.peek().kind == 'n' && !keywords[p.peek().text] {
			n, _ := p.name()
			t.Interfaces = append(t.Interfaces, n)
			if p.is("&") {
				p.at++
			}
		}
	}
	if _, err := p.directives(); err != nil {
		return err
	}
	switch kw {
	case "type", "interface", "input":
		fs, err := p.fields()
		if err != nil {
			return err
		}
		t.Fields = append(t.Fields, fs...)
	case "enum":
		if !p.is("{") {
			return nil
		}
		p.at++
		for !p.is("}") {
			if p.done() {
				return errEOF
			}
			v := Value{Description: p.description()}
			if v.Name, err = p.name(); err != nil {
				return err
			}
			if v.Deprecated, err = p.directives(); err != nil {
				return err
			}
			t.Values = append(t.Values, v)
		}
		p.at++
	case "union":
		if !p.is("=") {
			return nil
		}
		p.at++
		if p.is("|") {
			p.at++
		}
		for {
			n, err := p.name()
			if err != nil {
				return err
			}
			t.Members = append(t.Members, n)
			if !p.is("|") {
				return nil
			}
			p.at++
		}
	}
	return nil
}
