// Package flashcards shows a deck of cards to study, as Anki and other
// spaced-repetition apps do: the top card shows its front, and a click or
// Space turns it over in 3D to show its back. Once turned, the card is
// thrown to the right when it was known and to the left to see it again:
// dragged past a third of its width and let go, flicked, or with → and
// ←; let go short of that, it springs back. It tilts as it goes, marked
// Again or Got it. The next two cards show below it, smaller, and move up
// as it goes.
//
//	ev := flashcards.Deck(c, &app.deck, len(app.due), func(i int, back bool) {
//		card := app.cards[app.due[i]]
//		if back {
//			ui.Text(c, card.answer)
//		} else {
//			ui.Text(c, card.question)
//		}
//	})
//	if ev.Graded {
//		app.grade(ev.Known)
//	}
//
// i counts down the deck from the top card, 0. The deck reports a grade
// once the card it threw is gone: the app takes that card off the top of
// its cards in the same frame, and may put it back further down to see it
// again. Give the deck a size, or grow it in its parent.
package flashcards

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// shown is how many cards of the deck show, the top one and those under
// it.
const shown = 3

// State is what a deck keeps: how many cards it dealt, whether the top
// one is turned, and where it was dragged or thrown. Keep it in the app's
// state, one for each deck.
type State struct {
	dealt  int
	turned bool
	offset ui.Point
	width  float32
}

// Turn turns the top card over, to its back or to its front again, as a
// Show Answer button does.
func (s *State) Turn() { s.turned = !s.turned }

// Turned reports whether the top card shows its back, which it must to be
// thrown.
func (s *State) Turned() bool { return s.turned }

// Throw throws the turned top card to the right, known, or to the left,
// as → and ← do.
func (s *State) Throw(known bool) {
	if !s.turned {
		return
	}
	s.offset = ui.Point{X: -s.away()}
	if known {
		s.offset.X = s.away()
	}
}

// away is how far a thrown card goes, out of sight.
func (s *State) away() float32 { return 2 * max(s.width, 200) }

// Events is a deck, and whether the card it threw went this frame.
type Events struct {
	ui.Element
	// Graded reports that the top card was thrown, to the right when
	// Known.
	Graded bool
	Known  bool
}

// Deck creates a deck of n cards, of which face(i, back) builds the front
// or the back of card i in a column filling the card.
func Deck(c *ui.Context, s *State, n int, face func(i int, back bool)) Events {
	t := c.Theme()
	view := ui.Box(c).Focusable().Label("Flashcards")
	ev := Events{Element: view}
	if n == 0 {
		s.turned, s.offset = false, ui.Point{}
		return ev
	}
	s.width = view.LayoutBounds().W
	if view.Shortcut(0, ui.KeySpace) {
		s.Turn()
	}
	if view.Shortcut(0, ui.KeyLeft) {
		s.Throw(false)
	}
	if view.Shortcut(0, ui.KeyRight) {
		s.Throw(true)
	}
	shows := "front"
	if s.turned {
		shows = "back"
	}
	view.Value(shows)

	step := t.Space(3)
	h := max(0, view.LayoutBounds().H-float32(shown-1)*step)
	view.Children(func() {
		for k := min(n, shown) - 1; k >= 0; k-- {
			card := ui.Column(c.Key(s.dealt + k)).Absolute().Left(0).Top(0).Right(0).Bottom(float32(shown-1) * step)
			// A card below shows step DIPs under the bottom of the one above,
			// however much it shrank.
			scale := 1 - 0.05*float32(k)
			card.AnimatePose(ui.Pose{Y: float32(k)*step + (1-scale)*h/2, Scale: scale}, ui.Snappy)
			if k > 0 {
				card.Children(func() { side(c, k, false, 0, face) })
				continue
			}
			top(c, s, card, &ev, face)
		}
	})
	return ev
}

// top makes card the top card: turned over by a click, dragged and thrown
// once turned, tilting and marked as it goes.
func top(c *ui.Context, s *State, card ui.Element, ev *Events, face func(i int, back bool)) {
	t := c.Theme()
	w := max(1, s.width)
	if s.turned {
		card.Cursor(ui.CursorGrab)
		card.Draggable(&s.offset, ui.DraggableOptions{Snap: func(p ui.Point) ui.Point {
			switch {
			case p.X > w/3:
				return ui.Point{X: s.away(), Y: p.Y}
			case p.X < -w/3:
				return ui.Point{X: -s.away(), Y: p.Y}
			}
			return ui.Point{}
		}})
	} else {
		card.Cursor(ui.CursorPointer)
	}
	if card.Clicked() {
		s.Turn()
	}
	at := card.DragOffset()
	if s.turned && !card.Dragging() && abs(at.X) >= s.away()-0.5 {
		ev.Graded, ev.Known = true, at.X > 0
		s.dealt++
		s.turned, s.offset = false, ui.Point{}
		return
	}
	card.Rotate(at.X / w * 12)
	card.Opacity(max(0, min(1, 2-abs(at.X)/w)))
	turn := float32(0)
	if s.turned {
		turn = 180
	}
	turn = card.AnimateTo("flashcards.turn", turn, ui.Snappy)
	card.Children(func() {
		side(c, 0, false, turn, face)
		side(c, 0, true, turn+180, face)
		lean := at.X / (w / 3)
		if lean == 0 {
			return
		}
		label, color := "Got it", t.Success
		if lean < 0 {
			label, color = "Again", t.Danger
		}
		stamp := ui.Text(c, label).Absolute().Top(t.Space(5)).FontSize(t.FontSize+6).Bold().TextColor(color).
			Padding(t.Space(1), t.Space(3)).Border(2, color).Radius(t.Radius).Opacity(min(1, abs(lean)))
		if lean > 0 {
			stamp.Left(t.Space(5)).Rotate(-12)
		} else {
			stamp.Right(t.Space(5)).Rotate(12)
		}
	})
}

// side builds the front or the back of card i, turned by deg about its
// upright axis, hidden while it faces away.
func side(c *ui.Context, i int, back bool, deg float32, face func(i int, back bool)) {
	t := c.Theme()
	f := ui.Column(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).Padding(t.Space(8)).Center().Gap(t.Space(3)).
		Radius(t.Radius*3).Background(t.Background).Border(1, t.Border).Shadow(0, 4, 16, 0, ui.RGB(0, 0, 0).Alpha(0.12))
	if deg != 0 || back {
		f.RotateY(deg).Perspective(1200).BackfaceHidden()
	}
	f.Children(func() { face(i, back) })
}

func abs(v float32) float32 { return float32(math.Abs(float64(v))) }
