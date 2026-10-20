// Package statechart draws a state machine from text, as Mermaid's
// stateDiagram-v2 does, and runs it. The states are laid out from left to
// right, each after the states it comes from, and ordered so that few
// transitions cross; curved arrows go between them, named by their events,
// and loop over a state for those back to it. A filled dot marks where the
// machine starts and a ringed one where it ends, and the states of a
// composite state show in a box of its own.
//
// The state the machine is in is lit, and the arrows it can take are
// darker. Under the diagram, a button for each event it takes: a click, or
// the app calling Send, moves the machine along that transition, lighting
// the arrow as a dot runs along it to the next state, and View reports the
// step in the same frame. Beside the buttons, the last steps, newest
// first. A diagram too big to fit scrolls both ways. The lines that are
// wrong are listed under it, by number with what is wrong, and the diagram
// shows what the others say.
//
//	ev := statechart.View(c, &app.order, app.source)
//	if st, ok := ev.Moved(); ok {
//		app.log(st.From, st.Event, st.To)
//	}
//
// The source has a statement a line; lines starting with %% are
// comments:
//
//	stateDiagram-v2       the header, which may be left out
//	A --> B : event       a transition from A to B on event
//	A --> B               one without an event
//	[*] --> A             where the machine starts
//	A --> [*]             where it ends
//	state A               a state, in the order declared
//	state P { ... }       a composite state, around the lines up to }
//
// A state is in the composite state it is first named in. Inside one, [*]
// is where it starts and ends: entering P enters the state its [*] goes
// to, and a transition from P leaves it from any state in it that does not
// take the same event. Composite states nest one level.
package statechart

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// move is how long the machine takes along a transition, and history how
// many steps it lists.
const (
	move    = 450 * time.Millisecond
	history = 5
)

// Step is a move of a machine: the state it left, the event, "" for a
// transition without one, and the state it entered, [*] for an end.
type Step struct {
	From, Event, To string
}

// Events are what moved a machine in a frame. The zero Events report
// nothing.
type Events struct {
	step Step
	ok   bool
}

// Moved returns the step the machine took in this frame, by a click on an
// event's button or by Send.
func (e Events) Moved() (Step, bool) { return e.step, e.ok }

// State is what a view keeps: the chart as it parsed and laid out the
// source it was last given, the state the machine is in, its last steps,
// and the events sent to it. Keep it in the app's state, one for each
// view; the zero State starts the machine.
type State struct {
	key key
	ch  *chart
	// cur is the node the machine is in, plus 1, so that the zero State
	// starts it; lit is the edge it last took, plus 1, at the time at,
	// from the node from.
	cur, lit, from int
	at             time.Time
	steps          []Step
	sent           []string
}

// Send moves the machine along the transition its state takes for event,
// as a click on the event's button does, in the next View, which reports
// it. A state that takes no such event stays.
func (s *State) Send(event string) { s.sent = append(s.sent, event) }

// key is what a layout is for: the source, and the theme's sizes.
type key struct {
	src         string
	font, space float32
}

type kind uint8

const (
	simple kind = iota
	composite
	start
	end
)

type node struct {
	name   string
	kind   kind
	parent int
	// loops are the edges back to the node. tw is the width of its name,
	// lw that of its loop's label; w and h are its size, x and y its
	// center, top how far its loop reaches above it, and iw how wide the
	// states in a composite state are.
	loops                       []int
	tw, lw, w, h, x, y, top, iw float32
}

type edge struct {
	from, to int
	event    string
	line     int
	// scope is the composite state the edge is laid out in, -1 for the
	// chart; pts are the points its curve passes level, from where it
	// leaves to where it arrives; lx, ly is the center of its label, lw
	// its width.
	scope      int
	pts        [][2]float32
	lx, ly, lw float32
}

type problem struct {
	line int
	msg  string
}

type chart struct {
	nodes []node
	edges []edge
	names map[string]int
	errs  []problem
	w, h  float32
	z     sizes
}

// valid reports whether name can name a state.
func valid(name string) bool { return name != "" && !strings.ContainsAny(name, " \t[]{}:") }

// parse reads the statements of src, skipping those that are wrong.
func parse(src string) *chart {
	ch := &chart{names: map[string]int{}}
	fail := func(line int, format string, args ...any) {
		ch.errs = append(ch.errs, problem{line, fmt.Sprintf(format, args...)})
	}
	add := func(n node) int {
		ch.nodes = append(ch.nodes, n)
		return len(ch.nodes) - 1
	}
	scope := -1
	named := func(name string) int {
		if i, ok := ch.names[name]; ok {
			return i
		}
		i := add(node{name: name, parent: scope})
		ch.names[name] = i
		return i
	}
	// open are the braces not yet closed: the scope around each, and its
	// line. One that fails to open a composite state keeps the scope, so
	// that its } closes it alone.
	type block struct{ scope, line int }
	var open []block
	starts := map[int]int{}
	for n, raw := range strings.Split(src, "\n") {
		line := n + 1
		s := strings.TrimSpace(raw)
		switch {
		case s == "" || strings.HasPrefix(s, "%%") || s == "stateDiagram-v2" || s == "stateDiagram":
		case s == "}":
			if len(open) == 0 {
				fail(line, "} closes no composite state")
				continue
			}
			scope, open = open[len(open)-1].scope, open[:len(open)-1]
		case strings.HasPrefix(s, "state "):
			name, brace := strings.CutSuffix(strings.TrimSpace(s[len("state "):]), "{")
			name = strings.TrimSpace(name)
			if brace {
				open = append(open, block{scope, line})
			}
			switch {
			case !valid(name):
				fail(line, "%q is not a state's name", name)
			case !brace:
				named(name)
			case scope >= 0:
				fail(line, "composite states nest one level: this is in %s", ch.nodes[scope].name)
			default:
				i := named(name)
				if p := ch.nodes[i].parent; p >= 0 {
					fail(line, "composite states nest one level: %s is in %s", name, ch.nodes[p].name)
					continue
				}
				ch.nodes[i].kind = composite
				scope = i
			}
		default:
			head, event, _ := strings.Cut(s, ":")
			a, b, ok := strings.Cut(head, "-->")
			a, b, event = strings.TrimSpace(a), strings.TrimSpace(b), strings.TrimSpace(event)
			switch {
			case !ok:
				fail(line, "%q is not a state, a transition or a composite state", s)
				continue
			case a != "[*]" && !valid(a):
				fail(line, "%q is not a state's name", a)
				continue
			case b != "[*]" && !valid(b):
				fail(line, "%q is not a state's name", b)
				continue
			}
			var from, to int
			if a == "[*]" {
				i, ok := starts[scope]
				if !ok {
					i = add(node{name: "[*]", kind: start, parent: scope})
					starts[scope] = i
				}
				from = i
			} else {
				from = named(a)
			}
			// Each end is a node of its own, beside the state it ends,
			// rather than one that transitions from all over cross to.
			if b == "[*]" {
				to = add(node{name: "[*]", kind: end, parent: scope})
			} else {
				to = named(b)
			}
			ch.edges = append(ch.edges, edge{from: from, to: to, event: event, line: line})
		}
	}
	for k := len(open) - 1; k >= 0; k-- {
		fail(open[k].line, "the composite state has no }")
	}
	ch.edges = slices.DeleteFunc(ch.edges, func(e edge) bool {
		if ch.nodes[e.from].parent == e.to || ch.nodes[e.to].parent == e.from {
			fail(e.line, "a transition goes between a composite state and a state in it")
			return true
		}
		return false
	})
	for i, e := range ch.edges {
		if e.from == e.to {
			ch.nodes[e.from].loops = append(ch.nodes[e.from].loops, i)
		}
	}
	slices.SortStableFunc(ch.errs, func(a, b problem) int { return cmp.Compare(a.line, b.line) })
	return ch
}

// enter returns the node the machine is in once it enters n: the state a
// composite state starts in, else n.
func (ch *chart) enter(n int) int {
	if ch.nodes[n].kind == composite {
		for _, e := range ch.edges {
			if f := ch.nodes[e.from]; f.kind == start && f.parent == n {
				return e.to
			}
		}
	}
	return n
}

// initial returns the node the machine starts in: where the chart's [*]
// goes, else its first state; -1 when it has none.
func (ch *chart) initial() int {
	for _, e := range ch.edges {
		if f := ch.nodes[e.from]; f.kind == start && f.parent < 0 {
			return ch.enter(e.to)
		}
	}
	for i, n := range ch.nodes {
		if n.parent < 0 && (n.kind == simple || n.kind == composite) {
			return ch.enter(i)
		}
	}
	return -1
}

// takes returns the edges the machine takes from node n: n's own, then
// those of the composite state around it for events n does not take.
func (ch *chart) takes(n int) []int {
	var out []int
	for i, e := range ch.edges {
		if e.from == n {
			out = append(out, i)
		}
	}
	own := len(out)
	if p := ch.nodes[n].parent; p >= 0 {
		for i, e := range ch.edges {
			if e.from == p && !slices.ContainsFunc(out[:own], func(k int) bool { return ch.edges[k].event == e.event }) {
				out = append(out, i)
			}
		}
	}
	return out
}

// loopLabel returns the events of the transitions from n back to it.
func (ch *chart) loopLabel(n node) string {
	var events []string
	for _, i := range n.loops {
		if e := ch.edges[i].event; e != "" {
			events = append(events, e)
		}
	}
	return strings.Join(events, ", ")
}

// sizes are the sizes of the parts of a chart, from the theme.
type sizes struct {
	th, lh, padX, boxH, radius, dot, gapX, gapY, lane, port, inset, title, loopH, pad float32
}

func measure(c *ui.Context) sizes {
	t := c.Theme()
	_, th := c.MeasureText(0, ui.Span{Text: "Ag", Size: t.FontSize})
	_, sh := c.MeasureText(0, ui.Span{Text: "Ag", Size: t.FontSize - 1})
	return sizes{th: th, lh: sh + t.Space(1.5), padX: t.Space(3), boxH: th + 2*t.Space(2.5), radius: t.Space(2),
		dot: t.Space(4), gapX: t.Space(3), gapY: t.Space(5), lane: t.Space(2), port: t.Space(2.5), inset: t.Space(4),
		title: th + t.Space(4), loopH: t.Space(6), pad: t.Space(5)}
}

// layout sizes the nodes and labels, lays out the states of each composite
// state and then the chart, and moves them where they show.
func layout(c *ui.Context, ch *chart) {
	t := c.Theme()
	z := measure(c)
	ch.z = z
	width := func(s string, size float32, weight int) float32 {
		w, _ := c.MeasureText(0, ui.Span{Text: s, Size: size, Weight: weight})
		return w
	}
	label := func(s string) float32 { return width(s, t.FontSize-1, 400) + 2*t.Space(1.5) }
	for i := range ch.edges {
		if e := &ch.edges[i]; e.event != "" {
			e.lw = label(e.event)
		}
	}
	for i := range ch.nodes {
		n := &ch.nodes[i]
		switch n.kind {
		case simple:
			n.tw = width(n.name, t.FontSize, 500)
			n.w, n.h = max(n.tw+2*z.padX, t.Space(16)), z.boxH
		case start, end:
			n.w, n.h = z.dot, z.dot
		}
		if len(n.loops) > 0 {
			n.top = z.loopH + z.lh/2
			if l := ch.loopLabel(*n); l != "" {
				n.lw = label(l)
			}
		}
	}
	for i := range ch.nodes {
		n := &ch.nodes[i]
		if n.kind != composite {
			continue
		}
		w, h := place(ch, z, i)
		n.tw = width(n.name, t.FontSize-1, 600)
		n.iw = w
		n.w, n.h = max(w, n.tw)+2*z.inset, z.title+h+z.inset
	}
	w, h := place(ch, z, -1)
	ch.w, ch.h = w+2*z.pad, h+2*z.pad
	shift := func(scope int) (float32, float32) {
		if scope < 0 {
			return z.pad, z.pad
		}
		p := ch.nodes[scope]
		return p.x - p.iw/2, p.y - p.h/2 + z.title
	}
	for _, inner := range []bool{false, true} {
		for i := range ch.nodes {
			if n := &ch.nodes[i]; (n.parent >= 0) == inner {
				dx, dy := shift(n.parent)
				n.x, n.y = n.x+dx, n.y+dy
			}
		}
	}
	side := func(pt [2]float32, n node) [2]float32 {
		if pt[0] > ch.nodes[n.parent].x {
			return [2]float32{n.x + n.w/2, n.y}
		}
		return [2]float32{n.x - n.w/2, n.y}
	}
	for i := range ch.edges {
		e := &ch.edges[i]
		if e.pts == nil {
			continue
		}
		dx, dy := shift(e.scope)
		for k := range e.pts {
			e.pts[k][0], e.pts[k][1] = e.pts[k][0]+dx, e.pts[k][1]+dy
		}
		e.lx, e.ly = e.lx+dx, e.ly+dy
		if e.scope >= 0 {
			continue
		}
		if a := ch.nodes[e.from]; a.parent >= 0 {
			e.pts[0] = side(e.pts[0], a)
		}
		if b := ch.nodes[e.to]; b.parent >= 0 {
			e.pts[len(e.pts)-1] = side(e.pts[len(e.pts)-1], b)
		}
	}
}

// place lays out the states in scope, -1 for the chart, and the
// transitions between them from left to right, as Sugiyama's method does:
// it turns around the transitions that close a cycle, puts each state a
// column after those it comes from, gives each transition a point in each
// column it passes and its label one in a column between states, orders
// the columns so that few transitions cross, and sets each point level
// with those it links to. It returns the size of what it laid out, from 0,
// 0, where the nodes and edges of scope now are.
func place(ch *chart, z sizes, scope int) (float32, float32) {
	type vert struct {
		node, layer int
		w, up, down float32
		at          float64
		y           float32
	}
	// A link is an edge in scope: the points it passes from the left, the
	// point of its label, -1 for none, and whether it goes right to left.
	// An end of it in a composite state is inner, off below the center of
	// the composite state.
	type link struct {
		edge, label int
		rev         bool
		path        []int
		inner       [2]bool
		off         [2]float32
	}
	var vs []vert
	of := map[int]int{}
	for i, n := range ch.nodes {
		if n.parent == scope {
			of[i] = len(vs)
			vs = append(vs, vert{node: i, w: max(n.w, n.lw), up: n.h/2 + n.top, down: n.h / 2})
		}
	}
	if len(vs) == 0 {
		return 0, 0
	}
	end := func(n int) (u int, inner bool, off float32, ok bool) {
		p := ch.nodes[n].parent
		if p == scope {
			return of[n], false, 0, true
		}
		if p < 0 || ch.nodes[p].parent != scope {
			return 0, false, 0, false
		}
		return of[p], true, z.title + ch.nodes[n].y - ch.nodes[p].h/2, true
	}
	var links []link
	for i, e := range ch.edges {
		u, iu, ou, ok := end(e.from)
		v, iv, ov, ok2 := end(e.to)
		if ok && ok2 && u != v {
			links = append(links, link{edge: i, label: -1, path: []int{u, v}, inner: [2]bool{iu, iv}, off: [2]float32{ou, ov}})
		}
	}
	out := make([][]int, len(vs))
	for k, l := range links {
		out[l.path[0]] = append(out[l.path[0]], k)
	}
	mark := make([]uint8, len(vs))
	var walk func(u int)
	walk = func(u int) {
		mark[u] = 1
		for _, k := range out[u] {
			switch v := links[k].path[1]; mark[v] {
			case 0:
				walk(v)
			case 1:
				links[k].rev = true
			}
		}
		mark[u] = 2
	}
	for _, fromStart := range []bool{true, false} {
		for u, v := range vs {
			if mark[u] == 0 && (!fromStart || ch.nodes[v.node].kind == start) {
				walk(u)
			}
		}
	}
	rank := make([]int, len(vs))
	for k := range links {
		if l := &links[k]; l.rev {
			l.path[0], l.path[1] = l.path[1], l.path[0]
			l.inner[0], l.inner[1] = l.inner[1], l.inner[0]
			l.off[0], l.off[1] = l.off[1], l.off[0]
		}
	}
	for range vs {
		for _, l := range links {
			rank[l.path[1]] = max(rank[l.path[1]], rank[l.path[0]]+1)
		}
	}
	layers := 0
	for u := range vs {
		vs[u].layer = 2 * rank[u]
		layers = max(layers, vs[u].layer+1)
	}
	type segment struct {
		a, b   int
		oa, ob float32
	}
	var segs []segment
	for k := range links {
		l := &links[k]
		a, b := l.path[0], l.path[1]
		mid := (vs[a].layer + vs[b].layer) / 2
		if mid%2 == 0 {
			mid--
		}
		l.path = []int{a}
		for L := vs[a].layer + 1; L < vs[b].layer; L++ {
			d := vert{node: -1, layer: L}
			if e := ch.edges[l.edge]; L == mid && e.event != "" {
				d.w, d.up, d.down = e.lw, z.lh/2, z.lh/2
				l.label = len(vs)
			}
			l.path = append(l.path, len(vs))
			vs = append(vs, d)
		}
		l.path = append(l.path, b)
		for i := 1; i < len(l.path); i++ {
			s := segment{a: l.path[i-1], b: l.path[i]}
			if i == 1 {
				s.oa = l.off[0]
			}
			if i == len(l.path)-1 {
				s.ob = l.off[1]
			}
			segs = append(segs, s)
		}
	}

	cols := make([][]int, layers)
	for u := range vs {
		vs[u].at = float64(len(cols[vs[u].layer]))
		cols[vs[u].layer] = append(cols[vs[u].layer], u)
	}
	// pos is where in its column a link meets u: an inner end ranks by
	// where it is in the composite state.
	pos := func(u int, off float32) float64 {
		if off == 0 {
			return vs[u].at
		}
		return vs[u].at + float64(off/(vs[u].up+vs[u].down))
	}
	sweep := func(col []int, down bool) {
		key := make(map[int]float64, len(col))
		for _, u := range col {
			var near []float64
			for _, s := range segs {
				if down && s.b == u {
					near = append(near, pos(s.a, s.oa))
				} else if !down && s.a == u {
					near = append(near, pos(s.b, s.ob))
				}
			}
			key[u] = vs[u].at
			if len(near) > 0 {
				slices.Sort(near)
				key[u] = near[len(near)/2]
			}
		}
		slices.SortStableFunc(col, func(a, b int) int { return cmp.Compare(key[a], key[b]) })
		for k, u := range col {
			vs[u].at = float64(k)
		}
	}
	crossings := func() int {
		n := 0
		for i, s := range segs {
			for _, r := range segs[i+1:] {
				if vs[s.a].layer == vs[r.a].layer && (pos(s.a, s.oa)-pos(r.a, r.oa))*(pos(s.b, s.ob)-pos(r.b, r.ob)) < 0 {
					n++
				}
			}
		}
		return n
	}
	save := func() []float64 {
		o := make([]float64, len(vs))
		for u := range vs {
			o[u] = vs[u].at
		}
		return o
	}
	best, least := save(), crossings()
	for range 8 {
		for L := 1; L < layers; L++ {
			sweep(cols[L], true)
		}
		for L := layers - 2; L >= 0; L-- {
			sweep(cols[L], false)
		}
		if n := crossings(); n < least {
			best, least = save(), n
		}
	}
	for u := range vs {
		vs[u].at = best[u]
	}
	for _, col := range cols {
		slices.SortFunc(col, func(a, b int) int { return cmp.Compare(vs[a].at, vs[b].at) })
	}

	// level sets the points of col as near the heights their links want
	// as they can be in their order, apart by their sizes, by the pool
	// adjacent violators algorithm: the height of point i less how far it
	// must be below the first must not fall down the column.
	level := func(col []int, down, up bool) {
		off := make([]float32, len(col))
		for i := 1; i < len(col); i++ {
			a, b := col[i-1], col[i]
			gap := z.gapY
			if vs[a].node < 0 && vs[b].node < 0 {
				gap = z.lane
			}
			off[i] = off[i-1] + vs[a].down + vs[b].up + gap
		}
		type pool struct {
			sum, n float32
			first  int
		}
		var ps []pool
		for i, u := range col {
			var sum float32
			n := 0
			for _, s := range segs {
				if down && s.b == u {
					sum, n = sum+vs[s.a].y+s.oa-s.ob, n+1
				} else if up && s.a == u {
					sum, n = sum+vs[s.b].y+s.ob-s.oa, n+1
				}
			}
			want := vs[u].y
			if n > 0 {
				want = sum / float32(n)
			}
			ps = append(ps, pool{want - off[i], 1, i})
			for len(ps) > 1 && ps[len(ps)-2].sum/ps[len(ps)-2].n > ps[len(ps)-1].sum/ps[len(ps)-1].n {
				last := ps[len(ps)-1]
				ps = ps[:len(ps)-1]
				ps[len(ps)-1].sum += last.sum
				ps[len(ps)-1].n += last.n
			}
		}
		for k, p := range ps {
			end := len(col)
			if k+1 < len(ps) {
				end = ps[k+1].first
			}
			for i := p.first; i < end; i++ {
				vs[col[i]].y = p.sum/p.n + off[i]
			}
		}
	}
	for _, col := range cols {
		level(col, false, false)
	}
	for range 10 {
		for L := 1; L < layers; L++ {
			level(cols[L], true, false)
		}
		for L := layers - 2; L >= 0; L-- {
			level(cols[L], false, true)
		}
	}
	for range 4 {
		for _, col := range cols {
			level(col, true, true)
		}
	}

	top, bottom := float32(math.Inf(1)), float32(math.Inf(-1))
	for _, v := range vs {
		top, bottom = min(top, v.y-v.up), max(bottom, v.y+v.down)
	}
	xs := make([]float32, layers)
	ws := make([]float32, layers)
	for u := range vs {
		vs[u].y -= top
		ws[vs[u].layer] = max(ws[vs[u].layer], vs[u].w)
	}
	for L := range ws {
		if L%2 == 1 {
			ws[L] = max(ws[L], z.gapX)
		}
		xs[L] = ws[L] / 2
		if L > 0 {
			xs[L] = xs[L-1] + ws[L-1]/2 + z.gapX + ws[L]/2
		}
	}
	for _, v := range vs {
		if v.node >= 0 {
			ch.nodes[v.node].x, ch.nodes[v.node].y = xs[v.layer], v.y
		}
	}

	// Each side of a state spreads the links it has there, in the order of
	// the points they go to next; those of a composite state leave it
	// level where they can, and inner ends are where their states are.
	type port struct {
		link   int
		leaves bool
		toward float32
	}
	sides := make([][2][]port, len(vs))
	ys := make([][2]float32, len(links))
	for k, l := range links {
		p := l.path
		a, b := p[0], p[len(p)-1]
		ys[k] = [2]float32{vs[a].y + l.off[0], vs[b].y + l.off[1]}
		if !l.inner[0] {
			sides[a][1] = append(sides[a][1], port{k, true, vs[p[1]].y})
		}
		if !l.inner[1] {
			sides[b][0] = append(sides[b][0], port{k, false, vs[p[len(p)-2]].y})
		}
	}
	for u, sd := range sides {
		if vs[u].node < 0 {
			continue
		}
		n := ch.nodes[vs[u].node]
		for _, ports := range sd {
			slices.SortStableFunc(ports, func(a, b port) int { return cmp.Compare(a.toward, b.toward) })
			var step float32
			if n.kind == simple {
				step = min(z.port, (n.h-2*z.radius)/float32(max(len(ports)-1, 1)))
			}
			for i, p := range ports {
				y := vs[u].y + (float32(i)-float32(len(ports)-1)/2)*step
				if n.kind == composite {
					y = max(vs[u].y-n.h/2+z.title, min(p.toward, vs[u].y+n.h/2-z.inset))
				}
				if p.leaves {
					ys[p.link][0] = y
				} else {
					ys[p.link][1] = y
				}
			}
		}
	}
	for k, l := range links {
		e := &ch.edges[l.edge]
		p := l.path
		a, b := vs[p[0]], vs[p[len(p)-1]]
		pts := [][2]float32{{xs[a.layer] + ch.nodes[a.node].w/2, ys[k][0]}}
		for _, d := range p[1 : len(p)-1] {
			v := vs[d]
			x := xs[v.layer]
			// A link runs level past the states of a column, which may be
			// wide, and curves across a column of labels.
			if v.layer%2 == 0 {
				pts = append(pts, [2]float32{x - ws[v.layer]/2, v.y}, [2]float32{x + ws[v.layer]/2, v.y})
			} else {
				pts = append(pts, [2]float32{x, v.y})
			}
		}
		pts = append(pts, [2]float32{xs[b.layer] - ch.nodes[b.node].w/2, ys[k][1]})
		if l.rev {
			slices.Reverse(pts)
		}
		e.pts, e.scope = pts, scope
		if l.label >= 0 {
			e.lx, e.ly = xs[vs[l.label].layer], vs[l.label].y
		}
	}
	return xs[layers-1] + ws[layers-1]/2, bottom - top
}

// seg is a cubic Bézier curve: its ends, and its control points between.
type seg [4][2]float32

// curve returns the curve of edge i, moved by ox, oy: through its points,
// level at each, or for an edge back to its state a loop over the state.
func (ch *chart) curve(i int, ox, oy float32) []seg {
	e := ch.edges[i]
	if e.from == e.to {
		n := ch.nodes[e.from]
		x, top := ox+n.x, oy+n.y-n.h/2
		a, b := min(n.w/4, ch.z.loopH), ch.z.loopH/2
		// A cubic reaches 3/4 of the way to its control points.
		h := ch.z.loopH * 4 / 3
		return []seg{{{x + a, top}, {x + a + b, top - h}, {x - a - b, top - h}, {x - a, top}}}
	}
	var out []seg
	for k := 1; k < len(e.pts); k++ {
		p, q := e.pts[k-1], e.pts[k]
		dx := (q[0] - p[0]) / 2
		out = append(out, seg{{ox + p[0], oy + p[1]}, {ox + p[0] + dx, oy + p[1]}, {ox + q[0] - dx, oy + q[1]}, {ox + q[0], oy + q[1]}})
	}
	return out
}

// path returns the path of the curve sg.
func path(sg []seg) *ui.Path {
	var p ui.Path
	p.MoveTo(sg[0][0][0], sg[0][0][1])
	for _, s := range sg {
		p.CubeTo(s[1][0], s[1][1], s[2][0], s[2][1], s[3][0], s[3][1])
	}
	return &p
}

// along returns the point at f of the length of the curve sg.
func along(sg []seg, f float32) (float32, float32) {
	const steps = 16
	var pts [][2]float32
	for _, s := range sg {
		for k := 0; k <= steps; k++ {
			u := float32(k) / steps
			v := 1 - u
			a, b, c, d := v*v*v, 3*v*v*u, 3*v*u*u, u*u*u
			pts = append(pts, [2]float32{a*s[0][0] + b*s[1][0] + c*s[2][0] + d*s[3][0], a*s[0][1] + b*s[1][1] + c*s[2][1] + d*s[3][1]})
		}
	}
	lens := make([]float32, len(pts))
	for k := 1; k < len(pts); k++ {
		lens[k] = lens[k-1] + float32(math.Hypot(float64(pts[k][0]-pts[k-1][0]), float64(pts[k][1]-pts[k-1][1])))
	}
	want := f * lens[len(lens)-1]
	for k := 1; k < len(pts); k++ {
		if lens[k] >= want && lens[k] > lens[k-1] {
			u := (want - lens[k-1]) / (lens[k] - lens[k-1])
			return pts[k-1][0] + (pts[k][0]-pts[k-1][0])*u, pts[k-1][1] + (pts[k][1]-pts[k-1][1])*u
		}
	}
	last := pts[len(pts)-1]
	return last[0], last[1]
}

// take moves the machine along edge i.
func (s *State) take(c *ui.Context, i int) Events {
	ch := s.ch
	from, to := s.cur-1, ch.enter(ch.edges[i].to)
	s.cur, s.lit, s.from, s.at = to+1, i+1, from, c.Now()
	st := Step{From: ch.nodes[from].name, Event: ch.edges[i].event, To: ch.nodes[to].name}
	s.steps = append(s.steps, st)
	if len(s.steps) > history {
		s.steps = s.steps[len(s.steps)-history:]
	}
	return Events{st, true}
}

// View creates the chart that src describes, which runs from its [*], and
// under it the buttons of the events the state it is in takes and the
// last steps. It fills its parent; the chart scrolls when it does not fit.
func View(c *ui.Context, s *State, src string) Events {
	t := c.Theme()
	k := key{src, t.FontSize, t.Space(1)}
	if s.ch == nil || s.key != k {
		name := ""
		if s.ch != nil && s.cur > 0 {
			name = s.ch.nodes[s.cur-1].name
		}
		s.key, s.ch = k, parse(src)
		layout(c, s.ch)
		s.cur, s.lit = 0, 0
		if i, ok := s.ch.names[name]; ok {
			s.cur = s.ch.enter(i) + 1
		}
	}
	ch, z := s.ch, s.ch.z
	if s.cur == 0 {
		s.cur = ch.initial() + 1
	}
	var ev Events
	if len(s.sent) > 0 {
		event := s.sent[0]
		s.sent = s.sent[1:]
		if s.cur > 0 {
			for _, i := range ch.takes(s.cur - 1) {
				if ch.edges[i].event == event {
					ev = s.take(c, i)
					break
				}
			}
		}
		if len(s.sent) > 0 {
			c.AnimationFrame()
		}
	}
	reduce := c.Preferences().ReduceMotion
	ui.Column(c).Fill().Children(func() {
		ui.ScrollBoth(c).Grow(1).Shrink(1).Children(func() {
			ui.Box(c).Size(ch.w, ch.h).Draw(func(p *ui.Painter, r ui.Rect) {
				draw(p, t, s, r.X, r.Y, reduce)
			})
		})
		if len(ch.errs) > 0 {
			ui.Column(c).Gap(t.Space(1)).Padding(t.Space(2), t.Space(4)).Children(func() {
				for i, e := range ch.errs {
					ui.Text(c.Key(i), fmt.Sprintf("Line %d: %s", e.line, e.msg)).FontSize(t.FontSize - 1).TextColor(t.Danger)
				}
			})
		}
		ui.Divider(c)
		ui.Row(c).Gap(t.Space(10)).Padding(t.Space(3), t.Space(4)).AlignItems(ui.Start).Children(func() {
			ui.Column(c).Grow(1).Shrink(1).Gap(t.Space(2)).Children(func() {
				ui.Text(c, "State").FontSize(t.FontSize - 1).TextColor(t.TextMuted)
				if s.cur == 0 {
					ui.Text(c, "No states").TextColor(t.TextMuted)
					return
				}
				n := ch.nodes[s.cur-1]
				ui.Row(c).Gap(t.Space(2)).AlignItems(ui.End).Children(func() {
					title := n.name
					if n.kind == end {
						title = "Finished"
					}
					ui.Text(c, title).FontSize(t.FontSize + 3).Bold().SingleLine()
					if n.parent >= 0 {
						ui.Text(c, "in "+ch.nodes[n.parent].name).TextColor(t.TextMuted).SingleLine()
					}
				})
				takes := ch.takes(s.cur - 1)
				if len(takes) == 0 && n.kind != end {
					ui.Text(c, "Takes no events").TextColor(t.TextMuted)
				}
				ui.Row(c).Wrap().Gap(t.Space(2)).Children(func() {
					for _, i := range takes {
						e := ch.edges[i]
						text := e.event
						if text == "" {
							text = "→ " + ch.nodes[e.to].name
						}
						if ui.Button(c.Key(i), text).Clicked() {
							if ev.ok {
								s.sent = append(s.sent, e.event)
								c.AnimationFrame()
							} else {
								ev = s.take(c, i)
							}
						}
					}
				})
			})
			ui.Column(c).Width(t.Space(76)).MinHeight(float32(history+1) * (z.th + t.Space(1.5))).Gap(t.Space(1.5)).Children(func() {
				ui.Text(c, "History").FontSize(t.FontSize - 1).TextColor(t.TextMuted)
				if len(s.steps) == 0 {
					ui.Text(c, "No steps yet").TextColor(t.TextMuted)
				}
				for k := len(s.steps) - 1; k >= 0; k-- {
					st := s.steps[k]
					ink := t.TextMuted
					if k == len(s.steps)-1 {
						ink = t.Text
					}
					ui.Row(c.Key(k)).Gap(t.Space(3)).Children(func() {
						event := st.Event
						if event == "" {
							event = "—"
						}
						ui.Text(c, event).Width(t.Space(24)).FontWeight(500).TextColor(ink).SingleLine()
						ui.Text(c, st.From+" → "+st.To).TextColor(t.TextMuted).SingleLine().Shrink(1)
					})
				}
			})
		})
	})
	return ev
}

// draw paints the chart of s with its left at ox and top at oy, lighting
// the state the machine is in and the edge it last took, along which a
// dot runs while it moves.
func draw(p *ui.Painter, t *ui.Theme, s *State, ox, oy float32, reduce bool) {
	ch, z := s.ch, s.ch.z
	f := float32(1)
	if s.lit > 0 && !reduce {
		f = min(float32(p.Now().Sub(s.at))/float32(move), 1)
		if f < 1 {
			p.AnimationFrame()
		}
		f = ui.EaseInOut(f)
	}
	lit := make([]float32, len(ch.nodes))
	var takes []int
	if s.cur > 0 {
		lit[s.cur-1] += f
		takes = ch.takes(s.cur - 1)
	}
	if s.lit > 0 && s.from != s.cur-1 {
		lit[s.from] += 1 - f
	}
	for i, n := range ch.nodes {
		if n.parent >= 0 {
			lit[n.parent] = min(1, lit[n.parent]+lit[i])
		}
	}
	text := func(x, y float32, s string, size float32, weight int, color ui.Color) {
		p.RichText(x, y, 0, ui.Span{Text: s, Size: size, Weight: weight, Color: color})
	}
	tint := t.Text.Alpha(0.035)
	box := func(n node) ui.Rect { return ui.Rect{X: ox + n.x - n.w/2, Y: oy + n.y - n.h/2, W: n.w, H: n.h} }

	for i, n := range ch.nodes {
		if n.kind != composite {
			continue
		}
		r := box(n)
		p.Fill(r, tint, z.radius*1.5)
		p.Stroke(r, t.Border.Mix(t.Accent.Alpha(0.6), lit[i]), z.radius*1.5, 1)
		text(r.X+z.inset, r.Y+(z.title-z.th)/2+1, n.name, t.FontSize-1, 600, t.TextMuted.Mix(t.Accent, lit[i]))
	}

	type mark struct {
		x, y, w float32
		label   string
		ink     ui.Color
		inner   bool
	}
	var labels []mark
	for i, e := range ch.edges {
		sg := ch.curve(i, ox, oy)
		color, width, ink := t.Text.Alpha(0.3), float32(1.25), t.TextMuted
		if slices.Contains(takes, i) {
			color, width, ink = t.Text.Alpha(0.7), 1.5, t.Text
		}
		p.StrokePath(path(sg), width, color)
		head := color
		if i == s.lit-1 {
			p.StrokePath(path(sg).Trim(0, f), 2, t.Accent)
			head, ink = color.Mix(t.Accent, f), t.Accent
		}
		last := sg[len(sg)-1]
		arrowhead(p, t, last[3], last[2], head)
		switch {
		case e.from == e.to:
			n := ch.nodes[e.from]
			if l := ch.loopLabel(n); l != "" && i == n.loops[0] {
				labels = append(labels, mark{ox + n.x, oy + n.y - n.h/2 - z.loopH, n.lw, l, ink, n.parent >= 0})
			}
		case e.event != "":
			labels = append(labels, mark{ox + e.lx, oy + e.ly, e.lw, e.event, ink, e.scope >= 0})
		}
	}
	if s.lit > 0 && f < 1 {
		x, y := along(ch.curve(s.lit-1, ox, oy), f)
		var halo, dot ui.Path
		halo.Circle(x, y, t.Space(1.75))
		dot.Circle(x, y, t.Space(1.25))
		p.FillPath(&halo, t.Background)
		p.FillPath(&dot, t.Accent)
	}
	for _, l := range labels {
		r := ui.Rect{X: l.x - l.w/2, Y: l.y - z.lh/2, W: l.w, H: z.lh}
		p.Fill(r, t.Background, z.lh/2)
		if l.inner {
			p.Fill(r, tint, z.lh/2)
		}
		w, h := p.MeasureText(0, ui.Span{Text: l.label, Size: t.FontSize - 1})
		text(l.x-w/2, l.y-h/2, l.label, t.FontSize-1, 400, l.ink)
	}

	for i, n := range ch.nodes {
		x, y, k := ox+n.x, oy+n.y, lit[i]
		switch n.kind {
		case start:
			var dot ui.Path
			dot.Circle(x, y, z.dot/2)
			p.FillPath(&dot, t.Text.Alpha(0.85))
		case end:
			color := t.Text.Alpha(0.85).Mix(t.Accent, k)
			var ring, dot ui.Path
			ring.Circle(x, y, z.dot/2-0.75)
			dot.Circle(x, y, z.dot/2-t.Space(1.25))
			p.Fill(box(n), t.Background, z.dot/2)
			p.StrokePath(&ring, 1.5, color)
			p.FillPath(&dot, color)
		case simple:
			r := box(n)
			if k > 0 {
				p.Shadow(r, z.radius, 0, 0, t.Space(4), 0, t.Accent.Alpha(0.45*k))
			}
			p.Fill(r, t.Surface.Mix(t.Accent, k), z.radius)
			p.Stroke(r, t.Border.Mix(t.Accent, k), z.radius, 1)
			text(x-n.tw/2, y-z.th/2, n.name, t.FontSize, 500, t.Text.Mix(t.AccentText, k))
		}
	}
}

// arrowhead draws a filled arrowhead with its tip at tip, pointing away
// from from.
func arrowhead(p *ui.Painter, t *ui.Theme, tip, from [2]float32, color ui.Color) {
	dx, dy := tip[0]-from[0], tip[1]-from[1]
	d := float32(math.Hypot(float64(dx), float64(dy)))
	if d == 0 {
		dx, dy, d = 1, 0, 1
	}
	dx, dy = dx/d, dy/d
	l, h := t.Space(2.25), t.Space(1.25)
	bx, by := tip[0]-dx*l, tip[1]-dy*l
	var tri ui.Path
	tri.MoveTo(tip[0], tip[1]).LineTo(bx-dy*h, by+dx*h).LineTo(bx+dy*h, by-dx*h).Close()
	p.FillPath(&tri, color)
}
