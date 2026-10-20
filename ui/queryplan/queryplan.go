// Package queryplan shows how PostgreSQL ran a query, as explain.depesz.com
// and pev2 show the output of EXPLAIN: a tree of the plan's nodes, each
// with its type and the table or index it reads, the rows the planner
// estimated and those it got, a misestimate of ten times or more flagged
// with its factor, its loops, the time in it alone and with the nodes
// under it, and its cost; and a bar of its share of the time, red for the
// slowest. A plan without ANALYZE shows the estimates, its bars the share
// of the cost.
//
// A node's arrow, or Right and Left, fold and unfold the nodes under it. A
// click, or Up and Down, choose a node, and a panel beside the tree shows
// all its properties: its filter, index condition, sort keys, buffers and
// the rest, but the counters at zero, which EXPLAIN's text leaves out too;
// with none chosen, those of the plan, as its planning and execution time.
//
//	plan, err := queryplan.Parse(out) // once, not every frame
//	...
//	queryplan.View(c, &app.explain, plan).Grow(1)
package queryplan

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Plan is a query plan, as EXPLAIN writes it in JSON.
type Plan struct {
	roots    []*node
	props    []prop
	analyzed bool
	nodes    int
	// total is the time the plan took, or its cost without ANALYZE.
	total               float64
	planning, execution float64
}

// node is a step of a plan: its type and what it reads, its properties,
// and the nodes under it.
type node struct {
	title, on        string
	props            []prop
	kids             []*node
	est, rows, loops float64
	cost             float64
	// time is the time in the node and those under it, in all its loops,
	// and self the time in it alone, in milliseconds.
	time, self float64
	// share is the node's share of the time, or of the cost without
	// ANALYZE, and rank its place among the slowest, from 1, or 0.
	share float64
	rank  int
}

// prop is a property of a node or a plan, its value as it reads.
type prop struct {
	key, value string
}

// Parse reads the output of EXPLAIN (FORMAT JSON), with ANALYZE and
// BUFFERS or without, as psql prints it or as the bare JSON.
func Parse(data []byte) (*Plan, error) {
	var b bytes.Buffer
	for line := range bytes.Lines(data) {
		// psql's aligned output ends the lines of a value that goes on with "+".
		b.Write(bytes.TrimSuffix(bytes.TrimRight(line, " \t\r\n"), []byte("+")))
		b.WriteByte('\n')
	}
	data = b.Bytes()
	if i := bytes.IndexAny(data, "[{"); i > 0 {
		data = data[i:]
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	v, err := decode(dec)
	if err != nil {
		return nil, fmt.Errorf("queryplan: %w", err)
	}
	list, ok := v.([]any)
	if !ok {
		list = []any{v}
	}
	p := &Plan{}
	for _, item := range list {
		o, _ := item.(*object)
		if o == nil {
			continue
		}
		top, _ := o.get("Plan").(*object)
		if top == nil {
			continue
		}
		p.analyzed = p.analyzed || top.get("Actual Loops") != nil
		p.planning += o.num("Planning Time")
		p.execution += o.num("Execution Time")
		p.props = append(p.props, o.props("Plan")...)
		p.roots = append(p.roots, build(top, 1))
	}
	if len(p.roots) == 0 {
		return nil, errors.New("queryplan: no plan in the data")
	}
	var all []*node
	var walk func(ns []*node)
	walk = func(ns []*node) {
		for _, n := range ns {
			all = append(all, n)
			walk(n.kids)
		}
	}
	walk(p.roots)
	p.nodes = len(all)
	for _, n := range p.roots {
		if p.analyzed {
			p.total += n.time
		} else {
			p.total += n.cost
		}
	}
	for _, n := range all {
		v := n.self
		if !p.analyzed {
			v = n.cost
			for _, k := range n.kids {
				v -= k.cost
			}
		}
		n.share = max(0, v) / max(p.total, 1e-9)
	}
	slices.SortStableFunc(all, func(a, b *node) int { return cmp.Compare(b.share, a.share) })
	for i, n := range all[:min(3, len(all))] {
		// In a plan of a few nodes, the three slowest may take next to nothing.
		if n.share >= 0.1 {
			n.rank = i + 1
		}
	}
	return p, nil
}

// build returns the node of o, in a part of the plan that procs
// processes run in parallel.
func build(o *object, procs float64) *node {
	n := &node{title: title(o), on: on(o), props: o.props("Plans"), est: o.num("Plan Rows"),
		rows: o.num("Actual Rows"), loops: o.num("Actual Loops"), cost: o.num("Total Cost")}
	// The loops of a parallel node count those of every process, which
	// run at once.
	n.time = o.num("Actual Total Time") * n.loops / procs
	if w := o.num("Workers Launched"); w > 0 {
		procs = w + 1
	}
	n.self = n.time
	plans, _ := o.get("Plans").([]any)
	for _, v := range plans {
		if k, ok := v.(*object); ok {
			kid := build(k, procs)
			n.kids = append(n.kids, kid)
			n.self -= kid.time
		}
	}
	n.self = max(0, n.self)
	return n
}

// title returns the name of the node of o as EXPLAIN's text names it:
// "HashAggregate", "Hash Left Join", "Parallel Seq Scan".
func title(o *object) string {
	t := o.str("Node Type")
	switch t {
	case "Aggregate":
		switch o.str("Strategy") {
		case "Hashed":
			t = "HashAggregate"
		case "Sorted":
			t = "GroupAggregate"
		case "Mixed":
			t = "MixedAggregate"
		}
		if m := o.str("Partial Mode"); m != "" && m != "Simple" {
			t = m + " " + t
		}
	case "Hash Join", "Merge Join", "Nested Loop":
		if j := o.str("Join Type"); j != "" && j != "Inner" {
			t = strings.TrimSuffix(t, " Join") + " " + j + " Join"
		}
	case "ModifyTable":
		t = cmp.Or(o.str("Operation"), t)
	}
	if o.get("Parallel Aware") == true {
		t = "Parallel " + t
	}
	if o.str("Scan Direction") == "Backward" {
		t += " Backward"
	}
	return t
}

// on returns what the node of o reads, as EXPLAIN's text writes it after
// the name: "using orders_pkey on orders o", and the subplan it is.
func on(o *object) string {
	var parts []string
	if idx := o.str("Index Name"); idx != "" {
		parts = append(parts, "using "+idx)
	}
	name := cmp.Or(o.str("Relation Name"), o.str("CTE Name"), o.str("Function Name"))
	full := name
	if s := o.str("Schema"); s != "" && name != "" {
		full = s + "." + name
	}
	if a := o.str("Alias"); a != "" && a != name {
		full = strings.TrimSpace(full + " " + a)
	}
	if full != "" {
		parts = append(parts, "on "+full)
	}
	if s := o.str("Subplan Name"); s != "" {
		parts = append(parts, "("+s+")")
	}
	return strings.Join(parts, " ")
}

// object is a JSON object, its members in the order written.
type object struct {
	keys []string
	vals []any // string, json.Number, bool, nil, []any or *object
}

func (o *object) get(key string) any {
	if i := slices.Index(o.keys, key); i >= 0 {
		return o.vals[i]
	}
	return nil
}

func (o *object) str(key string) string {
	s, _ := o.get(key).(string)
	return s
}

func (o *object) num(key string) float64 {
	n, _ := o.get(key).(json.Number)
	f, _ := n.Float64()
	return f
}

// props returns the members of o but skip, their values as they read,
// leaving out those without any and, as EXPLAIN's text does, the
// counters of blocks, I/O time and rows removed at zero.
func (o *object) props(skip string) []prop {
	var out []prop
	for i, k := range o.keys {
		counter := strings.HasSuffix(k, " Blocks") || strings.Contains(k, "I/O") || strings.HasPrefix(k, "Rows Removed")
		n, _ := o.vals[i].(json.Number)
		f, err := n.Float64()
		v := text(k, o.vals[i])
		if k == skip || v == "" || counter && err == nil && f == 0 {
			continue
		}
		out = append(out, prop{k, v})
	}
	return out
}

// decode reads the next value of dec.
func decode(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	d, ok := tok.(json.Delim)
	if !ok {
		return tok, nil
	}
	if d == '[' {
		list := []any{}
		for dec.More() {
			v, err := decode(dec)
			if err != nil {
				return nil, err
			}
			list = append(list, v)
		}
		_, err := dec.Token()
		return list, err
	}
	o := &object{}
	for dec.More() {
		k, err := dec.Token()
		if err != nil {
			return nil, err
		}
		v, err := decode(dec)
		if err != nil {
			return nil, err
		}
		o.keys, o.vals = append(o.keys, k.(string)), append(o.vals, v)
	}
	_, err = dec.Token()
	return o, err
}

// text returns the value v of key as it reads: a number with its unit,
// a list a line an item, and an object a line a member.
func text(key string, v any) string {
	switch v := v.(type) {
	case string:
		return v
	case bool:
		return strconv.FormatBool(v)
	case json.Number:
		s := group(v.String())
		switch {
		case strings.HasSuffix(key, " Time"):
			s += " ms"
		case strings.HasSuffix(key, " Usage"), strings.HasSuffix(key, " Space Used"):
			s += " kB"
		}
		return s
	case []any:
		var lines []string
		for _, e := range v {
			lines = append(lines, text(key, e))
		}
		return strings.Join(lines, "\n")
	case *object:
		var lines []string
		for _, p := range v.props("") {
			lines = append(lines, p.key+": "+p.value)
		}
		return strings.Join(lines, "\n")
	}
	return ""
}

// State is what a view keeps: the nodes folded, the node chosen and the
// width of the panel. Keep it in the app's state, one for each view.
type State struct {
	tree  ui.OutlineState[*node]
	row   int
	split float32
	plan  *Plan
}

// chosen returns the node chosen, or nil.
func (s *State) chosen() *node {
	if s.row < 0 || s.row >= s.tree.Rows() {
		return nil
	}
	return s.tree.Item(s.row)
}

// View creates the view of plan: the tree of its nodes, all unfolded at
// first, a line below it of the plan's nodes and time, and the panel of
// the node chosen beside it. Give it a size, or Grow.
func View(c *ui.Context, s *State, plan *Plan) ui.Element {
	t := c.Theme()
	if s.plan != plan {
		s.tree, s.row, s.plan = ui.OutlineState[*node]{}, -1, plan
		var open func(ns []*node)
		open = func(ns []*node) {
			for _, n := range ns {
				if n.kids != nil {
					s.tree.Open.Add(n)
					open(n.kids)
				}
			}
		}
		open(plan.roots)
	}
	if s.split == 0 {
		s.split = 70
	}
	s.tree.List.Selected = &s.row
	num := func(title string, w float32) ui.TableColumn {
		return ui.TableColumn{Title: title, Width: t.Space(w), Align: ui.End}
	}
	cols := []ui.TableColumn{{Title: "Node", MinWidth: t.Space(40)}}
	if plan.analyzed {
		cols = append(cols, num("Planned", 20), num("Actual", 20), num("Estimate", 23), num("Loops", 16), num("Self", 20), num("Total", 20))
	} else {
		cols = append(cols, num("Rows", 20))
	}
	cols = append(cols, num("Cost", 20), ui.TableColumn{Title: "Share", Width: t.Space(36)})
	return ui.SplitPercent(c, &s.split, func() {
		ui.Column(c).Fill().Children(func() {
			ui.OutlineTable(c, &s.tree, cols, plan.roots, func(n *node) []*node { return n.kids }, func(n *node, col int) {
				s.cell(c, n, cols[col].Title)
			}).Label("Query plan").Grow(1)
			ui.Divider(c)
			ui.Text(c, plan.status()).FontSize(t.FontSize-1).TextColor(t.TextMuted).FontFeatures("tnum").SingleLine().
				Padding(t.Space(1.5), t.Space(3))
		})
	}, func() {
		s.panel(c, plan)
	})
}

// cell shows the value of n in the column titled col.
func (s *State) cell(c *ui.Context, n *node, col string) {
	t := c.Theme()
	muted, warn, slow, bar, track := t.TextMuted, t.Warning, t.Danger, t.Accent, t.Text.Alpha(0.08)
	// The colors of the other rows are unreadable on the accent of the
	// row chosen.
	if s.chosen() == n {
		a := t.AccentText
		muted, warn, slow, bar, track = a.Alpha(0.75), a, a, a, a.Alpha(0.25)
	}
	num := func(v string) ui.Element { return ui.Text(c, v).FontFeatures("tnum").SingleLine() }
	switch col {
	case "Node":
		ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Shrink(1).Children(func() {
			ui.Text(c, n.title).SingleLine().Shrink(0)
			if n.on != "" {
				ui.Text(c, n.on).TextColor(muted).SingleLine().Shrink(1)
			}
		})
	case "Planned", "Rows":
		num(rows(n.est))
	case "Actual":
		num(rows(n.rows))
	case "Estimate":
		if m := n.misestimate(); m != "" {
			num(m).TextColor(warn).Bold()
		}
	case "Loops":
		num(count(n.loops))
	case "Self":
		e := num(ms(n.self))
		if n.rank > 0 {
			e.TextColor(slow).Bold()
		}
	case "Total":
		num(ms(n.time))
	case "Cost":
		num(cost(n.cost))
	case "Share":
		if n.rank > 0 {
			bar = slow
		}
		ui.Row(c).Grow(1).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			share := float32(n.share)
			ui.Box(c).Grow(1).Height(t.Space(1.5)).Draw(func(p *ui.Painter, r ui.Rect) {
				p.Fill(r, track, r.H/2)
				if share > 0 {
					p.Fill(ui.Rect{X: r.X, Y: r.Y, W: max(r.H, share*r.W), H: r.H}, bar, r.H/2)
				}
			})
			num(fmt.Sprintf("%.1f%%", 100*n.share)).FontSize(t.FontSize - 1).TextColor(muted).Width(t.Space(11)).TextAlign(ui.End).Shrink(0)
		})
	}
}

// panel shows the properties of the node chosen, or with none, of plan.
func (s *State) panel(c *ui.Context, plan *Plan) {
	t := c.Theme()
	n := s.chosen()
	title, on, props := "Plan", "", plan.props
	if n != nil {
		title, on, props = n.title, n.on, n.props
	}
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).Gap(t.Space(2.5)).Padding(t.Space(4)).Children(func() {
			ui.Column(c).Gap(t.Space(1)).Children(func() {
				ui.Text(c, title).Bold().FontSize(t.FontSize + 2)
				if on != "" {
					ui.Text(c, on).TextColor(t.TextMuted)
				}
			})
			ui.Divider(c)
			for i, p := range props {
				ui.Row(c.Key(i)).Gap(t.Space(3)).AlignItems(ui.Start).Children(func() {
					ui.Text(c, p.key).TextColor(t.TextMuted).Width(t.Space(38)).Shrink(0)
					ui.Text(c, p.value).FontFeatures("tnum").Selectable().Shrink(1)
				})
			}
			if n == nil {
				ui.Text(c, "Choose a node to see its properties").TextColor(t.TextMuted)
			}
		})
	})
}

// status returns the line below the tree: how many nodes the plan has,
// and how long it took, or what it costs.
func (p *Plan) status() string {
	parts := []string{fmt.Sprintf("%d nodes", p.nodes)}
	if p.planning > 0 {
		parts = append(parts, "planning "+ms(p.planning))
	}
	if p.execution > 0 {
		parts = append(parts, "execution "+ms(p.execution))
	}
	if !p.analyzed {
		parts = append(parts, "cost "+cost(p.total))
	}
	return strings.Join(parts, "  ·  ")
}

// misestimate returns how far the planner's estimate of n's rows was off,
// "24× under", or "" for less than ten times.
func (n *node) misestimate() string {
	if n.loops == 0 {
		return ""
	}
	f := max(n.est, n.rows) / max(1, min(n.est, n.rows))
	if f < 10 {
		return ""
	}
	if n.est > n.rows {
		return count(f) + "× over"
	}
	return count(f) + "× under"
}

// ms formats a time in milliseconds: "0.09 ms", "44.1 ms", "918 ms", "1.18 s".
func ms(v float64) string {
	switch {
	case v >= 1000:
		return fmt.Sprintf("%.2f s", v/1000)
	case v >= 100:
		return fmt.Sprintf("%.0f ms", v)
	case v >= 10:
		return fmt.Sprintf("%.1f ms", v)
	}
	return fmt.Sprintf("%.2f ms", v)
}

// cost formats a cost, whole from 1,000 on: "83.98", "48,173".
func cost(v float64) string {
	if v >= 1000 {
		return count(v)
	}
	return fmt.Sprintf("%.2f", v)
}

// rows formats a count of rows, which PostgreSQL 18 averages over loops
// to a fraction.
func rows(v float64) string {
	if v < 100 && v != math.Trunc(v) {
		return fmt.Sprintf("%.2f", v)
	}
	return count(v)
}

// count formats v whole, with thousands separated, "12,345".
func count(v float64) string {
	return group(strconv.FormatInt(int64(math.Round(v)), 10))
}

// group separates the thousands of the number s as written, "13,520.37".
func group(s string) string {
	whole, frac, ok := strings.Cut(s, ".")
	for i := len(whole) - 3; i > 0 && whole[i-1] != '-'; i -= 3 {
		whole = whole[:i] + "," + whole[i:]
	}
	if ok {
		return whole + "." + frac
	}
	return whole
}
