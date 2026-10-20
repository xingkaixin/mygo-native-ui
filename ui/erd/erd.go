// Package erd shows the tables of a database and the foreign keys between
// them, as dbdiagram and DataGrip draw them: each table a card of its name
// and a row for each column, with a key by those of the primary key, a
// link by the foreign keys, and the type at the right, marked ? for a
// column that may be null. A line runs from the row of each foreign key to
// the row of the column it references, from the sides of the tables facing
// each other, in crow's foot notation: many at the foreign key, one at the
// column it references, after a circle for a foreign key that may be null
// and a bar for one that may not. A table referencing itself loops on its
// left side.
//
// The tables start in columns, each right of the tables it references,
// ordered so that few lines cross, and dragging a table by its name moves
// it. The pointer over a table lights its lines and the tables at their
// other ends; a click on a column chooses it and lights its lines, and a
// click again lets it go. Parse reads the tables from SQL.
//
//	if ui.TextArea(c, &app.sql).Changed() {
//		app.tables, app.errs = erd.Parse(app.sql)
//	}
//	erd.View(c, &app.diagram, app.tables).Grow(1)
package erd

import (
	"cmp"
	"math"
	"slices"

	"github.com/egoist/mygo/ui"
)

// Table is a table of a database: its name and its columns, in order.
type Table struct {
	Name    string
	Columns []Column
}

// Column is a column of a table: its name; its type as written, as
// varchar(255); whether it is of the primary key, and whether it may be
// null; and, for a foreign key, the table and the column it references.
type Column struct {
	Name, Type          string
	Key, Null           bool
	RefTable, RefColumn string
}

// State is what a diagram keeps: where each table is, by its name, so
// that tables stay where they are as the tables change, and the column
// chosen. Keep it in the app's state, one for each diagram.
type State struct {
	at map[string]ui.Point
	// routes are the slots of the relations between tables more than a
	// column apart, by the table and column at either end, as laid out,
	// until either table moves.
	routes map[[4]string][]slot
	// chosen is the table and the column chosen.
	chosen [2]string
}

// rel is a foreign key: column fk of table from references column pk of
// table to.
type rel struct {
	from, fk, to, pk int
	null             bool
}

// View creates the diagram of tables. Give it a size, or Grow; it scrolls
// both ways when it does not fit. A table new to s goes in the column the
// layout puts it in, below the tables there.
func View(c *ui.Context, s *State, tables []Table) ui.Element {
	t := c.Theme()
	size := t.FontSize - 1
	headH, rowH, padX, iconW := t.Space(9), t.Space(7), t.Space(3), t.Space(5)
	gapX, gapY, pad, stub := t.Space(22), t.Space(8), t.Space(6), t.Space(5)
	// The tables start right of room for the loop of a table referencing
	// itself.
	ox, oy := pad+2*stub, pad

	index := make(map[string]int, len(tables))
	for i, tb := range tables {
		index[tb.Name] = i
	}
	var rels []rel
	for i, tb := range tables {
		for k, col := range tb.Columns {
			j, ok := index[col.RefTable]
			m := -1
			if ok {
				m = slices.IndexFunc(tables[j].Columns, func(c Column) bool { return c.Name == col.RefColumn })
			}
			if m < 0 {
				continue
			}
			rels = append(rels, rel{from: i, fk: k, to: j, pk: m, null: col.Null})
		}
	}
	key := func(r rel) [4]string {
		return [4]string{tables[r.from].Name, tables[r.from].Columns[r.fk].Name, tables[r.to].Name, tables[r.to].Columns[r.pk].Name}
	}

	w, h := make([]float32, len(tables)), make([]float32, len(tables))
	icons := make([]int, len(tables))
	for i, tb := range tables {
		name, _ := c.MeasureText(0, ui.Span{Text: tb.Name, Weight: 700})
		w[i] = max(t.Space(44), name+2*padX)
		for _, col := range tb.Columns {
			icons[i] = max(icons[i], marks(col))
		}
		for _, col := range tb.Columns {
			name, _ := c.MeasureText(0, ui.Span{Text: col.Name, Size: size})
			typ, _ := c.MeasureText(0, ui.Span{Text: typeOf(col), Size: size})
			w[i] = max(w[i], 2*padX+float32(icons[i])*iconW+name+t.Space(6)+typ)
		}
		// The border, 1 DIP wide, is inside the card.
		w[i], h[i] = float32(math.Ceil(float64(w[i])))+2, headH+float32(len(tb.Columns))*rowH+2
	}

	if s.at == nil {
		s.at = map[string]ui.Point{}
	}
	var fresh []int
	for i, tb := range tables {
		if _, ok := s.at[tb.Name]; !ok {
			fresh = append(fresh, i)
		}
	}
	if len(fresh) > 0 {
		laid, slots := layout(rels, w, h, gapX, gapY, t.Space(2))
		if len(fresh) == len(tables) {
			s.routes = map[[4]string][]slot{}
			for n, rl := range rels {
				if slots[n] != nil {
					s.routes[key(rl)] = slots[n]
				}
			}
		}
		for _, i := range fresh {
			at := laid[i]
			if len(fresh) < len(tables) {
				at.Y = 0
				for k, tb := range tables {
					if q, ok := s.at[tb.Name]; ok && q.X < at.X+w[i] && at.X < q.X+w[k] {
						at.Y = max(at.Y, q.Y+h[k]+gapY)
					}
				}
			}
			s.at[tables[i].Name] = at
		}
	}

	return ui.ScrollBoth(c).Children(func() {
		box := ui.Box(c)
		hovered := -1
		if px, py, ok := box.PointerPosition(); ok {
			for i, tb := range tables {
				at := s.at[tb.Name]
				if px >= ox+at.X && px < ox+at.X+w[i] && py >= oy+at.Y && py < oy+at.Y+h[i] {
					hovered = i
				}
			}
		}
		chosen := func(i, k int) bool { return s.chosen == [2]string{tables[i].Name, tables[i].Columns[k].Name} }
		lit := make([]bool, len(rels))
		reached := make([]bool, len(tables))
		for n, r := range rels {
			if r.from == hovered || r.to == hovered {
				lit[n], reached[r.from], reached[r.to] = true, true, true
			}
			lit[n] = lit[n] || chosen(r.from, r.fk) || chosen(r.to, r.pk)
		}
		if hovered >= 0 {
			reached[hovered] = true
		}

		radius := t.Space(2)
		box.Children(func() {
			for i, tb := range tables {
				card := ui.Column(c.Key(tb.Name)).Absolute().Width(w[i]).Radius(radius).Clip().
					Background(t.Background).Border(1, t.Border).Shadow(0, 1, 3, 0, ui.RGB(0, 0, 0).Alpha(0.08)).
					Role(ui.RoleGroup).Label(tb.Name)
				if reached[i] {
					card.Border(1, t.Accent)
				}
				card.Children(func() {
					head := ui.Row(c).Height(headH).Padding(0, padX).AlignItems(ui.Center).
						Background(t.Text.Alpha(0.05)).Cursor(ui.CursorGrab)
					if head.Pressed() {
						head.Cursor(ui.CursorGrabbing)
					}
					if dx, dy, ok := head.Dragged(); ok && (dx != 0 || dy != 0) {
						at := s.at[tb.Name]
						s.at[tb.Name] = ui.Point{X: max(0, at.X+dx), Y: max(0, at.Y+dy)}
						for k := range s.routes {
							if k[0] == tb.Name || k[2] == tb.Name {
								delete(s.routes, k)
							}
						}
					}
					head.Children(func() {
						ui.Text(c, tb.Name).Bold().SingleLine()
					})
					for k, col := range tb.Columns {
						row := ui.ButtonBase(c.Key(k)).Height(rowH).Padding(0, padX).Label(col.Name)
						if row.Clicked() {
							if chosen(i, k) {
								s.chosen = [2]string{}
							} else {
								s.chosen = [2]string{tb.Name, col.Name}
							}
						}
						switch {
						case chosen(i, k):
							row.Background(t.Selection)
						case row.Hovered():
							row.Background(t.Text.Alpha(0.04))
						}
						row.Children(func() {
							if icons[i] > 0 {
								mark(c, col, icons[i], iconW)
							}
							ui.Text(c, col.Name).FontSize(size).SingleLine().Grow(1)
							ui.Text(c, typeOf(col)).FontSize(size).TextColor(t.TextMuted).SingleLine()
						})
					}
				})
				at := s.at[tb.Name]
				card.Left(ox + at.X).Top(oy + at.Y)
			}
		})

		rects := make([]ui.Rect, len(tables))
		var right, bottom float32
		for i, tb := range tables {
			at := s.at[tb.Name]
			rects[i] = ui.Rect{X: ox + at.X, Y: oy + at.Y, W: w[i], H: h[i]}
			right, bottom = max(right, at.X+w[i]), max(bottom, at.Y+h[i])
		}
		box.Size(ox+right+pad, oy+bottom+pad)

		row := func(i, k int) float32 { return rects[i].Y + 1 + headH + (float32(k)+0.5)*rowH }
		type line struct {
			a, b end
			via  []slot
		}
		lines := make([]line, len(rels))
		meet := map[[3]int][]int{}
		for n, rl := range rels {
			l := &lines[n]
			l.a, l.b = ends(rects[rl.from], rects[rl.to], row(rl.from, rl.fk), row(rl.to, rl.pk), rl.from == rl.to)
			for _, v := range s.routes[key(rl)] {
				l.via = append(l.via, slot{ox + v.x0, ox + v.x1, oy + v.y})
			}
			k := [3]int{rl.to, rl.pk, int(l.b.d)}
			meet[k] = append(meet[k], n)
		}
		// comes returns the height line n comes into its column from.
		comes := func(n int) float32 {
			if v := lines[n].via; len(v) > 0 {
				return v[len(v)-1].y
			}
			return lines[n].a.y
		}
		// Lines meeting at a row side by side, in the order they come from,
		// keep their marks apart.
		for _, ns := range meet {
			slices.SortFunc(ns, func(a, b int) int { return cmp.Compare(comes(a), comes(b)) })
			step := min(t.Space(2), (rowH-t.Space(2))/float32(max(1, len(ns)-1)))
			for k, n := range ns {
				lines[n].b.y += (float32(k) - float32(len(ns)-1)/2) * step
			}
		}
		box.Draw(func(p *ui.Painter, r ui.Rect) {
			p.Transform(ui.Translation(r.X, r.Y), 0, 0, func() {
				for _, on := range []bool{false, true} {
					for n, rl := range rels {
						if lit[n] != on {
							continue
						}
						color, width := t.Text.Alpha(0.35), float32(1.25)
						if on {
							color, width = t.Accent, 2
						}
						relation(p, lines[n].a, lines[n].b, lines[n].via, rl.null, stub, width, color)
					}
				}
			})
		})
	})
}

// typeOf returns the type of col as a row shows it, with ? after it when
// it may be null.
func typeOf(col Column) string {
	if col.Null {
		return col.Type + "?"
	}
	return col.Type
}

// marks returns how many marks the row of col shows: a key, a link, or
// both.
func marks(col Column) int {
	n := 0
	if col.Key {
		n++
	}
	if col.RefTable != "" {
		n++
	}
	return n
}

// mark shows the marks of col, a key for the primary key and a link for a
// foreign key, in n places w wide.
func mark(c *ui.Context, col Column, n int, w float32) {
	t := c.Theme()
	s := t.Space(3.5)
	ui.Box(c).Size(float32(n)*w, s).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		x, y := r.X, r.Y
		if col.Key {
			var bow ui.Path
			bow.Circle(x+s*0.27, y+s*0.5, s*0.22)
			p.StrokePath(&bow, 1.5, t.Warning)
			p.StrokePolyline([][2]float32{{x + s*0.49, y + s*0.5}, {x + s, y + s*0.5}, {x + s, y + s*0.75}}, 1.5, t.Warning)
			p.Line(x+s*0.78, y+s*0.5, x+s*0.78, y+s*0.7, 1.5, t.Warning)
			x += w
		}
		if col.RefTable != "" {
			cx, cy := x+s/2, y+s/2
			p.Transform(ui.Rotation(-45), cx, cy, func() {
				link := ui.Rect{Y: cy - s*0.17, W: s * 0.58, H: s * 0.34}
				for _, lx := range []float32{cx - s*0.5, cx - s*0.08} {
					link.X = lx
					p.Stroke(link, t.Accent, s*0.17, 1.5)
				}
			})
		}
	})
}

// end is an end of a line: where it meets its table, and the way it
// leaves it, 1 to the right and -1 to the left.
type end struct{ x, y, d float32 }

// ends returns the ends of the line from a foreign key's row, ya high in
// its table a, to the row of the column it references, yb high in table
// b: on the sides facing each other, on the right sides when the tables
// overlap, and on the left side, away from the lines of the tables
// referencing it, when self is set, as one table.
func ends(a, b ui.Rect, ya, yb float32, self bool) (end, end) {
	ea, eb := end{a.X + a.W, ya, 1}, end{b.X + b.W, yb, 1}
	switch {
	case self:
		ea, eb = end{a.X, ya, -1}, end{b.X, yb, -1}
	case a.X+a.W <= b.X:
		eb.x, eb.d = b.X, -1
	case b.X+b.W <= a.X:
		ea.x, ea.d = a.X, -1
	}
	return ea, eb
}

// slot is where a relation passes a column between its tables: across it
// from x0 to x1, y high.
type slot struct{ x0, x1, y float32 }

// relation draws a line from a, a foreign key, through the slots of via,
// to b, the column it references: a crow's foot at a, and at b a bar
// after a circle for a foreign key that may be null and after a bar for
// one that may not.
func relation(p *ui.Painter, a, b end, via []slot, null bool, stub, width float32, color ui.Color) {
	var path ui.Path
	x, y, d := a.x+a.d*stub, a.y, a.d
	path.MoveTo(a.x, a.y).LineTo(x, y)
	// curve goes on from x, y the way d, to x1, y1, coming in level from
	// the way out.
	curve := func(x1, y1, out float32) {
		bend := max(stub, abs(x1-x)/2)
		if out == d {
			bend = max(bend, 2*stub)
		}
		path.CubeTo(x+d*bend, y, x1+out*bend, y1, x1, y1)
		x, y = x1, y1
	}
	for _, v := range via {
		in, across := v.x0, v.x1
		if d < 0 {
			in, across = across, in
		}
		curve(in, v.y, -d)
		path.LineTo(across, v.y)
		x = across
	}
	curve(b.x+b.d*stub, b.y, b.d)
	p.StrokePath(&path, width, color)

	foot := stub * 0.35
	p.StrokePolyline([][2]float32{{a.x, a.y - foot}, {a.x + a.d*stub*0.6, a.y}, {a.x, a.y + foot}}, width, color)
	// along draws from o0 to o1 DIPs out from b.
	along := func(o0, o1 float32) { p.Line(b.x+b.d*o0, b.y, b.x+b.d*o1, b.y, width, color) }
	bar := func(o float32) { p.Line(b.x+b.d*o, b.y-stub*0.25, b.x+b.d*o, b.y+stub*0.25, width, color) }
	bar(stub * 0.3)
	if !null {
		along(0, stub)
		bar(stub * 0.55)
		return
	}
	o, rad := stub*0.72, stub*0.18
	along(0, o-rad)
	along(o+rad, stub)
	var circle ui.Path
	circle.Circle(b.x+b.d*o, b.y, rad)
	p.StrokePath(&circle, width, color)
}

func abs(v float32) float32 { return max(v, -v) }

// layout returns where each table goes first, and the slots each
// relation passes, as Sugiyama's method lays out graphs: the tables in
// columns, one right of the last of those it references, a relation
// between tables more than a column apart passing the columns between
// them in lanes of its own, thin slots among their tables; and in each
// column the tables and lanes ordered by where those they link to are,
// as dag orders its jobs, so that few lines cross. A column is as wide as
// its widest table, and centered on the tallest column.
func layout(rels []rel, w, h []float32, gapX, gapY, laneGap float32) ([]ui.Point, [][]slot) {
	n := len(w)
	needs := make([][]int, n)
	for _, r := range rels {
		if r.from != r.to {
			needs[r.from] = append(needs[r.from], r.to)
		}
	}
	col := make([]int, n)
	state := make([]uint8, n)
	var place func(i int) int
	place = func(i int) int {
		switch state[i] {
		case 1:
			// A table that needs itself, by a cycle, goes first.
			return -1
		case 2:
			return col[i]
		}
		state[i] = 1
		c := 0
		for _, k := range needs[i] {
			c = max(c, place(k)+1)
		}
		col[i], state[i] = c, 2
		return c
	}
	m := 0
	for i := range n {
		m = max(m, place(i)+1)
	}

	// The nodes past the tables are lanes, and adj links each node to
	// those next to it on the way of a relation.
	adj := make([][]int, n)
	lanes := make([][]int, len(rels))
	link := func(a, b int) { adj[a], adj[b] = append(adj[a], b), append(adj[b], a) }
	for k, r := range rels {
		a, b := r.from, r.to
		if a == b {
			continue
		}
		step, prev := 1, a
		if col[b] < col[a] {
			step = -1
		}
		for c := col[a] + step; col[a] != col[b] && c != col[b]; c += step {
			v := len(col)
			col, adj = append(col, c), append(adj, nil)
			link(prev, v)
			lanes[k], prev = append(lanes[k], v), v
		}
		link(prev, b)
	}
	cols := make([][]int, m)
	for i, c := range col {
		cols[c] = append(cols[c], i)
	}
	at := make([]float64, len(col))
	for _, ids := range cols {
		for k, i := range ids {
			at[i] = float64(k)
		}
	}
	sweep := func(ids []int, left bool) {
		key := make(map[int]float64, len(ids))
		for _, i := range ids {
			var sum float64
			var links int
			for _, k := range adj[i] {
				if left && col[k] < col[i] || !left && col[k] > col[i] {
					sum, links = sum+at[k], links+1
				}
			}
			key[i] = at[i]
			if links > 0 {
				key[i] = sum / float64(links)
			}
		}
		slices.SortStableFunc(ids, func(a, b int) int { return cmp.Compare(key[a], key[b]) })
		for k, i := range ids {
			at[i] = float64(k)
		}
	}
	for range 4 {
		for c := 1; c < m; c++ {
			sweep(cols[c], true)
		}
		for c := m - 2; c >= 0; c-- {
			sweep(cols[c], false)
		}
	}

	size := func(i int) float32 {
		if i < n {
			return h[i]
		}
		return 0
	}
	gap := func(i, j int) float32 {
		switch {
		case i < n && j < n:
			return gapY
		case i >= n && j >= n:
			return laneGap
		}
		return gapY * 3 / 4
	}
	tall := make([]float32, m)
	for c, ids := range cols {
		for k, i := range ids {
			tall[c] += size(i)
			if k > 0 {
				tall[c] += gap(ids[k-1], i)
			}
		}
	}
	tallest := slices.Max(tall)
	pos := make([]ui.Point, len(col))
	wide := make([]float32, m)
	var x float32
	for c, ids := range cols {
		y := (tallest - tall[c]) / 2
		for k, i := range ids {
			if k > 0 {
				y += gap(ids[k-1], i)
			}
			pos[i] = ui.Point{X: x, Y: y}
			y += size(i)
			if i < n {
				wide[c] = max(wide[c], w[i])
			}
		}
		x += wide[c] + gapX
	}
	slots := make([][]slot, len(rels))
	for k, vs := range lanes {
		for _, v := range vs {
			slots[k] = append(slots[k], slot{pos[v].X, pos[v].X + wide[col[v]], pos[v].Y})
		}
	}
	return pos[:n], slots
}
