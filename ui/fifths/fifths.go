// Package fifths is the circle of fifths, as music theory apps and
// songwriters' charts draw it: the twelve major keys around the outer
// ring, each a fifth above the one before it, clockwise from C at the top;
// their relative minors in the ring inside; and outside, how many sharps
// or flats each key signature has.
//
// A click on a key chooses it. The key chosen and the keys beside it show
// the six chords of the key, each marked with its numeral: I, IV and V and
// ii, iii and vi in a major key, i, iv and v and III, VI and VII in a
// minor one. Beside the circle, the key's scale spelled as it is written
// and its seven chords. ← and → step around the circle and ↑ and ↓ go
// between a major key and its relative minor.
//
//	fifths.View(c, &app.key)
package fifths

import (
	"fmt"
	"math"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Key is a key: where its tonic stands on the circle, from 0 for C or A
// minor to 11 for F or D minor, and whether it is minor.
type Key struct {
	At    int
	Minor bool
}

var (
	majors = [12]string{"C", "G", "D", "A", "E", "B", "F♯", "D♭", "A♭", "E♭", "B♭", "F"}
	minors = [12]string{"Am", "Em", "Bm", "F♯m", "C♯m", "G♯m", "D♯m", "B♭m", "Fm", "Cm", "Gm", "Dm"}
	// alt are the other names of the keys where the circle wraps.
	altMajors = map[int]string{5: "C♭", 6: "G♭", 7: "C♯"}
	altMinors = map[int]string{5: "A♭m", 6: "E♭m", 7: "A♯m"}
)

// Name returns the key's name, as "F♯ major" or "B♭ minor".
func (k Key) Name() string {
	at := ((k.At % 12) + 12) % 12
	if k.Minor {
		return strings.TrimSuffix(minors[at], "m") + " minor"
	}
	return majors[at] + " major"
}

// Signature returns how many sharps, above 0, or flats, below 0, the key
// is written with.
func (k Key) Signature() int {
	at := ((k.At % 12) + 12) % 12
	if at <= 6 {
		return at
	}
	return at - 12
}

var naturals = map[byte]int{'C': 0, 'D': 2, 'E': 4, 'F': 5, 'G': 7, 'A': 9, 'B': 11}
var letters = "CDEFGAB"

// Scale returns the seven notes of the key's scale, natural minor for a
// minor key, each letter once.
func (k Key) Scale() []string {
	at := ((k.At % 12) + 12) % 12
	tonic, steps := majors[at], []int{0, 2, 4, 5, 7, 9, 11}
	if k.Minor {
		tonic, steps = strings.TrimSuffix(minors[at], "m"), []int{0, 2, 3, 5, 7, 8, 10}
	}
	pitch := naturals[tonic[0]] + accidental(tonic[1:])
	from := strings.IndexByte(letters, tonic[0])
	notes := make([]string, 7)
	for i, step := range steps {
		l := letters[(from+i)%7]
		off := ((pitch+step-naturals[l])%12 + 18) % 12
		notes[i] = string(l) + []string{"♭♭♭", "♭♭", "♭", "", "♯", "♯♯"}[off-6+3]
	}
	return notes
}

func accidental(s string) int {
	return strings.Count(s, "♯") - strings.Count(s, "♭")
}

var (
	majorNumerals = []string{"I", "ii", "iii", "IV", "V", "vi", "vii°"}
	minorNumerals = []string{"i", "ii°", "III", "iv", "v", "VI", "VII"}
	majorKinds    = []string{"", "m", "m", "", "", "m", "°"}
	minorKinds    = []string{"m", "°", "", "m", "m", "", ""}
)

// Chords returns the seven chords built on the key's scale, as "Em" or
// "F♯°", with their numerals.
func (k Key) Chords() (chords, numerals []string) {
	kinds, nums := majorKinds, majorNumerals
	if k.Minor {
		kinds, nums = minorKinds, minorNumerals
	}
	for i, n := range k.Scale() {
		chords = append(chords, n+kinds[i])
	}
	return chords, nums
}

// numeral returns the numeral of the chord at the circle's position at in
// the ring of minor keys or major keys, or "" for one outside k.
func (k Key) numeral(at int, minor bool) string {
	d := ((at-k.At)%12 + 12) % 12
	if d == 11 {
		d = -1
	}
	if d > 1 {
		return ""
	}
	if k.Minor {
		if minor {
			return [...]string{"iv", "i", "v"}[d+1]
		}
		return [...]string{"VI", "III", "VII"}[d+1]
	}
	if minor {
		return [...]string{"ii", "vi", "iii"}[d+1]
	}
	return [...]string{"IV", "I", "V"}[d+1]
}

// View creates the circle of fifths, with *key chosen.
func View(c *ui.Context, key *Key) ui.Element {
	t := c.Theme()
	key.At = ((key.At % 12) + 12) % 12
	root := ui.Row(c).Gap(t.Space(8)).AlignItems(ui.Center).Focusable().FocusRing(false).Label("Circle of fifths").Value(key.Name())
	switch {
	case root.Shortcut(0, ui.KeyRight):
		key.At = (key.At + 1) % 12
	case root.Shortcut(0, ui.KeyLeft):
		key.At = (key.At + 11) % 12
	case root.Shortcut(0, ui.KeyUp), root.Shortcut(0, ui.KeyDown):
		key.Minor = !key.Minor
	}
	root.Children(func() {
		circle(c, key)
		panel(c, *key)
	})
	return root
}

const (
	outer float32 = 1.0
	sig   float32 = 0.88
	mid   float32 = 0.62
	inner float32 = 0.38
)

func circle(c *ui.Context, key *Key) {
	t := c.Theme()
	size := t.Space(96)
	e := ui.Box(c).Size(size, size).Shrink(0).Cursor(ui.CursorPointer)
	px, py, over := e.PointerPosition()
	R := size / 2
	hoverAt, hoverMinor := -1, false
	if over {
		dx, dy := px-R, py-R
		d := float32(math.Hypot(float64(dx), float64(dy))) / R
		if d >= inner && d <= sig {
			a := math.Atan2(float64(dx), -float64(dy))
			hoverAt = int(math.Round(a/(math.Pi/6)+12)) % 12
			hoverMinor = d < mid
		}
	}
	if e.Clicked() && hoverAt >= 0 {
		key.At, key.Minor = hoverAt, hoverMinor
		e.SetChanged()
	}
	k := *key
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		R := r.W / 2
		wedge := func(at int, r0, r1 float32) *ui.Path {
			var path ui.Path
			a0, a1 := (float64(at)-0.5)*math.Pi/6, (float64(at)+0.5)*math.Pi/6
			const n = 12
			for i := 0; i <= n; i++ {
				a := a0 + (a1-a0)*float64(i)/n
				x, y := cx+R*r1*float32(math.Sin(a)), cy-R*r1*float32(math.Cos(a))
				if i == 0 {
					path.MoveTo(x, y)
				} else {
					path.LineTo(x, y)
				}
			}
			for i := n; i >= 0; i-- {
				a := a0 + (a1-a0)*float64(i)/n
				path.LineTo(cx+R*r0*float32(math.Sin(a)), cy-R*r0*float32(math.Cos(a)))
			}
			path.Close()
			return &path
		}
		label := func(at int, rr float32, spans ...ui.Span) {
			a := float64(at) * math.Pi / 6
			x, y := cx+R*rr*float32(math.Sin(a)), cy-R*rr*float32(math.Cos(a))
			w, h := p.MeasureText(0, spans...)
			p.RichText(x-w/2, y-h/2, 0, spans...)
		}
		for at := range 12 {
			for _, minor := range []bool{false, true} {
				r0, r1 := mid, sig
				if minor {
					r0, r1 = inner, mid
				}
				num := k.numeral(at, minor)
				fill := t.Text.Alpha(0.04)
				if minor {
					fill = t.Text.Alpha(0.08)
				}
				textCol := t.Text
				switch {
				case at == k.At && minor == k.Minor:
					fill, textCol = t.Accent, t.AccentText
				case num != "":
					fill = t.Accent.Alpha(0.22)
				}
				if at == hoverAt && minor == hoverMinor && !(at == k.At && minor == k.Minor) {
					fill = fill.Mix(t.Text, 0.12)
				}
				w := wedge(at, r0, r1)
				p.FillPath(w, fill)
				p.StrokePath(w, 1.5, t.Background)
				name := majors[at]
				size := t.FontSize + 4
				if minor {
					name, size = minors[at], t.FontSize
				}
				spans := []ui.Span{{Text: name, Size: size, Weight: 700, Color: textCol}}
				if alt := map[bool]map[int]string{false: altMajors, true: altMinors}[minor][at]; alt != "" && at == 6 {
					spans = append(spans, ui.Span{Text: "\n" + alt, Size: size - 4, Color: textCol.Alpha(0.7)})
				}
				if num != "" {
					spans = append(spans, ui.Span{Text: "\n" + num, Size: t.FontSize - 3, Weight: 600, Color: textCol.Alpha(0.75)})
				}
				label(at, (r0+r1)/2, spans...)
			}
			s := Key{At: at}.Signature()
			text := "♮"
			switch {
			case s > 0:
				text = fmt.Sprintf("%d♯", s)
			case s < 0:
				text = fmt.Sprintf("%d♭", -s)
			}
			col := t.TextMuted
			if at == k.At {
				col = t.Text
			}
			label(at, (sig+outer)/2, ui.Span{Text: text, Size: t.FontSize - 2, Color: col})
		}
		name := strings.Fields(k.Name())
		w1, h1 := p.MeasureText(0, ui.Span{Text: name[0], Size: t.FontSize + 14, Weight: 700})
		w2, h2 := p.MeasureText(0, ui.Span{Text: name[1], Size: t.FontSize - 1})
		top := cy - (h1+h2)/2
		p.RichText(cx-w1/2, top, 0, ui.Span{Text: name[0], Size: t.FontSize + 14, Weight: 700, Color: t.Text})
		p.RichText(cx-w2/2, top+h1, 0, ui.Span{Text: name[1], Size: t.FontSize - 1, Color: t.TextMuted})
	})
}

// panel shows the key's signature, scale and chords.
func panel(c *ui.Context, k Key) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(5)).Width(t.Space(60)).Children(func() {
		ui.Column(c).Gap(t.Space(1)).Children(func() {
			ui.Text(c, k.Name()).FontSize(t.FontSize + 6).Bold()
			s := k.Signature()
			var sign string
			switch {
			case s == 0:
				sign = "No sharps or flats"
			case s == 1:
				sign = "1 sharp"
			case s > 0:
				sign = fmt.Sprintf("%d sharps", s)
			case s == -1:
				sign = "1 flat"
			default:
				sign = fmt.Sprintf("%d flats", -s)
			}
			rel := Key{At: k.At, Minor: !k.Minor}
			ui.Text(c, sign+" · relative "+rel.Name()).TextColor(t.TextMuted)
		})
		section := func(title string, fn func()) {
			ui.Column(c).Gap(t.Space(2)).Children(func() {
				ui.Text(c, title).FontSize(t.FontSize - 2).Bold().TextColor(t.TextMuted)
				fn()
			})
		}
		section("SCALE", func() {
			ui.Row(c).Gap(t.Space(1.5)).Wrap().Children(func() {
				for i, n := range k.Scale() {
					pill := ui.Text(c.Key(i), n).Bold().TextAlign(ui.Center).MinWidth(t.Space(9)).
						Padding(t.Space(1.5), t.Space(2)).Radius(t.Space(1.5)).Background(t.Text.Alpha(0.06))
					if i == 0 {
						pill.Background(t.Accent).TextColor(t.AccentText)
					}
				}
			})
		})
		section("CHORDS", func() {
			chords, nums := k.Chords()
			ui.Grid(c).Columns(7).GapX(t.Space(1.5)).GapY(t.Space(1)).Children(func() {
				for i := range chords {
					ui.Text(c.Key(i), nums[i]).FontSize(t.FontSize - 2).TextColor(t.TextMuted).TextAlign(ui.Center)
				}
				for i, ch := range chords {
					ui.Text(c.Key(-1-i), ch).Bold().TextAlign(ui.Center).Padding(t.Space(1.5), 0).
						Radius(t.Space(1.5)).Border(1, t.Border)
				}
			})
		})
	})
}
