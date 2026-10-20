// Package pianoroll is the piano roll of a music editor: a row for each
// key of a piano, drawn on the left, and four bars of beats across, the
// notes on them. Pressing an empty place adds a note there and dragging
// draws it out; dragging a note moves it in time and pitch, and dragging
// its right end sets how long it lasts, all on the grid chosen. A double
// click or Delete removes a note, and Quantize moves every note onto the
// grid. Play loops the bars at the tempo through a synth, each note on
// the sample it falls on, the playhead across them as they are heard; a
// click on the ruler plays from there. Changed reports an edit.
//
//	pianoroll.View(c, &app.roll, &app.notes, &app.bpm, app.synth)
package pianoroll

import (
	"math"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"
)

// Beats is how many beats the roll has: four bars of four.
const Beats = 16

// Note is a note: its key, as MIDI numbers it, 60 for middle C, and when
// it starts and how long it lasts, in beats.
type Note struct {
	Pitch         int
	Start, Length float64
}

// End returns when the note ends.
func (n Note) End() float64 { return n.Start + n.Length }

// Wave is the shape of the synth's wave.
type Wave uint8

// Waves.
const (
	Saw Wave = iota
	Square
	Triangle
	Sine
)

// Synth is how the notes sound: the wave, how many seconds a note takes
// to sound fully and to die away, the frequency above which its
// harmonics fade, and its volume from 0 to 1.
type Synth struct {
	Wave            Wave
	Attack, Release float64
	Cutoff          float64
	Volume          float64
}

const (
	rate     = 48000
	lowest   = 36 // C2
	highest  = 96 // C7
	harmonic = 32
)

func hz(pitch int) float64 { return 440 * math.Pow(2, float64(pitch-69)/12) }

// amps returns the weight of each harmonic of a note at f under s: the
// wave's own, then a two-pole low-pass at the cutoff, none past Nyquist.
func amps(f float64, s Synth) []float64 {
	out := make([]float64, min(harmonic, int(rate/2/f)))
	for i := range out {
		k := float64(i + 1)
		var a float64
		switch s.Wave {
		case Saw:
			a = 2 / math.Pi / k
		case Square:
			if i%2 == 0 {
				a = 4 / math.Pi / k
			}
		case Triangle:
			if i%2 == 0 {
				a = 8 / (math.Pi * math.Pi) / (k * k)
				if i%4 == 2 {
					a = -a
				}
			}
		case Sine:
			if i == 0 {
				a = 1
			}
		}
		r := k * f / max(s.Cutoff, 20)
		out[i] = a / math.Sqrt(1+r*r*r*r)
	}
	return out
}

// voice makes the sound of the notes looped at a tempo, adding up the
// harmonics of each note, so that a sample depends on nothing before it
// and playing starts anywhere.
type voice struct {
	mu    sync.Mutex
	notes []Note
	synth Synth
	beat  float64 // frames a beat
	pos   int64
}

func (v *voice) SampleRate() int { return rate }
func (v *voice) Channels() int   { return 1 }
func (v *voice) Len() int64      { return -1 }

func (v *voice) SetPosition(frame int64) error {
	v.mu.Lock()
	v.pos = frame
	v.mu.Unlock()
	return nil
}

func (v *voice) Read(buf []float32) (int, error) {
	v.mu.Lock()
	notes, sy, beat, pos := v.notes, v.synth, v.beat, v.pos
	v.pos += int64(len(buf))
	v.mu.Unlock()
	clear(buf)
	loop := Beats * beat
	attack := max(0.002, sy.Attack) * rate
	release := max(0.005, sy.Release) * rate
	for _, n := range notes {
		f := hz(n.Pitch)
		w := amps(f, sy)
		on := n.Length * beat
		start := n.Start * beat
		step := 2 * math.Pi * f / rate
		for i := range buf {
			local := math.Mod(float64(pos+int64(i)), loop)
			// A note's release goes on into the next time round.
			for _, off := range [2]float64{local - start, local + loop - start} {
				if off < 0 || off >= on+release {
					continue
				}
				env := min(1, off/attack)
				if off >= on {
					env = min(1, on/attack) * (1 - (off-on)/release)
				}
				x := off * step
				s0, s1, c := 0.0, math.Sin(x), 2*math.Cos(x)
				sum := 0.0
				for _, a := range w {
					sum += a * s1
					s0, s1 = s1, c*s1-s0
				}
				buf[i] += float32(sum * env * sy.Volume * 0.25)
			}
		}
	}
	for i, x := range buf {
		buf[i] = float32(math.Tanh(float64(x)))
	}
	return len(buf), nil
}

type mode uint8

const (
	idle mode = iota
	moving
	sizing
)

// State is what a piano roll keeps: its player, its grid, the note
// chosen and the drag going on. Keep it in the app's state, one for each
// roll.
type State struct {
	player *audio.Player
	voice  *voice
	bpm    float64
	from   float64
	grid   int
	// chosen is the index of the note chosen, plus 1, so that the zero
	// State chooses none.
	chosen int
	mode   mode
	note   Note
	down   bool
	px, py float32
	x, y   float32
	length float64
	scroll ui.ScrollState
	placed bool
}

var grids = []struct {
	name string
	step float64
}{{"1/4", 1}, {"1/8", 0.5}, {"1/16", 0.25}, {"1/32", 0.125}, {"Off", 0}}

func snap(v, step float64) float64 {
	if step == 0 {
		return v
	}
	return math.Round(v/step) * step
}

// Playing reports whether the roll plays.
func (s *State) Playing() bool { return s.player != nil && s.player.Playing() }

// beat returns the beat heard.
func (s *State) beat() float64 {
	return math.Mod(s.player.Position().Seconds()*rate/s.voice.beat, Beats)
}

func name(pitch int) string {
	return [...]string{"C", "C♯", "D", "D♯", "E", "F", "F♯", "G", "G♯", "A", "A♯", "B"}[pitch%12] + strconv.Itoa(pitch/12-1)
}

func black(pitch int) bool {
	switch pitch % 12 {
	case 1, 3, 6, 8, 10:
		return true
	}
	return false
}

// View creates the piano roll of *notes, played at *bpm beats a minute
// through synth.
func View(c *ui.Context, s *State, notes *[]Note, bpm *float64, synth Synth) ui.Element {
	t := c.Theme()
	*bpm = max(40, min(240, *bpm))
	if s.voice == nil {
		s.voice = &voice{beat: rate * 60 / *bpm}
		s.bpm = *bpm
		s.player = audio.NewPlayer(s.voice)
		s.length = 0.5
	}
	step := grids[s.grid].step
	s.voice.mu.Lock()
	changed := !slices.Equal(s.voice.notes, *notes) || s.voice.synth != synth || s.bpm != *bpm
	if changed {
		s.voice.notes = slices.Clone(*notes)
	}
	s.voice.synth = synth
	old := s.voice.beat
	s.voice.beat = rate * 60 / *bpm
	s.bpm = *bpm
	s.voice.mu.Unlock()
	playing := s.Playing()
	if changed && playing {
		// What the player decoded ahead was of the old notes; from where
		// it is heard, in beats, it decodes the new ones.
		pos := s.player.Position()
		s.player.Seek(time.Duration(float64(pos) * s.voice.beat / old))
	}
	head := -1.0
	if playing {
		head = s.beat()
		c.After(time.Second / 60)
	}
	rowH, keysW := t.Space(3.5), t.Space(14)
	root := ui.Column(c).Gap(t.Space(3))
	root.Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			label := "Play"
			if playing {
				label = "Stop"
			}
			if ui.PrimaryButton(c, label).Width(t.Space(20)).Clicked() {
				if playing {
					s.player.Stop()
				} else {
					s.player.Seek(time.Duration(s.from * s.voice.beat / rate * float64(time.Second)))
					s.player.Play()
				}
			}
			ui.NumberInput(c, bpm, 40, 240, 1).Label("Tempo")
			ui.Text(c, "BPM").TextColor(t.TextMuted)
			ui.Box(c).Grow(1)
			ui.Text(c, "Grid").TextColor(t.TextMuted)
			names := make([]string, len(grids))
			for i, g := range grids {
				names[i] = g.name
			}
			ui.Segmented(c, &s.grid, names...)
			if ui.Button(c, "Quantize").Disabled(step == 0 || len(*notes) == 0).Clicked() {
				for i, n := range *notes {
					(*notes)[i].Start = min(Beats-step, snap(n.Start, step))
					(*notes)[i].Length = max(step, snap(n.Length, step))
				}
				root.SetChanged()
			}
		})
		ui.Column(c).Grow(1).Border(1, t.Border).Radius(t.Space(2)).Clip().Children(func() {
			ui.Row(c).Height(t.Space(6)).Background(t.Surface).Children(func() {
				ui.Box(c).Width(keysW)
				r := ui.Box(c).Grow(1).Label("Ruler").Cursor(ui.CursorPointer)
				if x, _, ok := r.PointerPosition(); ok && r.Clicked() {
					w := r.LayoutBounds().W
					s.from = max(0, min(Beats, math.Floor(float64(x/w)*Beats*4)/4))
					if playing {
						s.player.Seek(time.Duration(s.from * s.voice.beat / rate * float64(time.Second)))
					}
				}
				from := s.from
				r.Draw(func(p *ui.Painter, b ui.Rect) {
					bw := b.W / Beats
					for k := 0; k < Beats; k++ {
						x := b.X + float32(k)*bw
						if k%4 == 0 {
							p.Line(x, b.Y+b.H*0.3, x, b.Y+b.H, 1, t.TextMuted)
							p.Text(x+4, b.Y+3, strconv.Itoa(k/4+1), t.FontSize-2, t.TextMuted)
						} else {
							p.Line(x, b.Y+b.H*0.7, x, b.Y+b.H, 1, t.TextMuted.Alpha(0.5))
						}
					}
					fx := b.X + float32(from)*bw
					var tri ui.Path
					tri.MoveTo(fx, b.Y+b.H).LineTo(fx-5, b.Y+b.H-7).LineTo(fx+5, b.Y+b.H-7).Close()
					p.FillPath(&tri, t.Accent)
				})
			})
			ui.Divider(c)
			sc := ui.Scroll(c).Grow(1).TrackScroll(&s.scroll)
			if !s.placed && s.scroll.MaxY > 0 {
				s.placed = true
				mid := 66.0
				if len(*notes) > 0 {
					lo, hi := highest, lowest
					for _, n := range *notes {
						lo, hi = min(lo, n.Pitch), max(hi, n.Pitch)
					}
					mid = float64(lo+hi) / 2
				}
				s.scroll.Y = float32(highest-mid)*rowH - sc.LayoutBounds().H/2
			}
			sc.Children(func() {
				ui.Row(c).Height(float32(highest-lowest+1) * rowH).Children(func() {
					keys(c, keysW, rowH)
					s.roll(c, notes, step, rowH, head, root)
				})
			})
		})
	})
	return root
}

// keys draws the piano down the left, a C named at each octave.
func keys(c *ui.Context, w, rowH float32) {
	t := c.Theme()
	// Black keys are dark in a dark theme too, darker than the white.
	ebony := ui.RGB(44, 44, 50)
	if t.Dark {
		ebony = ui.RGB(8, 8, 10)
	}
	ui.Box(c).Width(w).AlignSelf(ui.Stretch).Background(t.Surface).Draw(func(p *ui.Painter, r ui.Rect) {
		for pitch := highest; pitch >= lowest; pitch-- {
			y := r.Y + float32(highest-pitch)*rowH
			if black(pitch) {
				p.Fill(ui.Rect{X: r.X, Y: y + 1, W: r.W * 0.6, H: rowH - 2}, ebony, 2)
			} else if pitch%12 == 4 || pitch%12 == 11 {
				p.Line(r.X, y, r.X+r.W, y, 1, t.Border)
			}
			if pitch%12 == 0 {
				p.Line(r.X, y+rowH, r.X+r.W, y+rowH, 1, t.TextMuted.Alpha(0.5))
				p.Text(r.X+r.W-24, y+1, name(pitch), t.FontSize-4, t.TextMuted)
			}
		}
		p.Line(r.X+r.W-0.5, r.Y, r.X+r.W-0.5, r.Y+r.H, 1, t.Border)
	})
}

// roll shows the notes on the grid, and adds, moves, sizes and removes
// them as the pointer and keys say.
func (s *State) roll(c *ui.Context, notes *[]Note, step float64, rowH float32, head float64, root ui.Element) {
	t := c.Theme()
	g := ui.Box(c).Grow(1).AlignSelf(ui.Stretch).Focusable().Cursor(ui.CursorCrosshair).Label("Notes")
	bw := g.LayoutBounds().W / Beats
	if bw == 0 {
		return
	}
	at := func(x, y float32) (float64, int) {
		return float64(x / bw), highest - int(math.Floor(float64(y/rowH)))
	}
	ns := *notes
	if i := s.chosen - 1; i >= 0 && i < len(ns) && (g.Shortcut(0, ui.KeyBackspace) || g.Shortcut(0, ui.KeyDelete)) {
		*notes = slices.Delete(*notes, i, i+1)
		s.chosen = 0
		root.SetChanged()
	}
	x, y, _ := g.PointerPosition()
	// Pressed ends as the pointer leaves the roll, but the drag goes on
	// until it is let go.
	dx, dy, held := g.Dragged()
	switch {
	case g.Pressed() && !s.down:
		s.down, s.px, s.py, s.x, s.y = true, x, y, x, y
		s.press(notes, x, y, bw, step, at)
		root.SetChanged()
	case held && s.down:
		s.x, s.y = s.x+dx, s.y+dy
		s.drag(*notes, bw, rowH, step)
		root.SetChanged()
	case !held && s.down:
		s.down, s.mode = false, idle
		if i := s.chosen - 1; i >= 0 && i < len(*notes) {
			s.length = (*notes)[i].Length
		}
	}
	if g.DoubleClicked() {
		beat, pitch := at(x, y)
		for i, n := range *notes {
			if n.Pitch == pitch && beat >= n.Start && beat < n.End() {
				*notes = slices.Delete(*notes, i, i+1)
				s.chosen = 0
				root.SetChanged()
				break
			}
		}
	}
	ns = *notes
	chosen := s.chosen
	g.Draw(func(p *ui.Painter, r ui.Rect) {
		for pitch := highest; pitch >= lowest; pitch-- {
			y := r.Y + float32(highest-pitch)*rowH
			if black(pitch) {
				p.Fill(ui.Rect{X: r.X, Y: y, W: r.W, H: rowH}, t.Text.Alpha(0.04), 0)
			}
			if pitch%12 == 0 || pitch%12 == 5 {
				p.Line(r.X, y+rowH, r.X+r.W, y+rowH, 1, t.Border)
			}
		}
		if step > 0 && step < 1 {
			for k := 0.0; k < Beats; k += step {
				p.Line(r.X+float32(k)*bw, r.Y, r.X+float32(k)*bw, r.Y+r.H, 1, t.Text.Alpha(0.04))
			}
		}
		for k := 0; k < Beats; k++ {
			col := t.Text.Alpha(0.1)
			if k%4 == 0 {
				col = t.Text.Alpha(0.25)
			}
			p.Line(r.X+float32(k)*bw, r.Y, r.X+float32(k)*bw, r.Y+r.H, 1, col)
		}
		for i, n := range ns {
			box := ui.Rect{X: r.X + float32(n.Start)*bw, Y: r.Y + float32(highest-n.Pitch)*rowH + 1, W: max(3, float32(n.Length)*bw-1), H: rowH - 2}
			color := ui.Oklch(0.66, 0.14, float32((n.Pitch%12)*30))
			if head >= n.Start && head < n.End() {
				color = color.Mix(ui.RGB(255, 255, 255), 0.4)
			}
			p.Fill(box, color, 3)
			if chosen == i+1 {
				p.Stroke(box, t.Text, 3, 1.5)
			}
			p.Fill(ui.Rect{X: box.X + box.W - 3, Y: box.Y + 3, W: 1.5, H: box.H - 6}, ui.RGB(255, 255, 255).Alpha(0.6), 0)
		}
		if head >= 0 {
			hx := r.X + float32(head)*bw
			p.Line(hx, r.Y, hx, r.Y+r.H, 2, t.Danger)
		}
	})
}

// press starts what a press at x, y does: sizing a note by its right end,
// moving one, or adding one to draw out.
func (s *State) press(notes *[]Note, x, y, bw float32, step float64, at func(x, y float32) (float64, int)) {
	beat, pitch := at(x, y)
	for i, n := range *notes {
		if n.Pitch != pitch || beat < n.Start || beat >= n.End() {
			continue
		}
		s.chosen, s.note = i+1, n
		s.mode = moving
		if float32(n.End()-beat)*bw <= 6 {
			s.mode = sizing
		}
		return
	}
	if pitch < lowest || pitch > highest || beat >= Beats {
		s.chosen, s.mode = 0, idle
		return
	}
	start := beat
	if step > 0 {
		start = math.Floor(beat/step) * step
	}
	n := Note{Pitch: pitch, Start: start, Length: min(s.length, Beats-start)}
	*notes = append(*notes, n)
	s.chosen, s.note, s.mode = len(*notes), n, sizing
}

// drag carries on what the press started, the pointer at s.x, s.y.
func (s *State) drag(notes []Note, bw, rowH float32, step float64) {
	i := s.chosen - 1
	if i < 0 || i >= len(notes) || s.mode == idle {
		return
	}
	from := s.note
	db := float64((s.x - s.px) / bw)
	n := &notes[i]
	least := step
	if least == 0 {
		least = 1.0 / 32
	}
	switch s.mode {
	case moving:
		n.Start = max(0, min(Beats-from.Length, snap(from.Start+db, step)))
		n.Pitch = max(lowest, min(highest, from.Pitch-int(math.Round(float64((s.y-s.py)/rowH)))))
	case sizing:
		end := min(Beats, snap(from.End()+db, step))
		n.Length = max(least, end-from.Start)
	}
}
