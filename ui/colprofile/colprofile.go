// Package colprofile shows a table whose header sums up each column, as
// Data Wrangler and Observable do: a histogram of a column of numbers
// with its least and greatest, the most common values of a column of text
// with their counts, and how many cells are empty. A click on a bar or a
// value keeps the rows in it, Shift-click on a bar widens that to the bars
// between, and another click on what is kept lets them all go; what each
// column keeps applies together. The histograms show the rows kept over
// all the rows.
//
//	colprofile.View(c, &app.profile, app.columns, app.rows).Grow(1)
//
// A column whose cells all read as numbers, the empty ones aside, is a
// column of numbers; an empty cell is a missing value.
package colprofile

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// bins is how many bars a histogram has at most: fewer for a column of
// whole numbers fewer apart, which has a bar for each.
const bins = 24

// top is how many of a text column's values its header names.
const top = 3

// State is what a view keeps: the rows' place, what each column keeps,
// and what it worked out of the rows. Keep it in the app's state, one for
// each view.
type State struct {
	list    ui.ListState
	filters map[int]filter
	version int
	made    made
	columns []column
	kept    []int
	counts  [][]int
}

// filter is what a column keeps: the bars from lo to hi of a column of
// numbers, or a value of a column of text.
type filter struct {
	lo, hi int
	value  string
}

// made is what the view's work came from: the rows, by their count and
// first row, and the filters, by the version their changes count.
type made struct {
	n       int
	first   *[]string
	filters int
}

// column is a column worked out: its numbers, NaN where empty, whether
// they are whole, their least and greatest, its bars, or its values most
// common first; and how many of its cells are empty, of all.
type column struct {
	numeric  bool
	whole    bool
	nums     []float64
	missing  int
	total    int
	min, max float64
	bars     []int
	values   []valueCount
}

type valueCount struct {
	value string
	count int
}

// View creates a table of rows under the columns' titles and summaries.
// Each row holds a cell for each column. Give it a size, or grow it in
// its parent.
func View(c *ui.Context, s *State, titles []string, rows [][]string) ui.Element {
	t := c.Theme()
	s.work(titles, rows)
	width := t.Space(46)
	root := ui.Column(c)
	root.Children(func() {
		ui.ScrollHorizontal(c).Grow(1).Children(func() {
			ui.Column(c).Width(float32(len(titles)) * width).FillHeight().Children(func() {
				ui.Row(c).Children(func() {
					for j, title := range titles {
						s.header(c.Key(j), j, title, width)
					}
				})
				ui.Divider(c)
				ui.List(c, &s.list, len(s.kept), func(i int) {
					row := rows[s.kept[i]]
					ui.Row(c).Children(func() {
						for j := range titles {
							s.cell(c, j, row, width)
						}
					})
				}).Grow(1)
			})
		})
		ui.Divider(c)
		ui.Row(c).Height(t.Space(10)).PaddingX(t.Space(3)).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, fmt.Sprintf("%d of %d rows", len(s.kept), len(rows))).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
			ui.Spacer(c)
			if len(s.filters) > 0 && ui.Button(c, "Show all").Clicked() {
				clear(s.filters)
				s.version++
			}
		})
	})
	return root
}

// work works the columns out of rows, and the rows the filters keep,
// when either changed.
func (s *State) work(titles []string, rows [][]string) {
	m := made{n: len(rows), filters: s.version}
	if len(rows) > 0 {
		m.first = &rows[0]
	}
	fresh := m.n != s.made.n || m.first != s.made.first || len(s.columns) != len(titles)
	if fresh {
		s.columns = make([]column, len(titles))
		for j := range titles {
			s.columns[j] = profile(rows, j)
		}
	}
	if !fresh && m == s.made {
		return
	}
	s.made = m
	s.kept = s.kept[:0]
	for i, row := range rows {
		if s.keeps(i, row) {
			s.kept = append(s.kept, i)
		}
	}
	s.counts = make([][]int, len(titles))
	for j, col := range s.columns {
		if !col.numeric {
			continue
		}
		s.counts[j] = make([]int, len(col.bars))
		for _, i := range s.kept {
			if v := col.nums[i]; !math.IsNaN(v) {
				s.counts[j][col.bin(v)]++
			}
		}
	}
}

// keeps reports whether the filters keep row i.
func (s *State) keeps(i int, row []string) bool {
	for j, f := range s.filters {
		col := s.columns[j]
		if col.numeric {
			v := col.nums[i]
			if math.IsNaN(v) {
				return false
			}
			if b := col.bin(v); b < f.lo || b > f.hi {
				return false
			}
		} else if j >= len(row) || row[j] != f.value {
			return false
		}
	}
	return true
}

func profile(rows [][]string, j int) column {
	col := column{numeric: true, nums: make([]float64, len(rows)), total: len(rows), min: math.Inf(1), max: math.Inf(-1)}
	counts := map[string]int{}
	for i, row := range rows {
		v := ""
		if j < len(row) {
			v = row[j]
		}
		if v == "" {
			col.missing++
			col.nums[i] = math.NaN()
			continue
		}
		counts[v]++
		n, err := strconv.ParseFloat(v, 64)
		if err != nil {
			col.numeric = false
		}
		col.nums[i] = n
		col.min, col.max = min(col.min, n), max(col.max, n)
	}
	if col.missing == len(rows) {
		col.numeric = false
	}
	if col.numeric {
		n := bins
		if whole(col.nums) && col.max-col.min < bins {
			n, col.whole = int(col.max-col.min)+1, true
		}
		col.bars = make([]int, n)
		for _, v := range col.nums {
			if !math.IsNaN(v) {
				col.bars[col.bin(v)]++
			}
		}
		return col
	}
	col.nums = nil
	for v, n := range counts {
		col.values = append(col.values, valueCount{v, n})
	}
	slices.SortFunc(col.values, func(a, b valueCount) int {
		return cmp.Or(b.count-a.count, cmp.Compare(a.value, b.value))
	})
	return col
}

func (col column) bin(v float64) int {
	n := len(col.bars)
	if col.whole {
		return int(v - col.min)
	}
	if col.max == col.min {
		return 0
	}
	return min(n-1, int((v-col.min)/(col.max-col.min)*float64(n)))
}

// whole reports whether nums are all whole numbers, NaN aside.
func whole(nums []float64) bool {
	for _, v := range nums {
		if !math.IsNaN(v) && v != math.Trunc(v) {
			return false
		}
	}
	return true
}

// header builds column j's title and summary.
func (s *State) header(c *ui.Context, j int, title string, width float32) {
	t := c.Theme()
	col := s.columns[j]
	f, filtered := s.filters[j]
	ui.Column(c).Width(width).Padding(t.Space(2), t.Space(3)).Gap(t.Space(1.5)).Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, title).Bold().SingleLine().Ellipsis("…").Shrink(1)
			ui.Spacer(c)
			if col.missing > 0 {
				ui.Text(c, fmt.Sprintf("%.0f%% empty", 100*float64(col.missing)/float64(col.total))).FontSize(t.FontSize - 2).TextColor(t.TextMuted).SingleLine()
			}
		})
		box := ui.Column(c).Height(t.Space(19)).Gap(t.Space(1))
		box.Children(func() {
			if col.numeric {
				s.histogram(c, j, col, f, filtered)
				return
			}
			s.values(c, j, col, f, filtered)
		})
	})
}

func (s *State) histogram(c *ui.Context, j int, col column, f filter, filtered bool) {
	t := c.Theme()
	kept := s.counts[j]
	bars := ui.Box(c).Grow(1).Cursor(ui.CursorPointer).Role(ui.RoleButton).Label("Histogram")
	if bars.Clicked() {
		if x, _, ok := bars.PointerPosition(); ok {
			n := len(col.bars)
			b := max(0, min(n-1, int(x/bars.Bounds().W*float32(n))))
			switch {
			case filtered && bars.ClickModifiers()&ui.Shift != 0:
				s.keep(j, filter{lo: min(f.lo, b), hi: max(f.hi, b)})
			case filtered && f.lo <= b && b <= f.hi:
				s.keep(j, filter{lo: -1})
			default:
				s.keep(j, filter{lo: b, hi: b})
			}
		}
	}
	peak := slices.Max(col.bars)
	bars.Draw(func(p *ui.Painter, r ui.Rect) {
		w := r.W / float32(len(col.bars))
		for b, n := range col.bars {
			if n == 0 {
				continue
			}
			h := max(1, r.H*float32(n)/float32(peak))
			x := r.X + float32(b)*w
			p.Fill(ui.Rect{X: x + 0.5, Y: r.Y + r.H - h, W: w - 1, H: h}, t.Text.Alpha(0.12), 1)
			if k := kept[b]; k > 0 {
				color := t.Accent
				if filtered && (b < f.lo || b > f.hi) {
					color = t.Accent.Alpha(0.4)
				}
				kh := max(1, r.H*float32(k)/float32(peak))
				p.Fill(ui.Rect{X: x + 0.5, Y: r.Y + r.H - kh, W: w - 1, H: kh}, color, 1)
			}
		}
	})
	lo, hi := col.min, col.max
	if filtered {
		step := (col.max - col.min) / float64(len(col.bars))
		lo, hi = col.min+float64(f.lo)*step, col.min+float64(f.hi+1)*step
		if col.whole {
			lo, hi = col.min+float64(f.lo), col.min+float64(f.hi)
		}
	}
	ui.Row(c).Children(func() {
		ui.Text(c, short(lo)).FontSize(t.FontSize - 2).TextColor(t.TextMuted).FontFeatures("tnum")
		ui.Spacer(c)
		ui.Text(c, short(hi)).FontSize(t.FontSize - 2).TextColor(t.TextMuted).FontFeatures("tnum")
	})
}

func (s *State) values(c *ui.Context, j int, col column, f filter, filtered bool) {
	t := c.Theme()
	total := max(1, sum(col.values))
	for k, vc := range col.values[:min(top, len(col.values))] {
		on := filtered && f.value == vc.value
		row := ui.Row(c.Key(k)).Height(t.Space(5)).PaddingX(t.Space(1.5)).AlignItems(ui.Center).Radius(t.Radius).
			Cursor(ui.CursorPointer).Role(ui.RoleButton).Selected(on)
		if row.Clicked() {
			if on {
				s.keep(j, filter{lo: -1})
			} else {
				s.keep(j, filter{value: vc.value})
			}
		}
		share := float32(vc.count) / float32(total)
		fill := t.Text.Alpha(0.08)
		if on {
			fill = t.Accent.Alpha(0.3)
		}
		row.Draw(func(p *ui.Painter, r ui.Rect) {
			p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W * share, H: r.H}, fill, t.Radius)
		})
		row.Children(func() {
			ui.Text(c, vc.value).FontSize(t.FontSize - 1).SingleLine().Ellipsis("…").Shrink(1)
			ui.Spacer(c)
			ui.Text(c, strconv.Itoa(vc.count)).FontSize(t.FontSize - 2).TextColor(t.TextMuted).FontFeatures("tnum")
		})
	}
	if more := len(col.values) - top; more > 0 {
		ui.Text(c, fmt.Sprintf("%d more", more)).FontSize(t.FontSize - 2).TextColor(t.TextMuted).PaddingX(t.Space(1.5))
	}
}

// keep sets what column j keeps; a filter with lo -1 lets all go.
func (s *State) keep(j int, f filter) {
	if s.filters == nil {
		s.filters = map[int]filter{}
	}
	if f.lo < 0 {
		delete(s.filters, j)
	} else {
		s.filters[j] = f
	}
	s.version++
}

func (s *State) cell(c *ui.Context, j int, row []string, width float32) {
	t := c.Theme()
	v := ""
	if j < len(row) {
		v = row[j]
	}
	cell := ui.Text(c.Key(j), v).Width(width).Padding(t.Space(1.5), t.Space(3)).SingleLine().Ellipsis("…")
	switch {
	case v == "":
		cell.TextColor(t.TextMuted.Alpha(0.5))
	case s.columns[j].numeric:
		cell.TextAlign(ui.End).FontFeatures("tnum")
	}
}

func sum(values []valueCount) int {
	n := 0
	for _, v := range values {
		n += v.count
	}
	return n
}

// short writes v in a few characters: 1.2k, 3.4M.
func short(v float64) string {
	a := math.Abs(v)
	switch {
	case a >= 1e9:
		return strconv.FormatFloat(v/1e9, 'f', 1, 64) + "B"
	case a >= 1e6:
		return strconv.FormatFloat(v/1e6, 'f', 1, 64) + "M"
	case a >= 1e4:
		return strconv.FormatFloat(v/1e3, 'f', 1, 64) + "k"
	case a == math.Trunc(a):
		return strconv.FormatFloat(v, 'f', 0, 64)
	}
	return strconv.FormatFloat(v, 'g', 3, 64)
}
