package plotter

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// form is what an expression draws.
type form uint8

const (
	formNone     form = iota // nothing, as an empty expression
	formY                    // y = f(x), or f(x) alone
	formX                    // x = g(y)
	formPolar                // r = f(θ)
	formParam                // (f(t), g(t))
	formPoint                // (a, b)
	formImplicit             // an equation of x and y, as x^2 + y^2 = 4
	formConst                // a = 2
)

// parsed is an expression read: what it draws, from f, and g for the y
// of a point or a parametric curve (f is the lhs − rhs of an implicit
// equation, and the value of a constant), the constant it defines, and
// the constants it uses; or what is wrong with it.
type parsed struct {
	form form
	f, g *node
	name string
	uses []string
	err  error
}

// node is a node of an expression's tree: a number 'n', a variable 'v',
// a function of one value 'f', min '<' and max '>', minus 'm', a point
// ',', or + - * / ^.
type node struct {
	op   byte
	num  float64
	name string
	slot int
	fn   func(float64) float64
	args []*node
}

var funcs = map[string]func(float64) float64{
	"sin": math.Sin, "cos": math.Cos, "tan": math.Tan,
	"asin": math.Asin, "acos": math.Acos, "atan": math.Atan,
	"sinh": math.Sinh, "cosh": math.Cosh, "tanh": math.Tanh,
	"sqrt": math.Sqrt, "cbrt": math.Cbrt, "abs": math.Abs,
	"ln": math.Log, "log": math.Log10, "exp": math.Exp,
	"floor": math.Floor, "ceil": math.Ceil, "round": math.Round,
	"sign": func(v float64) float64 {
		if v == 0 || math.IsNaN(v) {
			return v
		}
		return math.Copysign(1, v)
	},
}

// names are the words a run of letters is split into, the longest first,
// as Desmos reads ax as a times x and sinx as sin x.
var names = func() []string {
	out := []string{"min", "max", "pi", "theta"}
	for name := range funcs {
		out = append(out, name)
	}
	slices.SortFunc(out, func(a, b string) int { return cmp.Compare(len(b), len(a)) })
	return out
}()

// reserved are the variables of the forms, which no constant takes.
var reserved = map[string]bool{"x": true, "y": true, "r": true, "t": true, "θ": true}

type token struct {
	kind byte // 'n' a number, 'a' a name, 0 the end, or the operator
	num  float64
	text string
}

func (t token) String() string {
	switch t.kind {
	case 'n':
		return strconv.FormatFloat(t.num, 'g', -1, 64)
	case 'a':
		return t.text
	}
	return string(t.kind)
}

var ops = map[rune]byte{
	'+': '+', '-': '-', '−': '-', '*': '*', '×': '*', '·': '*', '/': '/', '÷': '/',
	'^': '^', '(': '(', ')': ')', ',': ',', '=': '=',
}

func lex(src string) ([]token, error) {
	var toks []token
	for i := 0; i < len(src); {
		r, size := utf8.DecodeRuneInString(src[i:])
		switch {
		case unicode.IsSpace(r):
			i += size
		case r >= '0' && r <= '9' || r == '.':
			j := i
			for j < len(src) && (src[j] >= '0' && src[j] <= '9' || src[j] == '.') {
				j++
			}
			v, err := strconv.ParseFloat(src[i:j], 64)
			if err != nil {
				return nil, fmt.Errorf("%s is not a number", src[i:j])
			}
			toks = append(toks, token{kind: 'n', num: v})
			i = j
		case unicode.IsLetter(r):
			j := i
			for j < len(src) {
				r, size := utf8.DecodeRuneInString(src[j:])
				if !unicode.IsLetter(r) {
					break
				}
				j += size
			}
			for _, name := range split(src[i:j]) {
				toks = append(toks, token{kind: 'a', text: name})
			}
			i = j
		default:
			op, ok := ops[r]
			if !ok {
				return nil, fmt.Errorf("%c is not allowed", r)
			}
			toks = append(toks, token{kind: op})
			i += size
		}
	}
	return append(toks, token{}), nil
}

// split splits a run of letters into names: the functions, pi and theta
// it begins with, longest first, or else a letter.
func split(word string) []string {
	var out []string
	for word != "" {
		n := 0
		for _, name := range names {
			if strings.HasPrefix(word, name) {
				n = len(name)
				break
			}
		}
		if n == 0 {
			_, n = utf8.DecodeRuneInString(word)
		}
		out = append(out, word[:n])
		word = word[n:]
	}
	return out
}

// parser reads an expression by recursive descent: sums, then products,
// written or implied as in 2x, then signs, then powers, which group to
// the right. It keeps the first error and stops there.
type parser struct {
	toks []token
	i    int
	err  error
}

func (p *parser) peek() byte { return p.toks[p.i].kind }

func (p *parser) next() token {
	t := p.toks[p.i]
	if t.kind != 0 {
		p.i++
	}
	return t
}

func (p *parser) fail(format string, a ...any) {
	if p.err == nil {
		p.err = fmt.Errorf(format, a...)
	}
}

func (p *parser) expect(kind byte) {
	if p.err == nil && p.next().kind != kind {
		p.fail("missing %c", kind)
	}
}

func (p *parser) sum() *node {
	n := p.product()
	for p.err == nil && (p.peek() == '+' || p.peek() == '-') {
		op := p.next().kind
		n = &node{op: op, args: []*node{n, p.product()}}
	}
	return n
}

func (p *parser) product() *node {
	n := p.unary()
	for p.err == nil {
		switch p.peek() {
		case '*', '/':
			op := p.next().kind
			n = &node{op: op, args: []*node{n, p.unary()}}
		case 'n', 'a', '(':
			n = &node{op: '*', args: []*node{n, p.power()}}
		default:
			return n
		}
	}
	return n
}

func (p *parser) unary() *node {
	switch p.peek() {
	case '-':
		p.next()
		return &node{op: 'm', args: []*node{p.unary()}}
	case '+':
		p.next()
		return p.unary()
	}
	return p.power()
}

func (p *parser) power() *node {
	n := p.primary()
	if p.err == nil && p.peek() == '^' {
		p.next()
		return &node{op: '^', args: []*node{n, p.unary()}}
	}
	return n
}

func (p *parser) primary() *node {
	t := p.next()
	switch t.kind {
	case 'n':
		return &node{op: 'n', num: t.num}
	case '(':
		n := p.sum()
		if p.err == nil && p.peek() == ',' {
			p.next()
			n = &node{op: ',', args: []*node{n, p.sum()}}
		}
		p.expect(')')
		return n
	case 'a':
		switch t.text {
		case "pi", "π":
			return &node{op: 'n', num: math.Pi}
		case "e":
			return &node{op: 'n', num: math.E}
		case "theta":
			return &node{op: 'v', name: "θ"}
		case "min", "max":
			args := p.call(t.text)
			return &node{op: map[string]byte{"min": '<', "max": '>'}[t.text], args: args}
		}
		if fn, ok := funcs[t.text]; ok {
			args := p.call(t.text)
			if p.err == nil && len(args) != 1 {
				p.fail("%s takes one value", t.text)
			}
			return &node{op: 'f', name: t.text, fn: fn, args: args[:min(1, len(args))]}
		}
		return &node{op: 'v', name: t.text}
	case 0:
		p.fail("ends too soon")
	default:
		p.fail("unexpected %s", t)
	}
	return &node{op: 'n'}
}

// call reads the values a function is called with, in parentheses.
func (p *parser) call(name string) []*node {
	if p.peek() != '(' {
		p.fail("%s needs parentheses, as %s(x)", name, name)
		return nil
	}
	p.next()
	args := []*node{p.sum()}
	for p.err == nil && p.peek() == ',' {
		p.next()
		args = append(args, p.sum())
	}
	p.expect(')')
	return args
}

// parse reads src and finds what it draws: an equation solved for y, x
// or r, a point or parametric curve, a constant's definition, or else an
// equation of x and y.
func parse(src string) *parsed {
	toks, err := lex(src)
	if err != nil {
		return &parsed{err: err}
	}
	if toks[0].kind == 0 {
		return &parsed{}
	}
	p := &parser{toks: toks}
	lhs := p.sum()
	var rhs *node
	if p.err == nil && p.peek() == '=' {
		p.next()
		rhs = p.sum()
	}
	switch {
	case p.err != nil:
	case p.peek() == '=':
		p.fail("only one = goes in an expression")
	case p.peek() != 0:
		p.fail("unexpected %s", p.toks[p.i])
	}
	if p.err != nil {
		return &parsed{err: p.err}
	}
	if rhs == nil {
		if lhs.op == ',' {
			e := &parsed{form: formParam, f: lhs.args[0], g: lhs.args[1]}
			if !uses(e.f, "t") && !uses(e.g, "t") {
				e.form = formPoint
			}
			return e.check("t")
		}
		return (&parsed{form: formY, f: lhs}).check("x")
	}
	if !isVar(lhs, "") && isVar(rhs, "") {
		lhs, rhs = rhs, lhs
	}
	switch {
	case isVar(lhs, "y") && !uses(rhs, "y"):
		return (&parsed{form: formY, f: rhs}).check("x")
	case isVar(lhs, "x") && !uses(rhs, "x"):
		return (&parsed{form: formX, f: rhs}).check("y")
	case isVar(lhs, "r"):
		return (&parsed{form: formPolar, f: rhs}).check("θ")
	case lhs.op == 'v' && !reserved[lhs.name] && !uses(rhs, "x", "y", "r", "t", "θ"):
		return (&parsed{form: formConst, f: rhs, name: lhs.name}).check()
	}
	f := &node{op: '-', args: []*node{lhs, rhs}}
	return (&parsed{form: formImplicit, f: f}).check("x", "y")
}

// check finds the constants e uses, and what is wrong with it: a
// variable not of its form, or a point inside an expression.
func (e *parsed) check(vars ...string) *parsed {
	var walk func(n *node)
	walk = func(n *node) {
		switch {
		case e.err != nil:
			return
		case n.op == ',':
			e.err = fmt.Errorf("a point goes alone, as (1, 2)")
		case n.op == 'v' && reserved[n.name] && !slices.Contains(vars, n.name):
			e.err = fmt.Errorf("%s can't be used here", n.name)
		case n.op == 'v' && !reserved[n.name] && !slices.Contains(e.uses, n.name):
			e.uses = append(e.uses, n.name)
		}
		for _, a := range n.args {
			walk(a)
		}
	}
	for _, n := range []*node{e.f, e.g} {
		if n != nil {
			walk(n)
		}
	}
	if e.err != nil {
		return &parsed{err: e.err}
	}
	return e
}

func isVar(n *node, name string) bool {
	return n.op == 'v' && reserved[n.name] && (name == "" || n.name == name)
}

func uses(n *node, vars ...string) bool {
	if n.op == 'v' && slices.Contains(vars, n.name) {
		return true
	}
	return slices.ContainsFunc(n.args, func(a *node) bool { return uses(a, vars...) })
}

// bind returns n with the values of consts for its constants, the slot
// of each variable it uses, and what does not vary worked out.
func bind(n *node, consts map[string]float64) *node {
	if n.op == 'v' {
		switch n.name {
		case "x":
			return &node{op: 'v', slot: 0}
		case "y":
			return &node{op: 'v', slot: 1}
		case "t", "θ":
			return &node{op: 'v', slot: 2}
		}
		return &node{op: 'n', num: consts[n.name]}
	}
	b := &node{op: n.op, num: n.num, fn: n.fn}
	fixed := true
	for _, a := range n.args {
		a = bind(a, consts)
		fixed = fixed && a.op == 'n'
		b.args = append(b.args, a)
	}
	if fixed && b.op != 'n' {
		return &node{op: 'n', num: b.eval(nil)}
	}
	return b
}

// eval returns n's value at the variables x, y and t in v.
func (n *node) eval(v *[3]float64) float64 {
	switch n.op {
	case 'n':
		return n.num
	case 'v':
		return v[n.slot]
	case 'f':
		return n.fn(n.args[0].eval(v))
	case 'm':
		return -n.args[0].eval(v)
	case '<', '>':
		out := n.args[0].eval(v)
		for _, a := range n.args[1:] {
			if n.op == '<' {
				out = math.Min(out, a.eval(v))
			} else {
				out = math.Max(out, a.eval(v))
			}
		}
		return out
	}
	a, b := n.args[0].eval(v), n.args[1].eval(v)
	switch n.op {
	case '+':
		return a + b
	case '-':
		return a - b
	case '*':
		return a * b
	case '/':
		return a / b
	}
	return math.Pow(a, b)
}

// resolve works out the constants that exprs define, each once and those
// it uses first, and what is wrong with each expression: its syntax, or a
// constant it uses that is not defined, defined twice or in terms of
// itself.
func resolve(exprs []*parsed) (map[string]float64, []error) {
	errs := make([]error, len(exprs))
	defs := map[string][]int{}
	for i, e := range exprs {
		errs[i] = e.err
		if e.form == formConst {
			defs[e.name] = append(defs[e.name], i)
		}
	}
	consts := map[string]float64{}
	failed := map[string]error{}
	working := map[string]bool{}
	var value func(name string) error
	value = func(name string) error {
		if _, ok := consts[name]; ok {
			return nil
		}
		if err, ok := failed[name]; ok {
			return err
		}
		if working[name] {
			return fmt.Errorf("%s is defined in terms of itself", name)
		}
		is := defs[name]
		var err error
		switch len(is) {
		case 0:
			return fmt.Errorf("%s is not defined", name)
		case 1:
			working[name] = true
			e := exprs[is[0]]
			for _, u := range e.uses {
				if err = value(u); err != nil {
					break
				}
			}
			working[name] = false
			if err == nil {
				consts[name] = bind(e.f, consts).eval(nil)
				return nil
			}
		default:
			err = fmt.Errorf("%s is defined more than once", name)
		}
		failed[name] = err
		for _, i := range is {
			errs[i] = err
		}
		return err
	}
	for i, e := range exprs {
		if e.err != nil {
			continue
		}
		if e.form == formConst {
			value(e.name)
			continue
		}
		for _, u := range e.uses {
			if err := value(u); err != nil {
				errs[i] = err
				break
			}
		}
	}
	return consts, errs
}

// Constant returns the name and value of a constant src defines as a
// number, as a = 0.5 or b = -2, which a slider can set; ok is false for
// other expressions.
func Constant(src string) (name string, value float64, ok bool) {
	e := parse(src)
	if e.form != formConst || e.err != nil {
		return "", 0, false
	}
	switch n := e.f; {
	case n.op == 'n':
		return e.name, n.num, true
	case n.op == 'm' && n.args[0].op == 'n':
		return e.name, -n.args[0].num, true
	}
	return "", 0, false
}

// Errors returns what is wrong with each of exprs, nil for those that
// draw or define a constant: its syntax, a variable its form doesn't
// have, as t in y = f(x), or a constant it uses that no expression
// defines, or that is defined twice or in terms of itself.
func Errors(exprs []Expr) []error {
	ps := make([]*parsed, len(exprs))
	for i, e := range exprs {
		ps[i] = parse(e.Source)
	}
	_, errs := resolve(ps)
	return errs
}
