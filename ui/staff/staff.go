// Package staff engraves a tune written in ABC notation on a staff, and
// plays it. The title shows above the music and the composer under it.
// Each system starts with the clef and the key's sharps or flats, the
// first with the time too, and the systems fill the width of the view,
// the measures of each spread evenly. Notes have heads, stems up or down
// by where they sit, flags, beams across eighths and shorter written
// together, dots, ledger lines and accidentals; rests, ties, triplets,
// bars and repeats show as engravers draw them, without a music font.
//
// Play plays the tune at its tempo, repeats and all, a tone for each
// note, the notes sounding lit and the view scrolling to keep them in
// sight; Stop stops it. A click on a note chooses it, which Chosen
// returns. The lines that are wrong are listed under the score, by
// number with what is wrong, and the score shows what the others say.
//
//	staff.View(c, &app.score, app.source).Grow(1)
//	if n, ok := app.score.Chosen(); ok {
//		... // n.Line, n.Column, n.Name
//	}
//
// It reads this much of ABC, a field or a line of music a line:
//
//	X:1                  the tune's number, skipped
//	T:Greensleeves       the title; C: the composer
//	M:6/8                the meter; C is common time, C| cut time
//	L:1/8                the unit note length: 1/8 unless set, 1/16 under 3/4
//	Q:3/8=60             the tempo, a beat and how many a minute; Q:120 counts quarters
//	K:F#m clef=bass      the key, major or minor, as G, Bb, F#m or A minor, and the clef
//	C D E F G A B c d e f g a b    notes from middle C up; , and ' move an octave
//	^C ^^C _B __B =F     sharp, double sharp, flat, double flat and natural, to the bar's end
//	A2 A/2 A/ A3/2       two units long, half of one, half again, one and a half
//	A>B A<B              the first dotted and the second halved, or the other way
//	z z2                 rests
//	| || |] [| |: :| ::  bars, end bars and repeats
//	A-A                  a tie
//	[CEG]                a chord
//	(3ABC                a triplet
//	ABcd efga            notes written together beam; a space breaks the beam
//	% a comment
//
// Other fields are skipped, and K:, M: and Q: are read in the header
// only. Slurs, decorations, chord symbols, grace notes, endings and
// fields in brackets are not read, and their lines are listed.
package staff

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"
)

// Note is a note or a chord of a tune: where it starts in the source, its
// line and column counted from 1; how it reads, as B♭4, a chord's notes
// lowest first, as C4 E4 G4; its pitches as MIDI numbers them, 60 for
// middle C; and how long it lasts as a fraction of a whole note,
// Num/Den, 3/8 for a dotted quarter and 1/12 for an eighth of a triplet.
type Note struct {
	Line, Column int
	Name         string
	Pitches      []int
	Num, Den     int
}

// State is what a view keeps: the tune as it parsed and engraved the
// source it was last given, its player, and the note chosen. Keep it in
// the app's state, one for each view.
type State struct {
	src    string
	tn     *tune
	sch    schedule
	sc     *score
	key    key
	player *audio.Player
	scroll ui.ScrollState
	// follow is the system the view last scrolled to while playing, plus
	// 1, so that the zero State follows none.
	follow int
	// chosen is the line and column of the note chosen, line 0 for none.
	chosen [2]int
}

// key is what an engraving is for: the width, and the theme's sizes.
type key struct {
	w, font, space float32
}

// Playing reports whether the tune plays.
func (s *State) Playing() bool { return s.player != nil && s.player.Playing() }

// Chosen returns the note chosen with a click, while the tune has a note
// where it was.
func (s *State) Chosen() (Note, bool) {
	i := s.find()
	if i < 0 {
		return Note{}, false
	}
	e := s.tn.events[i]
	g := gcd(e.ticks, whole)
	n := Note{Line: e.line, Column: e.col, Num: e.ticks / g, Den: whole / g}
	var names []string
	for _, h := range e.heads {
		names = append(names, h.name())
		n.Pitches = append(n.Pitches, h.pitch())
	}
	n.Name = strings.Join(names, " ")
	return n, true
}

// find returns the event chosen, -1 for none.
func (s *State) find() int {
	if s.tn == nil || s.chosen[0] == 0 {
		return -1
	}
	return slices.IndexFunc(s.tn.events, func(e event) bool {
		return len(e.heads) > 0 && e.line == s.chosen[0] && e.col == s.chosen[1]
	})
}

// play plays the tune from at, over what played before.
func (s *State) play(at time.Duration) {
	if s.player != nil {
		s.player.Stop()
	}
	s.player = audio.NewPlayer(&voice{sounds: s.sch.sounds, length: int64(s.sch.length * rate)})
	s.player.Seek(at)
	s.player.Play()
}

// View creates the score of src, ABC notation, with Play above it. Give
// it a size, or Grow; it scrolls when it does not fit, and the lines that
// are wrong show under it.
func View(c *ui.Context, s *State, src string) ui.Element {
	t := c.Theme()
	if s.tn == nil || s.src != src {
		old := s.sch.sounds
		s.src, s.tn, s.sc = src, parse(src), nil
		s.sch = plan(s.tn)
		if s.Playing() && !slices.Equal(old, s.sch.sounds) {
			s.play(s.player.Position())
		}
	}
	playing := s.Playing()
	lit := map[int]bool{}
	if playing {
		now := s.player.Position().Seconds()
		next := s.sch.length
		for _, sp := range s.sch.spans {
			if sp.start <= now && now < sp.end {
				lit[sp.ev] = true
			}
			if sp.start > now {
				next = min(next, sp.start)
				break
			}
			if sp.end > now {
				next = min(next, sp.end)
			}
		}
		c.After(time.Duration((next-now)*float64(time.Second)) + time.Millisecond)
	} else {
		s.follow = 0
	}
	return ui.Column(c).Children(func() {
		ui.Row(c).Padding(t.Space(3), t.Space(4)).Children(func() {
			label := "Play"
			if playing {
				label = "Stop"
			}
			if ui.PrimaryButton(c, label).Width(t.Space(20)).Disabled(len(s.sch.spans) == 0).Clicked() {
				if playing {
					s.player.Stop()
				} else {
					s.play(0)
				}
			}
		})
		ui.Divider(c)
		sc := ui.Scroll(c).Grow(1).Shrink(1).TrackScroll(&s.scroll)
		view := sc.LayoutBounds()
		if k := (key{view.W, t.FontSize, t.Space(1)}); s.sc == nil || s.key != k {
			s.sc, s.key = engrave(c, s.tn, view.W), k
			// The width comes from the last frame; one more frame engraves
			// to the width this one lays out.
			c.AnimationFrame()
		}
		score := s.sc
		if playing && len(lit) > 0 {
			for i := range lit {
				if sys := score.evs[i].sys; sys+1 != s.follow {
					s.follow = sys + 1
					y := score.systems[sys]
					top, bottom := y.y+y.top-score.sp, y.y+y.bottom+score.sp
					if top < s.scroll.Y || bottom > s.scroll.Y+view.H {
						s.scroll.ScrollTo(0, top, ui.Smooth)
					}
				}
				break
			}
		}
		sc.Children(func() {
			box := ui.Box(c).AlignSelf(ui.Stretch).Height(score.h).Role(ui.RoleImage).Label(cmp.Or(s.tn.title, "Score"))
			hot := -1
			if x, y, ok := box.PointerPosition(); ok {
				hot = score.at(x, y)
			}
			if hot >= 0 {
				box.Cursor(ui.CursorPointer)
			}
			if box.Clicked() {
				s.chosen = [2]int{}
				if hot >= 0 {
					e := s.tn.events[hot]
					s.chosen = [2]int{e.line, e.col}
				}
			}
			chosen := s.find()
			box.Draw(func(p *ui.Painter, r ui.Rect) { score.paint(p, t, r.X, r.Y, lit, chosen) })
		})
		if len(s.tn.errs) == 0 {
			return
		}
		ui.Divider(c)
		ui.Column(c).Gap(t.Space(1)).Padding(t.Space(2), t.Space(4)).Children(func() {
			for i, e := range s.tn.errs {
				ui.Text(c.Key(i), fmt.Sprintf("Line %d: %s", e.line, e.msg)).FontSize(t.FontSize - 1).TextColor(t.Danger)
			}
		})
	})
}

// at returns the note or chord at x, y, -1 for none.
func (sc *score) at(x, y float32) int {
	for i, p := range sc.evs {
		if len(sc.tn.events[i].heads) == 0 {
			continue
		}
		sy := sc.systems[p.sys].y
		if x >= p.x-p.l-0.2*sc.sp && x <= p.x+p.r+0.2*sc.sp && y >= sy+p.y0 && y <= sy+p.y1 {
			return i
		}
	}
	return -1
}
