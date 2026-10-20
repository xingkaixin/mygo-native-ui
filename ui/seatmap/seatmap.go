// Package seatmap is the seat map of a cinema or a plane: its seats in
// rows, colored by their price zone, the screen or the nose in front, a
// legend of the zones, and the seats chosen with their total below.
// Clicking a seat chooses it or gives it back, up to a limit; a sold seat
// takes no click. Hovering a seat shows its name and price. Changed
// reports a seat chosen or given back.
//
//	plan := seatmap.Plan{
//		Front: "Screen",
//		Rows:  []string{"0000  0000", "1111  1111"},
//		Zones: []seatmap.Zone{{Name: "Standard", Price: 12}, {Name: "Premium", Price: 16}},
//		Sold:  map[seatmap.Seat]bool{{Row: 0, Num: 3}: true},
//		Currency: "$",
//	}
//	seatmap.View(c, plan, &app.seats, 4)
package seatmap

import (
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
)

// Seat is a seat by its row, from 0, rows of no seats not counting, and
// its number in the row, from 1, gaps not counting.
type Seat struct {
	Row, Num int
}

// Zone is a price zone: what it is called and what a seat in it costs.
type Zone struct {
	Name  string
	Price float64
}

// Plan is the seats of a hall or a plane.
type Plan struct {
	// Front is what the seats face, as "Screen" or "Stage", drawn above
	// them; "" draws nothing.
	Front string
	// Rows are the rows from the front, a seat of zone k a digit k and a
	// gap a space; an empty row is an aisle across.
	Rows  []string
	Zones []Zone
	Sold  map[Seat]bool
	// Letters names the rows by number and the seats in a row by letter,
	// as a plane does (12C), rather than the rows by letter and the seats
	// by number, as a cinema does (C12).
	Letters bool
	// Currency is written before prices, as "$".
	Currency string
}

// Name returns the name of seat s, as C12, or 12C with Letters.
func (p Plan) Name(s Seat) string {
	if p.Letters {
		return strconv.Itoa(s.Row+1) + string(rune('A'+s.Num-1))
	}
	return row(s.Row) + strconv.Itoa(s.Num)
}

// Zone returns the zone of seat s, or false when there is no such seat.
func (p Plan) Zone(s Seat) (Zone, bool) {
	k := -1
	for _, line := range p.Rows {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if k++; k != s.Row {
			continue
		}
		n := 0
		for _, ch := range line {
			if ch < '0' || ch > '9' {
				continue
			}
			if n++; n == s.Num && int(ch-'0') < len(p.Zones) {
				return p.Zones[ch-'0'], true
			}
		}
	}
	return Zone{}, false
}

// Total returns what the seats cost.
func (p Plan) Total(seats []Seat) float64 {
	sum := 0.0
	for _, s := range seats {
		z, _ := p.Zone(s)
		sum += z.Price
	}
	return sum
}

// row returns the letters of row k: A to Z, then AA.
func row(k int) string {
	s := ""
	for k++; k > 0; k = (k - 1) / 26 {
		s = string(rune('A'+(k-1)%26)) + s
	}
	return s
}

// View creates the seat map of plan, the seats chosen in *chosen, at
// most limit of them, 0 for any number.
func View(c *ui.Context, plan Plan, chosen *[]Seat, limit int) ui.Element {
	t := c.Theme()
	locale := c.Region().FormatLocale(c.Locale())
	money := func(v float64) string {
		return plan.Currency + i18n.FormatNumber(locale, v, i18n.NumberOptions{MinFractionDigits: 2, MaxFractionDigits: 2})
	}
	full := limit > 0 && len(*chosen) >= limit
	size, gap := t.Space(6), t.Space(1.5)
	widest := 0
	for _, line := range plan.Rows {
		widest = max(widest, len(line))
	}
	width := float32(widest)*(size+gap) - gap
	root := ui.Column(c).Gap(t.Space(5)).AlignItems(ui.Center)
	root.Children(func() {
		if plan.Front != "" {
			ui.Column(c).Width(width).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(width, t.Space(4)).Draw(func(p *ui.Painter, r ui.Rect) {
					var arc ui.Path
					arc.MoveTo(r.X, r.Y+r.H).CubeTo(r.X+r.W*0.25, r.Y, r.X+r.W*0.75, r.Y, r.X+r.W, r.Y+r.H)
					p.StrokePath(&arc, 3, t.TextMuted.Alpha(0.6))
				})
				ui.Text(c, plan.Front).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
			})
		}
		ui.Column(c).Gap(gap).Children(func() {
			k := -1
			for i, line := range plan.Rows {
				if strings.TrimSpace(line) == "" {
					ui.Box(c.Key(i)).Height(size / 2)
					continue
				}
				k++
				r := k
				name := row(r)
				if plan.Letters {
					name = strconv.Itoa(r + 1)
				}
				ui.Row(c.Key(i)).Gap(gap).AlignItems(ui.Center).Children(func() {
					label := func() {
						ui.Text(c, name).FontSize(t.FontSize - 3).TextColor(t.TextMuted).Width(t.Space(6)).TextAlign(ui.Center)
					}
					label()
					pad := widest - len(line)
					for range pad / 2 {
						ui.Box(c).Size(size, size)
					}
					n := 0
					for col, ch := range line {
						if ch < '0' || ch > '9' || int(ch-'0') >= len(plan.Zones) {
							ui.Box(c.Key(col)).Size(size, size)
							continue
						}
						n++
						s := Seat{r, n}
						seat(c.Key(col), plan, s, int(ch-'0'), chosen, full, size, money, root)
					}
					for range pad - pad/2 {
						ui.Box(c).Size(size, size)
					}
					label()
				})
			}
		})
		legend(c, plan, size, money)
		footer(c, plan, *chosen, limit, money)
	})
	return root
}

// seat shows seat s of zone z, and chooses or gives it back as it is
// clicked.
func seat(c *ui.Context, plan Plan, s Seat, z int, chosen *[]Seat, full bool, size float32, money func(float64) string, root ui.Element) {
	t := c.Theme()
	color := hue(z)
	zone := plan.Zones[z]
	at := slices.Index(*chosen, s)
	sold := plan.Sold[s]
	name := plan.Name(s)
	el := ui.Box(c).Size(size, size).Radius(size * 0.3).Justify(ui.Center).AlignItems(ui.Center).Role(ui.RoleCheckBox).Checked(at >= 0).
		Label(name).Tooltip(name + " · " + zone.Name + " · " + money(zone.Price))
	switch {
	case sold:
		el.Disabled(true).Background(t.Text.Alpha(0.1)).Cursor(ui.CursorNotAllowed).Tooltip(name + " · Sold")
	case at >= 0:
		el.Background(color).Cursor(ui.CursorPointer)
	case full:
		el.Background(color.Alpha(0.12)).Border(1.5, color.Alpha(0.4)).Tooltip(name + " · " + money(zone.Price) + " · Limit reached")
	default:
		el.Focusable().Background(color.Alpha(0.18)).Border(1.5, color).Cursor(ui.CursorPointer)
		if el.Hovered() {
			el.Background(color.Alpha(0.45))
		}
	}
	if at >= 0 {
		el.Focusable()
	}
	if el.Clicked() && !sold {
		switch {
		case at >= 0:
			*chosen = slices.Delete(*chosen, at, at+1)
			root.SetChanged()
		case !full:
			*chosen = append(*chosen, s)
			root.SetChanged()
		}
	}
	el.Children(func() {
		if at >= 0 {
			mark := strconv.Itoa(s.Num)
			if plan.Letters {
				mark = string(rune('A' + s.Num - 1))
			}
			ui.Text(c, mark).FontSize(t.FontSize - 4).Bold().TextColor(ui.RGB(255, 255, 255))
		}
	})
}

// legend shows a swatch of each zone with its price, then of a seat
// chosen and one sold.
func legend(c *ui.Context, plan Plan, size float32, money func(float64) string) {
	t := c.Theme()
	swatch := func(key any, fill, border ui.Color, text string) {
		ui.Row(c.Key(key)).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
			b := ui.Box(c).Size(size*0.7, size*0.7).Radius(size * 0.2).Background(fill)
			if border.A > 0 {
				b.Border(1.5, border)
			}
			ui.Text(c, text).FontSize(t.FontSize - 1)
		})
	}
	ui.Row(c).Gap(t.Space(5)).Wrap().Justify(ui.Center).Children(func() {
		for i, z := range plan.Zones {
			swatch(i, hue(i).Alpha(0.18), hue(i), z.Name+" "+money(z.Price))
		}
		swatch("chosen", hue(0), ui.Color{}, "Chosen")
		swatch("sold", t.Text.Alpha(0.1), ui.Color{}, "Sold")
	})
}

// footer shows the seats chosen, in order, and what they cost.
func footer(c *ui.Context, plan Plan, chosen []Seat, limit int, money func(float64) string) {
	t := c.Theme()
	seats := slices.Clone(chosen)
	slices.SortFunc(seats, func(a, b Seat) int {
		if a.Row != b.Row {
			return a.Row - b.Row
		}
		return a.Num - b.Num
	})
	names := make([]string, len(seats))
	for i, s := range seats {
		names[i] = plan.Name(s)
	}
	ui.Row(c).AlignSelf(ui.Stretch).Gap(t.Space(4)).AlignItems(ui.Center).Padding(t.Space(3), t.Space(4)).
		Background(t.Surface).Radius(t.Space(2)).Border(1, t.Border).Children(func() {
		ui.Column(c).Grow(1).Gap(t.Space(0.5)).Children(func() {
			count := strconv.Itoa(len(seats)) + " seats"
			if len(seats) == 1 {
				count = "1 seat"
			}
			if limit > 0 {
				count += " of up to " + strconv.Itoa(limit)
			}
			ui.Text(c, count).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
			list := strings.Join(names, ", ")
			if list == "" {
				list = "Click seats to choose them"
			}
			ui.Text(c, list).Bold()
		})
		ui.Text(c, money(plan.Total(seats))).FontSize(t.FontSize + 6).Bold().FontFeatures("tnum")
	})
}

// hue returns zone k's color.
func hue(k int) ui.Color {
	return ui.Oklch(0.62, 0.14, float32((250+k*95)%360))
}
