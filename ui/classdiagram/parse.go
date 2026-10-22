package classdiagram

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

type direction uint8

const (
	down direction = iota
	up
	right
	left
)

var directions = map[string]direction{"TB": down, "BT": up, "LR": right, "RL": left}

// end is what a relation ends in at a class.
type end uint8

const (
	bare end = iota
	arrow
	triangle
	filled
	hollow
)

// member is a field or a method as it shows: head is its visibility and
// name, with a method's parameters, and tail its type after ": ", "" for
// none.
type member struct {
	head, tail       string
	static, abstract bool
}

type problem struct {
	line int
	msg  string
}

// skipped are the statements that style a diagram or link it elsewhere,
// which a view does not show.
var skipped = []string{"style", "classDef", "cssClass", "click", "link", "callback"}

func idRune(r rune) bool { return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) }

// idLen returns how long the class's name at the start of s is. A dash
// joins words, as in api-gateway, but does not start a relation.
func idLen(s string) int {
	n := 0
	for n < len(s) {
		r, w := utf8.DecodeRuneInString(s[n:])
		if r == '-' && n > 0 {
			next, _ := utf8.DecodeRuneInString(s[n+w:])
			if idRune(next) {
				n += w
				continue
			}
		}
		if !idRune(r) {
			break
		}
		n += w
	}
	return n
}

// snippet returns the start of s, to name it in a message.
func snippet(s string) string {
	if f := strings.Fields(s); len(f) > 0 {
		return f[0]
	}
	return s
}

// generics returns s with each pair of ~ around type parameters as < and
// >, as List~T~ shows as List<T>; ok is false when a ~ has no pair. A ~
// closes when one is open and no name follows it, so that List~List~T~~
// nests.
func generics(s string) (string, bool) {
	if !strings.Contains(s, "~") {
		return s, true
	}
	var b strings.Builder
	depth := 0
	for i, r := range s {
		if r != '~' {
			b.WriteRune(r)
			continue
		}
		next, _ := utf8.DecodeRuneInString(s[i+1:])
		if depth > 0 && !idRune(next) {
			b.WriteByte('>')
			depth--
		} else {
			b.WriteByte('<')
			depth++
		}
	}
	return b.String(), depth == 0
}

// readMember reads a member, as a class's body or "A : member" gives it:
// a method when it has parentheses, else a field.
func readMember(s string) (m member, method bool, err string) {
	text := s
	switch {
	case strings.HasSuffix(s, "$"):
		m.static, s = true, strings.TrimSpace(s[:len(s)-1])
	case strings.HasSuffix(s, "*"):
		m.abstract, s = true, strings.TrimSpace(s[:len(s)-1])
	}
	if s == "" {
		return m, false, "the member is empty"
	}
	vis := ""
	// A ~ first is package visibility unless it opens type parameters,
	// which pair it with another.
	if strings.ContainsRune("+-#", rune(s[0])) || s[0] == '~' && strings.Count(s, "~")%2 == 1 {
		vis, s = s[:1], strings.TrimSpace(s[1:])
	}
	g, ok := generics(s)
	if !ok {
		return m, false, fmt.Sprintf("a ~ in %q has no pair", text)
	}
	s = g
	if open := strings.IndexByte(s, '('); open >= 0 {
		name := strings.TrimSpace(s[:open])
		closing, depth := -1, 0
		for i := open; i < len(s) && closing < 0; i++ {
			switch s[i] {
			case '(':
				depth++
			case ')':
				if depth--; depth == 0 {
					closing = i
				}
			}
		}
		switch {
		case name == "":
			return m, true, fmt.Sprintf("the method %q has no name", text)
		case closing < 0:
			return m, true, fmt.Sprintf("the method %s has no )", name)
		}
		m.head = vis + name + "(" + strings.TrimSpace(s[open+1:closing]) + ")"
		ret := strings.TrimSpace(s[closing+1:])
		if ret = strings.TrimSpace(strings.TrimPrefix(ret, ":")); ret != "" {
			m.tail = ": " + ret
		}
		return m, true, ""
	}
	name, typ := s, ""
	if a, b, ok := strings.Cut(s, ":"); ok {
		name, typ = strings.TrimSpace(a), strings.TrimSpace(b)
	} else if i := strings.LastIndexAny(s, " \t"); i >= 0 {
		typ, name = strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:])
	}
	if name == "" {
		return m, false, fmt.Sprintf("the field %q has no name", text)
	}
	m.head = vis + name
	if typ != "" {
		m.tail = ": " + typ
	}
	return m, false, ""
}

// readRelation reads the relation at the start of s, as <|-- or ..>,
// and returns the rest; ok is false when s does not start with one.
func readRelation(s string) (ends [2]end, dashed bool, rest string, ok bool) {
	i := 0
	switch {
	case strings.HasPrefix(s, "<|"):
		ends[0], i = triangle, 2
	case strings.HasPrefix(s, "*"):
		ends[0], i = filled, 1
	case strings.HasPrefix(s, "o"):
		ends[0], i = hollow, 1
	case strings.HasPrefix(s, "<"):
		ends[0], i = arrow, 1
	}
	switch {
	case strings.HasPrefix(s[i:], "--"):
	case strings.HasPrefix(s[i:], ".."):
		dashed = true
	default:
		return ends, false, s, false
	}
	i += 2
	switch rest := s[i:]; {
	case strings.HasPrefix(rest, "|>"):
		ends[1], i = triangle, i+2
	case strings.HasPrefix(rest, "*"):
		ends[1], i = filled, i+1
	case strings.HasPrefix(rest, "o") && idLen(rest[1:]) == 0:
		ends[1], i = hollow, i+1
	case strings.HasPrefix(rest, ">"):
		ends[1], i = arrow, i+1
	}
	if rest = s[i:]; strings.HasPrefix(rest, "-") || strings.HasPrefix(rest, ".") {
		return ends, false, s, false
	}
	return ends, dashed, rest, true
}

// quoted reads the text in quotes at the start of s, a cardinality, and
// returns the rest; ok is false when s does not start with a quote.
func quoted(s string) (text, rest string, ok bool, err string) {
	after, ok := strings.CutPrefix(s, `"`)
	if !ok {
		return "", s, false, ""
	}
	i := strings.IndexByte(after, '"')
	if i < 0 {
		return "", s, false, "the cardinality has no closing quote"
	}
	return after[:i], strings.TrimLeft(after[i+1:], " \t"), true, ""
}

// parser reads a diagram's statements in turn.
type parser struct {
	d   *diagram
	ids map[string]int
}

// class returns the class of id, making it if it is new.
func (p *parser) class(id string) int {
	if i, ok := p.ids[id]; ok {
		return i
	}
	p.ids[id] = len(p.d.classes)
	p.d.classes = append(p.d.classes, class{id: id, name: id})
	return len(p.d.classes) - 1
}

// add adds m to class i, as a method or a field.
func (p *parser) add(i int, m member, method bool) {
	cl := &p.d.classes[i]
	if method {
		cl.methods = append(cl.methods, m)
	} else {
		cl.fields = append(cl.fields, m)
	}
}

// declare reads "class A", "class A~T~ {" or "class A {}" after the word
// class, and returns the class it opens the body of, -1 for none.
func (p *parser) declare(s string) (int, string) {
	n := idLen(s)
	if s == "" {
		return -1, "the class has no name"
	}
	if n == 0 {
		return -1, fmt.Sprintf("%q is not a class's name", snippet(s))
	}
	id, rest := s[:n], s[n:]
	tok := rest
	if i := strings.IndexAny(rest, " \t{"); i >= 0 {
		tok = rest[:i]
	}
	if tok != "" && !strings.HasPrefix(tok, "~") {
		return -1, fmt.Sprintf("%q is not a class's name", id+tok)
	}
	name, ok := generics(id + tok)
	if !ok {
		return -1, fmt.Sprintf("a ~ in %q has no pair", id+tok)
	}
	body := -1
	switch rest = strings.Join(strings.Fields(rest[len(tok):]), ""); rest {
	case "", "{}":
	case "{":
		body = 0
	default:
		return -1, fmt.Sprintf("%q after class %s is not {", rest, id)
	}
	i := p.class(id)
	p.d.classes[i].name = name
	if body == 0 {
		body = i
	}
	return body, ""
}

// annotate reads "<<interface>> A", or "<<interface>>" in the body of
// class in, and annotates the class.
func (p *parser) annotate(s string, in int) string {
	j := strings.Index(s, ">>")
	if j < 0 {
		return "the annotation has no >>"
	}
	note, rest := strings.TrimSpace(s[2:j]), strings.TrimSpace(s[j+2:])
	switch {
	case in >= 0 && rest != "":
		return fmt.Sprintf("%q after the annotation: it takes a line of its own", rest)
	case in >= 0:
	case rest == "":
		return "the annotation names no class"
	case idLen(rest) != len(rest):
		return fmt.Sprintf("%q is not a class's name", rest)
	default:
		in = p.class(rest)
	}
	p.d.classes[in].note = note
	return ""
}

// statement reads "A : member" or a relation, as A "1" --> "*" B : text,
// adding it whole or, when it is wrong, not at all.
func (p *parser) statement(line int, s string) string {
	n := idLen(s)
	if n == 0 {
		return fmt.Sprintf("%q is not a class, a member or a relation", snippet(s))
	}
	a := s[:n]
	rest := strings.TrimLeft(s[n:], " \t")
	if after, ok := strings.CutPrefix(rest, ":"); ok {
		m, method, err := readMember(strings.TrimSpace(after))
		if err != "" {
			return err
		}
		p.add(p.class(a), m, method)
		return ""
	}
	if rest == "" {
		return fmt.Sprintf("%s alone is not a statement: declare it as class %s", a, a)
	}
	r := relation{line: line}
	card, rest, _, err := quoted(rest)
	if err != "" {
		return err
	}
	r.cards[0] = card
	ends, dashed, rest, ok := readRelation(rest)
	if !ok {
		return fmt.Sprintf("%q after %s is not a relation, as <|--, *--, o--, -->, --, ..>, ..|> or ..", snippet(rest), a)
	}
	r.ends, r.dashed = ends, dashed
	rest = strings.TrimLeft(rest, " \t")
	if card, rest, _, err = quoted(rest); err != "" {
		return err
	}
	r.cards[1] = card
	m := idLen(rest)
	if m == 0 {
		if rest == "" {
			return "the relation after " + a + " goes to no class"
		}
		return fmt.Sprintf("%q is not a class's name", snippet(rest))
	}
	b := rest[:m]
	if rest = strings.TrimSpace(rest[m:]); rest != "" {
		label, ok := strings.CutPrefix(rest, ":")
		if !ok {
			return fmt.Sprintf("%q after %s is not a label, as : text", snippet(rest), b)
		}
		r.label = strings.TrimSpace(label)
	}
	r.from, r.to = p.class(a), p.class(b)
	p.d.rels = append(p.d.rels, r)
	return ""
}

// parse reads the statements of src, skipping those that are wrong.
func parse(src string) *diagram {
	d := &diagram{}
	p := &parser{d: d, ids: map[string]int{}}
	fail := func(line int, format string, args ...any) {
		d.errs = append(d.errs, problem{line, fmt.Sprintf(format, args...)})
	}
	body, opened := -1, 0
	for n, raw := range strings.Split(src, "\n") {
		line := n + 1
		s := strings.TrimSpace(raw)
		if s == "" || strings.HasPrefix(s, "%%") {
			continue
		}
		if body >= 0 {
			switch {
			case s == "}":
				body = -1
			case strings.HasPrefix(s, "<<"):
				if err := p.annotate(s, body); err != "" {
					fail(line, "%s", err)
				}
			default:
				m, method, err := readMember(s)
				if err != "" {
					fail(line, "%s", err)
					continue
				}
				p.add(body, m, method)
			}
			continue
		}
		word, rest := s, ""
		if i := strings.IndexFunc(s, unicode.IsSpace); i >= 0 {
			word, rest = s[:i], strings.TrimSpace(s[i:])
		}
		switch {
		case s == "classDiagram" || s == "classDiagram-v2":
		case word == "direction":
			dir, ok := directions[strings.ToUpper(rest)]
			if !ok {
				fail(line, "%q is not a direction: TB, BT, LR or RL", rest)
				continue
			}
			d.dir = dir
		case strings.HasPrefix(s, "<<"):
			if err := p.annotate(s, -1); err != "" {
				fail(line, "%s", err)
			}
		case word == "class":
			i, err := p.declare(rest)
			if err != "" {
				fail(line, "%s", err)
				continue
			}
			body, opened = i, line
		case s == "}":
			fail(line, "} closes no class")
		case slices.Contains(skipped, word):
		default:
			if err := p.statement(line, s); err != "" {
				fail(line, "%s", err)
			}
		}
	}
	if body >= 0 {
		fail(opened, "the class %s has no }", d.classes[body].id)
	}
	for k, r := range d.rels {
		if r.from == r.to {
			d.classes[r.from].loops = append(d.classes[r.from].loops, k)
		}
	}
	slices.SortStableFunc(d.errs, func(a, b problem) int { return cmp.Compare(a.line, b.line) })
	return d
}
