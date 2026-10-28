// Package presence shows the other people in a shared document, as Figma
// and other multiplayer editors do: each one's pointer, an arrow in their
// color with their name beside it, gliding from one position to the next
// as they arrive, and what they have selected outlined in their color.
// A name fades once its pointer has rested a few seconds, and the arrow
// dims after a minute. Over the document, their faces in a row, those
// resting dimmed.
//
//	presence.Avatars(c, &app.presence, app.peers)
//	presence.Cursors(c, &app.presence, app.peers) // inside the document's view
//	if id := app.presence.Following(); id != "" {
//		app.panTo(id)
//	}
//
// A click on a face follows that person: the view shows a frame in their
// color with their name at its top, and Following reports them, for the
// app to move its view along with theirs. Another click on the face, a
// click on the name, or Escape stops following, as does the person
// leaving.
package presence

import (
	"hash/fnv"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// Peer is another person in the document: who they are, their color, or
// the zero Color for one picked from their ID, their pointer in the
// coordinates of the view Cursors is in, when it last moved, and what
// they have selected, or an empty Rect.
type Peer struct {
	ID, Name  string
	Color     ui.Color
	X, Y      float32
	Moved     time.Time
	Selection ui.Rect
}

var palette = []ui.Color{
	ui.Hex("#e5484d"), ui.Hex("#f76b15"), ui.Hex("#ffb224"), ui.Hex("#30a46c"),
	ui.Hex("#12a594"), ui.Hex("#0090ff"), ui.Hex("#6e56cf"), ui.Hex("#d6409f"),
}

func (p Peer) color() ui.Color {
	if p.Color != (ui.Color{}) {
		return p.Color
	}
	h := fnv.New32a()
	h.Write([]byte(p.ID))
	return palette[h.Sum32()%uint32(len(palette))]
}

const (
	rest  = 4 * time.Second
	sleep = time.Minute
)

// State is what presence keeps: the person followed. Keep it in the app's
// state, one for each document.
type State struct {
	following string
}

// Following returns the ID of the person followed, or "" for none.
func (s *State) Following() string { return s.following }

func find(peers []Peer, id string) (Peer, bool) {
	for _, p := range peers {
		if p.ID == id {
			return p, true
		}
	}
	return Peer{}, false
}

// Cursors creates the pointers and selections of peers, over the whole
// of its parent, which it lets the pointer through to.
func Cursors(c *ui.Context, s *State, peers []Peer) ui.Element {
	t := c.Theme()
	now := c.Now()
	followed, following := find(peers, s.following)
	if !following {
		s.following = ""
	}
	layer := ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).PassThrough()
	type mark struct {
		p           Peer
		x, y, label float32
		dim         bool
	}
	marks := make([]mark, 0, len(peers))
	var next time.Duration
	for _, p := range peers {
		idle := now.Sub(p.Moved)
		for _, at := range []time.Duration{rest, sleep} {
			if idle < at && (next == 0 || at-idle < next) {
				next = at - idle
			}
		}
		show := float32(1)
		if idle >= rest {
			show = 0
		}
		marks = append(marks, mark{
			p:     p,
			x:     layer.Spring("presence.x."+p.ID, p.X),
			y:     layer.Spring("presence.y."+p.ID, p.Y),
			label: layer.Animate("presence.label."+p.ID, show, 200*time.Millisecond),
			dim:   idle >= sleep,
		})
	}
	if next > 0 {
		c.After(next)
	}
	if following && c.Shortcut(0, ui.KeyEscape) {
		s.following, following = "", false
	}
	layer.Draw(func(pa *ui.Painter, r ui.Rect) {
		for _, m := range marks {
			sel := m.p.Selection
			if sel.W <= 0 || sel.H <= 0 {
				continue
			}
			color := m.p.color()
			box := ui.Rect{X: r.X + sel.X, Y: r.Y + sel.Y, W: sel.W, H: sel.H}
			pa.Fill(box, color.Alpha(0.06), 2)
			pa.Stroke(box, color, 2, 1.5)
		}
		for _, m := range marks {
			color := m.p.color()
			if m.dim {
				color = color.Alpha(0.4)
			}
			x, y := r.X+m.x, r.Y+m.y
			arrow(pa, x, y, color)
			if m.label <= 0.01 {
				continue
			}
			span := ui.Span{Text: m.p.Name, Size: t.FontSize - 1, Weight: 500, Color: ui.RGBA(255, 255, 255, m.label)}
			w, h := pa.MeasureText(0, span)
			pill := ui.Rect{X: x + 12, Y: y + 18, W: w + 12, H: h + 6}
			pa.Fill(pill, color.Alpha(m.label), pill.H/2)
			pa.RichText(pill.X+6, pill.Y+3, 0, span)
		}
	})
	if !following {
		return layer
	}
	color := followed.color()
	return layer.Children(func() {
		ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).PassThrough().Border(3, color)
		ui.Row(c).Absolute().Left(0).Top(0).Right(0).Justify(ui.Center).PassThrough().Children(func() {
			tag := ui.ButtonBase(c).Padding(t.Space(1), t.Space(3)).Radius(0, 0, t.Space(2), t.Space(2)).Background(color).
				Label("Stop following " + followed.Name)
			if tag.Clicked() {
				s.following = ""
			}
			tag.Children(func() {
				ui.Text(c, "Following "+followed.Name).FontSize(t.FontSize - 1).Bold().TextColor(ui.RGB(255, 255, 255))
			})
		})
	})
}

// arrow draws a pointer with its tip at (x, y).
func arrow(p *ui.Painter, x, y float32, color ui.Color) {
	var pa ui.Path
	pa.MoveTo(x, y).LineTo(x, y+16).LineTo(x+4.5, y+12).LineTo(x+7.5, y+18.5).LineTo(x+10, y+17.3).
		LineTo(x+7.2, y+11).LineTo(x+12.5, y+11).Close()
	p.FillPath(&pa, color)
	p.StrokePath(&pa, 1.2, ui.RGB(255, 255, 255))
}

// Avatars creates the faces of peers, overlapping, the first five and how
// many more, each in a ring of its color while followed.
func Avatars(c *ui.Context, s *State, peers []Peer) ui.Element {
	t := c.Theme()
	now := c.Now()
	size := t.Space(8)
	const most = 5
	return ui.Row(c).AlignItems(ui.Center).Children(func() {
		for i, p := range peers {
			if i == most {
				ui.Text(c, "+"+strconv.Itoa(len(peers)-most)).FontSize(t.FontSize-2).Bold().Size(size, size).Radius(size/2).
					Background(t.Surface).Border(2, t.Background).TextAlign(ui.Center).Padding(t.Space(1.5), 0).
					Margin(0, 0, 0, -t.Space(2)).Tooltip(names(peers[most:]))
				break
			}
			color := p.color()
			b := ui.ButtonBase(c.Key(p.ID)).Size(size, size).Radius(size/2).Background(color).Border(2, t.Background).
				Label("Follow " + p.Name).Tooltip(p.Name)
			if i > 0 {
				b.Margin(0, 0, 0, -t.Space(2))
			}
			if s.following == p.ID {
				b.Border(2, t.Background).Shadow(0, 0, 0, 2, color)
			}
			if now.Sub(p.Moved) >= sleep {
				b.Opacity(0.5)
			}
			if b.Clicked() {
				if s.following == p.ID {
					s.following = ""
				} else {
					s.following = p.ID
				}
			}
			b.Children(func() {
				ui.Text(c, initials(p.Name)).FontSize(t.FontSize - 2).Bold().TextColor(ui.RGB(255, 255, 255)).Margin(ui.Auto)
			})
		}
	})
}

func initials(name string) string {
	var out []rune
	for _, w := range strings.Fields(name) {
		out = append(out, []rune(strings.ToUpper(w))[0])
		if len(out) == 2 {
			break
		}
	}
	return string(out)
}

func names(peers []Peer) string {
	out := make([]string, len(peers))
	for i, p := range peers {
		out[i] = p.Name
	}
	return strings.Join(out, ", ")
}
