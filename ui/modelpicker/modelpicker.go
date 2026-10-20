// Package modelpicker chooses the model a chat or an agent talks to, as
// the model selectors of Cursor and T3 Chat do, for the row of controls
// under a promptinput: a button of the model chosen, its provider's mark
// and its name, which opens a panel of the models under their providers.
// Each shows icons of what it can do, read images, call tools and reason,
// which name themselves on hover, its context window, "200K", its price in
// and out per million tokens, "$3 / $15", and a check if chosen.
//
// Typing finds models by their names and providers; Up and Down move
// among them, as the pointer does, and Enter or a click chooses one, which
// closes the panel, as Escape or a click outside it does.
//
//	promptinput.View(c, &app.prompt, "Ask anything", app.busy, func() {
//		modelpicker.View(c, &app.model, models)
//	})
package modelpicker

import (
	"hash/fnv"
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// Model is a model to choose: its ID, which the app keeps, the name and
// provider it shows with, its context window in tokens, its prices in
// dollars per million tokens read and written, and whether it reads
// images, calls tools and reasons before it answers.
type Model struct {
	ID, Name, Provider       string
	Context                  int
	In, Out                  float64
	Vision, Tools, Reasoning bool
}

// state is what a picker keeps: whether its panel shows, the text typed
// in it and the text the models were found for, the model the arrows are
// on, as an index into the models, the one last under the pointer, -1 for
// none, and whether the arrows moved to a model not yet scrolled to.
type state struct {
	open    bool
	query   string
	found   string
	on      int
	hovered int
	scroll  bool
}

// View creates the button choosing *chosen, the ID of one of models,
// which opens the panel of them. Changed reports a model chosen.
func View(c *ui.Context, chosen *string, models []Model) ui.Element {
	t := c.Theme()
	before := *chosen
	current := slices.IndexFunc(models, func(m Model) bool { return m.ID == *chosen })
	button := ui.ButtonBase(c).Gap(t.Space(1.5)).Padding(t.Space(1), t.Space(2), t.Space(1), t.Space(1.5)).
		Radius(t.Space(2)).Role(ui.RolePopUpButton).Label("Model")
	st := ui.Local(button, "modelpicker.state", func() state { return state{hovered: -1} })
	if button.Clicked() {
		st.open = !st.open
		st.query, st.found, st.on = "", "", current
	}
	if button.Hovered() || st.open {
		button.Background(t.Text.Alpha(0.06))
	}
	button.Expanded(st.open).Children(func() {
		if current < 0 {
			ui.Text(c, "Choose a model").TextColor(t.TextMuted).SingleLine()
		} else {
			m := models[current]
			mark(c, m.Provider, t.Space(4.5))
			ui.Text(c, m.Name).SingleLine().Shrink(1)
		}
		chevron(c)
	})
	ui.PopoverBase(c, button, &st.open, func(panel ui.Element) {
		panel.Width(t.Space(118)).Margin(t.Space(1.5), 0, 0, 0).Radius(t.Space(3)).Clip().Background(t.Background).
			Border(1, t.Border).Shadow(0, 8, 24, 0, ui.RGBA(0, 0, 0, 0.16)).Role(ui.RoleDialog).Label("Models")
		found := find(models, st.query)
		if st.found != st.query {
			st.found = st.query
			st.on = -1
		}
		if !slices.Contains(found, st.on) {
			st.on = -1
			if len(found) > 0 {
				st.on = found[0]
			}
		}
		step := func(d int) {
			if i := slices.Index(found, st.on) + d; i >= 0 && i < len(found) {
				st.on, st.scroll = found[i], true
			}
		}
		if panel.OverlayShortcut(0, ui.KeyDown) {
			step(1)
		}
		if panel.OverlayShortcut(0, ui.KeyUp) {
			step(-1)
		}
		panel.Children(func() {
			var input, active ui.Element
			ui.Row(c).Gap(t.Space(2)).Padding(t.Space(2.5), t.Space(3)).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(t.Space(4), t.Space(4)).Draw(func(p *ui.Painter, r ui.Rect) { magnifier(p, r, t.TextMuted) })
				input = ui.TextInputBase(c, &st.query).AutoFocus().Grow(1).Placeholder("Search models").
					Label("Search models").Role(ui.RoleComboBox).Expanded(true)
			})
			if input.Submitted() && st.on >= 0 {
				*chosen, st.open = models[st.on].ID, false
			}
			ui.Divider(c)
			hovered := -1
			ui.Scroll(c).MaxHeight(t.Space(96)).Padding(t.Space(1)).Children(func() {
				if len(found) == 0 {
					ui.Text(c, "No models match").TextColor(t.TextMuted).Padding(t.Space(3))
				}
				provider := ""
				for _, i := range found {
					m := models[i]
					if m.Provider != provider {
						provider = m.Provider
						heading(c.Key("heading "+provider), provider)
					}
					r := row(c.Key(i), m, i == st.on, m.ID == *chosen)
					if r.Hovered() {
						hovered = i
					}
					if r.Clicked() {
						*chosen, st.open = m.ID, false
					}
					if i == st.on {
						active = r
						if st.scroll {
							r.ScrollIntoView()
							st.scroll = false
						}
					}
				}
			})
			// The pointer moving onto a model puts the arrows there; resting
			// there, it leaves the arrows to move on.
			if hovered != st.hovered && hovered >= 0 {
				st.on = hovered
			}
			st.hovered = hovered
			if active.Valid() {
				input.ActiveDescendant(active)
			}
		})
	})
	if *chosen != before {
		button.SetChanged()
	}
	return button
}

// find returns the indexes of the models whose names and providers hold
// every word of query, case aside, grouped by provider in the order the
// providers first come.
func find(models []Model, query string) []int {
	words := strings.Fields(strings.ToLower(query))
	var providers []string
	for _, m := range models {
		if !slices.Contains(providers, m.Provider) {
			providers = append(providers, m.Provider)
		}
	}
	var found []int
	for _, p := range providers {
		for i, m := range models {
			if m.Provider != p {
				continue
			}
			text := strings.ToLower(m.Name + " " + m.Provider)
			if !slices.ContainsFunc(words, func(w string) bool { return !strings.Contains(text, w) }) {
				found = append(found, i)
			}
		}
	}
	return found
}

// heading shows provider's mark and name above its models.
func heading(c *ui.Context, provider string) {
	t := c.Theme()
	ui.Row(c).Gap(t.Space(2)).Padding(t.Space(2), t.Space(2), t.Space(1)).AlignItems(ui.Center).Children(func() {
		mark(c, provider, t.Space(4))
		ui.Text(c, provider).FontSize(t.FontSize - 2).Bold().TextColor(t.TextMuted)
	})
}

// row creates the row of m, lit while on, with a check if chosen.
func row(c *ui.Context, m Model, on, chosen bool) ui.Element {
	t := c.Theme()
	small := t.FontSize - 1
	r := ui.ButtonBase(c).Gap(t.Space(2)).Padding(t.Space(1.5), t.Space(2)).Radius(t.Space(1.5)).Label(m.Name)
	if on {
		r.Background(t.Text.Alpha(0.07))
	}
	return r.Children(func() {
		ui.Text(c, m.Name).SingleLine().Grow(1).Shrink(1)
		ui.Row(c).Gap(t.Space(1)).Children(func() {
			capability(c, m.Vision, "Vision: reads images", eye)
			capability(c, m.Tools, "Tools: calls functions", wrench)
			capability(c, m.Reasoning, "Reasoning: thinks before it answers", bulb)
		})
		ui.Text(c, tokens(m.Context)).FontSize(small).TextColor(t.TextMuted).FontFeatures("tnum").
			Width(t.Space(10)).TextAlign(ui.End).Tooltip(tokens(m.Context) + " tokens of context")
		price := dollars(m.In) + " / " + dollars(m.Out)
		ui.Text(c, price).FontSize(small).TextColor(t.TextMuted).FontFeatures("tnum").
			Width(t.Space(22)).TextAlign(ui.End).Tooltip(dollars(m.In) + " in, " + dollars(m.Out) + " out per million tokens")
		ui.Box(c).Size(t.Space(4), t.Space(4)).Draw(func(p *ui.Painter, r ui.Rect) {
			if chosen {
				check(p, r, t.Accent)
			}
		})
	})
}

// capability shows icon, named label, which its tooltip shows, or room
// for it unless has.
func capability(c *ui.Context, has bool, label string, icon func(p *ui.Painter, r ui.Rect, color ui.Color)) {
	t := c.Theme()
	b := ui.Box(c).Size(t.Space(4.5), t.Space(4.5))
	if !has {
		return
	}
	b.Label(label).Tooltip(label).Draw(func(p *ui.Painter, r ui.Rect) { icon(p, r, t.TextMuted) })
}

// mark shows the first letter of provider in a circle d across, of a
// color of its own, as the provider's logo.
func mark(c *ui.Context, provider string, d float32) {
	h := fnv.New32a()
	h.Write([]byte(provider))
	color := ui.Oklch(0.6, 0.13, float32(h.Sum32()%360))
	r, _ := utf8.DecodeRuneInString(provider)
	ui.Row(c).Size(d, d).Radius(d / 2).Background(color).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
		ui.Text(c, strings.ToUpper(string(r))).FontSize(d * 0.55).Bold().TextColor(ui.RGB(255, 255, 255))
	})
}

// tokens returns n tokens in thousands or millions, "200K", "1M".
func tokens(n int) string {
	switch {
	case n >= 1_000_000:
		return strconv.FormatFloat(math.Round(float64(n)/1e5)/10, 'f', -1, 64) + "M"
	case n >= 1_000:
		return strconv.FormatFloat(math.Round(float64(n)/1e2)/10, 'f', -1, 64) + "K"
	}
	return strconv.Itoa(n)
}

// dollars returns v in dollars, "$3", or with cents, "$0.25".
func dollars(v float64) string {
	if v == math.Trunc(v) {
		return "$" + strconv.FormatFloat(v, 'f', 0, 64)
	}
	return "$" + strconv.FormatFloat(v, 'f', 2, 64)
}

// chevron shows an arrow pointing down.
func chevron(c *ui.Context) {
	t := c.Theme()
	ui.Box(c).Size(t.Space(3), t.Space(3)).Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W/5
		p.StrokePolyline([][2]float32{{cx - k, cy - k/2}, {cx, cy + k/2}, {cx + k, cy - k/2}}, 1.5, t.TextMuted)
	})
}

// The icons below draw in the middle of r, k half their size.

func check(p *ui.Painter, r ui.Rect, color ui.Color) {
	cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W*0.3
	p.StrokePolyline([][2]float32{{cx - k, cy}, {cx - k/3, cy + k*0.7}, {cx + k, cy - k*0.7}}, 1.75, color)
}

func magnifier(p *ui.Painter, r ui.Rect, color ui.Color) {
	cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W*0.4
	var lens ui.Path
	lens.Circle(cx-k*0.2, cy-k*0.2, k*0.6)
	p.StrokePath(&lens, 1.5, color)
	p.StrokePolyline([][2]float32{{cx + k*0.25, cy + k*0.25}, {cx + k*0.85, cy + k*0.85}}, 1.5, color)
}

func eye(p *ui.Painter, r ui.Rect, color ui.Color) {
	cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W*0.42
	var lid, pupil ui.Path
	lid.MoveTo(cx-k, cy).QuadTo(cx, cy-k*1.15, cx+k, cy).QuadTo(cx, cy+k*1.15, cx-k, cy).Close()
	p.StrokePath(&lid, 1.25, color)
	pupil.Circle(cx, cy, k*0.3)
	p.FillPath(&pupil, color)
}

func wrench(p *ui.Painter, r ui.Rect, color ui.Color) {
	cx, cy, k := r.X+r.W/2, r.Y+r.H/2, float64(r.W*0.42)
	// The head is a ring open to the top right, its jaws, with the handle
	// going down to the left from its other side.
	hx, hy, hr := float64(cx)+k*0.35, float64(cy)-k*0.35, k*0.5
	p.StrokePolyline(arc(hx, hy, hr, -math.Pi/4+0.8, -math.Pi/4-0.8+2*math.Pi), 1.25, color)
	from := [2]float32{float32(hx + hr*math.Cos(3*math.Pi/4)), float32(hy + hr*math.Sin(3*math.Pi/4))}
	p.StrokePolyline([][2]float32{from, {cx - float32(k)*0.9, cy + float32(k)*0.9}}, 1.5, color)
}

func bulb(p *ui.Painter, r ui.Rect, color ui.Color) {
	cx, cy, k := r.X+r.W/2, r.Y+r.H/2, float64(r.W*0.42)
	gx, gy, gr := float64(cx), float64(cy)-k*0.25, k*0.62
	glass := arc(gx, gy, gr, math.Pi*0.75, math.Pi*2.25)
	p.StrokePolyline(glass, 1.25, color)
	left, right := glass[0], glass[len(glass)-1]
	base := float32(gy + k*0.75)
	p.StrokePolyline([][2]float32{left, {left[0], base}, {right[0], base}, right}, 1.25, color)
	p.StrokePolyline([][2]float32{{left[0] + 1, base + float32(k)*0.3}, {right[0] - 1, base + float32(k)*0.3}}, 1.25, color)
}

// arc returns points along the circle at x, y of radius r, from angle
// from to to, clockwise.
func arc(x, y, r, from, to float64) [][2]float32 {
	var points [][2]float32
	for i := range 21 {
		a := from + float64(i)/20*(to-from)
		points = append(points, [2]float32{float32(x + r*math.Cos(a)), float32(y + r*math.Sin(a))})
	}
	return points
}
