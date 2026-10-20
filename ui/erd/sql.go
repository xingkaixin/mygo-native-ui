package erd

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Error is a mistake in the source: the line it is on, from 1, and what
// is wrong.
type Error struct {
	Line int
	Msg  string
}

func (e Error) Error() string { return fmt.Sprintf("line %d: %s", e.Line, e.Msg) }

// Parse reads the CREATE TABLE statements of src, as PostgreSQL and
// SQLite write them, and returns their tables in order, and what is wrong
// in src by line. It reads columns with their types and arguments, as
// varchar(255) and numeric(10,2), PRIMARY KEY, NOT NULL, NULL and
// REFERENCES t(col), or REFERENCES t for its primary key, and the table
// constraints PRIMARY KEY (a, b) and FOREIGN KEY (a) REFERENCES t(b),
// named or not. It skips UNIQUE, DEFAULT and CHECK, and the other
// statements. Names may be quoted, as "x" and `x`, and schema-qualified,
// which leaves the schema out. A statement with a mistake is left out,
// and a reference to a table or column not in src.
func Parse(src string) ([]Table, []Error) {
	toks, lexErr := lex(src)
	if lexErr != nil {
		// The statement the error is in ends early: leave it out.
		for len(toks) > 0 && !toks[len(toks)-1].punct(";") {
			toks = toks[:len(toks)-1]
		}
	}
	var tables []Table
	var refs []ref
	var errs []Error
	for len(toks) > 0 {
		stmt, end := toks, toks[len(toks)-1].line
		toks = nil
		if n := slices.IndexFunc(stmt, func(t token) bool { return t.punct(";") }); n >= 0 {
			stmt, end, toks = stmt[:n], stmt[n].line, stmt[n+1:]
		}
		if len(stmt) < 2 || !stmt[0].is("create") || !stmt[1].is("table") {
			continue
		}
		p := &parser{toks: stmt[2:], end: end}
		t, rs, err := p.table()
		switch {
		case err != nil:
			errs = append(errs, *err)
		case slices.ContainsFunc(tables, func(o Table) bool { return strings.EqualFold(o.Name, t.Name) }):
			errs = append(errs, Error{stmt[0].line, fmt.Sprintf("table %q exists already", t.Name)})
		default:
			for i := range rs {
				rs[i].table = len(tables)
			}
			tables, refs = append(tables, t), append(refs, rs...)
		}
	}
	if lexErr != nil {
		errs = append(errs, *lexErr)
	}
	errs = append(errs, resolve(tables, refs)...)
	slices.SortStableFunc(errs, func(a, b Error) int { return cmp.Compare(a.Line, b.Line) })
	return tables, errs
}

// ref is a foreign key: columns cols of table table reference the columns
// to of table into, or its primary key without them.
type ref struct {
	table int
	cols  []int
	into  string
	to    []string
	line  int
}

// resolve sets the columns refs reference, and returns the references to
// what is not in tables.
func resolve(tables []Table, refs []ref) []Error {
	var errs []Error
	find := func(name string, n int, of func(int) string) int {
		for i := range n {
			if strings.EqualFold(of(i), name) {
				return i
			}
		}
		return -1
	}
next:
	for _, r := range refs {
		j := find(r.into, len(tables), func(i int) string { return tables[i].Name })
		if j < 0 {
			errs = append(errs, Error{r.line, fmt.Sprintf("no table %q", r.into)})
			continue
		}
		into := tables[j]
		to := r.to
		if to == nil {
			for _, c := range into.Columns {
				if c.Key {
					to = append(to, c.Name)
				}
			}
			if to == nil {
				errs = append(errs, Error{r.line, fmt.Sprintf("%q has no primary key", into.Name)})
				continue
			}
		}
		if len(to) != len(r.cols) {
			errs = append(errs, Error{r.line, "numbers of referencing and referenced columns differ"})
			continue
		}
		at := make([]int, len(to))
		for k, name := range to {
			at[k] = find(name, len(into.Columns), func(i int) string { return into.Columns[i].Name })
			if at[k] < 0 {
				errs = append(errs, Error{r.line, fmt.Sprintf("%q has no column %q", into.Name, name)})
				continue next
			}
		}
		for k, i := range r.cols {
			col := &tables[r.table].Columns[i]
			col.RefTable, col.RefColumn = into.Name, into.Columns[at[k]].Name
		}
	}
	return errs
}

// token is a word, a name in quotes, a string, or a mark of punctuation,
// and the line it starts on.
type token struct {
	text string
	kind byte // 'w' word, 'q' quoted name, 's' string, 'p' punctuation
	line int
}

func (t token) is(word string) bool { return t.kind == 'w' && strings.EqualFold(t.text, word) }

func (t token) punct(s string) bool { return t.kind == 'p' && t.text == s }

// lex splits src into tokens, leaving out spaces and comments, up to a
// string, quoted name or comment not closed, which it returns as an
// error.
func lex(src string) ([]token, *Error) {
	var toks []token
	line := 1
	for i := 0; i < len(src); {
		r, n := utf8.DecodeRuneInString(src[i:])
		switch {
		case r == '\n':
			line++
			i++
		case unicode.IsSpace(r):
			i += n
		case strings.HasPrefix(src[i:], "--"):
			for i < len(src) && src[i] != '\n' {
				i++
			}
		case strings.HasPrefix(src[i:], "/*"):
			end := strings.Index(src[i+2:], "*/")
			if end < 0 {
				return toks, &Error{line, "comment not closed"}
			}
			line += strings.Count(src[i:i+2+end], "\n")
			i += end + 4
		case r == '\'' || r == '"' || r == '`':
			q := src[i]
			start := line
			var b strings.Builder
			j := i + 1
			for ; j < len(src) && (src[j] != q || j+1 < len(src) && src[j+1] == q); j++ {
				// A quote doubled stands for one.
				if src[j] == q {
					j++
				}
				if src[j] == '\n' {
					line++
				}
				b.WriteByte(src[j])
			}
			kind, what := byte('q'), "name"
			if q == '\'' {
				kind, what = 's', "string"
			}
			if j >= len(src) {
				return toks, &Error{start, what + " not closed"}
			}
			toks = append(toks, token{b.String(), kind, start})
			i = j + 1
		case wordRune(r):
			j := i
			for j < len(src) {
				r, n := utf8.DecodeRuneInString(src[j:])
				if !wordRune(r) {
					break
				}
				j += n
			}
			toks = append(toks, token{src[i:j], 'w', line})
			i = j
		default:
			toks = append(toks, token{src[i : i+n], 'p', line})
			i += n
		}
	}
	return toks, nil
}

func wordRune(r rune) bool { return r == '_' || r == '$' || unicode.IsLetter(r) || unicode.IsDigit(r) }

// stops are the words that end a column's type, and an expression after
// DEFAULT.
var stops = map[string]bool{
	"constraint": true, "primary": true, "not": true, "null": true, "unique": true,
	"default": true, "check": true, "references": true, "collate": true, "generated": true,
}

// parser reads a CREATE TABLE statement after its first two words; end
// is the line where the statement ends.
type parser struct {
	toks []token
	i    int
	end  int
}

func (p *parser) peek() token {
	if p.i < len(p.toks) {
		return p.toks[p.i]
	}
	return token{}
}

// fail stops the statement with a mistake at the token next, or at its
// end.
func (p *parser) fail(format string, args ...any) {
	line := p.end
	if p.i < len(p.toks) {
		line = p.toks[p.i].line
	}
	panic(Error{line, fmt.Sprintf(format, args...)})
}

func (p *parser) accept(word string) bool {
	if p.peek().is(word) {
		p.i++
		return true
	}
	return false
}

func (p *parser) acceptPunct(s string) bool {
	if p.peek().punct(s) {
		p.i++
		return true
	}
	return false
}

func (p *parser) want(word string) {
	if !p.accept(word) {
		p.fail("expected %s", strings.ToUpper(word))
	}
}

func (p *parser) wantPunct(s string) {
	if !p.acceptPunct(s) {
		p.fail("expected %q", s)
	}
}

func (p *parser) ident() string {
	t := p.peek()
	if t.kind != 'w' && t.kind != 'q' {
		p.fail("expected a name")
	}
	p.i++
	return t.text
}

// name reads a name, schema-qualified or not, and returns it without the
// schema.
func (p *parser) name() string {
	n := p.ident()
	for p.acceptPunct(".") {
		n = p.ident()
	}
	return n
}

// names reads names in parentheses, separated by commas.
func (p *parser) names() []string {
	p.wantPunct("(")
	var out []string
	for {
		out = append(out, p.ident())
		if !p.acceptPunct(",") {
			break
		}
	}
	p.wantPunct(")")
	return out
}

// group skips tokens in parentheses, from the opening one to the one
// that closes it, and returns them as text.
func (p *parser) group() string {
	var b strings.Builder
	depth := 0
	for prev := byte(0); ; p.i++ {
		t := p.peek()
		switch {
		case t.kind == 0:
			p.fail(`expected ")"`)
		case t.punct("("):
			depth++
		case t.punct(")"):
			depth--
		}
		if t.kind == 'w' && prev == 'w' {
			b.WriteByte(' ')
		}
		if t.kind == 's' {
			fmt.Fprintf(&b, "'%s'", t.text)
		} else {
			b.WriteString(t.text)
		}
		prev = t.kind
		if depth == 0 {
			p.i++
			return b.String()
		}
	}
}

// done reports whether the next token ends a column or a table
// constraint.
func (p *parser) done() bool {
	t := p.peek()
	return t.kind == 0 || t.punct(",") || t.punct(")")
}

// stop reports whether the next token is a word of stops.
func (p *parser) stop() bool {
	t := p.peek()
	return t.kind == 'w' && stops[strings.ToLower(t.text)]
}

// skip skips the next token, or the tokens in parentheses from it.
func (p *parser) skip() {
	if p.peek().punct("(") {
		p.group()
	} else {
		p.i++
	}
}

// table reads the name and the columns of a table, and returns the
// foreign keys it has, their table not yet set.
func (p *parser) table() (t Table, refs []ref, err *Error) {
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(Error)
			if !ok {
				panic(r)
			}
			err = &e
		}
	}()
	if p.accept("if") {
		p.want("not")
		p.want("exists")
	}
	t.Name = p.name()
	p.wantPunct("(")
	for {
		if p.accept("constraint") {
			p.ident()
		}
		switch {
		case p.accept("primary"):
			p.want("key")
			for _, i := range p.columns(&t) {
				t.Columns[i].Key, t.Columns[i].Null = true, false
			}
		case p.accept("foreign"):
			p.want("key")
			cols := p.columns(&t)
			p.want("references")
			refs = append(refs, p.reference(cols))
		case p.accept("unique"), p.accept("check"):
		default:
			if r, ok := p.column(&t); ok {
				refs = append(refs, r)
			}
		}
		for !p.done() {
			p.skip()
		}
		if !p.acceptPunct(",") {
			break
		}
	}
	if !p.acceptPunct(")") {
		p.fail(`expected "," or ")"`)
	}
	return t, refs, nil
}

// columns reads names of columns of t in parentheses and returns their
// indexes.
func (p *parser) columns(t *Table) []int {
	line := p.peek().line
	var out []int
	for _, name := range p.names() {
		i := slices.IndexFunc(t.Columns, func(c Column) bool { return strings.EqualFold(c.Name, name) })
		if i < 0 {
			panic(Error{line, fmt.Sprintf("%q has no column %q", t.Name, name)})
		}
		out = append(out, i)
	}
	return out
}

// reference reads what cols reference, after REFERENCES, and the actions
// on delete and update.
func (p *parser) reference(cols []int) ref {
	r := ref{cols: cols, line: p.peek().line}
	r.into = p.name()
	if p.peek().punct("(") {
		r.to = p.names()
	}
	for {
		switch {
		case p.accept("on"):
			// DELETE or UPDATE, then SET NULL, SET DEFAULT, NO ACTION, or a word.
			p.i++
			if !p.accept("set") {
				p.accept("no")
			}
			p.i++
		case p.accept("match"):
			p.i++
		default:
			return r
		}
	}
}

// column reads a column of t, and returns the foreign key it has, if any.
func (p *parser) column(t *Table) (r ref, fk bool) {
	line := p.peek().line
	col := Column{Name: p.ident(), Null: true}
	if slices.ContainsFunc(t.Columns, func(c Column) bool { return strings.EqualFold(c.Name, col.Name) }) {
		panic(Error{line, fmt.Sprintf("column %q appears twice", col.Name)})
	}
	var typ strings.Builder
	for !p.done() && !p.stop() {
		tk := p.peek()
		switch {
		case tk.punct("("):
			typ.WriteString(p.group())
			continue
		case tk.kind != 'p' && typ.Len() > 0:
			typ.WriteByte(' ')
		}
		typ.WriteString(tk.text)
		p.i++
	}
	col.Type = typ.String()
	for !p.done() {
		switch {
		case p.accept("constraint"):
			p.ident()
		case p.accept("primary"):
			p.want("key")
			col.Key, col.Null = true, false
		case p.accept("not"):
			p.want("null")
			col.Null = false
		case p.accept("default"):
			for first := true; !p.done() && (first || !p.stop()); first = false {
				p.skip()
			}
		case p.accept("check"):
			p.group()
		case p.accept("references"):
			r, fk = p.reference([]int{len(t.Columns)}), true
		case p.accept("collate"):
			p.name()
		default:
			// NULL, UNIQUE, and what is not read, as AUTOINCREMENT and
			// GENERATED ALWAYS AS IDENTITY.
			p.skip()
		}
	}
	t.Columns = append(t.Columns, col)
	return r, fk
}
