// Package callgrid lays out the people in a video call, as meeting apps
// do: in a grid of equal tiles as large as the window allows, or with
// the one speaking large and the others in a strip below. A screen being
// shared takes the stage, the people beside it. Each tile shows the
// person's video, or their initials while their camera is off, their
// name with a slashed microphone when muted, a hand when raised, and a
// green outline that brightens with their voice. A double click pins a
// person to the stage, and another unpins them.
//
//	callgrid.View(c, &app.call, app.people).Grow(1)
package callgrid

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Participant is a person in the call: their name, which tells them
// apart, what their camera and their shared screen show, nil when off,
// how loud they speak now, from 0 to 1, whether they are muted or raise
// their hand, and the color of their initials' tile.
type Participant struct {
	Name   string
	Video  func(p *ui.Painter, r ui.Rect)
	Screen func(p *ui.Painter, r ui.Rect)
	Level  float32
	Muted  bool
	Hand   bool
	Color  ui.Color
}

// Layout is how the tiles are laid out.
type Layout int

const (
	Grid Layout = iota
	Speaker
)

// State is what a call keeps: its layout, the person pinned, and who
// spoke last. Keep it in the app's state, one for each call.
type State struct {
	// Layout is the tiles' layout, which the app switches.
	Layout  Layout
	pinned  string
	speaker string
}

// speaking is the level above which a person is speaking.
const speaking = 0.12

func svg(body string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` + body + `</svg>`))
}

var (
	micOff = svg(`<path d="M9 9v3a3 3 0 0 0 5.1 2.1M15 9.3V5a3 3 0 0 0-5.9-.6"/><path d="M17 16.9A7 7 0 0 1 5 12M19 12a7 7 0 0 1-.1 1.2M12 19v3M3 3l18 18"/>`)
	pin    = svg(`<path d="M12 17v5M9 10.8V4h6v6.8l2 3.2H7z"/>`)
)

// tile is where a person shows: their index, and whether as a screen.
type tile struct {
	who    int
	screen bool
	r      ui.Rect
}

// grid returns n tiles of 16:9, in rows as large as fit w by h, gap
// apart, the last row centered.
func grid(n int, w, h, gap float32) []ui.Rect {
	if n == 0 {
		return nil
	}
	bestW, bestCols := float32(0), 1
	for cols := 1; cols <= n; cols++ {
		rows := (n + cols - 1) / cols
		tw := min((w-gap*float32(cols-1))/float32(cols), (h-gap*float32(rows-1))/float32(rows)*16/9)
		if tw > bestW {
			bestW, bestCols = tw, cols
		}
	}
	tw, th := bestW, bestW*9/16
	rows := (n + bestCols - 1) / bestCols
	y0 := (h - (th*float32(rows) + gap*float32(rows-1))) / 2
	out := make([]ui.Rect, n)
	for i := range n {
		row, col := i/bestCols, i%bestCols
		inRow := min(bestCols, n-row*bestCols)
		x0 := (w - (tw*float32(inRow) + gap*float32(inRow-1))) / 2
		out[i] = ui.Rect{X: x0 + float32(col)*(tw+gap), Y: y0 + float32(row)*(th+gap), W: tw, H: th}
	}
	return out
}

// strip returns up to n tiles of 16:9 in a row w wide and h tall,
// centered, and how many fit.
func strip(n int, x, y, w, h, gap float32, across bool) ([]ui.Rect, int) {
	tw, th := h*16/9, h
	if !across {
		tw, th = w, w*9/16
	}
	along, size := w, tw
	if !across {
		along, size = h, th
	}
	fit := max(1, int((along+gap)/(size+gap)))
	fit = min(fit, n)
	start := (along - (size*float32(fit) + gap*float32(fit-1))) / 2
	out := make([]ui.Rect, fit)
	for i := range fit {
		off := start + float32(i)*(size+gap)
		if across {
			out[i] = ui.Rect{X: x + off, Y: y, W: tw, H: th}
		} else {
			out[i] = ui.Rect{X: x, Y: y + off, W: tw, H: th}
		}
	}
	return out, fit
}

// View creates the call of people. Give it a size, or grow it in its
// parent.
func View(c *ui.Context, s *State, people []Participant) ui.Element {
	t := c.Theme()
	view := ui.Box(c).Background(ui.RGB(16, 16, 18)).Role(ui.RoleGroup).Label("Call")
	b := view.LayoutBounds()
	gap := t.Space(2)
	area := ui.Rect{X: gap, Y: gap, W: max(1, b.W-2*gap), H: max(1, b.H-2*gap)}
	index := func(name string) int {
		for i, p := range people {
			if p.Name == name {
				return i
			}
		}
		return -1
	}
	if index(s.pinned) < 0 {
		s.pinned = ""
	}
	loudest := float32(speaking)
	for _, p := range people {
		if p.Level > loudest {
			loudest, s.speaker = p.Level, p.Name
		}
	}
	sharer := -1
	for i, p := range people {
		if p.Screen != nil {
			sharer = i
			break
		}
	}
	var tiles []tile
	more := 0
	switch {
	case sharer >= 0 || s.Layout == Speaker || s.pinned != "":
		stage := tile{who: sharer, screen: sharer >= 0}
		if !stage.screen {
			stage.who = max(0, index(s.pinned))
			if s.pinned == "" && index(s.speaker) >= 0 {
				stage.who = index(s.speaker)
			}
		}
		var rest []int
		for i := range people {
			if stage.screen || i != stage.who {
				rest = append(rest, i)
			}
		}
		var rects []ui.Rect
		var fit int
		if stage.screen {
			side := min(area.W*0.24, t.Space(60))
			stage.r = ui.Rect{X: area.X, Y: area.Y, W: area.W - side - gap, H: area.H}
			rects, fit = strip(len(rest), area.X+area.W-side, area.Y, side, area.H, gap, false)
		} else {
			band := min(area.H*0.2, t.Space(36))
			stage.r = ui.Rect{X: area.X, Y: area.Y, W: area.W, H: area.H - band - gap}
			if len(rest) == 0 {
				stage.r = area
			} else {
				rects, fit = strip(len(rest), area.X, area.Y+area.H-band, area.W, band, gap, true)
			}
		}
		tiles = append(tiles, stage)
		// When not all fit, the last tile counts the others.
		if fit < len(rest) {
			more = len(rest) - fit + 1
			fit--
		}
		for i := range fit {
			tiles = append(tiles, tile{who: rest[i], r: rects[i]})
		}
		if more > 0 {
			tiles = append(tiles, tile{who: -1, r: rects[len(rects)-1]})
		}
	default:
		for i, r := range grid(len(people), area.W, area.H, gap) {
			r.X, r.Y = r.X+area.X, r.Y+area.Y
			tiles = append(tiles, tile{who: i, r: r})
		}
	}
	if view.DoubleClicked() {
		if px, py, ok := view.PointerPosition(); ok {
			for _, tl := range tiles {
				if tl.r.Contains(px, py) && tl.who >= 0 && !tl.screen {
					name := people[tl.who].Name
					if s.pinned == name {
						s.pinned = ""
					} else {
						s.pinned = name
					}
					view.SetChanged()
				}
			}
		}
	}
	pinned := s.pinned
	view.Draw(func(p *ui.Painter, r ui.Rect) {
		for _, tl := range tiles {
			q := ui.Rect{X: r.X + tl.r.X, Y: r.Y + tl.r.Y, W: tl.r.W, H: tl.r.H}
			switch {
			case tl.who < 0:
				p.Fill(q, ui.RGB(44, 44, 48), 8)
				span := ui.Span{Text: fmt.Sprintf("+%d", more), Size: q.H * 0.22, Weight: 600, Color: ui.RGB(230, 230, 235)}
				w, h := p.MeasureText(0, span)
				p.RichText(q.X+q.W/2-w/2, q.Y+q.H/2-h/2, 0, span)
			case tl.screen:
				person := people[tl.who]
				p.Fill(q, ui.RGB(0, 0, 0), 8)
				p.Clip(q, 8, func() { person.Screen(p, q) })
				label(p, q, "Shared by "+person.Name, false, t)
			default:
				face(p, q, people[tl.who], t, people[tl.who].Name == pinned)
			}
		}
	})
	return view
}

// face draws person in the tile q.
func face(p *ui.Painter, q ui.Rect, person Participant, t *ui.Theme, pinned bool) {
	p.Fill(q, ui.RGB(44, 44, 48), 8)
	if person.Video != nil {
		p.Clip(q, 8, func() { person.Video(p, q) })
	} else {
		rad := min(q.W, q.H) * 0.2
		col := person.Color
		if col == (ui.Color{}) {
			col = ui.RGB(90, 100, 130)
		}
		dot := ui.Rect{X: q.X + q.W/2 - rad, Y: q.Y + q.H/2 - rad, W: 2 * rad, H: 2 * rad}
		p.Fill(dot, col, rad)
		span := ui.Span{Text: initials(person.Name), Size: rad * 0.8, Weight: 600, Color: ui.RGB(255, 255, 255)}
		w, h := p.MeasureText(0, span)
		p.RichText(dot.X+rad-w/2, dot.Y+rad-h/2, 0, span)
	}
	if person.Level > speaking && !person.Muted {
		a := 0.45 + 0.55*min(1, (person.Level-speaking)/(1-speaking)*1.5)
		p.Stroke(q, ui.RGB(52, 199, 89).Alpha(a), 8, 3)
	}
	label(p, q, person.Name, person.Muted, t)
	x := q.X + 8
	if person.Hand {
		span := ui.Span{Text: "✋", Size: max(12, min(20, q.H*0.09))}
		w, h := p.MeasureText(0, span)
		badge := ui.Rect{X: x, Y: q.Y + 8, W: w + 10, H: h + 4}
		p.Fill(badge, ui.RGB(255, 196, 0), badge.H/2)
		p.RichText(badge.X+5, badge.Y+2, 0, span)
		x += badge.W + 6
	}
	if pinned {
		sz := max(14, min(20, q.H*0.08))
		badge := ui.Rect{X: x, Y: q.Y + 8, W: sz + 10, H: sz + 6}
		p.Fill(badge, ui.RGB(0, 0, 0).Alpha(0.55), badge.H/2)
		p.Icon(pin, ui.Rect{X: badge.X + 5, Y: badge.Y + 3, W: sz, H: sz}, ui.RGB(255, 255, 255))
	}
}

// label draws name in a pill at the bottom left of q, a slashed
// microphone before it when muted.
func label(p *ui.Painter, q ui.Rect, name string, muted bool, t *ui.Theme) {
	size := max(10, min(t.FontSize, q.H*0.08))
	span := ui.Span{Text: name, Size: size, Weight: 500, Color: ui.RGB(255, 255, 255)}
	w, h := p.MeasureText(0, span)
	icon := float32(0)
	if muted {
		icon = h + 2
	}
	w = min(w, q.W-24-icon)
	pill := ui.Rect{X: q.X + 8, Y: q.Y + q.H - h - 14, W: w + icon + 14, H: h + 6}
	p.Fill(pill, ui.RGB(0, 0, 0).Alpha(0.55), 5)
	if muted {
		p.Icon(micOff, ui.Rect{X: pill.X + 6, Y: pill.Y + 3, W: h, H: h}, ui.RGB(255, 90, 80))
	}
	p.Clip(ui.Rect{X: pill.X + 7 + icon, Y: pill.Y, W: w, H: pill.H}, 0, func() {
		p.RichText(pill.X+7+icon, pill.Y+3, 0, span)
	})
}

// initials returns the first letters of name's first and last words.
func initials(name string) string {
	f := strings.Fields(name)
	if len(f) == 0 {
		return "?"
	}
	out := []rune(f[0])[:1]
	if len(f) > 1 {
		out = append(out, []rune(f[len(f)-1])[0])
	}
	return strings.ToUpper(string(out))
}
