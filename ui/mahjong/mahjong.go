// Package mahjong draws mahjong tiles and holds a hand of them. A tile
// shows its face on ivory over a green back: the dots as coins, the
// bamboo as sticks, the characters as a numeral over 萬, the winds and
// dragons as their characters, or its back when face down.
//
// A hand stands its tiles in a row, the tile just drawn a little apart and
// the melds called after it. The tile pointed at rises; a click chooses a
// tile and a second click, a double-click or Enter discards it, and ← and
// → move the choice. Below the hand, the tiles it waits on once it is one
// tile from complete, and with a tile too many, a dot under each tile
// whose discard leaves it waiting, and for the tile chosen, what it would
// wait on.
//
//	ev := mahjong.Hand(c, &app.hand, &app.tiles, app.melds)
//	if ev.Discarded != nil {
//		app.river = append(app.river, *ev.Discarded)
//	}
package mahjong

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// Suit is a tile's suit, or its kind for the honors.
type Suit uint8

const (
	Dots Suit = iota
	Bamboo
	Characters
	Winds
	Dragons
)

// Tile is a mahjong tile: its suit and its rank, from 1 to 9 in the three
// suits, East, South, West and North from 1 for the winds, and red, green
// and white from 1 for the dragons.
type Tile struct {
	Suit Suit
	Rank int
}

// The honors by name.
var (
	East  = Tile{Winds, 1}
	South = Tile{Winds, 2}
	West  = Tile{Winds, 3}
	North = Tile{Winds, 4}
	Red   = Tile{Dragons, 1}
	Green = Tile{Dragons, 2}
	White = Tile{Dragons, 3}
)

var (
	numbers   = [...]string{"", "One", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight", "Nine"}
	numerals  = [...]string{"", "一", "二", "三", "四", "五", "六", "七", "八", "九"}
	suitNames = [...]string{"Dots", "Bamboo", "Characters"}
	winds     = [...]string{"", "東", "南", "西", "北"}
	windNames = [...]string{"", "East", "South", "West", "North"}
	dragons   = [...]string{"", "中", "發", ""}
	dragNames = [...]string{"", "Red", "Green", "White"}
)

func (t Tile) name() string {
	switch t.Suit {
	case Winds:
		return windNames[t.Rank] + " wind"
	case Dragons:
		return dragNames[t.Rank] + " dragon"
	}
	return numbers[t.Rank] + " of " + suitNames[t.Suit]
}

// index numbers the 34 kinds of tile in the order Sort puts them.
func (t Tile) index() int {
	if t.Suit <= Characters {
		return int(t.Suit)*9 + t.Rank - 1
	}
	if t.Suit == Winds {
		return 27 + t.Rank - 1
	}
	return 31 + t.Rank - 1
}

func tileAt(i int) Tile {
	switch {
	case i < 27:
		return Tile{Suit(i / 9), i%9 + 1}
	case i < 31:
		return Tile{Winds, i - 26}
	}
	return Tile{Dragons, i - 30}
}

// Set returns the 136 tiles: four of each suit's nine and of each honor.
func Set() []Tile {
	var s []Tile
	for i := range 34 {
		for range 4 {
			s = append(s, tileAt(i))
		}
	}
	return s
}

// Sort puts tiles in order: dots, bamboo and characters by rank, then the
// winds and the dragons.
func Sort(tiles []Tile) {
	slices.SortStableFunc(tiles, func(a, b Tile) int { return a.index() - b.index() })
}

// Meld is a set called from another player's discard, or a kong declared:
// three or four tiles, shown after the hand, the outer two of a concealed
// kong face down.
type Meld struct {
	Tiles     []Tile
	Concealed bool
}

// Waits returns the tiles that would complete hand, given how many melds
// it has called, in Sort's order: four sets and a pair, or seven pairs.
func Waits(hand []Tile, melds int) []Tile {
	var n [34]int
	for _, t := range hand {
		n[t.index()]++
	}
	var out []Tile
	for i := range 34 {
		if n[i] == 4 {
			continue
		}
		n[i]++
		if complete(&n, 4-melds) {
			out = append(out, tileAt(i))
		}
		n[i]--
	}
	return out
}

// Complete reports whether hand, with its called melds, is a winning hand.
func Complete(hand []Tile, melds int) bool {
	var n [34]int
	for _, t := range hand {
		n[t.index()]++
	}
	return complete(&n, 4-melds)
}

func complete(n *[34]int, sets int) bool {
	total, pairs := 0, 0
	for _, k := range n {
		total += k
		if k == 2 {
			pairs++
		}
	}
	if total != 3*sets+2 {
		return false
	}
	if sets == 4 && pairs == 7 {
		return true
	}
	for i := range n {
		if n[i] >= 2 {
			n[i] -= 2
			ok := decompose(n, sets)
			n[i] += 2
			if ok {
				return true
			}
		}
	}
	return false
}

// decompose reports whether n splits into sets pungs and chows.
func decompose(n *[34]int, sets int) bool {
	i := slices.IndexFunc(n[:], func(k int) bool { return k > 0 })
	if i < 0 {
		return sets == 0
	}
	if n[i] >= 3 {
		n[i] -= 3
		ok := decompose(n, sets-1)
		n[i] += 3
		if ok {
			return true
		}
	}
	if i < 27 && i%9 <= 6 && n[i+1] > 0 && n[i+2] > 0 {
		n[i]--
		n[i+1]--
		n[i+2]--
		ok := decompose(n, sets-1)
		n[i]++
		n[i+1]++
		n[i+2]++
		return ok
	}
	return false
}

// Size returns the size of a tile in a hand.
func Size(c *ui.Context) (float32, float32) {
	t := c.Theme()
	return t.Space(10), t.Space(13.5)
}

// Face creates a tile, its face up or down, the size of a hand's; Size
// on it draws it larger or smaller.
func Face(c *ui.Context, tile Tile, up bool) ui.Element {
	w, h := Size(c)
	e := ui.Box(c).Size(w, h).Shrink(0).Label(tile.name())
	if !up {
		e.Label("Face down")
	}
	return dress(e, tile, up)
}

var (
	ivory = ui.Hex("#fbf7ec")
	back  = ui.Hex("#1f7a63")
	ink   = ui.Hex("#1d2a3a")
	red   = ui.Hex("#c8302c")
	green = ui.Hex("#23845a")
	blue  = ui.Hex("#2257a6")
)

// dress draws tile on e.
func dress(e ui.Element, tile Tile, up bool) ui.Element {
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		rad, depth := r.W*0.14, r.H*0.09
		p.Shadow(r, rad, 0, 1.5, 3, 0, ui.RGBA(0, 0, 0, 0.25))
		p.Fill(r, back, rad)
		f := ui.Rect{X: r.X, Y: r.Y, W: r.W, H: r.H - depth}
		if !up {
			p.Fill(f, back.Mix(ui.RGB(255, 255, 255), 0.18), rad)
			return
		}
		p.Fill(f, ivory, rad)
		p.Stroke(f, ui.RGBA(0, 0, 0, 0.12), rad, 1)
		in := ui.Rect{X: f.X + f.W*0.12, Y: f.Y + f.H*0.1, W: f.W * 0.76, H: f.H * 0.8}
		switch tile.Suit {
		case Dots:
			drawDots(p, in, tile.Rank)
		case Bamboo:
			drawBamboo(p, in, tile.Rank)
		case Characters:
			glyph(p, in.X+in.W/2, in.Y+in.H*0.28, numerals[tile.Rank], in.H*0.4, ink)
			glyph(p, in.X+in.W/2, in.Y+in.H*0.74, "萬", in.H*0.46, red)
		case Winds:
			glyph(p, in.X+in.W/2, in.Y+in.H/2, winds[tile.Rank], in.H*0.62, ink)
		case Dragons:
			if tile.Rank == 3 {
				p.Stroke(ui.Rect{X: in.X + in.W*0.08, Y: in.Y + in.H*0.12, W: in.W * 0.84, H: in.H * 0.76}, blue, 2, 2)
				break
			}
			glyph(p, in.X+in.W/2, in.Y+in.H/2, dragons[tile.Rank], in.H*0.62, []ui.Color{red, red, green}[tile.Rank])
		}
	})
	return e
}

// glyph draws s centered on (x, y).
func glyph(p *ui.Painter, x, y float32, s string, size float32, col ui.Color) {
	span := ui.Span{Text: s, Size: size, Weight: 700, Color: col, Lang: "zh-Hant"}
	w, h := p.MeasureText(0, span)
	p.RichText(x-w/2, y-h/2, 0, span)
}

// dotAt are where a dots tile's coins stand, by rank.
var dotAt = [...][][2]float32{
	1: {{0.5, 0.5}},
	2: {{0.5, 0.25}, {0.5, 0.75}},
	3: {{0.22, 0.18}, {0.5, 0.5}, {0.78, 0.82}},
	4: {{0.27, 0.27}, {0.73, 0.27}, {0.27, 0.73}, {0.73, 0.73}},
	5: {{0.25, 0.2}, {0.75, 0.2}, {0.5, 0.5}, {0.25, 0.8}, {0.75, 0.8}},
	6: {{0.28, 0.17}, {0.72, 0.17}, {0.28, 0.55}, {0.72, 0.55}, {0.28, 0.85}, {0.72, 0.85}},
	7: {{0.2, 0.12}, {0.5, 0.22}, {0.8, 0.32}, {0.28, 0.6}, {0.72, 0.6}, {0.28, 0.87}, {0.72, 0.87}},
	8: {{0.28, 0.12}, {0.72, 0.12}, {0.28, 0.37}, {0.72, 0.37}, {0.28, 0.63}, {0.72, 0.63}, {0.28, 0.88}, {0.72, 0.88}},
	9: {{0.2, 0.17}, {0.5, 0.17}, {0.8, 0.17}, {0.2, 0.5}, {0.5, 0.5}, {0.8, 0.5}, {0.2, 0.83}, {0.5, 0.83}, {0.8, 0.83}},
}

// dotColor returns the color of a dots tile's ith coin.
func dotColor(rank, i int) ui.Color {
	switch rank {
	case 1, 5:
		if rank == 1 || i == 2 {
			return red
		}
		return blue
	case 2:
		return []ui.Color{green, blue}[i]
	case 3:
		return []ui.Color{blue, red, green}[i]
	case 4:
		return []ui.Color{blue, green, green, blue}[i]
	case 6:
		if i < 2 {
			return green
		}
		return red
	case 7:
		if i < 3 {
			return green
		}
		return red
	case 8:
		return blue
	}
	return []ui.Color{blue, red, green}[i/3]
}

func drawDots(p *ui.Painter, r ui.Rect, rank int) {
	rad := r.W * 0.17
	switch {
	case rank == 1:
		rad = r.W * 0.46
	case rank <= 4:
		rad = r.W * 0.22
	}
	for i, at := range dotAt[rank] {
		x, y := r.X+r.W*at[0], r.Y+r.H*at[1]
		col := dotColor(rank, i)
		var d, ring, mid ui.Path
		d.Circle(x, y, rad)
		p.FillPath(&d, col)
		ring.Circle(x, y, rad*0.62)
		p.FillPath(&ring, ivory)
		mid.Circle(x, y, rad*0.32)
		p.FillPath(&mid, col)
	}
}

// stickAt are where a bamboo tile's sticks stand, by rank: their centers,
// and how tall they are, as parts of the face.
var stickAt = [...][][3]float32{
	1: {{0.5, 0.5, 0.86}},
	2: {{0.5, 0.25, 0.4}, {0.5, 0.75, 0.4}},
	3: {{0.5, 0.25, 0.4}, {0.28, 0.75, 0.4}, {0.72, 0.75, 0.4}},
	4: {{0.28, 0.25, 0.4}, {0.72, 0.25, 0.4}, {0.28, 0.75, 0.4}, {0.72, 0.75, 0.4}},
	5: {{0.18, 0.25, 0.4}, {0.82, 0.25, 0.4}, {0.5, 0.5, 0.4}, {0.18, 0.75, 0.4}, {0.82, 0.75, 0.4}},
	6: {{0.18, 0.25, 0.4}, {0.5, 0.25, 0.4}, {0.82, 0.25, 0.4}, {0.18, 0.75, 0.4}, {0.5, 0.75, 0.4}, {0.82, 0.75, 0.4}},
	7: {{0.5, 0.16, 0.28}, {0.18, 0.5, 0.28}, {0.5, 0.5, 0.28}, {0.82, 0.5, 0.28}, {0.18, 0.84, 0.28}, {0.5, 0.84, 0.28}, {0.82, 0.84, 0.28}},
	8: {{0.12, 0.25, 0.4}, {0.37, 0.25, 0.4}, {0.63, 0.25, 0.4}, {0.88, 0.25, 0.4}, {0.12, 0.75, 0.4}, {0.37, 0.75, 0.4}, {0.63, 0.75, 0.4}, {0.88, 0.75, 0.4}},
	9: {{0.18, 0.16, 0.28}, {0.5, 0.16, 0.28}, {0.82, 0.16, 0.28}, {0.18, 0.5, 0.28}, {0.5, 0.5, 0.28}, {0.82, 0.5, 0.28}, {0.18, 0.84, 0.28}, {0.5, 0.84, 0.28}, {0.82, 0.84, 0.28}},
}

func drawBamboo(p *ui.Painter, r ui.Rect, rank int) {
	w := r.W * 0.17
	if rank == 1 {
		w = r.W * 0.3
	}
	for i, at := range stickAt[rank] {
		col := green
		if rank == 1 || rank == 5 && i == 2 || rank == 7 && i == 0 || rank == 9 && i%3 == 1 {
			col = red
		}
		h := r.H * at[2]
		s := ui.Rect{X: r.X + r.W*at[0] - w/2, Y: r.Y + r.H*at[1] - h/2, W: w, H: h}
		p.Fill(s, col, w/2)
		for _, k := range []float32{0.5, 0.06, 0.94} {
			y := s.Y + s.H*k
			p.Line(s.X, y, s.X+s.W, y, max(1, w*0.18), ivory)
		}
		p.Fill(ui.Rect{X: s.X + s.W*0.42, Y: s.Y + s.H*0.1, W: s.W * 0.16, H: s.H * 0.8}, ivory.Alpha(0.5), 0)
	}
}

// State is what a hand keeps: the tile chosen. Keep it in the app's state,
// one for each hand.
type State struct {
	chosen    int
	picked    bool
	discarded time.Time
}

// Events is a hand, and the tile discarded in the frame, taken out of it.
type Events struct {
	ui.Element
	Discarded *Tile
}

// Hand creates *hand, with melds called. With a tile more than a waiting
// hand holds, the last stands apart as the tile drawn.
func Hand(c *ui.Context, s *State, hand *[]Tile, melds []Meld) Events {
	t := c.Theme()
	var ev Events
	tiles := *hand
	n := len(tiles)
	drawn := n%3 == 2
	if s.chosen >= n {
		s.picked = false
	}
	discard := func(i int) {
		d := tiles[i]
		*hand = slices.Delete(slices.Clone(tiles), i, i+1)
		ev.Discarded = &d
		s.picked, s.discarded = false, c.Now()
	}
	// leaves are the waits after discarding each tile, by its index.
	var leaves map[int][]Tile
	if drawn {
		leaves = map[int][]Tile{}
		for i, x := range tiles {
			if _, ok := leaves[x.index()]; ok {
				continue
			}
			rest := slices.Delete(slices.Clone(tiles), i, i+1)
			leaves[x.index()] = Waits(rest, len(melds))
		}
	}
	root := ui.Column(c).Gap(t.Space(4)).Focusable().FocusRing(false).Label("Hand").Role(ui.RoleGroup)
	switch {
	case n == 0:
	case root.Shortcut(0, ui.KeyLeft):
		if !s.picked {
			s.chosen = n
		}
		s.chosen, s.picked = max(0, s.chosen-1), true
	case root.Shortcut(0, ui.KeyRight):
		if !s.picked {
			s.chosen = -1
		}
		s.chosen, s.picked = min(n-1, s.chosen+1), true
	case root.Shortcut(0, ui.KeyEnter) && s.picked:
		discard(s.chosen)
	}
	hovered := -1
	root.Children(func() {
		ui.Row(c).AlignItems(ui.End).Gap(t.Space(0.5)).Height(t.Space(19)).Children(func() {
			for i, tile := range tiles {
				if ev.Discarded != nil {
					break
				}
				if drawn && i == n-1 {
					ui.Box(c.Key("gap")).Width(t.Space(3))
				}
				chosen := s.picked && s.chosen == i
				w, h := Size(c)
				e := ui.ButtonBase(c.Key(i)).Size(w, h).Shrink(0).Label(tile.name()).Cursor(ui.CursorPointer)
				if chosen {
					e.Selected(true)
				}
				lift := float32(0)
				switch {
				case chosen:
					lift = t.Space(4)
				case e.Hovered():
					lift = t.Space(1.5)
					hovered = i
				}
				e.Translate(0, -e.Spring("mahjong.lift", lift))
				dress(e, tile, true)
				leaves := drawn && len(leaves[tile.index()]) > 0
				e.DrawOver(func(p *ui.Painter, r ui.Rect) {
					if chosen {
						p.Stroke(ui.Rect{X: r.X - 1.5, Y: r.Y - 1.5, W: r.W + 3, H: r.H + 3}, t.Accent, r.W*0.16, 2)
					}
					if leaves {
						var d ui.Path
						d.Circle(r.X+r.W/2, r.Y+r.H+t.Space(2)+lift, 2)
						p.FillPath(&d, t.Accent)
					}
				})
				switch {
				// The hand is built again in the frame of a discard, where
				// the click that made it would choose the tile now there.
				case c.Now().Equal(s.discarded):
				case e.DoubleClicked(), e.Clicked() && chosen:
					discard(i)
				case e.Clicked():
					s.chosen, s.picked = i, true
					e.SetChanged()
				}
			}
			for k, m := range melds {
				ui.Row(c.Key(fmt.Sprint("meld", k))).MarginStart(t.Space(4)).Gap(1).Label("Meld").Role(ui.RoleGroup).Children(func() {
					for j, tile := range m.Tiles {
						up := !(m.Concealed && (j == 0 || j == len(m.Tiles)-1))
						w, h := Size(c)
						Face(c.Key(j), tile, up).Size(w*0.82, h*0.82)
					}
				})
			}
		})
		status(c, s, tiles, melds, drawn, hovered, leaves)
	})
	if ev.Discarded != nil {
		root.SetChanged()
	}
	ev.Element = root
	return ev
}

// status shows what the hand waits on, or, with a tile drawn, what it
// would wait on after discarding the tile chosen or pointed at.
func status(c *ui.Context, s *State, tiles []Tile, melds []Meld, drawn bool, hovered int, leaves map[int][]Tile) {
	t := c.Theme()
	var label string
	var waits []Tile
	switch {
	case drawn && Complete(tiles, len(melds)):
		label = "Complete hand"
	case drawn:
		at := hovered
		if at < 0 && s.picked && s.chosen < len(tiles) {
			at = s.chosen
		}
		if at < 0 {
			k := 0
			for _, w := range leaves {
				if len(w) > 0 {
					k++
				}
			}
			if k > 0 {
				label = fmt.Sprintf("%d discards leave the hand waiting", k)
			}
			break
		}
		waits = leaves[tiles[at].index()]
		if len(waits) > 0 {
			label = "Discard " + tiles[at].name() + " to wait on"
		}
	default:
		waits = Waits(tiles, len(melds))
		if len(waits) > 0 {
			label = "Waiting on"
		}
	}
	names := make([]string, len(waits))
	for i, w := range waits {
		names[i] = w.name()
	}
	row := ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Height(t.Space(9))
	if label != "" {
		row.Label(strings.TrimSpace(label + " " + strings.Join(names, ", ")))
	}
	row.Children(func() {
		if label == "" {
			return
		}
		ui.Text(c, label).TextColor(t.TextMuted)
		for i, w := range waits {
			sw, sh := Size(c)
			Face(c.Key(i), w, true).Size(sw*0.6, sh*0.6)
		}
	})
}
