package dot

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

type direction uint8

const (
	tb direction = iota
	bt
	lr
	rl
)

var rankdirs = map[string]direction{"TB": tb, "BT": bt, "LR": lr, "RL": rl}

type shape uint8

const (
	ellipse shape = iota
	box
	circle
	doublecircle
	diamond
	plaintext
	plain
	point
	note
	cylinder
)

var shapes = map[string]shape{
	"ellipse": ellipse, "oval": ellipse, "box": box, "rect": box, "rectangle": box, "circle": circle,
	"doublecircle": doublecircle, "diamond": diamond, "plaintext": plaintext, "none": plaintext,
	"plain": plain, "point": point, "note": note, "cylinder": cylinder,
}

type style uint8

const (
	filled style = 1 << iota
	rounded
	dashed
	dotted
	bold
	invis
)

var styles = map[string]style{
	"filled": filled, "rounded": rounded, "dashed": dashed, "dotted": dotted, "bold": bold,
	"invis": invis, "invisible": invis, "solid": 0,
}

// arrow is what ends an edge. The zero arrow is normal, as in Graphviz.
type arrow uint8

const (
	normalHead arrow = iota
	emptyHead
	veeHead
	dotHead
	odotHead
	diamondHead
	noHead
)

var arrows = map[string]arrow{
	"normal": normalHead, "empty": emptyHead, "onormal": emptyHead, "vee": veeHead, "dot": dotHead,
	"odot": odotHead, "diamond": diamondHead, "none": noHead,
}

// dirs are the ends of an edge that show arrowheads: 1 for its head, 2
// for its tail.
var dirs = map[string]uint8{"forward": 1, "back": 2, "both": 3, "none": 0}

// ink is a color the source gives, if it gives one.
type ink struct {
	c  ui.Color
	ok bool
}

func (k ink) or(c ui.Color) ui.Color {
	if k.ok {
		return k.c
	}
	return c
}

// look is how a node, an edge or a cluster draws, as its attributes say:
// its label as written, its style, its colors and how wide its lines
// are, 1 as Graphviz's 1 point.
type look struct {
	label             string
	style             style
	color, fill, font ink
	pen               float32
}

// set gives l the attribute key, and returns what is wrong with val, if
// anything. It keeps the parts of a style that are right, and skips the
// attributes it does not draw.
func (l *look) set(key, val string) string {
	switch key {
	case "label":
		l.label = val
	case "style":
		l.style = 0
		wrong := ""
		for _, s := range strings.Split(val, ",") {
			s = strings.TrimSpace(s)
			st, ok := styles[strings.ToLower(s)]
			if !ok && wrong == "" {
				wrong = fmt.Sprintf("%q is not a style: filled, rounded, dashed, dotted, bold or invis", s)
			}
			l.style |= st
		}
		return wrong
	case "color", "fillcolor", "fontcolor":
		c, ok := parseColor(val)
		if !ok {
			return fmt.Sprintf("%q is not a color: #rrggbb, #rrggbbaa or an X11 name", val)
		}
		switch key {
		case "color":
			l.color = ink{c, true}
		case "fillcolor":
			l.fill = ink{c, true}
		default:
			l.font = ink{c, true}
		}
	case "penwidth":
		f, err := strconv.ParseFloat(strings.TrimSpace(val), 32)
		if err != nil || f < 0 {
			return fmt.Sprintf("%q is not a pen width", val)
		}
		l.pen = float32(f)
	}
	return ""
}

// filling returns the color l fills with when filled: its fillcolor, else
// its color.
func (l look) filling() ink {
	if l.fill.ok {
		return l.fill
	}
	return l.color
}

func (n *node) set(key, val string) string {
	if key != "shape" {
		return n.look.set(key, val)
	}
	sh, ok := shapes[strings.ToLower(val)]
	if !ok {
		return fmt.Sprintf("%q is not a shape: box, ellipse, circle, doublecircle, diamond, note, cylinder, point, plaintext or none", val)
	}
	n.shape = sh
	return ""
}

func (e *edge) set(key, val string) string {
	switch key {
	case "arrowhead", "arrowtail":
		h, ok := arrows[strings.ToLower(val)]
		if !ok {
			return fmt.Sprintf("%q is not an arrowhead: normal, empty, vee, dot, odot, diamond or none", val)
		}
		if key == "arrowhead" {
			e.heads[1] = h
		} else {
			e.heads[0] = h
		}
	case "dir":
		d, ok := dirs[strings.ToLower(val)]
		if !ok {
			return fmt.Sprintf("%q is not a dir: forward, back, both or none", val)
		}
		e.dir = d
	default:
		return e.look.set(key, val)
	}
	return ""
}

// label returns s as it shows: \n, \l and \r break lines, \N is the
// node's ID and \G the graph's, and a backslash before anything else
// stands for it.
func label(s, id, graph string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 == len(s) {
			b.WriteByte(s[i])
			continue
		}
		i++
		switch s[i] {
		case 'n', 'l', 'r':
			b.WriteByte('\n')
		case 'N':
			b.WriteString(id)
		case 'G':
			b.WriteString(graph)
		default:
			b.WriteByte(s[i])
		}
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// Kinds of tokens; punctuation is its own byte.
const (
	tEOF byte = iota
	tID
	tQuoted
	tHTML
	tArrow
	tLine
)

type token struct {
	kind byte
	text string
	line int
}

func idByte(c byte) bool {
	return c == '_' || c >= 0x80 || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9'
}

// lex splits src into tokens, skipping comments and the lines a C
// preprocessor leaves, which start with #.
func lex(src string) ([]token, []problem) {
	var toks []token
	var errs []problem
	line, bol := 1, true
	for i := 0; i < len(src); {
		c := src[i]
		switch c {
		case '\n':
			line, bol = line+1, true
			i++
			continue
		case ' ', '\t', '\r', '\f', '\v':
			i++
			continue
		}
		start := line
		switch {
		case c == '#' && bol, strings.HasPrefix(src[i:], "//"):
			if j := strings.IndexByte(src[i:], '\n'); j >= 0 {
				i += j
			} else {
				i = len(src)
			}
		case strings.HasPrefix(src[i:], "/*"):
			j := strings.Index(src[i+2:], "*/")
			if j < 0 {
				errs = append(errs, problem{line, "the comment has no */"})
				i = len(src)
				break
			}
			line += strings.Count(src[i:i+2+j], "\n")
			i += j + 4
		case c == '"':
			// Only \" stands for something else here; the other escapes
			// are the label's.
			var b strings.Builder
			j := i + 1
			for ; j < len(src) && src[j] != '"'; j++ {
				switch {
				case src[j] == '\\' && j+1 < len(src) && src[j+1] == '"':
					b.WriteByte('"')
					j++
				case src[j] == '\\' && j+1 < len(src) && src[j+1] == '\n':
					line++
					j++
				case src[j] == '\\' && j+1 < len(src):
					b.WriteString(src[j : j+2])
					j++
				default:
					if src[j] == '\n' {
						line++
					}
					b.WriteByte(src[j])
				}
			}
			if j == len(src) {
				errs = append(errs, problem{start, "the string has no closing quote"})
			}
			toks = append(toks, token{tQuoted, b.String(), start})
			i = min(j+1, len(src))
		case c == '<':
			depth, j := 0, i
			for ; j < len(src); j++ {
				if src[j] == '<' {
					depth++
				} else if src[j] == '>' {
					if depth--; depth == 0 {
						break
					}
				} else if src[j] == '\n' {
					line++
				}
			}
			toks = append(toks, token{tHTML, src[i:min(j+1, len(src))], start})
			i = min(j+1, len(src))
		case c == '-' && i+1 < len(src) && src[i+1] == '>':
			toks = append(toks, token{tArrow, "->", line})
			i += 2
		case c == '-' && i+1 < len(src) && src[i+1] == '-':
			toks = append(toks, token{tLine, "--", line})
			i += 2
		case idByte(c) || c == '.' || c == '-' && i+1 < len(src) && (src[i+1] == '.' || '0' <= src[i+1] && src[i+1] <= '9'):
			j := i + 1
			for j < len(src) && (idByte(src[j]) || src[j] == '.') {
				j++
			}
			toks = append(toks, token{tID, src[i:j], line})
			i = j
		case strings.IndexByte("{}[];,=:+", c) >= 0:
			toks = append(toks, token{c, string(c), line})
			i++
		default:
			r, w := utf8.DecodeRuneInString(src[i:])
			errs = append(errs, problem{line, fmt.Sprintf("%q is not part of DOT", r)})
			i += w
		}
		bol = false
	}
	return append(toks, token{tEOF, "", line}), errs
}

// stmt is a statement as written: a node with its attributes, edges
// between its ends, defaults for the nodes, the edges or the graph after
// it, an attribute of the graph, or a subgraph.
type stmt struct {
	kind  byte // 'n', 'e', 'd', '=' or 's'
	line  int
	of    string
	ends  []operand
	attrs []attr
}

// operand is a node's ID, or a subgraph standing for its nodes.
type operand struct {
	id  string
	sub *sub
}

type sub struct {
	id   string
	body []stmt
}

type attr struct {
	key, val string
	line     int
}

// syntax is what is wrong with a statement, which leaves it.
type syntax problem

type parser struct {
	toks     []token
	i        int
	errs     []problem
	directed bool
}

func (p *parser) peek() token { return p.toks[p.i] }

func (p *parser) next() token {
	t := p.toks[p.i]
	if t.kind != tEOF {
		p.i++
	}
	return t
}

func (p *parser) fail(line int, format string, args ...any) {
	panic(syntax{line, fmt.Sprintf(format, args...)})
}

func keyword(t token, words ...string) bool {
	if t.kind != tID {
		return false
	}
	for _, w := range words {
		if strings.EqualFold(t.text, w) {
			return true
		}
	}
	return false
}

var keywords = []string{"graph", "digraph", "subgraph", "node", "edge", "strict"}

// body reads statements up to the } that ends them, or the end.
func (p *parser) body() []stmt {
	var out []stmt
	for {
		switch p.peek().kind {
		case '}', tEOF:
			return out
		case ';':
			p.next()
			continue
		}
		at := p.i
		if s, ok := p.stmt(); ok {
			out = append(out, s)
		}
		if p.i == at {
			p.next()
		}
	}
}

// stmt reads a statement, or notes what is wrong with it and skips it.
func (p *parser) stmt() (s stmt, ok bool) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		e, is := r.(syntax)
		if !is {
			panic(r)
		}
		p.errs = append(p.errs, problem(e))
		p.skip(e.line)
		ok = false
	}()
	return p.statement(), true
}

// skip passes the rest of a wrong statement: up to a semicolon, the }
// around it, or the next line, past the brackets and braces it opens.
func (p *parser) skip(line int) {
	depth := 0
	for {
		t := p.peek()
		switch {
		case t.kind == tEOF:
			return
		case depth == 0 && (t.line > line || t.kind == '}'):
			return
		case depth == 0 && t.kind == ';':
			p.next()
			return
		case t.kind == '[' || t.kind == '{':
			depth++
		case (t.kind == ']' || t.kind == '}') && depth > 0:
			depth--
		}
		p.next()
	}
}

func (p *parser) statement() stmt {
	t := p.peek()
	if keyword(t, "graph", "node", "edge") {
		p.next()
		of := strings.ToLower(t.text)
		if p.peek().kind != '[' {
			p.fail(t.line, "%s takes its attributes in [ ], as %s [color=gray]", of, of)
		}
		return stmt{kind: 'd', line: t.line, of: of, attrs: p.attrs()}
	}
	if keyword(t, "digraph", "strict") {
		p.fail(t.line, "%s may only start the graph", t.text)
	}
	if (t.kind == tID && !keyword(t, keywords...) || t.kind == tQuoted) && p.toks[p.i+1].kind == '=' {
		key := p.id()
		p.next()
		return stmt{kind: '=', line: t.line, attrs: []attr{{key, p.value(key), t.line}}}
	}
	first := p.operand(true)
	if k := p.peek().kind; k == tArrow || k == tLine {
		ends := []operand{first}
		for k := p.peek().kind; k == tArrow || k == tLine; k = p.peek().kind {
			op := p.next()
			switch {
			case k == tLine && p.directed:
				p.fail(op.line, "-- joins nodes in a graph; a digraph links them with ->")
			case k == tArrow && !p.directed:
				p.fail(op.line, "-> links nodes in a digraph; a graph joins them with --")
			}
			if n := p.peek(); n.kind != tID && n.kind != tQuoted && n.kind != '{' {
				p.fail(op.line, "the edge after %s goes to no node", ends[len(ends)-1].name())
			}
			ends = append(ends, p.operand(false))
		}
		return stmt{kind: 'e', line: t.line, ends: ends, attrs: p.attrs()}
	}
	if first.sub != nil {
		return stmt{kind: 's', line: t.line, ends: []operand{first}}
	}
	return stmt{kind: 'n', line: t.line, ends: []operand{first}, attrs: p.attrs()}
}

// name returns how a message names o.
func (o operand) name() string {
	if o.sub != nil {
		return "the subgraph"
	}
	return o.id
}

// operand reads a node's ID, with a port after it, which it skips as
// edges end where they meet the node, or a subgraph.
func (p *parser) operand(first bool) operand {
	t := p.peek()
	switch {
	case t.kind == '{' || keyword(t, "subgraph"):
		return operand{sub: p.subgraph()}
	case t.kind == tHTML:
		p.fail(t.line, "HTML labels are not supported")
	case keyword(t, keywords...):
		p.fail(t.line, "%s is a keyword: put it in quotes to name a node", t.text)
	case t.kind == tID || t.kind == tQuoted:
		id := p.id()
		for p.peek().kind == ':' {
			colon := p.next()
			if k := p.peek().kind; k != tID && k != tQuoted {
				p.fail(colon.line, "the port of %s has no name", id)
			}
			p.id()
		}
		return operand{id: id}
	}
	if first {
		p.fail(t.line, "%q does not start a statement", t.text)
	}
	p.fail(t.line, "%q is not a node", t.text)
	return operand{}
}

func (p *parser) subgraph() *sub {
	sb := &sub{}
	if t := p.peek(); keyword(t, "subgraph") {
		p.next()
		if n := p.peek(); n.kind == tID && !keyword(n, keywords...) || n.kind == tQuoted {
			sb.id = p.id()
		}
		if p.peek().kind != '{' {
			p.fail(t.line, "the subgraph has no {")
		}
	}
	open := p.next()
	sb.body = p.body()
	if p.peek().kind != '}' {
		p.fail(open.line, "the subgraph's { has no }")
	}
	p.next()
	return sb
}

// attrs reads attribute lists, as [a=1, b=2][c=3], if there are any.
func (p *parser) attrs() []attr {
	var out []attr
	for p.peek().kind == '[' {
		open := p.next()
		for p.peek().kind != ']' {
			t := p.peek()
			switch t.kind {
			case tEOF:
				p.fail(open.line, "the [ has no ]")
			case tID, tQuoted:
			default:
				p.fail(t.line, "%q is not an attribute", t.text)
			}
			key := p.id()
			if p.peek().kind != '=' {
				p.fail(t.line, "%s has no value, as %s=value", key, key)
			}
			p.next()
			out = append(out, attr{key, p.value(key), t.line})
			if k := p.peek().kind; k == ',' || k == ';' {
				p.next()
			}
		}
		p.next()
	}
	return out
}

func (p *parser) value(key string) string {
	switch t := p.peek(); t.kind {
	case tID, tQuoted:
		return p.id()
	case tHTML:
		p.fail(t.line, "HTML labels are not supported")
	default:
		p.fail(t.line, "%s has no value", key)
	}
	return ""
}

// id reads an ID, joining strings in quotes with + between them.
func (p *parser) id() string {
	t := p.next()
	s := t.text
	for t.kind == tQuoted && p.peek().kind == '+' && p.toks[p.i+1].kind == tQuoted {
		p.next()
		t = p.next()
		s += t.text
	}
	return s
}

// scope is where statements are: the defaults of the nodes and edges
// they make, the cluster they are in, and the cluster whose body they
// are, -1 for none. same is whether the subgraph keeps its nodes in one
// rank.
type scope struct {
	nodes, edges []attr
	cluster, own int
	root, same   bool
}

// builder makes a graph of its statements.
type builder struct {
	g        *graph
	ids      map[string]int
	clusters map[string]int
}

func (b *builder) fail(line int, msg string) {
	if msg != "" {
		b.g.errs = append(b.g.errs, problem{line, msg})
	}
}

// check notes what is wrong with attrs, set on a node or an edge that
// shows nowhere, so that a wrong value is told once.
func (b *builder) check(attrs []attr, edges bool) {
	for _, a := range attrs {
		if edges {
			b.fail(a.line, new(edge).set(a.key, a.val))
		} else {
			b.fail(a.line, new(node).set(a.key, a.val))
		}
	}
}

// run makes what body says in sc, and returns the nodes it names, in the
// order it first names them.
func (b *builder) run(body []stmt, sc *scope) []int {
	var in []int
	seen := map[int]bool{}
	mention := func(u int) {
		if !seen[u] {
			seen[u] = true
			in = append(in, u)
		}
	}
	for _, s := range body {
		switch s.kind {
		case 'd':
			switch s.of {
			case "node":
				b.check(s.attrs, false)
				sc.nodes = slices.Concat(sc.nodes, s.attrs)
			case "edge":
				b.check(s.attrs, true)
				sc.edges = slices.Concat(sc.edges, s.attrs)
			default:
				for _, a := range s.attrs {
					b.fail(a.line, b.setGraph(sc, a.key, a.val))
				}
			}
		case '=':
			a := s.attrs[0]
			b.fail(a.line, b.setGraph(sc, a.key, a.val))
		case 'n':
			b.check(s.attrs, false)
			u := b.node(s.ends[0].id, sc)
			for _, a := range s.attrs {
				b.g.nodes[u].set(a.key, a.val)
			}
			mention(u)
		case 's':
			for _, u := range b.sub(s.ends[0].sub, sc) {
				mention(u)
			}
		case 'e':
			b.check(s.attrs, true)
			var prev []int
			for k, o := range s.ends {
				cur := []int{}
				if o.sub != nil {
					cur = b.sub(o.sub, sc)
				} else {
					cur = append(cur, b.node(o.id, sc))
				}
				for _, u := range cur {
					mention(u)
				}
				if k > 0 {
					for _, from := range prev {
						for _, to := range cur {
							b.edge(from, to, sc.edges, s.attrs, s.line)
						}
					}
				}
				prev = cur
			}
		}
	}
	return in
}

// sub makes subgraph sb in parent, and returns its nodes. A subgraph whose
// ID starts with cluster is a cluster, and one ID names one cluster
// however many times it is written.
func (b *builder) sub(sb *sub, parent *scope) []int {
	sc := &scope{nodes: parent.nodes, edges: parent.edges, cluster: parent.cluster, own: -1}
	if len(sb.id) >= 7 && strings.EqualFold(sb.id[:7], "cluster") {
		k, ok := b.clusters[sb.id]
		if !ok {
			k = len(b.g.clusters)
			b.clusters[sb.id] = k
			b.g.clusters = append(b.g.clusters, cluster{id: sb.id, look: look{pen: 1}, parent: parent.cluster})
		}
		sc.cluster, sc.own = k, k
	}
	in := b.run(sb.body, sc)
	if sc.same && len(in) > 1 {
		b.g.sames = append(b.g.sames, in)
	}
	return in
}

func (b *builder) setGraph(sc *scope, key, val string) string {
	switch key {
	case "rankdir":
		d, ok := rankdirs[strings.ToUpper(strings.TrimSpace(val))]
		if !ok {
			return fmt.Sprintf("%q is not a rankdir: TB, LR, BT or RL", val)
		}
		if sc.root {
			b.g.dir = d
		}
	case "rank":
		sc.same = val == "same"
	default:
		l := &look{}
		if sc.own >= 0 {
			l = &b.g.clusters[sc.own].look
		}
		return l.set(key, val)
	}
	return ""
}

// node returns the node of id, making it in sc with sc's defaults if it is
// new. A node named again in a cluster inside its own moves into it.
func (b *builder) node(id string, sc *scope) int {
	g := b.g
	u, ok := b.ids[id]
	if !ok {
		u = len(g.nodes)
		b.ids[id] = u
		n := node{id: id, look: look{label: `\N`, pen: 1}, cluster: sc.cluster}
		for _, a := range sc.nodes {
			n.set(a.key, a.val)
		}
		g.nodes = append(g.nodes, n)
		return u
	}
	if c := g.nodes[u].cluster; c != sc.cluster && g.within(sc.cluster, c) {
		g.nodes[u].cluster = sc.cluster
	}
	return u
}

// edge makes an edge from one node to another; in a strict graph, an edge
// between nodes that have one takes its attributes instead.
func (b *builder) edge(from, to int, defaults, attrs []attr, line int) {
	g := b.g
	k := -1
	if g.strict {
		k = slices.IndexFunc(g.edges, func(e edge) bool {
			return e.from == from && e.to == to || !g.directed && e.from == to && e.to == from
		})
	}
	if k < 0 {
		e := edge{from: from, to: to, line: line, look: look{pen: 1}}
		if g.directed {
			e.dir = 1
		}
		for _, a := range defaults {
			e.set(a.key, a.val)
		}
		k = len(g.edges)
		g.edges = append(g.edges, e)
	}
	for _, a := range attrs {
		g.edges[k].set(a.key, a.val)
	}
}

// parse reads src, skipping the statements that are wrong.
func parse(src string) *graph {
	toks, errs := lex(src)
	p := &parser{toks: toks, errs: errs}
	g := &graph{}
	t := p.peek()
	if keyword(t, "strict") {
		g.strict = true
		p.next()
		t = p.peek()
	}
	headed := keyword(t, "digraph", "graph")
	if headed {
		g.directed = keyword(t, "digraph")
		p.next()
		if n := p.peek(); n.kind == tID && !keyword(n, keywords...) || n.kind == tQuoted {
			g.id = p.id()
		}
	} else {
		// Read the statements as a digraph's, to show what they say.
		p.errs = append(p.errs, problem{t.line, "the source does not start with graph or digraph, as digraph name {"})
		g.directed = true
	}
	p.directed = g.directed
	open := p.peek()
	if open.kind == '{' {
		p.next()
	} else if headed {
		p.errs = append(p.errs, problem{open.line, "the graph has no {"})
	}
	body := p.body()
	if p.peek().kind == '}' {
		p.next()
		if t := p.peek(); t.kind != tEOF {
			p.errs = append(p.errs, problem{t.line, "the graph has ended: this is after its }"})
		}
	} else if open.kind == '{' {
		p.errs = append(p.errs, problem{open.line, "the graph's { has no }"})
	}

	g.errs = p.errs
	b := &builder{g: g, ids: map[string]int{}, clusters: map[string]int{}}
	b.run(body, &scope{cluster: -1, own: -1, root: true})
	for i := range g.nodes {
		n := &g.nodes[i]
		n.text = label(n.label, n.id, g.id)
	}
	for i := range g.edges {
		e := &g.edges[i]
		e.text = label(e.label, "", g.id)
		if e.from == e.to {
			g.nodes[e.from].loops = append(g.nodes[e.from].loops, i)
		}
	}
	for k := range g.clusters {
		cl := &g.clusters[k]
		cl.text = label(cl.label, cl.id, g.id)
	}
	slices.SortStableFunc(g.errs, func(a, b problem) int { return cmp.Compare(a.line, b.line) })
	return g
}
