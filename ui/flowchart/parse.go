package flowchart

import (
	"cmp"
	"fmt"
	"regexp"
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

var directions = map[string]direction{"TB": down, "TD": down, "BT": up, "LR": right, "RL": left}

type shape uint8

const (
	rect shape = iota
	round
	stadium
	subroutine
	cylinder
	circle
	double
	diamond
	hexagon
	flag
	lean
	leanLeft
	trapezoid
	inverted
)

// brackets are what may follow a node's id to give it a shape, each
// opening with the ways it may close: the longest first, so that (( is
// not read as (.
var brackets = []struct {
	open   string
	closes []string
	shapes []shape
}{
	{"(((", []string{")))"}, []shape{double}},
	{"([", []string{"])"}, []shape{stadium}},
	{"((", []string{"))"}, []shape{circle}},
	{"(", []string{")"}, []shape{round}},
	{"[[", []string{"]]"}, []shape{subroutine}},
	{"[(", []string{")]"}, []shape{cylinder}},
	{"[/", []string{"/]", `\]`}, []shape{lean, trapezoid}},
	{`[\`, []string{`\]`, "/]"}, []shape{leanLeft, inverted}},
	{"[", []string{"]"}, []shape{rect}},
	{"{{", []string{"}}"}, []shape{hexagon}},
	{"{", []string{"}"}, []shape{diamond}},
	{">", []string{"]"}, []shape{flag}},
}

type stroke uint8

const (
	solid stroke = iota
	dotted
	thick
	hidden
)

type head uint8

const (
	bare head = iota
	arrow
	dot
	cross
)

// link is how a link is drawn: its stroke, what ends it at either end,
// how many ranks it spans at least, and its text.
type link struct {
	stroke stroke
	heads  [2]head
	length int
	text   string
}

type problem struct {
	line int
	msg  string
}

// ref is a node as a statement names it: its id, and the shape and text
// in brackets after it, if any.
type ref struct {
	id     string
	shape  shape
	text   string
	shaped bool
}

var breaks = regexp.MustCompile(`(?i)<br\s*/?>`)

// label returns s as it shows: <br> breaks a line.
func label(s string) string {
	lines := breaks.Split(s, -1)
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	return strings.Join(lines, "\n")
}

// unquote returns s without the double quotes around it.
func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

func idRune(r rune) bool { return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) }

// idLen returns how long the id at the start of s is. A dash joins words,
// as in api-gateway, but does not start a link.
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

// readNode reads the node at the start of s, and returns the rest.
func readNode(s string) (ref, string, string) {
	n := idLen(s)
	if n == 0 {
		return ref{}, s, fmt.Sprintf("%q is not a node", snippet(s))
	}
	r := ref{id: s[:n]}
	s = s[n:]
	missing := ""
	for _, b := range brackets {
		if !strings.HasPrefix(s, b.open) {
			continue
		}
		text, k, rest, ok := enclosed(s[len(b.open):], b.closes)
		if !ok {
			if missing == "" {
				missing = b.closes[0]
			}
			// The next bracket may still close, as [ does for [(text].
			continue
		}
		r.shape, r.text, r.shaped = b.shapes[k], label(text), true
		s = rest
		break
	}
	if !r.shaped && missing != "" {
		return r, s, fmt.Sprintf("the text of %s has no %s", r.id, missing)
	}
	if rest, ok := strings.CutPrefix(s, ":::"); ok {
		s = rest[idLen(rest):]
	}
	return r, s, ""
}

// enclosed reads text up to the first of closes, in quotes or not, and
// returns which closed it and what follows.
func enclosed(s string, closes []string) (text string, k int, rest string, ok bool) {
	if q := strings.TrimLeft(s, " "); strings.HasPrefix(q, `"`) {
		end := strings.IndexByte(q[1:], '"')
		if end < 0 {
			return "", 0, "", false
		}
		after := strings.TrimLeft(q[end+2:], " ")
		for k, cl := range closes {
			if strings.HasPrefix(after, cl) {
				return q[1 : end+1], k, after[len(cl):], true
			}
		}
		return "", 0, "", false
	}
	at := -1
	for i, cl := range closes {
		if j := strings.Index(s, cl); j >= 0 && (at < 0 || j < at) {
			at, k = j, i
		}
	}
	if at < 0 {
		return "", 0, "", false
	}
	return s[:at], k, s[at+len(closes[k]):], true
}

// readLink reads the link at the start of s, and returns the rest; ok is
// false when s does not start with one.
func readLink(s string) (l link, rest string, ok bool, err string) {
	l.length = 1
	i := 0
	headAt := func(c byte) head {
		switch c {
		case '>':
			return arrow
		case 'o':
			return dot
		case 'x':
			return cross
		}
		return bare
	}
	if len(s) > 1 && strings.ContainsRune("-=.", rune(s[1])) {
		switch s[0] {
		case '<':
			l.heads[0], i = arrow, 1
		case 'o', 'x':
			l.heads[0], i = headAt(s[0]), 1
		}
	}
	run := func(c byte) int {
		n := 0
		for i+n < len(s) && s[i+n] == c {
			n++
		}
		return n
	}
	end := func() bool {
		if i < len(s) {
			if h := headAt(s[i]); h != bare {
				l.heads[1] = h
				i++
				return true
			}
		}
		return false
	}
	switch {
	case i+1 < len(s) && s[i] == '-' && s[i+1] == '.':
		l.stroke = dotted
		i++
		if dots := run('.'); i+dots < len(s) && s[i+dots] == '-' {
			l.length = dots
			i += dots + 1
			end()
			break
		}
		i++
		j := strings.Index(s[i:], ".-")
		if j < 0 {
			return l, s, false, "the link's text has no end, as .->"
		}
		k := i + j
		for k > i && s[k-1] == '.' {
			k--
		}
		l.text, l.length = label(strings.TrimSpace(s[i:k])), i+j+1-k
		i += j + 2
		end()
	case i < len(s) && (s[i] == '-' || s[i] == '='):
		c := s[i]
		if c == '=' {
			l.stroke = thick
		}
		n := run(c)
		if n < 2 {
			return l, s, false, ""
		}
		i += n
		switch {
		case end():
			l.length = n - 1
		case n > 2:
			l.length = n - 2
		default:
			// The text runs up to the rest of the link, as in "-- text -->".
			j := strings.Index(s[i:], string([]byte{c, c}))
			if j < 0 {
				return l, s, false, fmt.Sprintf("the link's text has no end, as %c%c>", c, c)
			}
			l.text = label(strings.TrimSpace(s[i : i+j]))
			i += j
			m := run(c)
			i += m
			if end() {
				l.length = m - 1
			} else {
				l.length = max(1, m-2)
			}
		}
	case strings.HasPrefix(s[i:], "~~~"):
		l.stroke = hidden
		n := run('~')
		l.length = n - 2
		i += n
	default:
		return l, s, false, ""
	}
	rest = strings.TrimLeft(s[i:], " \t")
	if after, ok := strings.CutPrefix(rest, "|"); ok {
		j := strings.IndexByte(after, '|')
		if j < 0 {
			return l, s, false, "the link's text has no closing |"
		}
		l.text = label(unquote(strings.TrimSpace(after[:j])))
		rest = after[j+1:]
	}
	return l, rest, true, ""
}

// statements splits a line at the semicolons between its statements,
// leaving those in quotes and brackets.
func statements(line string) []string {
	var out []string
	depth, quoted, start := 0, false, 0
	for i := 0; i < len(line); i++ {
		switch c := line[i]; {
		case c == '"':
			quoted = !quoted
		case quoted:
		case c == '[' || c == '(' || c == '{':
			depth++
		case c == ']' || c == ')' || c == '}':
			depth = max(0, depth-1)
		case c == ';' && depth == 0:
			out = append(out, line[start:i])
			start = i + 1
		}
	}
	return append(out, line[start:])
}

// parser reads a chart's statements in turn: scope is the subgraph they
// are in, -1 for none, and open the subgraphs not yet ended.
type parser struct {
	ch    *chart
	ids   map[string]int
	scope int
	open  []int
}

// node returns the node r names, making it if it is new and giving it the
// shape r gives. A node is in the subgraph it is first named in.
func (p *parser) node(r ref) int {
	i, ok := p.ids[r.id]
	if !ok {
		i = len(p.ch.nodes)
		p.ids[r.id] = i
		p.ch.nodes = append(p.ch.nodes, node{id: r.id, text: r.id, group: -1})
	}
	n := &p.ch.nodes[i]
	if r.shaped {
		n.shape, n.text = r.shape, r.text
	}
	if n.group < 0 {
		n.group = p.scope
	}
	return i
}

// chain reads a statement of nodes and links between them, as
// "A & B --> C -- text --> D", adding it whole or, when it is wrong, not
// at all.
func (p *parser) chain(line int, s string) string {
	var groups [][]ref
	var links []link
	for {
		var group []ref
		for {
			r, rest, err := readNode(s)
			if err != "" {
				return err
			}
			group = append(group, r)
			s = strings.TrimLeft(rest, " \t")
			after, ok := strings.CutPrefix(s, "&")
			if !ok {
				break
			}
			s = strings.TrimLeft(after, " \t")
		}
		groups = append(groups, group)
		if s == "" {
			break
		}
		last := group[len(group)-1].id
		l, rest, ok, err := readLink(s)
		if err != "" {
			return err
		}
		if !ok {
			return fmt.Sprintf("%q after %s is not a link, as -->, ---, -.->, ==> or ~~~", snippet(s), last)
		}
		s = strings.TrimLeft(rest, " \t")
		if s == "" {
			return "the link after " + last + " goes to no node"
		}
		links = append(links, l)
	}
	ids := make([][]int, len(groups))
	for k, g := range groups {
		for _, r := range g {
			ids[k] = append(ids[k], p.node(r))
		}
	}
	for k, l := range links {
		for _, a := range ids[k] {
			for _, b := range ids[k+1] {
				p.ch.edges = append(p.ch.edges, edge{from: a, to: b, link: l, line: line, clip: [2]int{-1, -1}, scope: -1})
			}
		}
	}
	return ""
}

// parse reads the statements of src, skipping those that are wrong.
func parse(src string) *chart {
	ch := &chart{}
	p := &parser{ch: ch, ids: map[string]int{}, scope: -1}
	fail := func(line int, format string, args ...any) {
		ch.errs = append(ch.errs, problem{line, fmt.Sprintf(format, args...)})
	}
	first := true
	for n, raw := range strings.Split(src, "\n") {
		line := n + 1
		if t := strings.TrimSpace(raw); t == "" || strings.HasPrefix(t, "%%") {
			continue
		}
		for _, s := range statements(raw) {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			word, rest := s, ""
			if i := strings.IndexFunc(s, unicode.IsSpace); i >= 0 {
				word, rest = s[:i], strings.TrimSpace(s[i:])
			}
			// Only end in lower case ends a subgraph, so that End may name a
			// node, as in Mermaid.
			header := first
			first = false
			switch w := strings.ToLower(word); {
			case header && (w == "flowchart" || w == "graph"):
				if rest == "" {
					break
				}
				d, ok := directions[strings.ToUpper(rest)]
				if !ok {
					fail(line, "%q is not a direction: TB, TD, BT, LR or RL", rest)
				}
				ch.dir = d
			case w == "subgraph":
				g := group{parent: p.scope, line: line}
				if i := strings.IndexByte(rest, '['); i > 0 && strings.HasSuffix(rest, "]") {
					g.id, g.title = strings.TrimSpace(rest[:i]), unquote(strings.TrimSpace(rest[i+1:len(rest)-1]))
				} else if g.title = unquote(rest); !strings.ContainsAny(g.title, " \t") {
					g.id = g.title
				}
				if g.id == "" {
					g.id = fmt.Sprintf("subGraph%d", len(ch.groups))
				}
				g.title = label(g.title)
				p.open = append(p.open, len(ch.groups))
				p.scope = len(ch.groups)
				ch.groups = append(ch.groups, g)
			case word == "end":
				if len(p.open) == 0 {
					fail(line, "end closes no subgraph")
					continue
				}
				p.open = p.open[:len(p.open)-1]
				p.scope = ch.groups[p.scope].parent
			case w == "direction":
				if _, ok := directions[strings.ToUpper(rest)]; !ok {
					fail(line, "%q is not a direction: TB, TD, BT, LR or RL", rest)
				}
			case w == "classdef" || w == "class" || w == "style" || w == "linkstyle" || w == "click" ||
				strings.HasPrefix(w, "acctitle") || strings.HasPrefix(w, "accdescr"):
			default:
				if err := p.chain(line, s); err != "" {
					fail(line, "%s", err)
				}
			}
		}
	}
	for k := len(p.open) - 1; k >= 0; k-- {
		g := ch.groups[p.open[k]]
		fail(g.line, "subgraph %s has no end", g.id)
	}
	resolve(ch, p.ids, fail)
	slices.SortStableFunc(ch.errs, func(a, b problem) int { return cmp.Compare(a.line, b.line) })
	return ch
}

// resolve turns the nodes named by a subgraph's id into that subgraph,
// the subgraphs with no nodes into nodes, and the links to subgraphs into
// links to a node in them, cut where they meet the subgraph's box.
func resolve(ch *chart, ids map[string]int, fail func(int, string, ...any)) {
	alias := make([]int, len(ch.nodes))
	for i := range alias {
		alias[i] = -1
	}
	for g, gr := range ch.groups {
		if i, ok := ids[gr.id]; ok {
			alias[i] = g
		}
	}
	// ends are what each end of each edge is: a node, or a subgraph as
	// ^g, the bitwise complement of its index.
	ends := make([][2]int, len(ch.edges))
	var nodes []node
	index := make([]int, len(ch.nodes))
	for i, n := range ch.nodes {
		index[i] = len(nodes)
		if alias[i] < 0 {
			nodes = append(nodes, n)
		}
	}
	for k, e := range ch.edges {
		for j, i := range [2]int{e.from, e.to} {
			ends[k][j] = index[i]
			if alias[i] >= 0 {
				ends[k][j] = ^alias[i]
			}
		}
	}
	ch.nodes = nodes

	// A subgraph with no nodes shows as a node of its title; those in it
	// come after it, so they turn first.
	becomes := make([]int, len(ch.groups))
	for g := len(ch.groups) - 1; g >= 0; g-- {
		becomes[g] = -1
		if slices.ContainsFunc(ch.nodes, func(n node) bool { return ch.within(n.group, g) }) {
			continue
		}
		gr := ch.groups[g]
		becomes[g] = len(ch.nodes)
		ch.nodes = append(ch.nodes, node{id: gr.id, text: cmp.Or(gr.title, gr.id), group: gr.parent})
	}
	var edges []edge
	for k, e := range ch.edges {
		for j := range 2 {
			if g := ^ends[k][j]; g >= 0 && becomes[g] >= 0 {
				ends[k][j] = becomes[g]
			}
		}
		in := func(j int) int {
			if ends[k][j] < 0 {
				return ^ends[k][j]
			}
			return ch.nodes[ends[k][j]].group
		}
		a, b := ends[k][0], ends[k][1]
		if a < 0 && ch.within(in(1), ^a) || b < 0 && ch.within(in(0), ^b) {
			fail(e.line, "a link goes between a subgraph and what is in it")
			continue
		}
		e.from, e.to = a, b
		edges = append(edges, e)
	}
	ch.edges = edges
	keep := make([]int, len(ch.groups))
	var groups []group
	for g, gr := range ch.groups {
		keep[g] = -1
		if becomes[g] < 0 {
			keep[g] = len(groups)
			groups = append(groups, gr)
		}
	}
	// A subgraph that turned into a node held none that stayed, and the
	// one around it stays.
	for g := range groups {
		if p := groups[g].parent; p >= 0 {
			groups[g].parent = keep[p]
		}
	}
	for i := range ch.nodes {
		if g := ch.nodes[i].group; g >= 0 {
			ch.nodes[i].group = keep[g]
		}
	}
	ch.groups = groups
	for k := range ch.edges {
		e := &ch.edges[k]
		if e.from < 0 {
			e.from = ^keep[^e.from]
		}
		if e.to < 0 {
			e.to = ^keep[^e.to]
		}
		if e.from < 0 {
			e.clip[0], e.from = ^e.from, ch.member(^e.from, false)
		}
		if e.to < 0 {
			e.clip[1], e.to = ^e.to, ch.member(^e.to, true)
		}
	}
	for k, e := range ch.edges {
		if e.from == e.to {
			ch.nodes[e.from].loops = append(ch.nodes[e.from].loops, k)
		}
	}
}

// member returns the node of subgraph g that links to it end at: the
// first that no link in g comes into, for a link coming in, else the last
// that no link in g leaves.
func (ch *chart) member(g int, in bool) int {
	var nodes []int
	for i, n := range ch.nodes {
		if ch.within(n.group, g) {
			nodes = append(nodes, i)
		}
	}
	if !in {
		slices.Reverse(nodes)
	}
	for _, i := range nodes {
		if !slices.ContainsFunc(ch.edges, func(e edge) bool {
			if e.from < 0 || e.to < 0 || e.from == e.to || e.clip != [2]int{-1, -1} {
				return false
			}
			if in {
				return e.to == i && ch.within(ch.nodes[e.from].group, g)
			}
			return e.from == i && ch.within(ch.nodes[e.to].group, g)
		}) {
			return i
		}
	}
	return nodes[0]
}

// within reports whether subgraph a, -1 for the chart, is g or in it.
func (ch *chart) within(a, g int) bool {
	for ; a >= 0; a = ch.groups[a].parent {
		if a == g {
			return true
		}
	}
	return false
}
