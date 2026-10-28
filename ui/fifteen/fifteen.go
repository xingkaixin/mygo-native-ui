// Package fifteen is the fifteen puzzle: numbered tiles in a square
// frame with one gap, to slide back into order. Clicking a tile in the
// gap's row or column slides it and the tiles between it and the gap
// over; the arrow keys slide the tile next to the gap into it. Tiles in
// their places show in the accent color. Above the frame, the moves and
// the time since the first; Shuffle deals a new puzzle, which can always
// be solved.
//
//	fifteen.View(c, &app.puzzle, &app.board)
package fifteen

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/egoist/mygo/ui"
)

// Board is a puzzle: its size in tiles, its tiles row by row with 0 for
// the gap, and the tiles moved so far.
type Board struct {
	Size  int
	Tiles []int
	Moves int
}

// New returns a puzzle of size by size, shuffled by random slides from
// seed, so that it can be solved.
func New(size int, seed uint64) Board {
	b := Board{Size: size, Tiles: make([]int, size*size)}
	for i := range b.Tiles {
		b.Tiles[i] = (i + 1) % len(b.Tiles)
	}
	r := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	back := -1
	for range size * size * size * 4 {
		gap := b.gap()
		var next []int
		for _, n := range b.neighbors(gap) {
			if n != back {
				next = append(next, n)
			}
		}
		n := next[r.IntN(len(next))]
		b.Tiles[gap], b.Tiles[n] = b.Tiles[n], 0
		back = gap
	}
	if b.Solved() {
		return New(size, seed+1)
	}
	return b
}

func (b Board) gap() int { return slices.Index(b.Tiles, 0) }

func (b Board) neighbors(i int) []int {
	var out []int
	x, y := i%b.Size, i/b.Size
	if x > 0 {
		out = append(out, i-1)
	}
	if x < b.Size-1 {
		out = append(out, i+1)
	}
	if y > 0 {
		out = append(out, i-b.Size)
	}
	if y < b.Size-1 {
		out = append(out, i+b.Size)
	}
	return out
}

// Solved reports whether every tile is in its place.
func (b Board) Solved() bool {
	for i, v := range b.Tiles {
		if v != (i+1)%len(b.Tiles) {
			return false
		}
	}
	return true
}

// Slide slides the tile at i, and those between it and the gap, toward
// the gap, if it is in the gap's row or column.
func (b *Board) Slide(i int) bool {
	gap := b.gap()
	gx, gy, x, y := gap%b.Size, gap/b.Size, i%b.Size, i/b.Size
	var step int
	switch {
	case i == gap:
		return false
	case y == gy:
		step = sign(x - gx)
	case x == gx:
		step = sign(y-gy) * b.Size
	default:
		return false
	}
	for g := gap; g != i; g += step {
		b.Tiles[g], b.Tiles[g+step] = b.Tiles[g+step], 0
		b.Moves++
	}
	return true
}

func sign(x int) int {
	if x < 0 {
		return -1
	}
	return 1
}

// State is what a puzzle keeps: when its first move was made and when it
// was solved. Keep it in the app's state, one for each puzzle.
type State struct {
	start, end time.Time
}

// View creates the puzzle of *b.
func View(c *ui.Context, s *State, b *Board) ui.Element {
	t := c.Theme()
	now := c.Now()
	n := b.Size
	moved := false
	slide := func(i int) {
		if i >= 0 && i < len(b.Tiles) && !b.Solved() && b.Slide(i) {
			moved = true
		}
	}
	gap := b.gap()
	gx, gy := gap%n, gap/n
	switch {
	case c.Shortcut(0, ui.KeyUp) && gy < n-1:
		slide(gap + n)
	case c.Shortcut(0, ui.KeyDown) && gy > 0:
		slide(gap - n)
	case c.Shortcut(0, ui.KeyLeft) && gx < n-1:
		slide(gap + 1)
	case c.Shortcut(0, ui.KeyRight) && gx > 0:
		slide(gap - 1)
	}
	tile, space := t.Space(18), t.Space(1.5)
	root := ui.Column(c).Gap(t.Space(4)).AlignItems(ui.Center)
	root.Children(func() {
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Width(float32(n)*(tile+space) + space).Children(func() {
			ui.Text(c, fmt.Sprintf("%d moves", b.Moves)).FontFeatures("tnum")
			if !s.start.IsZero() {
				end := now
				if !s.end.IsZero() {
					end = s.end
				} else {
					c.After(time.Second - now.Sub(s.start)%time.Second)
				}
				d := end.Sub(s.start).Truncate(time.Second)
				ui.Text(c, fmt.Sprintf("%d:%02d", int(d.Minutes()), int(d.Seconds())%60)).FontFeatures("tnum").TextColor(t.TextMuted)
			}
			ui.Spacer(c)
			if ui.Button(c, "Shuffle").Clicked() {
				*b, moved = New(n, uint64(now.UnixNano())), true
			}
		})
		frame := float32(n)*(tile+space) + space
		ui.Box(c).Size(frame, frame).Radius(t.Radius+4).Background(t.Surface).Border(1, t.Border).Children(func() {
			solved := b.Solved()
			for i, v := range b.Tiles {
				if v == 0 {
					continue
				}
				e := ui.ButtonBase(c.Key(v)).Size(tile, tile).Radius(t.Radius + 2).Center().Label(fmt.Sprint(v))
				x := e.AnimateTo("fifteen.x", space+float32(i%n)*(tile+space), ui.Snappy)
				y := e.AnimateTo("fifteen.y", space+float32(i/n)*(tile+space), ui.Snappy)
				e.Absolute().Left(x).Top(y)
				home := v == i+1
				switch {
				case home || solved:
					e.Background(t.Accent)
				case e.Hovered():
					e.Background(t.SurfaceHover)
				default:
					e.Background(t.Background)
				}
				e.Border(1, t.Border).Children(func() {
					ink := t.Text
					if home || solved {
						ink = t.AccentText
					}
					ui.Text(c, fmt.Sprint(v)).FontSize(tile * 0.36).Bold().FontFeatures("tnum").TextColor(ink)
				})
				if e.Clicked() {
					slide(i)
				}
			}
		})
		if b.Solved() && b.Moves > 0 {
			ui.Text(c, fmt.Sprintf("Solved in %d moves", b.Moves)).Bold().TextColor(t.Success)
		}
	})
	switch {
	case b.Moves == 0:
		s.start, s.end = time.Time{}, time.Time{}
	case s.start.IsZero():
		s.start = now
	case b.Solved() && s.end.IsZero():
		s.end = now
	}
	if moved {
		root.SetChanged()
	}
	return root
}
