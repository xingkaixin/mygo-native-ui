// Package password is a field for a new password, as sign-up pages show
// one: what is typed hidden until the eye button at its end shows it, a
// meter under it filling four bars as the password gets harder to guess,
// from Weak to Strong, and the rules the app sets, each checked once the
// password meets it. Changed reports the password edited.
//
//	password.Field(c, &app.password, []password.Rule{
//		{Text: "At least 8 characters", Met: func(s string) bool { return utf8.RuneCountInString(s) >= 8 }},
//	})
//	ok := password.Strength(app.password) >= 3
package password

import (
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// Rule is what a password must be, as the field lists it, and what tells
// whether it is.
type Rule struct {
	Text string
	Met  func(string) bool
}

var levels = []string{"", "Weak", "Fair", "Good", "Strong"}

// Field creates the field editing *value, with the rules listed under it.
func Field(c *ui.Context, value *string, rules []Rule) ui.Element {
	t := c.Theme()
	before := *value
	root := ui.Column(c).Gap(t.Space(2))
	shown := ui.Local(root, "password.shown", func() bool { return false })
	root.Children(func() {
		field := ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Padding(t.Space(1), t.Space(1.5), t.Space(1), t.Space(2.5)).
			Radius(t.Radius).Background(t.Surface).Cursor(ui.CursorText)
		var input ui.Element
		field.Children(func() {
			input = ui.TextInputBase(c, value).Grow(1).Shrink(1).Label("Password")
			if !*shown {
				input.Password()
			}
			label := "Show password"
			if *shown {
				label = "Hide password"
			}
			eye := ui.ButtonBase(c).Size(t.Space(5), t.Space(5)).Radius(t.Space(1)).Label(label).Tooltip(label).
				Checked(*shown).KeepFocus().Cursor(ui.CursorDefault)
			if eye.Clicked() {
				*shown = !*shown
			}
			color := t.TextMuted
			if eye.Hovered() {
				color = t.Text
			}
			eye.Draw(func(p *ui.Painter, r ui.Rect) { drawEye(p, r, color, !*shown) })
		})
		if field.Clicked() {
			input.Focus()
		}
		switch {
		case input.Focused():
			field.Border(1, t.Accent)
		case field.Hovered():
			field.Border(1, t.Border.Mix(t.Text, 0.25))
		default:
			field.Border(1, t.Border)
		}

		n := Strength(*value)
		color := [...]ui.Color{t.Border, t.Danger, t.Warning, t.Success, t.Success}[n]
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Role(ui.RoleMeter).Label("Strength").Value(levels[n]).Children(func() {
			ui.Row(c).Gap(t.Space(1)).Grow(1).Children(func() {
				for i := range 4 {
					bar := ui.Box(c.Key(i)).Height(t.Space(1)).Grow(1).Radius(t.Space(0.5)).Background(t.Text.Alpha(0.1))
					if i < n {
						bar.Background(color)
					}
				}
			})
			ui.Text(c, levels[n]).FontSize(t.FontSize - 1).TextColor(color).Width(t.Space(12)).TextAlign(ui.End)
		})

		if len(rules) > 0 {
			ui.Column(c).Gap(t.Space(1)).Role(ui.RoleList).Label("Requirements").Children(func() {
				for i, r := range rules {
					met := *value != "" && r.Met(*value)
					ui.Row(c.Key(i)).Gap(t.Space(2)).AlignItems(ui.Center).Role(ui.RoleListItem).Label(r.Text).Checked(met).Children(func() {
						ui.Box(c).Size(t.Space(3.5), t.Space(3.5)).Draw(func(p *ui.Painter, b ui.Rect) { mark(p, b, t, met) })
						text := ui.Text(c, r.Text).FontSize(t.FontSize - 1)
						if !met {
							text.TextColor(t.TextMuted)
						}
					})
				}
			})
		}
	})
	if *value != before {
		root.SetChanged()
	}
	return root
}

// Strength rates how hard s is to guess, from 0 for none to 4: 1 Weak, 2
// Fair, 3 Good, 4 Strong. It counts the bits of s from its length and the
// kinds of characters in it, a character repeating the one before or
// following it in a run, as "aaa", "abc" and "321", adding none, and
// rates the most common passwords Weak whatever is added at their ends.
func Strength(s string) int {
	if s == "" {
		return 0
	}
	if common(s) {
		return 1
	}
	var lower, upper, digit, symbol, other bool
	count := 0
	prev, step := rune(-1), 0
	for _, r := range s {
		switch {
		case r < 128 && unicode.IsLower(r):
			lower = true
		case r < 128 && unicode.IsUpper(r):
			upper = true
		case unicode.IsDigit(r):
			digit = true
		case r < 128:
			symbol = true
		default:
			other = true
		}
		d := int(r - prev)
		if prev < 0 || d < -1 || d > 1 || (d != 0 && d != step && step != 0) {
			count++
			step = 0
		} else {
			step = d
		}
		prev = r
	}
	pool := 0
	for _, k := range []struct {
		on bool
		n  int
	}{{lower, 26}, {upper, 26}, {digit, 10}, {symbol, 33}, {other, 100}} {
		if k.on {
			pool += k.n
		}
	}
	switch bits := float64(count) * math.Log2(float64(pool)); {
	case bits < 36:
		return 1
	case bits < 60:
		return 2
	case bits < 80:
		return 3
	default:
		return 4
	}
}

var commons = strings.Fields(`password passw0rd p@ssword qwerty qwertyuiop asdfgh asdfghjkl zxcvbn
	letmein welcome admin administrator login iloveyou monkey dragon master sunshine princess
	football baseball shadow superman batman trustno1 hello freedom whatever starwars
	abc123 123abc 1q2w3e4r 1qaz2wsx qazwsx changeme secret default`)

// common reports whether s is one of the most common passwords, with or
// without digits and symbols at its ends, as "Password1!".
func common(s string) bool {
	core := strings.ToLower(strings.TrimFunc(s, func(r rune) bool { return !unicode.IsLetter(r) }))
	if core == "" {
		return utf8.RuneCountInString(s) < 10
	}
	for _, w := range commons {
		if core == w || strings.ToLower(s) == w {
			return true
		}
	}
	return false
}

// drawEye draws an eye, struck through when slash is true.
func drawEye(p *ui.Painter, r ui.Rect, color ui.Color, slash bool) {
	cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W*0.32
	var lid, pupil ui.Path
	lid.MoveTo(cx-k, cy).QuadTo(cx, cy-k*1.25, cx+k, cy).QuadTo(cx, cy+k*1.25, cx-k, cy)
	p.StrokePath(&lid, 1.25, color)
	pupil.Circle(cx, cy, k*0.32)
	p.StrokePath(&pupil, 1.25, color)
	if slash {
		p.StrokePolyline([][2]float32{{cx - k*0.85, cy - k*0.85}, {cx + k*0.85, cy + k*0.85}}, 1.25, color)
	}
}

// mark draws a check in a filled circle for a rule met, an empty circle
// for one not.
func mark(p *ui.Painter, r ui.Rect, t *ui.Theme, met bool) {
	cx, cy, rad := r.X+r.W/2, r.Y+r.H/2, r.W/2-0.5
	var dot ui.Path
	dot.Circle(cx, cy, rad)
	if !met {
		p.StrokePath(&dot, 1, t.TextMuted.Alpha(0.6))
		return
	}
	p.FillPath(&dot, t.Success)
	k := rad * 0.5
	p.StrokePolyline([][2]float32{{cx - k, cy}, {cx - k*0.25, cy + k*0.7}, {cx + k, cy - k*0.6}}, 1.5, ui.RGB(255, 255, 255))
}
