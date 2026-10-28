// Package resistor reads and sets the color bands of a resistor: four,
// five or six bands on a drawn resistor, beige for carbon and blue for
// metal film, whose value, tolerance and range, and temperature
// coefficient show beside it. Clicking a band picks it, and the colors it
// may take show below to pick from. Typing a value, as 4.7k, 4k7 or 1M,
// sets the bands to it.
//
//	resistor.View(c, &app.resistor, &app.bands)
//	ohms, tolerance, _, err := resistor.Value(app.bands)
package resistor

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Color is a band's color.
type Color int

const (
	Black Color = iota
	Brown
	Red
	Orange
	Yellow
	Green
	Blue
	Violet
	Grey
	White
	Gold
	Silver
)

var colors = [...]struct {
	name      string
	paint     ui.Color
	tolerance float64
	tempco    int
}{
	{"Black", ui.Hex("#1B1B1B"), 0, 250},
	{"Brown", ui.Hex("#7A4A23"), 1, 100},
	{"Red", ui.Hex("#D2302C"), 2, 50},
	{"Orange", ui.Hex("#F07A1A"), 0.05, 15},
	{"Yellow", ui.Hex("#F2CC1D"), 0.02, 25},
	{"Green", ui.Hex("#2E9B45"), 0.5, 20},
	{"Blue", ui.Hex("#2A63D4"), 0.25, 10},
	{"Violet", ui.Hex("#8A4FC8"), 0.1, 5},
	{"Grey", ui.Hex("#8E8E93"), 0.01, 1},
	{"White", ui.Hex("#F5F5F2"), 0, 0},
	{"Gold", ui.Hex("#C9A43A"), 5, 0},
	{"Silver", ui.Hex("#B8BEC6"), 10, 0},
}

func (c Color) String() string { return colors[c].name }

type role uint8

const (
	digit role = iota
	multiplier
	tolerance
	tempco
)

var roleNames = [...]string{"Digit", "Multiplier", "Tolerance", "Temperature coefficient"}

func roles(n int) []role {
	switch n {
	case 4:
		return []role{digit, digit, multiplier, tolerance}
	case 6:
		return []role{digit, digit, digit, multiplier, tolerance, tempco}
	}
	return []role{digit, digit, digit, multiplier, tolerance}
}

// allowed reports whether c may be a band of role r.
func allowed(r role, c Color) bool {
	switch r {
	case digit:
		return c <= White
	case multiplier:
		return true
	case tolerance:
		return colors[c].tolerance > 0
	}
	return colors[c].tempco > 0
}

// Value returns what bands mean: the resistance in ohms, the tolerance
// in percent, and the temperature coefficient in ppm/K for six bands.
func Value(bands []Color) (ohms, tol float64, ppm int, err error) {
	if len(bands) < 4 || len(bands) > 6 {
		return 0, 0, 0, errors.New("a resistor has four, five or six bands")
	}
	rs := roles(len(bands))
	mantissa := 0
	for i, b := range bands {
		if b < Black || b > Silver || !allowed(rs[i], b) {
			return 0, 0, 0, fmt.Errorf("%s is not a %s band", b, strings.ToLower(roleNames[rs[i]]))
		}
		switch rs[i] {
		case digit:
			mantissa = mantissa*10 + int(b)
		case multiplier:
			exp := int(b)
			switch b {
			case Gold:
				exp = -1
			case Silver:
				exp = -2
			}
			ohms = float64(mantissa) * math.Pow10(exp)
		case tolerance:
			tol = colors[b].tolerance
		case tempco:
			ppm = colors[b].tempco
		}
	}
	return ohms, tol, ppm, nil
}

// Encode returns the bands of n, four to six, for ohms, keeping the
// tolerance and temperature bands of like, if it has them.
func Encode(ohms float64, n int, like []Color) ([]Color, error) {
	digits := 3
	if n == 4 {
		digits = 2
	}
	if ohms <= 0 {
		return nil, errors.New("a resistance is more than 0 Ω")
	}
	for exp := -2; exp <= 9; exp++ {
		m := ohms / math.Pow10(exp)
		r := math.Round(m)
		if r >= math.Pow10(digits) || math.Abs(m-r) > 1e-6*max(1, m) {
			continue
		}
		var out []Color
		for k := digits - 1; k >= 0; k-- {
			out = append(out, Color(int(r)/int(math.Pow10(k))%10))
		}
		mult := Color(exp)
		switch exp {
		case -1:
			mult = Gold
		case -2:
			mult = Silver
		}
		tol, temp := Gold, Brown
		if k := len(like); k >= 4 && k <= 6 {
			tol = like[k-1-boolInt(k == 6)]
			if k == 6 {
				temp = like[5]
			}
		}
		out = append(out, mult, tol)
		if n == 6 {
			out = append(out, temp)
		}
		return out, nil
	}
	return nil, fmt.Errorf("%s needs more than %d digits", Format(ohms), digits)
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Parse reads a resistance as 470, 4.7k, 4k7, 2R2, 1M or 1.5 MΩ.
func Parse(s string) (float64, error) {
	s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSpace(s), "Ω"), "ohm"))
	s = strings.ReplaceAll(s, " ", "")
	if s == "" {
		return 0, errors.New("empty")
	}
	scale := map[byte]float64{'R': 1, 'r': 1, 'k': 1e3, 'K': 1e3, 'M': 1e6, 'G': 1e9}
	for i := 0; i < len(s); i++ {
		if f, ok := scale[s[i]]; ok {
			num := s[:i]
			if rest := s[i+1:]; rest != "" {
				if strings.Contains(num, ".") {
					return 0, fmt.Errorf("%q has two decimal points", s)
				}
				num += "." + rest
			}
			v, err := strconv.ParseFloat(num, 64)
			if err != nil {
				return 0, fmt.Errorf("%q is not a resistance", s)
			}
			return v * f, nil
		}
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("%q is not a resistance", s)
	}
	return v, nil
}

// Format writes ohms with a prefix, as 4.7 kΩ.
func Format(ohms float64) string {
	unit := ""
	switch {
	case ohms >= 1e9:
		ohms, unit = ohms/1e9, "G"
	case ohms >= 1e6:
		ohms, unit = ohms/1e6, "M"
	case ohms >= 1e3:
		ohms, unit = ohms/1e3, "k"
	}
	return strconv.FormatFloat(ohms, 'f', -1, 64) + " " + unit + "Ω"
}

// State is what a resistor keeps: the band picked and the value typed.
// Keep it in the app's state, one for each resistor.
type State struct {
	picked int
	typed  string
	err    string
}

// View creates the resistor of *bands, four to six.
func View(c *ui.Context, s *State, bands *[]Color) ui.Element {
	t := c.Theme()
	if n := len(*bands); n < 4 || n > 6 {
		*bands = []Color{Yellow, Violet, Red, Gold}
	}
	b := *bands
	rs := roles(len(b))
	s.picked = min(max(s.picked, 0), len(b)-1)
	changed := false
	ohms, tol, ppm, err := Value(b)
	root := ui.Column(c).Gap(t.Space(6))
	root.Children(func() {
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
			count := len(b) - 4
			if ui.Segmented(c, &count, "4 bands", "5 bands", "6 bands").Label("Bands").Changed() {
				if next, e := Encode(ohms, count+4, b); e == nil {
					*bands, changed = next, true
				} else {
					next := make([]Color, count+4)
					copy(next, b)
					for i, r := range roles(count + 4) {
						if !allowed(r, next[i]) || i >= len(b) {
							next[i] = map[role]Color{digit: Brown, multiplier: Black, tolerance: Gold, tempco: Brown}[r]
						}
					}
					*bands, changed = next, true
				}
			}
			field := ui.TextInput(c, &s.typed).Placeholder("4.7k").Width(t.Space(32)).Label("Resistance")
			if field.Changed() {
				s.err = ""
				if v, e := Parse(s.typed); e != nil {
					s.err = e.Error()
				} else if next, e := Encode(v, len(b), b); e != nil {
					s.err = e.Error()
				} else {
					*bands, changed = next, true
				}
			}
			if s.err != "" && s.typed != "" {
				ui.Text(c, s.err).TextColor(t.Danger).FontSize(t.FontSize - 1)
			}
		})
		b = *bands
		rs = roles(len(b))
		ui.Box(c).Size(t.Space(160), t.Space(30)).Draw(func(p *ui.Painter, r ui.Rect) {
			drawResistor(p, r, b, s.picked, t)
		}).Children(func() {
			for i := range b {
				x, w := bandAt(ui.Rect{W: t.Space(160)}, len(b), i)
				hit := ui.ButtonBase(c.Key(i)).Absolute().Left(x-4).Top(t.Space(6)).Size(w+8, t.Space(18)).Label(fmt.Sprintf("Band %d, %s", i+1, b[i]))
				if hit.Clicked() {
					s.picked = i
				}
			}
		})
		ui.Column(c).Gap(t.Space(2)).Children(func() {
			ui.Text(c, fmt.Sprintf("Band %d · %s", s.picked+1, roleNames[rs[s.picked]])).TextColor(t.TextMuted)
			ui.Row(c).Wrap().Gap(t.Space(1.5)).Children(func() {
				for col := Black; col <= Silver; col++ {
					if !allowed(rs[s.picked], col) {
						continue
					}
					sw := ui.ButtonBase(c.Key(col)).Padding(t.Space(1.5), t.Space(2.5)).Radius(t.Radius).Gap(t.Space(2)).AlignItems(ui.Center).Label(col.String())
					if b[s.picked] == col {
						sw.Background(t.Accent.Alpha(0.15)).Border(1.5, t.Accent)
					} else {
						sw.Background(t.Surface).Border(1, t.Border)
					}
					sw.Children(func() {
						ui.Box(c).Size(14, 14).Radius(7).Background(colors[col].paint).Border(1, t.Text.Alpha(0.2))
						ui.Text(c, col.String()+note(rs[s.picked], col)).FontSize(t.FontSize - 1)
					})
					if sw.Clicked() && b[s.picked] != col {
						next := append([]Color(nil), b...)
						next[s.picked] = col
						*bands, changed = next, true
					}
				}
			})
		})
		if err != nil {
			ui.Text(c, err.Error()).TextColor(t.Danger)
			return
		}
		ui.Column(c).Gap(t.Space(1)).Children(func() {
			ui.Text(c, fmt.Sprintf("%s ±%g%%", Format(ohms), tol)).FontSize(t.FontSize * 2.2).Bold().FontFeatures("tnum")
			detail := fmt.Sprintf("%s to %s", Format(round(ohms*(1-tol/100))), Format(round(ohms*(1+tol/100))))
			if ppm > 0 {
				detail += fmt.Sprintf(" · %d ppm/K", ppm)
			}
			ui.Text(c, detail).TextColor(t.TextMuted).FontFeatures("tnum")
		})
	})
	if changed {
		root.SetChanged()
	}
	return root
}

func round(v float64) float64 {
	if v == 0 {
		return 0
	}
	p := math.Pow10(4 - int(math.Ceil(math.Log10(math.Abs(v)))))
	return math.Round(v*p) / p
}

// note returns what color c stands for as a band of role r.
func note(r role, c Color) string {
	switch r {
	case digit:
		return fmt.Sprintf(" %d", c)
	case multiplier:
		switch c {
		case Gold:
			return " ×0.1"
		case Silver:
			return " ×0.01"
		}
		return " ×" + strings.TrimSuffix(Format(math.Pow10(int(c))), "Ω")
	case tolerance:
		return fmt.Sprintf(" ±%g%%", colors[c].tolerance)
	}
	return fmt.Sprintf(" %d ppm", colors[c].tempco)
}

// bandAt returns where band i of n is across a resistor drawn in r.
func bandAt(r ui.Rect, n, i int) (x, w float32) {
	body := ui.Rect{X: r.X + r.W*0.18, W: r.W * 0.64}
	w = body.W * 0.06
	start, gap := body.X+body.W*0.14, body.W*0.1
	if tol := n - 1 - boolInt(n == 6); i >= tol {
		// The tolerance band, and the temperature band after it, stand
		// apart toward the other end, so the bands read one way.
		return body.X + body.W*0.72 + float32(i-tol)*gap, w
	}
	return start + float32(i)*gap, w
}

func drawResistor(p *ui.Painter, r ui.Rect, bands []Color, picked int, t *ui.Theme) {
	cy := r.Y + r.H/2
	lead := t.Text.Alpha(0.35)
	p.Fill(ui.Rect{X: r.X, Y: cy - 1.5, W: r.W, H: 3}, lead, 1.5)
	body := ui.Rect{X: r.X + r.W*0.18, Y: cy - r.H*0.3, W: r.W * 0.64, H: r.H * 0.6}
	tint := ui.Hex("#E3CBA0")
	if len(bands) > 4 {
		tint = ui.Hex("#8FB8E0")
	}
	p.Shadow(body, body.H/2, 0, 2, 6, 0, ui.RGBA(0, 0, 0, 0.2))
	p.Fill(body, tint, body.H/2)
	p.FillGradient(body, ui.LinearGradient{From: ui.RGBA(255, 255, 255, 0.35), To: ui.RGBA(0, 0, 0, 0.12), Angle: 90}, body.H/2)
	for i, b := range bands {
		x, w := bandAt(r, len(bands), i)
		band := ui.Rect{X: x, Y: body.Y, W: w, H: body.H}
		p.Fill(band, colors[b].paint, 0)
		if i == picked {
			p.Stroke(ui.Rect{X: x - 4, Y: body.Y - 8, W: w + 8, H: body.H + 16}, t.Accent, 4, 2)
		}
	}
}
