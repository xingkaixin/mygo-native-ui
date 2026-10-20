package spreadsheet

import (
	"math"
	"strconv"
	"strings"
	"unicode"
)

// Cell is a cell of a sheet by its column and row, from 0: A1 is {0, 0}.
type Cell struct {
	Col, Row int
}

// Name returns the cell's name, as A1 or AB12.
func (c Cell) Name() string { return column(c.Col) + strconv.Itoa(c.Row+1) }

// column returns the letters of column k: A to Z, then AA.
func column(k int) string {
	s := ""
	for k++; k > 0; k = (k - 1) / 26 {
		s = string(rune('A'+(k-1)%26)) + s
	}
	return s
}

// parseCell reads a cell's name, as A1, in any case.
func parseCell(s string) (Cell, bool) {
	i := 0
	col := 0
	for i < len(s) && unicode.IsLetter(rune(s[i])) {
		col = col*26 + int(unicode.ToUpper(rune(s[i]))-'A'+1)
		i++
	}
	if i == 0 || i == len(s) {
		return Cell{}, false
	}
	row, err := strconv.Atoi(s[i:])
	if err != nil || row < 1 {
		return Cell{}, false
	}
	return Cell{col - 1, row - 1}, true
}

// value is what a cell shows: a number, text, or an error such as
// #DIV/0!.
type value struct {
	num   float64
	text  string
	isNum bool
	err   string
}

func (v value) String() string {
	switch {
	case v.err != "":
		return v.err
	case v.isNum:
		return strconv.FormatFloat(v.num, 'g', 10, 64)
	}
	return v.text
}

// sheet works out the values of cells, each once, those a formula refers
// to first.
type sheet struct {
	cells map[Cell]string
	memo  map[Cell]value
	state map[Cell]uint8
}

func newSheet(cells map[Cell]string) *sheet {
	return &sheet{cells: cells, memo: map[Cell]value{}, state: map[Cell]uint8{}}
}

func (sh *sheet) get(c Cell) value {
	switch sh.state[c] {
	case 1:
		return value{err: "#CYCLE!"}
	case 2:
		return sh.memo[c]
	}
	sh.state[c] = 1
	v := sh.compute(sh.cells[c])
	sh.memo[c], sh.state[c] = v, 2
	return v
}

func (sh *sheet) compute(raw string) value {
	s := strings.TrimSpace(raw)
	if s == "" {
		return value{}
	}
	if s[0] == '=' {
		p := &parser{sh: sh, src: s[1:]}
		p.next()
		v := p.compare()
		if p.tok.kind != end && v.err == "" {
			return value{err: "#ERROR!"}
		}
		if v == (value{}) {
			// A formula of an empty cell is 0, as other spreadsheets make it.
			return n(0)
		}
		return v
	}
	if f, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", ""), 64); err == nil {
		return value{num: f, isNum: true}
	}
	return value{text: raw}
}

type kind uint8

const (
	end kind = iota
	number
	name
	op
	bad
)

type token struct {
	kind kind
	text string
	num  float64
}

// parser works a formula out as it reads it, by recursive descent: the
// comparisons, then sums, then products, then signs and powers.
type parser struct {
	sh  *sheet
	src string
	pos int
	tok token
}

func (p *parser) next() {
	for p.pos < len(p.src) && p.src[p.pos] == ' ' {
		p.pos++
	}
	if p.pos >= len(p.src) {
		p.tok = token{kind: end}
		return
	}
	start := p.pos
	ch := p.src[p.pos]
	switch {
	case ch >= '0' && ch <= '9' || ch == '.':
		for p.pos < len(p.src) && (p.src[p.pos] >= '0' && p.src[p.pos] <= '9' || p.src[p.pos] == '.') {
			p.pos++
		}
		f, err := strconv.ParseFloat(p.src[start:p.pos], 64)
		if err != nil {
			p.tok = token{kind: bad}
			return
		}
		p.tok = token{kind: number, num: f}
	case unicode.IsLetter(rune(ch)):
		for p.pos < len(p.src) && (unicode.IsLetter(rune(p.src[p.pos])) || unicode.IsDigit(rune(p.src[p.pos]))) {
			p.pos++
		}
		p.tok = token{kind: name, text: strings.ToUpper(p.src[start:p.pos])}
	default:
		p.pos++
		if p.pos < len(p.src) {
			if two := p.src[start : p.pos+1]; two == "<=" || two == ">=" || two == "<>" {
				p.pos++
			}
		}
		p.tok = token{kind: op, text: p.src[start:p.pos]}
	}
}

func (p *parser) is(s string) bool { return p.tok.kind == op && p.tok.text == s }

func num(v value) (float64, value) {
	switch {
	case v.err != "":
		return 0, v
	case v.isNum:
		return v.num, value{}
	case v.text == "":
		return 0, value{}
	}
	return 0, value{err: "#VALUE!"}
}

func arith(a, b value, f func(x, y float64) value) value {
	x, e := num(a)
	if e.err != "" {
		return e
	}
	y, e := num(b)
	if e.err != "" {
		return e
	}
	return f(x, y)
}

func n(f float64) value { return value{num: f, isNum: true} }

func (p *parser) compare() value {
	v := p.sum()
	for p.tok.kind == op {
		o := p.tok.text
		var test func(x, y float64) bool
		switch o {
		case "=":
			test = func(x, y float64) bool { return x == y }
		case "<>":
			test = func(x, y float64) bool { return x != y }
		case "<":
			test = func(x, y float64) bool { return x < y }
		case ">":
			test = func(x, y float64) bool { return x > y }
		case "<=":
			test = func(x, y float64) bool { return x <= y }
		case ">=":
			test = func(x, y float64) bool { return x >= y }
		default:
			return v
		}
		p.next()
		w := p.sum()
		v = arith(v, w, func(x, y float64) value {
			if test(x, y) {
				return n(1)
			}
			return n(0)
		})
	}
	return v
}

func (p *parser) sum() value {
	v := p.product()
	for p.is("+") || p.is("-") {
		minus := p.is("-")
		p.next()
		w := p.product()
		v = arith(v, w, func(x, y float64) value {
			if minus {
				return n(x - y)
			}
			return n(x + y)
		})
	}
	return v
}

func (p *parser) product() value {
	v := p.power()
	for p.is("*") || p.is("/") {
		div := p.is("/")
		p.next()
		w := p.power()
		v = arith(v, w, func(x, y float64) value {
			if !div {
				return n(x * y)
			}
			if y == 0 {
				return value{err: "#DIV/0!"}
			}
			return n(x / y)
		})
	}
	return v
}

func (p *parser) power() value {
	v := p.unary()
	if p.is("^") {
		p.next()
		w := p.power()
		v = arith(v, w, func(x, y float64) value { return n(math.Pow(x, y)) })
	}
	return v
}

func (p *parser) unary() value {
	if p.is("-") {
		p.next()
		return arith(n(0), p.unary(), func(_, y float64) value { return n(-y) })
	}
	if p.is("+") {
		p.next()
		return p.unary()
	}
	return p.primary()
}

func (p *parser) primary() value {
	t := p.tok
	switch {
	case t.kind == number:
		p.next()
		return n(t.num)
	case p.is("("):
		p.next()
		v := p.compare()
		if !p.is(")") {
			return value{err: "#ERROR!"}
		}
		p.next()
		return v
	case t.kind == name:
		p.next()
		if p.is("(") {
			p.next()
			return p.call(t.text)
		}
		c, ok := parseCell(t.text)
		if !ok {
			return value{err: "#NAME?"}
		}
		return p.sh.get(c)
	}
	return value{err: "#ERROR!"}
}

// args reads a function's arguments up to its ")", each a value or the
// values of a range, as A1:B3.
func (p *parser) args() ([][]value, bool) {
	var out [][]value
	if p.is(")") {
		p.next()
		return out, true
	}
	for {
		if p.tok.kind == name && p.pos < len(p.src) && strings.HasPrefix(strings.TrimLeft(p.src[p.pos:], " "), ":") {
			from, ok := parseCell(p.tok.text)
			p.next()
			p.next()
			to, ok2 := parseCell(p.tok.text)
			if !ok || !ok2 || p.tok.kind != name {
				return nil, false
			}
			p.next()
			var vs []value
			for r := min(from.Row, to.Row); r <= max(from.Row, to.Row); r++ {
				for c := min(from.Col, to.Col); c <= max(from.Col, to.Col); c++ {
					vs = append(vs, p.sh.get(Cell{c, r}))
				}
			}
			out = append(out, vs)
		} else {
			out = append(out, []value{p.compare()})
		}
		switch {
		case p.is(","):
			p.next()
		case p.is(")"):
			p.next()
			return out, true
		default:
			return nil, false
		}
	}
}

func (p *parser) call(fn string) value {
	args, ok := p.args()
	if !ok {
		return value{err: "#ERROR!"}
	}
	var nums []float64
	for _, a := range args {
		for _, v := range a {
			if v.err != "" {
				return v
			}
			if v.isNum {
				nums = append(nums, v.num)
			}
		}
	}
	arg := func(i int) (float64, value) {
		if i >= len(args) || len(args[i]) != 1 {
			return 0, value{err: "#ERROR!"}
		}
		return num(args[i][0])
	}
	switch fn {
	case "SUM":
		s := 0.0
		for _, x := range nums {
			s += x
		}
		return n(s)
	case "AVERAGE", "AVG":
		if len(nums) == 0 {
			return value{err: "#DIV/0!"}
		}
		s := 0.0
		for _, x := range nums {
			s += x
		}
		return n(s / float64(len(nums)))
	case "MIN", "MAX":
		if len(nums) == 0 {
			return n(0)
		}
		m := nums[0]
		for _, x := range nums[1:] {
			if fn == "MIN" {
				m = math.Min(m, x)
			} else {
				m = math.Max(m, x)
			}
		}
		return n(m)
	case "COUNT":
		return n(float64(len(nums)))
	case "ABS":
		x, e := arg(0)
		if e.err != "" {
			return e
		}
		return n(math.Abs(x))
	case "ROUND":
		x, e := arg(0)
		if e.err != "" {
			return e
		}
		d := 0.0
		if len(args) > 1 {
			if d, e = arg(1); e.err != "" {
				return e
			}
		}
		k := math.Pow(10, d)
		return n(math.Round(x*k) / k)
	case "IF":
		if len(args) != 3 {
			return value{err: "#ERROR!"}
		}
		test, e := arg(0)
		if e.err != "" {
			return e
		}
		if test != 0 {
			return args[1][0]
		}
		return args[2][0]
	}
	return value{err: "#NAME?"}
}
