// Package upset shows how items fall among sets as an UpSet plot, which
// reads where a Venn diagram of more than three sets does not: a row for
// each set, with a bar of how many items are in it and its name on the
// left, and a column for each combination of sets, with dots filled in the
// rows of its sets and joined by a line, under a bar of how many items are
// in exactly those sets and no other, with its count. The columns sort by
// their size, the largest first, or by their degree, the fewest sets
// first, and show the first Limit of them; with Empty, the combinations no
// item is in show as well. Items in none of the sets show nowhere.
//
// Pointing at a column lights it, its dots and its sets, with a card naming
// its combination, "In Chrome and Firefox, not Safari", and how many items
// are in it; pointing at a set lights the columns it is in. A click selects
// a column, in the accent color, and the part of each set in it fills its
// set's bar; Shift-click adds a column or takes it away, and a click beside
// the columns or Esc clears them. The app reads the items selected in the
// frame they change:
//
//	ev := upset.View(c, &app.plot, sets, items) // items[i] lists the sets of item i
//	if ev.Changed() {
//		app.chosen = ev.Selected()
//	}
package upset

import (
	"cmp"
	"math/bits"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Order is how a plot sorts its columns.
type Order uint8

const (
	// BySize puts the columns with the most items first, and those with as
	// many by their degree.
	BySize Order = iota
	// ByDegree puts the columns of the fewest sets first, and those of as
	// many sets by their size.
	ByDegree
)

// State is what a plot keeps: the columns selected, by their sets, so that
// they hold as the items change, until they no longer show. Order, Limit
// and Empty, which the app sets, sort the columns, show at most Limit of
// them, or all for 0, and show the combinations no item is in as well, for
// up to 12 sets. Keep it in the app's state, one for each plot; the zero
// State shows every combination with items, the largest first, and
// selects none.
type State struct {
	Order Order
	Limit int
	Empty bool

	selected map[uint64]bool
	// chosen is the items selected in the last frame, for Events, and
	// masks, sizes, totals, cols and pick hold what a frame computes, for
	// their storage.
	chosen []int
	masks  []uint64
	sizes  map[uint64]int
	totals []int
	cols   []column
	pick   []int
}

// column is a combination of sets, a bit for each, how many items are in
// exactly those sets, and how many sets it has.
type column struct {
	mask   uint64
	size   int
	degree int
}

// most is how many sets a plot shows, a bit of a mask each, and emptyMost
// how many it shows the empty combinations of.
const (
	most      = 64
	emptyMost = 12
)

// tally counts the items in each of n sets and in each combination of them,
// into totals and cols, sorted, and keeps each item's combination in masks.
func (s *State) tally(n int, items [][]int) {
	s.masks = s.masks[:0]
	s.totals = slices.Grow(s.totals[:0], n)[:n]
	clear(s.totals)
	if s.sizes == nil {
		s.sizes = map[uint64]int{}
	}
	clear(s.sizes)
	for _, sets := range items {
		var m uint64
		for _, k := range sets {
			if k >= 0 && k < n {
				m |= 1 << k
			}
		}
		s.masks = append(s.masks, m)
		if m != 0 {
			s.sizes[m]++
		}
		for b := m; b != 0; b &= b - 1 {
			s.totals[bits.TrailingZeros64(b)]++
		}
	}
	s.cols = s.cols[:0]
	for m, size := range s.sizes {
		s.cols = append(s.cols, column{m, size, bits.OnesCount64(m)})
	}
	if s.Empty && n <= emptyMost {
		for m := uint64(1); m < 1<<n; m++ {
			if s.sizes[m] == 0 {
				s.cols = append(s.cols, column{m, 0, bits.OnesCount64(m)})
			}
		}
	}
	by := s.Order
	slices.SortFunc(s.cols, func(a, b column) int {
		size, degree := cmp.Compare(b.size, a.size), cmp.Compare(a.degree, b.degree)
		if by == ByDegree {
			size, degree = degree, size
		}
		// Combinations alike otherwise go in the order of their first sets.
		first := 0
		if d := a.mask ^ b.mask; d != 0 {
			first = 1
			if a.mask&(d&-d) != 0 {
				first = -1
			}
		}
		return cmp.Or(size, degree, first)
	})
}

// click selects the column of mask alone, or with add as well as those
// selected, taking it away where it was; a mask of 0, a click beside the
// columns, clears them, unless add.
func (s *State) click(mask uint64, add bool) {
	if s.selected == nil {
		s.selected = map[uint64]bool{}
	}
	switch {
	case mask == 0 && !add:
		clear(s.selected)
	case mask == 0:
	case add && s.selected[mask]:
		delete(s.selected, mask)
	case add:
		s.selected[mask] = true
	case s.selected[mask] && len(s.selected) == 1:
		clear(s.selected)
	default:
		clear(s.selected)
		s.selected[mask] = true
	}
}

// Events is what a plot's selection did in a frame.
type Events struct {
	selected []int
	changed  bool
}

// Selected returns the indexes of the items in the columns selected, in
// order: none while no column is. It is a new slice each time they change,
// so one the app keeps stays as it was.
func (e Events) Selected() []int { return e.selected }

// Changed reports whether the items selected changed in this frame, as the
// user selected a column, or the items or the columns showing did.
func (e Events) Changed() bool { return e.changed }

// layout is where a plot's parts go: the sets' bars end at barX, barW long
// at most, their names start at nameX, nameW wide at most, and the columns
// at colX, colW wide each; the columns' bars rise from base as far as top,
// and the rows of the sets start at rowY, rowH high each.
type layout struct {
	barX, barW, nameX, nameW, colX, colW float32
	top, base, rowY, rowH                float32
}

// newLayout lays out a plot w by h of sets rows and cols columns, the sets'
// counts countW wide at most and their names nameW, its small text th
// high.
func newLayout(t *ui.Theme, w, h float32, sets, cols int, countW, nameW, th float32) layout {
	l := layout{nameW: nameW, top: th + t.Space(1.5)}
	l.rowH = max(t.Space(4), min(t.Space(7), h*0.5/float32(max(1, sets))))
	l.rowY = h - float32(sets)*l.rowH
	l.base = l.rowY - t.Space(2)
	l.barW = max(t.Space(12), min(t.Space(30), w*0.12))
	l.barX = countW + t.Space(1.5) + l.barW
	l.nameX = l.barX + t.Space(3)
	l.colX = l.nameX + nameW + t.Space(3)
	l.colW = min(t.Space(8), max(0, w-l.colX)/float32(max(1, cols)))
	return l
}

// hit returns the column at x, y in the plot, or the set whose row it is
// in beside the columns, -1 for none.
func (l layout) hit(x, y float32, sets, cols int) (col, set int) {
	right, bottom := l.colX+float32(cols)*l.colW, l.rowY+float32(sets)*l.rowH
	switch {
	case y < 0 || y >= bottom:
	case x >= l.colX && x < right:
		return min(cols-1, int((x-l.colX)/l.colW)), -1
	case y >= l.rowY && x >= 0:
		return -1, min(sets-1, int((y-l.rowY)/l.rowH))
	}
	return -1, -1
}

// View creates the plot of items among sets, items[i] the indexes of the
// sets item i is in, in any order; sets past the 64th are left out. It
// fills its parent. The Events returned tell which items are selected.
func View(c *ui.Context, s *State, sets []string, items [][]int) Events {
	t := c.Theme()
	sets = sets[:min(len(sets), most)]
	ns := len(sets)
	s.tally(ns, items)
	cols := s.cols
	if s.Limit > 0 {
		cols = cols[:min(len(cols), s.Limit)]
	}
	for m := range s.selected {
		if !slices.ContainsFunc(cols, func(c column) bool { return c.mask == m }) {
			delete(s.selected, m)
		}
	}

	small, nameSize := t.FontSize-2, t.FontSize-1
	_, th := c.MeasureText(0, ui.Span{Text: "0", Size: small})
	_, nh := c.MeasureText(0, ui.Span{Text: "M", Size: nameSize, Weight: 600})
	countW, nameW := float32(0), float32(0)
	for k, name := range sets {
		w, _ := c.MeasureText(0, ui.Span{Text: number(s.totals[k]), Size: small})
		countW = max(countW, w)
		w, _ = c.MeasureText(0, ui.Span{Text: name, Size: nameSize, Weight: 600})
		nameW = max(nameW, w)
	}
	nameW = min(nameW, t.Space(40))

	box := ui.Box(c).Fill().Focusable().Label("UpSet plot")
	b := box.LayoutBounds()
	l := newLayout(t, b.W, b.H, ns, len(cols), countW, nameW, th)
	hotCol, hotSet := -1, -1
	px, py, over := box.PointerPosition()
	if over {
		hotCol, hotSet = l.hit(px, py, ns, len(cols))
	}
	if hotCol >= 0 {
		box.Cursor(ui.CursorPointer)
	}
	if box.Clicked() && over {
		mask := uint64(0)
		if hotCol >= 0 {
			mask = cols[hotCol].mask
		}
		s.click(mask, box.ClickModifiers()&ui.Shift != 0)
	}
	if len(s.selected) > 0 && box.Shortcut(0, ui.KeyEscape) {
		clear(s.selected)
	}

	pick := s.pick[:0]
	if len(s.selected) > 0 {
		for i, m := range s.masks {
			if s.selected[m] {
				pick = append(pick, i)
			}
		}
	}
	s.pick = pick
	var ev Events
	if !slices.Equal(pick, s.chosen) {
		s.chosen, ev.changed = append([]int{}, pick...), true
	}
	ev.selected = s.chosen

	// inSel is how many of each set's items are selected, and lit which
	// columns the pointer lights.
	inSel := make([]int, ns)
	lit := make([]bool, len(cols))
	maxSize, maxTotal := 1, 1
	for i, col := range cols {
		maxSize = max(maxSize, col.size)
		lit[i] = i == hotCol || hotSet >= 0 && col.mask>>hotSet&1 != 0
		if !s.selected[col.mask] {
			continue
		}
		for b := col.mask; b != 0; b &= b - 1 {
			inSel[bits.TrailingZeros64(b)] += col.size
		}
	}
	for _, n := range s.totals {
		maxTotal = max(maxTotal, n)
	}
	hotMask := uint64(0)
	if hotCol >= 0 {
		hotMask = cols[hotCol].mask
	}

	shown := number(len(cols)) + " intersections"
	switch {
	case len(cols) == 1 && len(s.cols) == 1:
		shown = "1 intersection"
	case len(cols) < len(s.cols):
		shown = "Top " + number(len(cols)) + " of " + number(len(s.cols)) + " intersections"
	}
	status, hint := "", []string{"Click a column to select it,", "Shift-click to add more"}
	if len(s.selected) > 0 {
		status = strconv.Itoa(len(s.selected)) + " selected: " + number(len(pick)) + " of " + number(len(items))
		hint = []string{"Shift-click to add or remove,", "Esc to clear"}
	}
	box.Value(strings.TrimSuffix(shown+", "+status, ", "))

	if hotCol >= 0 {
		col := cols[hotCol]
		// The card goes by the top of the column's bar, high enough above
		// the dots not to hide them.
		y := min(l.base-(l.base-l.top)*float32(col.size)/float32(maxSize), l.base-t.Space(18))
		box.Children(func() {
			anchor := ui.Box(c).Absolute().PassThrough().Left(l.colX+float32(hotCol)*l.colW).Top(max(0, y)).
				Size(l.colW, 1)
			card(c, anchor, phrase(sets, col.mask), col.size, len(items))
		})
	}

	selected := s.selected
	totals := s.totals
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		l := newLayout(t, r.W, r.H, ns, len(cols), countW, nameW, th)
		ox, oy := r.X, r.Y
		right, bottom := ox+l.colX+float32(len(cols))*l.colW, oy+l.rowY+float32(ns)*l.rowH
		hover := hotCol >= 0 || hotSet >= 0

		for k := range ns {
			y := oy + l.rowY + float32(k)*l.rowH
			band := ui.Rect{X: ox + l.nameX - t.Space(1.5), Y: y, W: right - ox - l.nameX + t.Space(1.5), H: l.rowH}
			switch {
			case k == hotSet:
				band.X, band.W = ox, right-ox
				p.Fill(band, t.Text.Alpha(0.07), 4)
			case k%2 == 1:
				p.Fill(band, t.Text.Alpha(0.035), 4)
			}
		}
		for i, col := range cols {
			x := ox + l.colX + float32(i)*l.colW
			if selected[col.mask] {
				p.Fill(ui.Rect{X: x, Y: oy, W: l.colW, H: bottom - oy}, t.Accent.Alpha(0.1), 4)
			}
			if i == hotCol {
				p.Fill(ui.Rect{X: x, Y: oy, W: l.colW, H: bottom - oy}, t.Text.Alpha(0.06), 4)
			}
		}

		// The sets: a bar of each one's size, ending by its name, and its
		// count past the bar's start.
		barH := max(2, l.rowH*0.5)
		for k, name := range sets {
			y := oy + l.rowY + float32(k)*l.rowH
			on := k == hotSet || hotMask>>k&1 != 0
			fill, text, weight := t.Text.Alpha(0.45), t.Text, 400
			switch {
			case on:
				fill, weight = t.Text.Alpha(0.8), 600
			case hover:
				fill, text = t.Text.Alpha(0.2), t.TextMuted
			}
			w := l.barW * float32(totals[k]) / float32(maxTotal)
			bar := ui.Rect{X: ox + l.barX - w, Y: y + (l.rowH-barH)/2, W: w, H: barH}
			p.Fill(bar, fill, 2)
			if inSel[k] > 0 {
				sw := max(1, l.barW*float32(inSel[k])/float32(maxTotal))
				p.Fill(ui.Rect{X: ox + l.barX - sw, Y: bar.Y, W: sw, H: barH}, t.Accent, 2)
			}
			label := number(totals[k])
			cw, _ := p.MeasureText(0, ui.Span{Text: label, Size: small})
			p.Text(bar.X-t.Space(1.5)-cw, y+(l.rowH-th)/2, label, small, t.TextMuted)
			name = fit(p, name, nameSize, l.nameW)
			p.RichText(ox+l.nameX, y+(l.rowH-nh)/2, 0, ui.Span{Text: name, Size: nameSize, Weight: weight, Color: text})
		}

		// The columns: a bar of each one's size with its count, and its
		// dots, those of its sets filled and joined.
		r0 := max(2, min(l.colW*0.3, l.rowH*0.26))
		bw := max(1, min(l.colW-2, l.colW*0.64))
		halo := t.Background.Alpha(0.85)
		// The counts show over every column where the largest fits, else
		// over the one the pointer is on alone.
		widest, _ := p.MeasureText(0, ui.Span{Text: number(maxSize), Size: small})
		counts := widest+t.Space(1) <= l.colW
		for i, col := range cols {
			cx := ox + l.colX + float32(i)*l.colW + l.colW/2
			color, label := t.Text.Alpha(0.75), t.TextMuted
			if selected[col.mask] {
				color = t.Accent
			}
			switch {
			case lit[i]:
				label = t.Text
				if !selected[col.mask] {
					color = t.Text.Alpha(0.9)
				}
			case hover:
				color, label = color.Alpha(float32(color.A)/255*0.3), label.Alpha(0.4)
			}
			h := (l.base - l.top) * float32(col.size) / float32(maxSize)
			if h > 0 {
				p.Fill(ui.Rect{X: cx - bw/2, Y: oy + l.base - h, W: bw, H: h}, color, min(2, bw/2))
			}
			text := number(col.size)
			w, _ := p.MeasureText(0, ui.Span{Text: text, Size: small})
			if counts || i == hotCol {
				ly := oy + l.base - h - t.Space(1) - th
				if i == hotCol {
					p.Fill(ui.Rect{X: cx - w/2 - 2, Y: ly, W: w + 4, H: th}, halo, 3)
				}
				p.Text(cx-w/2, ly, text, small, label)
			}
			// The line runs over the dots of the sets left out, and under those
			// of its own.
			first, last := float32(-1), float32(-1)
			for k := range ns {
				cy := oy + l.rowY + float32(k)*l.rowH + l.rowH/2
				if col.mask>>k&1 == 0 {
					p.Fill(ui.Rect{X: cx - r0, Y: cy - r0, W: 2 * r0, H: 2 * r0}, t.Text.Alpha(0.1), r0)
					continue
				}
				if first < 0 {
					first = cy
				}
				last = cy
			}
			if last > first {
				p.Line(cx, first, cx, last, max(1.5, r0*0.45), color)
			}
			for b := col.mask; b != 0; b &= b - 1 {
				cy := oy + l.rowY + float32(bits.TrailingZeros64(b))*l.rowH + l.rowH/2
				p.Fill(ui.Rect{X: cx - r0, Y: cy - r0, W: 2 * r0, H: 2 * r0}, color, r0)
			}
		}

		// The corner above the sets: how many columns show, what is selected
		// and what a click does, and what the sets' bars are.
		width := l.colX - t.Space(4)
		y := oy
		_, h := p.RichText(ox, y, width, ui.Span{Text: shown, Size: nameSize, Weight: 600, Color: t.Text})
		y += h + t.Space(1)
		if status != "" {
			_, h = p.RichText(ox, y, width, ui.Span{Text: status, Size: small, Weight: 600, Color: t.Accent})
			y += h + t.Space(0.5)
		}
		for _, line := range hint {
			_, h = p.RichText(ox, y, width, ui.Span{Text: line, Size: small, Color: t.TextMuted})
			y += h
		}
		w, _ := p.MeasureText(0, ui.Span{Text: "Set size", Size: small})
		p.Text(ox+l.barX-w, oy+l.rowY-th-t.Space(1), "Set size", small, t.TextMuted)
	})
	return ev
}

// card shows by anchor, the bar of a column, the card of its combination,
// named, with size of all its items.
func card(c *ui.Context, anchor ui.Element, name string, size, all int) {
	t := c.Theme()
	share := ""
	if all > 0 {
		share = " · " + strconv.FormatFloat(100*float64(size)/float64(all), 'f', 1, 64) + "%"
	}
	ui.Overlay(c, func() {
		// The card and all in it let the pointer through to the plot under
		// it, which would otherwise lose the hover and the card.
		tip := ui.Column(c).Absolute().PassThrough().Role(ui.RoleTooltip).MaxWidth(t.Space(72)).
			AttachTo(anchor, ui.AnchorTopRight, ui.AnchorTopLeft).Margin(0, t.Space(2)).
			Padding(t.Space(2.5), t.Space(3)).Gap(t.Space(1)).Radius(t.Space(2)).
			Background(t.Background).Border(1, t.Border).Shadow(0, 4, 16, 0, ui.RGBA(0, 0, 0, 0.18))
		tip.Children(func() {
			ui.Text(c, name).PassThrough().Bold()
			ui.Row(c).PassThrough().Gap(t.Space(1.5)).AlignItems(ui.End).Children(func() {
				ui.Text(c, number(size)).PassThrough().FontSize(t.FontSize + 4).FontWeight(600).FontFeatures("tnum")
				ui.Text(c, "of "+number(all)+share).PassThrough().TextColor(t.TextMuted).FontFeatures("tnum")
			})
		})
	})
}

// phrase names the combination mask of sets: "In Chrome and Firefox, not
// Safari", "Only in Chrome", or "In Chrome, Firefox and Safari" for all of
// them.
func phrase(sets []string, mask uint64) string {
	var in, out []string
	for k, name := range sets {
		if mask>>k&1 != 0 {
			in = append(in, name)
		} else {
			out = append(out, name)
		}
	}
	switch {
	case len(out) == 0:
		return "In " + join(in, "and")
	case len(in) == 1:
		return "Only in " + in[0]
	}
	return "In " + join(in, "and") + ", not " + join(out, "or")
}

// join writes names as a list ending with word: "A", "A and B", "A, B and
// C".
func join(names []string, word string) string {
	if len(names) < 2 {
		return strings.Join(names, "")
	}
	return strings.Join(names[:len(names)-1], ", ") + " " + word + " " + names[len(names)-1]
}

// fit returns name, cut short with an ellipsis to be width wide at most.
func fit(p *ui.Painter, name string, size, width float32) string {
	w, _ := p.MeasureText(0, ui.Span{Text: name, Size: size, Weight: 600})
	if w <= width {
		return name
	}
	runes := []rune(name)
	for n := len(runes) - 1; n > 0; n-- {
		cut := string(runes[:n]) + "…"
		if w, _ := p.MeasureText(0, ui.Span{Text: cut, Size: size, Weight: 600}); w <= width {
			return cut
		}
	}
	return "…"
}

// number writes n with commas: 1,000.
func number(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
