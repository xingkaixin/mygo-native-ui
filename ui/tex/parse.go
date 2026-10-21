package tex

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

type kind uint8

const (
	// sym is the character r in alphabet alpha.
	sym kind = iota
	// text is text, in alpha: \text's, a function's name, a mistake's
	// source.
	text
	// group is body as one atom.
	group
	// frac is body over den, without a rule but between r and r2 for a
	// binomial.
	frac
	// radical is the root of body, den its degree.
	radical
	// fence is body between r and r2, sized to it, with middle atoms in
	// it.
	fence
	middle
	// delim is r size ems tall.
	delim
	// accent is r over body, as wide as body when wide.
	accent
	// bar is a line over body, or under it when under.
	bar
	// brace is the brace r over or under body.
	brace
	// array is the cells of rows, laid out as environment env.
	array
	// space is size mu of room.
	space
	// setStyle sets the style of what follows in the list.
	setStyle
)

// atom is a part of a formula: what it is, its class, its scripts, and
// the parts each kind has.
type atom struct {
	kind  kind
	class class
	r, r2 rune
	text  string
	alpha alphabet
	ink   ink
	// limits puts the scripts above and below when 1, beside when -1, and
	// as the style says when 0.
	limits int8
	// style is the style a fraction or setStyle sets, or noStyle.
	style     style
	under     bool
	wide      bool
	size      float32
	body, den []*atom
	sub, sup  []*atom
	primes    int
	rows      [][][]*atom
	env       string
}

// ink is the color of an atom: the theme's text when zero.
type ink struct {
	c      ui.Color
	danger bool
}

func (k ink) color(t *ui.Theme) ui.Color {
	switch {
	case k.danger:
		return t.Danger
	case k.c.A != 0:
		return k.c
	}
	return t.Text
}

type parser struct {
	src   string
	pos   int
	errs  []error
	ink   ink
	alpha alphabet
	// groups, fences, envs and brackets count what is open, which the
	// tokens closing them end lists in.
	groups, fences, envs, brackets int
}

// parse reads the formula src, typesetting its mistakes as their source in
// red.
func parse(src string) ([]*atom, []error) {
	p := &parser{src: src}
	return p.list(), p.errs
}

func isLetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

func (p *parser) skipSpace() {
	for p.pos < len(p.src) {
		switch p.src[p.pos] {
		case ' ', '\t', '\n', '\r':
			p.pos++
		case '%':
			for p.pos < len(p.src) && p.src[p.pos] != '\n' {
				p.pos++
			}
		default:
			return
		}
	}
}

// peek returns the next token: a command with its backslash, as `\frac`,
// `\{` and `\\`, or a character; "" at the end.
func (p *parser) peek() string {
	p.skipSpace()
	if p.pos >= len(p.src) {
		return ""
	}
	i := p.pos + 1
	if p.src[p.pos] == '\\' && i < len(p.src) {
		if isLetter(p.src[i]) {
			for i < len(p.src) && isLetter(p.src[i]) {
				i++
			}
		} else {
			_, n := utf8.DecodeRuneInString(p.src[i:])
			i += n
		}
		return p.src[p.pos:i]
	}
	_, n := utf8.DecodeRuneInString(p.src[p.pos:])
	return p.src[p.pos : p.pos+n]
}

func (p *parser) next() string {
	t := p.peek()
	p.pos += len(t)
	return t
}

// fail notes a mistake and returns the atom showing src[start:p.pos].
func (p *parser) fail(start int, format string, args ...any) *atom {
	p.errs = append(p.errs, fmt.Errorf(format, args...))
	return &atom{kind: text, text: p.src[start:p.pos], alpha: roman, ink: ink{danger: true}}
}

// closes reports whether tok ends the list being read, closing what is
// open around it.
func (p *parser) closes(tok string) bool {
	switch tok {
	case "":
		return true
	case "}":
		return p.groups > 0
	case "]":
		return p.brackets > 0
	case `\right`, `\middle`:
		return p.fences > 0
	case "&", `\\`, `\end`:
		return p.envs > 0
	}
	return false
}

// list reads atoms up to what closes it, which it leaves; \color holds to
// its end.
func (p *parser) list() []*atom {
	saved := p.ink
	defer func() { p.ink = saved }()
	var out []*atom
	for {
		start := p.pos
		tok := p.peek()
		if p.closes(tok) {
			return out
		}
		switch tok {
		case "^", "_":
			out = p.script(out, tok == "^")
		case "'":
			p.next()
			last(&out).primes++
		case `\limits`, `\nolimits`:
			p.next()
			if n := len(out); n > 0 && out[n-1].class == op {
				out[n-1].limits = map[bool]int8{true: 1, false: -1}[tok == `\limits`]
			} else {
				out = append(out, p.fail(start, "%s follows no operator", tok))
			}
		case "}", `\right`, `\middle`, "&", `\\`, `\end`:
			p.next()
			out = append(out, p.fail(start, "%s closes nothing", tok))
		default:
			out = append(out, p.atom()...)
		}
	}
}

// last returns the last atom of list, adding an empty one to bear scripts
// where there is none.
func last(list *[]*atom) *atom {
	if n := len(*list); n > 0 && (*list)[n-1].kind != space && (*list)[n-1].kind != setStyle {
		return (*list)[n-1]
	}
	a := &atom{kind: group}
	*list = append(*list, a)
	return a
}

func (p *parser) script(out []*atom, up bool) []*atom {
	start := p.pos
	tok := p.next()
	arg, ok := p.arg()
	if !ok {
		return append(out, p.fail(start, "%s needs a script after it", tok))
	}
	base := last(&out)
	field := &base.sub
	if up {
		field = &base.sup
	}
	if *field != nil {
		return append(out, p.fail(start, "two %ss on one atom", map[bool]string{true: "superscript", false: "subscript"}[up]))
	}
	*field = arg
	if *field == nil {
		*field = []*atom{}
	}
	return out
}

// arg reads an argument, a group or a token, false when there is none.
func (p *parser) arg() ([]*atom, bool) {
	switch tok := p.peek(); tok {
	case "{":
		return p.group(), true
	case "", "}", "^", "_", "&", `\\`, `\end`, `\right`, `\middle`:
		return nil, false
	case "]":
		if p.brackets > 0 {
			return nil, false
		}
	}
	return p.atom(), true
}

// group reads {…}; without its }, the { shows red before the rest.
func (p *parser) group() []*atom {
	start := p.pos
	p.next()
	p.groups++
	body := p.list()
	p.groups--
	if p.peek() == "}" {
		p.next()
		return body
	}
	end := p.pos
	p.pos = start + 1
	brace := p.fail(start, "{ has no }")
	p.pos = end
	return append([]*atom{brace}, body...)
}

// args reads n arguments, false when one is missing.
func (p *parser) args(n int) ([][]*atom, bool) {
	out := make([][]*atom, n)
	for i := range out {
		a, ok := p.arg()
		if !ok {
			return nil, false
		}
		out[i] = a
	}
	return out, true
}

func (p *parser) char(r rune) *atom {
	switch r {
	case '-':
		r = '−'
	case '*':
		r = '∗'
	}
	return &atom{kind: sym, r: r, class: classes[r], alpha: p.alpha, ink: p.ink}
}

// atom reads what a token starts: none, one atom, or the atoms of a
// mistake and what follows it.
func (p *parser) atom() []*atom {
	start := p.pos
	tok := p.peek()
	if tok == "" {
		return nil
	}
	if tok == "{" {
		return []*atom{{kind: group, body: p.group(), ink: p.ink}}
	}
	p.next()
	if tok == "~" {
		return []*atom{{kind: space, size: spaces[tok]}}
	}
	if tok[0] != '\\' || len(tok) == 1 {
		r, _ := utf8.DecodeRuneInString(tok)
		return []*atom{p.char(r)}
	}
	name := tok[1:]
	one := func(a *atom) []*atom {
		if a.ink == (ink{}) {
			a.ink = p.ink
		}
		return []*atom{a}
	}
	missing := func() []*atom { return []*atom{p.fail(start, "%s needs an argument", tok)} }
	if s, ok := symbols[name]; ok {
		a := &atom{kind: sym, r: s.r, class: s.class, alpha: p.alpha}
		if s.class == op && strings.ContainsRune(integrals, s.r) {
			a.limits = -1
		}
		return one(a)
	}
	if w, ok := spaces[tok]; ok {
		return []*atom{{kind: space, size: w}}
	}
	if limits, ok := functions[name]; ok {
		return one(&atom{kind: text, text: name, alpha: roman, class: op, limits: map[bool]int8{true: 0, false: -1}[limits]})
	}
	if a, ok := alphabets[name]; ok {
		saved := p.alpha
		p.alpha = a
		body, ok := p.arg()
		p.alpha = saved
		if !ok {
			return missing()
		}
		return one(&atom{kind: group, body: body})
	}
	if a, ok := textAlphabets[name]; ok {
		if p.peek() != "{" {
			return missing()
		}
		return one(&atom{kind: group, body: p.text(a)})
	}
	if acc, ok := accents[name]; ok {
		body, ok := p.arg()
		if !ok {
			return missing()
		}
		return one(&atom{kind: accent, r: acc.r, wide: acc.wide, body: body})
	}
	if size, cl, ok := bigDelim(name); ok {
		r, ok := p.delim()
		if !ok {
			return []*atom{p.fail(start, "%s needs a delimiter", tok)}
		}
		return one(&atom{kind: delim, r: r, size: size, class: cl})
	}
	switch name {
	case "frac", "dfrac", "tfrac", "cfrac", "binom", "dbinom", "tbinom":
		a, ok := p.args(2)
		if !ok {
			return missing()
		}
		f := &atom{kind: frac, class: inner, body: a[0], den: a[1], style: noStyle}
		switch name[0] {
		case 'd':
			f.style = display
		case 't':
			f.style = textStyle
		case 'c':
			f.style = display
			f.body = append([]*atom{{kind: setStyle, style: display}}, f.body...)
			f.den = append([]*atom{{kind: setStyle, style: display}}, f.den...)
		}
		if strings.HasSuffix(name, "binom") {
			f.r, f.r2 = '(', ')'
		}
		return one(f)
	case "sqrt":
		var degree []*atom
		if p.peek() == "[" {
			p.next()
			p.brackets++
			degree = p.list()
			p.brackets--
			if p.peek() != "]" {
				return []*atom{p.fail(start, `\sqrt[ has no ]`)}
			}
			p.next()
		}
		body, ok := p.arg()
		if !ok {
			return missing()
		}
		return one(&atom{kind: radical, body: body, den: degree})
	case "operatorname":
		limits := int8(-1)
		if strings.HasPrefix(p.src[p.pos:], "*") {
			p.pos++
			limits = 0
		}
		saved := p.alpha
		p.alpha = roman
		body, ok := p.arg()
		p.alpha = saved
		if !ok {
			return missing()
		}
		return one(&atom{kind: group, class: op, body: body, limits: limits})
	case "overline", "underline":
		body, ok := p.arg()
		if !ok {
			return missing()
		}
		return one(&atom{kind: bar, body: body, under: name == "underline"})
	case "overbrace", "underbrace":
		body, ok := p.arg()
		if !ok {
			return missing()
		}
		b := &atom{kind: brace, class: op, limits: 1, body: body, under: name == "underbrace", r: '⏞'}
		if b.under {
			b.r = '⏟'
		}
		return one(b)
	case "overset", "underset", "stackrel":
		a, ok := p.args(2)
		if !ok {
			return missing()
		}
		g := &atom{kind: group, body: a[1], limits: 1}
		if len(a[1]) == 1 {
			g.class = a[1][0].class
		}
		if name == "underset" {
			g.sub = a[0]
		} else {
			g.sup = a[0]
		}
		return one(g)
	case "not":
		next, _ := p.arg()
		if len(next) == 1 && next[0].kind == sym {
			if r, ok := negations[next[0].r]; ok {
				next[0].r = r
				return next
			}
		}
		return append([]*atom{p.fail(start, `\not cannot strike that out`)}, next...)
	case "displaystyle", "textstyle", "scriptstyle", "scriptscriptstyle":
		st := map[string]style{"displaystyle": display, "textstyle": textStyle, "scriptstyle": scriptStyle, "scriptscriptstyle": scriptScript}[name]
		return []*atom{{kind: setStyle, style: st}}
	case "color", "textcolor":
		c, ok := p.color()
		if !ok {
			return []*atom{p.fail(start, "%s needs a color, as {red} or {#e11d48}", tok)}
		}
		if name == "color" {
			p.ink = ink{c: c}
			return nil
		}
		saved := p.ink
		p.ink = ink{c: c}
		body, ok := p.arg()
		p.ink = saved
		if !ok {
			return missing()
		}
		return []*atom{{kind: group, body: body}}
	case "left":
		return p.fence(start)
	case "begin":
		return p.env(start)
	}
	return []*atom{p.fail(start, "unknown command %s", tok)}
}

// bigDelim returns the size, in ems, and the class of \big, \Bigl and
// the others.
func bigDelim(name string) (float32, class, bool) {
	for i, base := range []string{"big", "Big", "bigg", "Bigg"} {
		if rest, ok := strings.CutPrefix(name, base); ok {
			if cl, ok := map[string]class{"": ord, "l": open, "r": close, "m": rel}[rest]; ok {
				return 1.2 + 0.6*float32(i), cl, true
			}
		}
	}
	return 0, 0, false
}

// delim reads a delimiter, 0 for ".".
func (p *parser) delim() (rune, bool) {
	tok := p.next()
	if tok == "." {
		return 0, true
	}
	r, ok := delimiters[tok]
	return r, ok
}

func (p *parser) fence(start int) []*atom {
	l, ok := p.delim()
	if !ok {
		return []*atom{p.fail(start, `\left needs a delimiter`)}
	}
	open := p.pos
	f := &atom{kind: fence, class: inner, r: l, ink: p.ink}
	p.fences++
	for {
		f.body = append(f.body, p.list()...)
		if p.peek() != `\middle` {
			break
		}
		at := p.pos
		p.next()
		r, ok := p.delim()
		if !ok {
			f.body = append(f.body, p.fail(at, `\middle needs a delimiter`))
			continue
		}
		f.body = append(f.body, &atom{kind: middle, r: r, ink: p.ink})
	}
	p.fences--
	if p.peek() != `\right` {
		end := p.pos
		p.pos = open
		left := p.fail(start, `\left has no \right`)
		p.pos = end
		return append([]*atom{left}, f.body...)
	}
	at := p.pos
	p.next()
	r, ok := p.delim()
	if !ok {
		return []*atom{f, p.fail(at, `\right needs a delimiter`)}
	}
	f.r2 = r
	return []*atom{f}
}

// envFences are the delimiters around environments.
var envFences = map[string][2]rune{
	"pmatrix": {'(', ')'}, "bmatrix": {'[', ']'}, "Bmatrix": {'{', '}'},
	"vmatrix": {'|', '|'}, "Vmatrix": {'‖', '‖'}, "cases": {'{', 0},
}

// layouts are how environments lay out their cells, by the one they are
// laid out as.
var layouts = map[string]string{
	"matrix": "matrix", "pmatrix": "matrix", "bmatrix": "matrix", "Bmatrix": "matrix",
	"vmatrix": "matrix", "Vmatrix": "matrix", "cases": "cases", "aligned": "aligned",
	"align": "aligned", "align*": "aligned", "split": "aligned", "gathered": "gathered",
	"gather": "gathered", "gather*": "gathered",
}

func (p *parser) envName() (string, bool) {
	p.skipSpace()
	rest := p.src[p.pos:]
	if !strings.HasPrefix(rest, "{") {
		return "", false
	}
	name, _, ok := strings.Cut(rest[1:], "}")
	if !ok {
		return "", false
	}
	p.pos += len(name) + 2
	return strings.TrimSpace(name), true
}

func (p *parser) env(start int) []*atom {
	name, ok := p.envName()
	if !ok {
		return []*atom{p.fail(start, `\begin needs an environment's name`)}
	}
	begin := p.pos
	a := &atom{kind: array, env: layouts[name], ink: p.ink}
	p.envs++
	row := [][]*atom{}
	for {
		row = append(row, p.list())
		switch p.peek() {
		case "&":
			p.next()
			continue
		case `\\`:
			p.next()
			a.rows, row = append(a.rows, row), [][]*atom{}
			continue
		}
		break
	}
	p.envs--
	if len(row) > 1 || len(row[0]) > 0 || len(a.rows) == 0 {
		a.rows = append(a.rows, row)
	}
	var head []*atom
	if a.env == "" {
		end := p.pos
		p.pos = begin
		head = append(head, p.fail(start, "unknown environment %s", name))
		p.pos = end
		a.env = "matrix"
	}
	var tail []*atom
	at := p.pos
	if p.peek() == `\end` {
		p.next()
		if got, _ := p.envName(); got != name {
			tail = append(tail, p.fail(at, `\begin{%s} ends with \end{%s}`, name, got))
		}
	} else {
		end := p.pos
		p.pos = begin
		head = append(head, p.fail(start, `\begin{%s} has no \end`, name))
		p.pos = end
	}
	body := a
	if d, ok := envFences[name]; ok {
		body = &atom{kind: fence, class: inner, r: d[0], r2: d[1], body: []*atom{a}, ink: p.ink}
	}
	return append(append(head, body), tail...)
}

// text reads the argument of \text: its words, with their spaces, and in
// red the commands that text has no meaning for.
func (p *parser) text(a alphabet) []*atom {
	start := p.pos
	p.next()
	var out []*atom
	var sb strings.Builder
	flush := func() {
		if sb.Len() > 0 {
			out = append(out, &atom{kind: text, text: sb.String(), alpha: a, ink: p.ink})
			sb.Reset()
		}
	}
	depth := 0
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		switch {
		case c == '{':
			depth++
		case c == '}' && depth == 0:
			p.pos++
			flush()
			return out
		case c == '}':
			depth--
		case c == ' ' || c == '\t' || c == '\n' || c == '~':
			if !strings.HasSuffix(sb.String(), " ") {
				sb.WriteByte(' ')
			}
		case c == '\\':
			at := p.pos
			tok := p.next()
			if len(tok) == 2 && strings.Contains(`{}$&%#_ \`, tok[1:]) {
				sb.WriteString(tok[1:])
			} else {
				flush()
				out = append(out, p.fail(at, "%s has no meaning in text", tok))
			}
			continue
		default:
			_, n := utf8.DecodeRuneInString(p.src[p.pos:])
			sb.WriteString(p.src[p.pos : p.pos+n])
			p.pos += n
			continue
		}
		p.pos++
	}
	flush()
	end := p.pos
	p.pos = start + 1
	brace := p.fail(start, "{ has no }")
	p.pos = end
	return append([]*atom{brace}, out...)
}

// colors are the named colors of LaTeX's xcolor.
var colors = map[string]string{
	"black": "#000000", "white": "#ffffff", "red": "#ff0000", "green": "#00a000",
	"blue": "#0000ff", "cyan": "#00b0b0", "magenta": "#ff00ff", "yellow": "#e0c000",
	"gray": "#808080", "grey": "#808080", "darkgray": "#404040", "lightgray": "#c0c0c0",
	"brown": "#a05a2c", "lime": "#80c000", "olive": "#808000", "orange": "#ff8000",
	"pink": "#ff8080", "purple": "#a000a0", "teal": "#008080", "violet": "#8000ff",
}

// color reads {name} or {#rrggbb}.
func (p *parser) color() (ui.Color, bool) {
	name, ok := p.envName()
	if !ok {
		return ui.Color{}, false
	}
	if hex, ok := colors[name]; ok {
		name = hex
	}
	hex := strings.TrimPrefix(name, "#")
	if len(hex) != 3 && len(hex) != 6 || strings.Trim(strings.ToLower(hex), "0123456789abcdef") != "" {
		return ui.Color{}, false
	}
	return ui.Hex("#" + hex), true
}

// Err returns what is wrong with formula src, the first of the mistakes
// View shows in red, or nil.
func Err(src string) error {
	_, errs := parse(src)
	if len(errs) == 0 {
		return nil
	}
	return errs[0]
}
