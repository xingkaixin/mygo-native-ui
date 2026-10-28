// Presence fills a planning board with four teammates: their pointers
// arrive every 800 ms, as from a server, and glide between, Sam has a
// note selected, and Lee has been away a while. Click a face to follow
// that person: the board moves along with them until you click it again
// or press Escape.
//
//	go run ./examples/presence
package main

import (
	"hash/fnv"
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/presence"
)

type note struct {
	x, y, w, h float32
	text       string
	color      ui.Color
}

var notes = []note{
	{60, 60, 200, 120, "Ship the beta to the first ten teams", ui.Hex("#fff3b0")},
	{320, 90, 200, 120, "Interview three admins about roles", ui.Hex("#c9f2d8")},
	{600, 50, 200, 120, "Billing page: annual plans", ui.Hex("#d6e4ff")},
	{120, 300, 200, 120, "Fix the slow export on big boards", ui.Hex("#ffd6d6")},
	{420, 330, 200, 120, "Write the launch post", ui.Hex("#fff3b0")},
	{760, 280, 200, 120, "Audit log for workspace owners", ui.Hex("#e6dcff")},
	{980, 120, 200, 120, "Keyboard shortcuts sheet", ui.Hex("#c9f2d8")},
	{1040, 420, 200, 120, "Price test in two markets", ui.Hex("#d6e4ff")},
}

var people = []struct {
	id, name string
	away     bool
}{
	{"ana", "Ana Ruiz", false},
	{"sam", "Sam Okafor", false},
	{"mira", "Mira Chen", false},
	{"lee", "Lee Park", true},
}

const tick = 800 * time.Millisecond

type app struct {
	presence presence.State
	start    time.Time
}

func jitter(id string, n int, span float32) float32 {
	h := fnv.New32a()
	h.Write([]byte(id))
	h.Write([]byte{byte(n), byte(n >> 8)})
	return float32(h.Sum32()%1000)/1000*span - span/2
}

// peers returns where each person is at now: drifting around a note,
// moving on to the next every few ticks.
func (a *app) peers(now time.Time) []presence.Peer {
	n := int(now.Sub(a.start) / tick)
	out := make([]presence.Peer, len(people))
	for i, p := range people {
		step := n
		moved := a.start.Add(time.Duration(n) * tick)
		if p.away {
			step, moved = 0, a.start.Add(-2*time.Minute)
		}
		target := notes[(i*3+step/4)%len(notes)]
		out[i] = presence.Peer{
			ID: p.id, Name: p.name, Moved: moved,
			X: target.x + target.w/2 + jitter(p.id, step, 160),
			Y: target.y + target.h/2 + jitter(p.id+"y", step, 110),
		}
		if p.id == "sam" {
			s := notes[(i*3+step/4)%len(notes)]
			out[i].Selection = ui.Rect{X: s.x - 4, Y: s.y - 4, W: s.w + 8, H: s.h + 8}
		}
	}
	return out
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	now := c.Now()
	if a.start.IsZero() {
		a.start = now
	}
	c.After(tick - now.Sub(a.start)%tick)
	peers := a.peers(now)
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Padding(t.Space(3), t.Space(5)).Children(func() {
			ui.Text(c, "Q3 planning").Bold().Grow(1)
			presence.Avatars(c, &a.presence, peers)
			ui.Button(c, "Share")
		})
		view := ui.Box(c).Grow(1).Clip().Background(t.Surface)
		w, h := view.Bounds().W, view.Bounds().H
		var ox, oy float32
		for _, p := range peers {
			if p.ID == a.presence.Following() {
				ox, oy = p.X-w/2, p.Y-h/2
			}
		}
		ox, oy = view.Spring("pan.x", ox), view.Spring("pan.y", oy)
		for i := range peers {
			peers[i].X -= ox
			peers[i].Y -= oy
			peers[i].Selection.X -= ox
			peers[i].Selection.Y -= oy
		}
		view.Children(func() {
			for i, n := range notes {
				ui.Text(c.Key(i), n.text).Absolute().Left(n.x-ox).Top(n.y-oy).Size(n.w, n.h).Padding(t.Space(3)).
					Background(n.color).Radius(4).Shadow(0, 2, 6, 0, ui.RGBA(0, 0, 0, 0.12)).TextColor(ui.RGB(30, 30, 30))
			}
			presence.Cursors(c, &a.presence, peers)
		})
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Presence",
			Width:   1000,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
