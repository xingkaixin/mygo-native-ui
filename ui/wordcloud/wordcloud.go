// Package wordcloud is a word cloud: the words most often come largest,
// placed on a spiral out from the middle so that none covers another, as
// many as fit. Hovering a word shows how often it comes; a click chooses
// it, dimming the others, and a click on it again lets it go. Changed
// reports a word chosen or let go.
//
//	wordcloud.View(c, &app.cloud, words, &app.word).Size(640, 400)
package wordcloud

import (
	"cmp"
	"hash/fnv"
	"math"
	"slices"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
)

// Word is a word and how many times it comes.
type Word struct {
	Text  string
	Count int
}

type placed struct {
	word       Word
	x, y, w, h float32
	size       float32
	rank       int
}

// State is what a cloud keeps: where its words go, until the words or
// its size change. Keep it in the app's state, one for each cloud.
type State struct {
	key    uint64
	placed []placed
}

// most is how many words a cloud tries to place.
const most = 150

func fingerprint(words []Word, w, h float32) uint64 {
	f := fnv.New64a()
	f.Write([]byte(strconv.Itoa(int(w)) + "x" + strconv.Itoa(int(h))))
	for _, wd := range words {
		f.Write([]byte(wd.Text + "\x00" + strconv.Itoa(wd.Count) + "\x00"))
	}
	return f.Sum64()
}

// place lays out words in a box of w by h, the most common first, each
// where a spiral from the middle first finds room.
func (s *State) place(c *ui.Context, words []Word, w, h float32) {
	ws := slices.Clone(words)
	slices.SortFunc(ws, func(a, b Word) int { return cmp.Or(cmp.Compare(b.Count, a.Count), cmp.Compare(a.Text, b.Text)) })
	ws = ws[:min(len(ws), most)]
	s.placed = s.placed[:0]
	if len(ws) == 0 {
		return
	}
	lo, hi := math.Log(float64(ws[len(ws)-1].Count)), math.Log(float64(ws[0].Count))
	small, large := float32(11), min(64, h/6)
	pad := float32(4)
	free := func(x, y, bw, bh float32) bool {
		if x < 0 || y < 0 || x+bw > w || y+bh > h {
			return false
		}
		for _, p := range s.placed {
			if x < p.x+p.w+pad && p.x < x+bw+pad && y < p.y+p.h+pad && p.y < y+bh+pad {
				return false
			}
		}
		return true
	}
	aspect := w / max(h, 1)
	for i, wd := range ws {
		f := float32(0.5)
		if hi > lo {
			f = float32((math.Log(float64(wd.Count)) - lo) / (hi - lo))
		}
		size := small + (large-small)*f
		bw, bh := c.MeasureText(0, ui.Span{Text: wd.Text, Size: size, Weight: 600})
		// Each word sets off at its own angle, so that the cloud grows
		// round rather than along one arm.
		start := float64(i) * 2.39996
		reach := math.Hypot(float64(w), float64(h)) / 2
		for th := 0.0; th*2 < reach; th += 0.12 {
			r := float32(th * 2)
			a := start + th
			x := w/2 + r*float32(math.Cos(a))*aspect - bw/2
			y := h/2 + r*float32(math.Sin(a)) - bh/2
			if free(x, y, bw, bh) {
				s.placed = append(s.placed, placed{wd, x, y, bw, bh, size, i})
				break
			}
		}
	}
}

var hues = []float32{255, 160, 35, 320, 200, 10}

// View creates the cloud of words, the word chosen in *chosen, "" for
// none.
func View(c *ui.Context, s *State, words []Word, chosen *string) ui.Element {
	t := c.Theme()
	root := ui.Box(c).Clip()
	b := root.LayoutBounds()
	if b.W == 0 || b.H == 0 {
		// Words go where the cloud's size lets them, known from its first
		// layout.
		c.After(time.Millisecond)
		return root
	}
	if key := fingerprint(words, b.W, b.H); key != s.key {
		s.key = key
		s.place(c, words, b.W, b.H)
	}
	root.Children(func() {
		for i, p := range s.placed {
			on := *chosen == p.word.Text
			hue := hues[p.rank%len(hues)]
			color := ui.Oklch(0.58-0.12*min(1, p.size/48), 0.15, hue)
			if t.Dark {
				color = ui.Oklch(0.72, 0.13, hue)
			}
			el := ui.Box(c.Key(i)).Absolute().Left(p.x).Top(p.y).Size(p.w, p.h).Cursor(ui.CursorPointer).
				Role(ui.RoleButton).Selected(on).Label(p.word.Text).Tooltip(p.word.Text + " · " + strconv.Itoa(p.word.Count))
			switch {
			case on:
				el.Background(color.Alpha(0.14)).Radius(p.size / 5)
			case *chosen != "":
				color = color.Alpha(0.3)
			case el.Hovered():
				color = color.Mix(t.Text, 0.35)
			}
			if el.Clicked() {
				if on {
					*chosen = ""
				} else {
					*chosen = p.word.Text
				}
				root.SetChanged()
			}
			el.Children(func() {
				ui.Text(c, p.word.Text).FontSize(p.size).FontWeight(600).TextColor(color).SingleLine()
			})
		}
	})
	return root
}
