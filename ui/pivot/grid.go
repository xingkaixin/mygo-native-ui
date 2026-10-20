package pivot

import (
	"cmp"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
)

// order is how the rows are sorted: by their labels, or by the values of
// value in the column of the group keyed col; desc reverses it.
type order struct {
	byValue bool
	col     string
	value   int
	desc    bool
}

// sortBy sorts the rows by o, the labels from A and the values from the
// largest, or reverses them when they are sorted by o already.
func (s *State) sortBy(o order) {
	o.desc = s.order.desc
	if o == s.order {
		s.order.desc = !s.order.desc
		return
	}
	o.desc = o.byValue
	s.order = o
}

// group is the records with the same values of the fields grouping rows,
// or columns, down to its depth: its label, a key naming those fields and
// values, unique in its tree, and the groups in it.
type group struct {
	label string
	key   string
	depth int
	id    int
	kids  []*group
	by    map[string]*group
}

func (g *group) find(key string) *group {
	if g.key == key {
		return g
	}
	for _, k := range g.kids {
		if strings.HasPrefix(key, k.key) {
			if f := k.find(key); f != nil {
				return f
			}
		}
	}
	return nil
}

// acc sums up the values of a field in a cell: how many there are, how
// many are numbers, and their sum and bounds.
type acc struct {
	n, nums     int
	sum, lo, hi float64
}

func (a *acc) add(x float64, number bool) {
	a.n++
	if !number {
		return
	}
	if a.nums == 0 || x < a.lo {
		a.lo = x
	}
	if a.nums == 0 || x > a.hi {
		a.hi = x
	}
	a.nums++
	a.sum += x
}

func (a acc) get(agg Agg) (float64, bool) {
	switch {
	case agg == Count:
		return float64(a.n), a.n > 0
	case a.nums == 0:
		return 0, false
	case agg == Average:
		return a.sum / float64(a.nums), true
	case agg == Min:
		return a.lo, true
	case agg == Max:
		return a.hi, true
	}
	return a.sum, true
}

// cube is the records grouped into rows and into columns, and what each
// cell sums up, by the ids of its row and its column.
type cube struct {
	rows, cols *group
	cells      map[[2]int][]acc
	ids        int
}

func (cb *cube) child(g *group, field, label string) *group {
	if k, ok := g.by[label]; ok {
		return k
	}
	if g.by == nil {
		g.by = map[string]*group{}
	}
	cb.ids++
	k := &group{label: label, key: g.key + "\x1f" + field + "=" + label, depth: g.depth + 1, id: cb.ids}
	g.by[label] = k
	g.kids = append(g.kids, k)
	return k
}

func (cb *cube) get(row, col *group, v int, agg Agg) (float64, bool) {
	cell := cb.cells[[2]int{row.id, col.id}]
	if cell == nil {
		return 0, false
	}
	return cell[v].get(agg)
}

// build groups records as s says, with their values summed up into each
// cell of their row and column groups and of the groups around them, and
// sorts the groups.
func build(s *State, fields []string, records [][]string, ks []kind) *cube {
	at := func(names []string) []int {
		idx := make([]int, len(names))
		for i, n := range names {
			idx[i] = slices.Index(fields, n)
		}
		return idx
	}
	rf, cf := at(s.Rows), at(s.Columns)
	vf := make([]int, len(s.Values))
	for i, v := range s.Values {
		vf[i] = slices.Index(fields, v.Field)
	}
	cb := &cube{rows: &group{}, cols: &group{}, cells: map[[2]int][]acc{}}
	path := func(p []*group, g *group, r []string, names []string, idx []int) []*group {
		p = append(p[:0], g)
		for k, f := range idx {
			g = cb.child(g, names[k], value(r, f))
			p = append(p, g)
		}
		return p
	}
	var rp, cp []*group
	xs := make([]float64, len(s.Values))
	has := make([]bool, len(s.Values))
	nums := make([]bool, len(s.Values))
	for _, r := range records {
		rp = path(rp, cb.rows, r, s.Rows, rf)
		cp = path(cp, cb.cols, r, s.Columns, cf)
		for v, f := range vf {
			text := value(r, f)
			has[v], nums[v] = text != "", false
			if has[v] && ks[f].number {
				xs[v], _ = strconv.ParseFloat(text, 64)
				nums[v] = true
			}
		}
		for _, a := range rp {
			for _, b := range cp {
				key := [2]int{a.id, b.id}
				cell := cb.cells[key]
				if cell == nil {
					cell = make([]acc, len(s.Values))
					cb.cells[key] = cell
				}
				for v := range cell {
					if has[v] {
						cell[v].add(xs[v], nums[v])
					}
				}
			}
		}
	}
	sortTree(cb.cols, func(a, b *group) int { return labels(a.label, b.label) })
	var by *group
	if s.order.byValue && s.order.value < len(s.Values) {
		by = cb.cols.find(s.order.col)
	}
	if by == nil && s.order.byValue {
		s.order = order{}
	}
	sign := 1
	if s.order.desc {
		sign = -1
	}
	sortTree(cb.rows, func(a, b *group) int {
		if by == nil {
			return sign * labels(a.label, b.label)
		}
		agg := s.Values[s.order.value].Agg
		x, okx := cb.get(a, by, s.order.value, agg)
		y, oky := cb.get(b, by, s.order.value, agg)
		switch {
		case okx != oky && okx:
			return -1
		case okx != oky:
			return 1
		}
		return cmp.Or(sign*cmp.Compare(x, y), labels(a.label, b.label))
	})
	return cb
}

// clipped paints with draw inside the box only, since a box clips its
// children but not what it paints.
func clipped(draw func(p *ui.Painter, r ui.Rect)) func(p *ui.Painter, r ui.Rect) {
	return func(p *ui.Painter, r ui.Rect) { p.Clip(r, 0, func() { draw(p, r) }) }
}

func value(r []string, f int) string {
	if f < 0 || f >= len(r) {
		return ""
	}
	return r[f]
}

func sortTree(g *group, by func(a, b *group) int) {
	slices.SortStableFunc(g.kids, by)
	for _, k := range g.kids {
		sortTree(k, by)
	}
}

// labels orders labels as numbers when both are, and as text when not.
func labels(a, b string) int {
	x, errx := strconv.ParseFloat(a, 64)
	y, erry := strconv.ParseFloat(b, 64)
	if errx == nil && erry == nil {
		return cmp.Compare(x, y)
	}
	return cmp.Or(strings.Compare(strings.ToLower(a), strings.ToLower(b)), strings.Compare(a, b))
}

// line is a row or a column showing: its group, and whether it is the
// group's total, as the subtotal column of a group unfolded, or Total.
type line struct {
	g     *group
	total bool
}

// lines adds the rows of the groups in g, each followed by those in it
// unless folded.
func (s *State) lines(g *group, out []line) []line {
	for _, k := range g.kids {
		out = append(out, line{g: k})
		if len(k.kids) > 0 && !s.folded["r"+k.key] {
			out = s.lines(k, out)
		}
	}
	return out
}

// head is a cell of the column headers: over data columns from to to, from
// header row row, rows tall, and the group it folds and unfolds, if any.
type head struct {
	from, to  int
	row, rows int
	label     string
	g         *group
}

// columns adds the columns of the groups in g, n data columns each, those
// of a group unfolded followed by its subtotal, and their headers, levels
// rows of them.
func (s *State) columns(g *group, cols []line, heads []head, n, levels int) ([]line, []head) {
	for _, k := range g.kids {
		from := len(cols) * n
		if len(k.kids) > 0 && !s.folded["c"+k.key] {
			cols, heads = s.columns(k, cols, heads, n, levels)
			heads = append(heads, head{from: from, to: len(cols) * n, row: k.depth - 1, rows: 1, label: name(k.label), g: k})
			cols = append(cols, line{g: k, total: true})
			heads = append(heads, head{from: len(cols)*n - n, to: len(cols) * n, row: k.depth - 1, rows: levels - k.depth + 1, label: name(k.label) + " Total"})
			continue
		}
		h := head{from: from, to: from + n, row: k.depth - 1, rows: levels - k.depth + 1, label: name(k.label)}
		if len(k.kids) > 0 {
			h.g = k
		}
		cols = append(cols, line{g: k})
		heads = append(heads, h)
	}
	return cols, heads
}

func name(label string) string {
	if label == "" {
		return "(blank)"
	}
	return label
}

// table shows the records grouped and summed up as s says.
func (s *State) table(c *ui.Context, fields []string, records [][]string, ks []kind) {
	t := c.Theme()
	cb := build(s, fields, records, ks)
	n, levels := len(s.Values), len(s.Columns)
	rows := append(s.lines(cb.rows, nil), line{g: cb.rows, total: true})
	cols, heads := s.columns(cb.cols, nil, nil, n, levels)
	if levels > 0 {
		heads = append(heads, head{from: len(cols) * n, to: len(cols)*n + n, rows: levels, label: "Total"})
		cols = append(cols, line{g: cb.cols, total: true})
	} else {
		cols = []line{{g: cb.cols}}
	}
	hrows := levels
	if n > 1 || levels == 0 {
		for d := range len(cols) * n {
			v := s.Values[d%n]
			heads = append(heads, head{from: d, to: d + 1, row: levels, rows: 1, label: v.Agg.String() + " of " + v.Field})
		}
		hrows++
	}

	locale := c.Region().FormatLocale(c.Locale())
	formats := make([]*i18n.NumberFormatter, n)
	for v, val := range s.Values {
		o := i18n.NumberOptions{MinFractionDigits: 2, MaxFractionDigits: 2}
		if f := slices.Index(fields, val.Field); val.Agg == Count || f >= 0 && ks[f].whole && val.Agg != Average {
			o = i18n.NumberOptions{Integer: true}
		}
		formats[v] = i18n.NewNumberFormatter(locale, o)
	}
	cellAt := func(row line, d int) (string, bool) {
		v := d % n
		x, ok := cb.get(row.g, cols[d/n].g, v, s.Values[v].Agg)
		if !ok {
			return "", false
		}
		return formats[v].Format(x), true
	}

	size, pad, rowH := t.FontSize-1, t.Space(2), t.Space(7)
	chev := t.Space(4)
	measure := func(text string, weight int) float32 {
		w, _ := c.MeasureText(0, ui.Span{Text: text, Size: size, Weight: weight, Features: "tnum"})
		return w
	}
	bottom := func(h head) bool { return h.row+h.rows == hrows && h.to-h.from == 1 }
	ws := make([]float32, len(cols)*n)
	for d := range ws {
		ws[d] = t.Space(20)
		widest, text := -1.0, ""
		for _, r := range rows {
			v := d % n
			if x, ok := cb.get(r.g, cols[d/n].g, v, s.Values[v].Agg); ok && math.Abs(x) > widest {
				widest, text = math.Abs(x), formats[v].Format(x)
			}
		}
		ws[d] = max(ws[d], measure(text, 700)+2*pad)
	}
	for _, h := range heads {
		if bottom(h) {
			ws[h.from] = max(ws[h.from], measure(h.label, 600)+2*pad+t.Space(3))
		}
	}
	xs := make([]float32, len(ws)+1)
	for d, w := range ws {
		xs[d+1] = xs[d] + w
	}
	slot := float32(0)
	if len(s.Rows) > 1 {
		slot = chev
	}
	indent := func(depth int) float32 { return pad + float32(max(depth, 1)-1)*t.Space(4) }
	labelW := measure(strings.Join(s.Rows, " / "), 600) + 2*pad + t.Space(3)
	for _, r := range rows {
		labelW = max(labelW, indent(r.g.depth)+slot+measure(name(r.g.label), 700)+pad)
	}
	labelW = min(max(labelW, t.Space(36)), t.Space(90))
	totalW, totalH, headH := xs[len(xs)-1], float32(len(rows))*rowH, float32(hrows)*rowH

	toggle := func(key string) {
		if s.folded[key] {
			delete(s.folded, key)
		} else {
			s.folded[key] = true
		}
	}
	hit := func(x, y float32) (fold *group, d int) {
		x += s.scroll.X
		row := int(y / rowH)
		for _, h := range heads {
			if row < h.row || row >= h.row+h.rows || x < xs[h.from] || x >= xs[h.to] {
				continue
			}
			if h.g != nil && (!bottom(h) || x < max(xs[h.from], s.scroll.X)+pad+chev) {
				return h.g, -1
			}
			if bottom(h) {
				return nil, h.from
			}
		}
		return nil, -1
	}
	sorted := func(d int) bool {
		return s.order.byValue && cols[d/n].g.key == s.order.col && d%n == s.order.value
	}
	text := func(p *ui.Painter, area ui.Rect, x float32, label string, weight int, color ui.Color) {
		span := ui.Span{Text: label, Size: size, Weight: weight, Color: color, Features: "tnum"}
		_, th := p.MeasureText(0, span)
		p.Clip(area, 0, func() { p.RichText(x, area.Y+(area.H-th)/2, 0, span) })
	}
	width := func(p *ui.Painter, label string, weight int) float32 {
		w, _ := p.MeasureText(0, ui.Span{Text: label, Size: size, Weight: weight, Features: "tnum"})
		return w
	}
	chevron := func(p *ui.Painter, x, cy float32, open bool) {
		cx, d := x+chev/2, float32(3)
		pts := [][2]float32{{cx - d/2, cy - d}, {cx + d/2, cy}, {cx - d/2, cy + d}}
		if open {
			pts = [][2]float32{{cx - d, cy - d/2}, {cx, cy + d/2}, {cx + d, cy - d/2}}
		}
		p.StrokePolyline(pts, 1.5, t.TextMuted)
	}
	arrow := func(p *ui.Painter, x, cy float32, desc bool) {
		d := float32(3)
		pts := [][2]float32{{x, cy + d/2}, {x + d, cy - d/2}, {x + 2*d, cy + d/2}}
		if desc {
			pts = [][2]float32{{x, cy - d/2}, {x + d, cy + d/2}, {x + 2*d, cy - d/2}}
		}
		p.StrokePolyline(pts, 1.5, t.Accent)
	}
	grid := t.Border.Alpha(0.6)

	frame := ui.Column(c).Grow(1).Shrink(1).MaxWidth(labelW+totalW+2).MaxHeight(headH+totalH+2).
		Border(1, t.Border).Radius(t.Radius).Clip()
	frame.Children(func() {
		ui.Row(c).Height(headH).Shrink(0).AlignItems(ui.Stretch).Children(func() {
			corner := ui.ButtonBase(c).Width(labelW).Shrink(0).Background(t.Surface).Label("Sort by " + strings.Join(s.Rows, ", "))
			if corner.Clicked() {
				s.sortBy(order{})
			}
			corner.Draw(func(p *ui.Painter, r ui.Rect) {
				area := ui.Rect{X: r.X, Y: r.Y + r.H - rowH, W: r.W, H: rowH}
				label := strings.Join(s.Rows, " / ")
				text(p, area, r.X+pad, label, 600, t.Text)
				if !s.order.byValue && len(s.Rows) > 0 {
					arrow(p, r.X+pad+width(p, label, 600)+t.Space(1.5), area.Y+rowH/2, s.order.desc)
				}
				p.Line(r.X+r.W-1, r.Y, r.X+r.W-1, r.Y+r.H, 1, t.Border)
				p.Line(r.X, r.Y+r.H-1, r.X+r.W, r.Y+r.H-1, 1, t.Border)
			})
			header := ui.Box(c).Grow(1).Shrink(1).Clip().Background(t.Surface).Label("Column headers")
			if x, y, ok := header.PointerPosition(); ok {
				if g, d := hit(x, y); g != nil || d >= 0 {
					header.Cursor(ui.CursorPointer)
				}
			}
			if header.Clicked() {
				if x, y, ok := header.PointerPosition(); ok {
					switch g, d := hit(x, y); {
					case g != nil:
						toggle("c" + g.key)
					case d >= 0:
						s.sortBy(order{byValue: true, col: cols[d/n].g.key, value: d % n})
					}
				}
			}
			header.Draw(clipped(func(p *ui.Painter, r ui.Rect) {
				for _, h := range heads {
					x0, x1 := r.X+xs[h.from]-s.scroll.X, r.X+xs[h.to]-s.scroll.X
					if x1 < r.X || x0 > r.X+r.W {
						continue
					}
					area := ui.Rect{X: x0, Y: r.Y + float32(h.row)*rowH, W: x1 - x0, H: float32(h.rows) * rowH}
					p.Line(x1, area.Y, x1, area.Y+area.H, 1, t.Border)
					p.Line(x0, area.Y+area.H-1, x1, area.Y+area.H-1, 1, t.Border)
					// A group's name stays in view while its first columns scroll out.
					x := max(x0, r.X) + pad
					if h.g != nil {
						chevron(p, x, area.Y+area.H/2, !s.folded["c"+h.g.key])
						x += chev
					}
					if bottom(h) {
						w := width(p, h.label, 600)
						x = x1 - pad - w
						if sorted(h.from) {
							arrow(p, x-t.Space(3), area.Y+area.H/2, s.order.desc)
						}
					}
					text(p, area, x, h.label, 600, t.Text)
				}
			}))
		})
		ui.Row(c).Grow(1).Shrink(1).AlignItems(ui.Stretch).Children(func() {
			side := ui.Box(c).Width(labelW).Shrink(0).Clip().Label("Row headers")
			at := func(y float32) int {
				i := int((y + s.scroll.Y) / rowH)
				if i < 0 || i >= len(rows) || rows[i].total || len(rows[i].g.kids) == 0 {
					return -1
				}
				return i
			}
			if _, y, ok := side.PointerPosition(); ok && at(y) >= 0 {
				side.Cursor(ui.CursorPointer)
			}
			if side.Clicked() {
				if _, y, ok := side.PointerPosition(); ok && at(y) >= 0 {
					toggle("r" + rows[at(y)].g.key)
				}
			}
			side.Draw(clipped(func(p *ui.Painter, r ui.Rect) {
				first := max(0, int(s.scroll.Y/rowH))
				for i := first; i < len(rows) && float32(i)*rowH-s.scroll.Y < r.H; i++ {
					row := rows[i]
					area := ui.Rect{X: r.X, Y: r.Y + float32(i)*rowH - s.scroll.Y, W: r.W, H: rowH}
					label, weight := name(row.g.label), 400
					switch {
					case row.total:
						p.Fill(area, t.Surface, 0)
						label, weight = "Total", 700
					case len(row.g.kids) > 0:
						weight = 600
						chevron(p, area.X+indent(row.g.depth), area.Y+rowH/2, !s.folded["r"+row.g.key])
					}
					text(p, area, area.X+indent(row.g.depth)+slot, label, weight, t.Text)
					p.Line(area.X, area.Y+rowH, area.X+area.W, area.Y+rowH, 1, grid)
				}
				p.Line(r.X+r.W-1, r.Y, r.X+r.W-1, r.Y+r.H, 1, t.Border)
			}))
			ui.ScrollBoth(c).Grow(1).Shrink(1).TrackScroll(&s.scroll).Children(func() {
				ui.Box(c).Size(totalW, totalH).Draw(func(p *ui.Painter, r ui.Rect) {
					vis := p.Visible()
					top, end := max(0, int((vis.Y-r.Y)/rowH)), min(len(rows), int((vis.Y+vis.H-r.Y)/rowH)+1)
					left := 0
					for left < len(ws) && r.X+xs[left+1] <= vis.X {
						left++
					}
					right := left
					for right < len(ws) && r.X+xs[right] < vis.X+vis.W {
						right++
					}
					for d := left; d < right; d++ {
						if cols[d/n].total {
							p.Fill(ui.Rect{X: r.X + xs[d], Y: r.Y, W: ws[d], H: r.H}, t.Surface.Alpha(0.5), 0)
						}
					}
					for i := top; i < end; i++ {
						row := rows[i]
						y := r.Y + float32(i)*rowH
						weight := 400
						switch {
						case row.total:
							p.Fill(ui.Rect{X: r.X, Y: y, W: r.W, H: rowH}, t.Surface, 0)
							weight = 700
						case len(row.g.kids) > 0:
							weight = 600
						}
						for d := left; d < right; d++ {
							if label, ok := cellAt(row, d); ok {
								area := ui.Rect{X: r.X + xs[d], Y: y, W: ws[d], H: rowH}
								text(p, area, area.X+area.W-pad-width(p, label, weight), label, weight, t.Text)
							}
						}
						p.Line(r.X, y+rowH, r.X+r.W, y+rowH, 1, grid)
					}
					for d := left; d < right; d++ {
						color := grid
						if (d+1)%n == 0 {
							color = t.Border
						}
						p.Line(r.X+xs[d+1], r.Y, r.X+xs[d+1], r.Y+r.H, 1, color)
					}
				})
			})
		})
	})
}
