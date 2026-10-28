// Package recipe shows a recipe to cook from: its ingredients beside its
// steps. The stepper by Serves scales every amount, and Metric and US
// switch the units, grams to ounces and milliliters to cups, and the
// oven's degrees in the steps. US amounts read as fractions, as ⅓ cup. A
// click on an ingredient ticks it off once it is out, and on a step's
// number marks it done. A step that takes a while has a timer: a click
// starts it, counting down around a ring, and pauses it; when it runs out
// it stays lit until clicked, and timers run side by side.
//
//	recipe.View(c, &app.cooking, app.recipe)
package recipe

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
)

// Ingredient is an amount of something: Unit is g, kg, ml, l, tsp, tbsp,
// cup, oz or lb, or empty for a count, as of eggs.
type Ingredient struct {
	Amount float64
	Unit   string
	Name   string
}

// Step is a step of the method, and how long its timer runs, if it has
// one. Degrees written as 180°C turn into °F with US units.
type Step struct {
	Text  string
	Timer time.Duration
}

// Recipe is a recipe: how many it serves, and what goes into it how.
type Recipe struct {
	Name        string
	Serves      int
	Ingredients []Ingredient
	Steps       []Step
}

// State is what a recipe keeps: the servings, the units, what is ticked
// off and the timers. Keep it in the app's state, one for each recipe.
type State struct {
	serves float64
	units  int
	have   map[int]bool
	done   map[int]bool
	timers map[int]*timer
}

type timer struct {
	end     time.Time
	left    time.Duration
	running bool
}

var (
	grams  = map[string]float64{"g": 1, "kg": 1000, "oz": 28.3495, "lb": 453.592}
	litres = map[string]float64{"ml": 1, "l": 1000, "tsp": 4.92892, "tbsp": 14.7868, "cup": 236.588}
	heat   = regexp.MustCompile(`(\d+)\s?°C`)
)

// View creates the recipe r.
func View(c *ui.Context, s *State, r Recipe) ui.Element {
	t := c.Theme()
	if s.serves == 0 {
		s.serves = float64(max(1, r.Serves))
	}
	if s.have == nil {
		s.have, s.done, s.timers = map[int]bool{}, map[int]bool{}, map[int]*timer{}
	}
	us := s.units == 1
	scale := s.serves / float64(max(1, r.Serves))
	now := c.Now()
	root := ui.Column(c).Gap(t.Space(6))
	root.Children(func() {
		ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Center).Wrap().Children(func() {
			ui.Text(c, r.Name).FontSize(t.FontSize + 10).Bold().Grow(1)
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Serves").TextColor(t.TextMuted)
				ui.Text(c, strconv.Itoa(int(s.serves))).Bold().FontFeatures("tnum").MinWidth(t.Space(5)).TextAlign(ui.End)
				ui.Stepper(c, &s.serves, 1, 24, 1).Label("Servings")
			})
			ui.Segmented(c, &s.units, "Metric", "US").Label("Units")
		})
		ui.Row(c).Gap(t.Space(10)).AlignItems(ui.Start).Children(func() {
			ui.Column(c).Width(t.Space(64)).Gap(t.Space(1)).Role(ui.RoleList).Label("Ingredients").Children(func() {
				ui.Text(c, "Ingredients").Bold().Margin(0, 0, t.Space(2), 0)
				for i, in := range r.Ingredients {
					have := s.have[i]
					row := ui.ButtonBase(c.Key(i)).Gap(t.Space(2.5)).AlignItems(ui.Center).Padding(t.Space(1.5), t.Space(2)).
						Radius(t.Space(1.5)).Role(ui.RoleCheckBox).Checked(have).Cursor(ui.CursorPointer)
					if row.Hovered() {
						row.Background(t.SurfaceHover)
					}
					if row.Clicked() {
						s.have[i] = !have
					}
					amount := measure(in, scale, us)
					row.Label(amount + " " + in.Name)
					row.Children(func() {
						box := ui.Box(c).Size(t.Space(4), t.Space(4)).Radius(t.Space(1)).Border(1.5, t.Border).Center()
						if have {
							box.Background(t.Accent).BorderColor(t.Accent).Children(func() {
								ui.Text(c, "✓").FontSize(t.FontSize - 3).Bold().TextColor(t.AccentText)
							})
						}
						a := ui.Text(c, amount).Bold().FontFeatures("tnum").MinWidth(t.Space(18))
						n := ui.Text(c, in.Name).Grow(1)
						if have {
							a.TextColor(t.TextMuted)
							n.TextColor(t.TextMuted).Strikethrough()
						}
					})
				}
			})
			ui.Column(c).Grow(1).Gap(t.Space(4)).Role(ui.RoleList).Label("Method").Children(func() {
				ui.Text(c, "Method").Bold()
				for i, st := range r.Steps {
					ui.Row(c.Key(i)).Gap(t.Space(3)).AlignItems(ui.Start).Children(func() {
						done := s.done[i]
						num := ui.ButtonBase(c).Size(t.Space(7), t.Space(7)).Radius(t.Space(3.5)).Center().
							Border(1.5, t.Border).Role(ui.RoleCheckBox).Checked(done).Label(fmt.Sprintf("Step %d", i+1)).Cursor(ui.CursorPointer)
						if done {
							num.Background(t.Accent).BorderColor(t.Accent)
						}
						if num.Clicked() {
							s.done[i] = !done
						}
						num.Children(func() {
							mark := ui.Text(c, strconv.Itoa(i+1)).FontSize(t.FontSize - 1).Bold()
							if done {
								mark.TextColor(t.AccentText)
							}
						})
						ui.Column(c).Grow(1).Gap(t.Space(2)).Children(func() {
							text := ui.Text(c, oven(st.Text, us)).LineHeight(1.5)
							if done {
								text.TextColor(t.TextMuted)
							}
							if st.Timer > 0 {
								ui.Row(c).Children(func() { clock(c, s, i, st.Timer, now) })
							}
						})
					})
				}
			})
		})
	})
	return root
}

// clock creates the timer of step i, which runs for d.
func clock(c *ui.Context, s *State, i int, d time.Duration, now time.Time) {
	t := c.Theme()
	tm := s.timers[i]
	left := d
	if tm != nil {
		left = tm.left
		if tm.running {
			left = max(0, tm.end.Sub(now))
		}
	}
	over := tm != nil && left == 0
	if tm != nil && tm.running && !over {
		c.After(left%time.Second + time.Millisecond)
	}
	chip := ui.ButtonBase(c).Gap(t.Space(2)).AlignItems(ui.Center).Padding(t.Space(1), t.Space(3), t.Space(1), t.Space(1.5)).
		Radius(t.Space(4)).Border(1, t.Border).Cursor(ui.CursorPointer)
	label := "Start timer"
	switch {
	case over:
		label = "Timer done"
		chip.Background(ui.Hex("#F59F00")).BorderColor(ui.Hex("#F59F00"))
	case tm != nil && tm.running:
		label = "Pause timer"
		chip.Background(t.Accent.Alpha(0.1)).BorderColor(t.Accent)
	case tm != nil:
		label = "Resume timer"
	case chip.Hovered():
		chip.Background(t.SurfaceHover)
	}
	chip.Label(label).Value(minutes(left))
	if chip.Clicked() {
		switch {
		case tm == nil:
			s.timers[i] = &timer{end: now.Add(d), running: true}
		case over:
			delete(s.timers, i)
		case tm.running:
			tm.left, tm.running = left, false
		default:
			tm.end, tm.running = now.Add(tm.left), true
		}
	}
	running := tm != nil && tm.running
	chip.Children(func() {
		frac := float32(1 - float64(left)/float64(d))
		ui.Box(c).Size(t.Space(5), t.Space(5)).Draw(func(p *ui.Painter, r ui.Rect) {
			cx, cy, rad := r.X+r.W/2, r.Y+r.H/2, r.W/2-1.5
			var ring ui.Path
			ring.Circle(cx, cy, rad)
			col := t.TextMuted
			if running {
				col = t.Accent
			}
			if over {
				col = ui.RGB(255, 255, 255)
			}
			p.StrokePath(&ring, 2, col.Alpha(0.3))
			if frac > 0 {
				var arc ui.Path
				steps := max(2, int(frac*48))
				for k := range steps + 1 {
					a := -math.Pi/2 + 2*math.Pi*float64(frac)*float64(k)/float64(steps)
					x, y := cx+rad*float32(math.Cos(a)), cy+rad*float32(math.Sin(a))
					if k == 0 {
						arc.MoveTo(x, y)
					} else {
						arc.LineTo(x, y)
					}
				}
				p.StrokePath(arc.Cap(ui.RoundCap), 2, col)
			}
		})
		text := ui.Text(c, minutes(left)).FontFeatures("tnum").FontSize(t.FontSize - 1).Bold()
		switch {
		case over:
			text.TextColor(ui.RGB(255, 255, 255))
		case running:
			text.TextColor(t.Accent)
		}
	})
}

// minutes writes d as 12:05, or 1:02:05 past an hour.
func minutes(d time.Duration) string {
	sec := int((d + time.Second - 1) / time.Second)
	if sec >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", sec/3600, sec/60%60, sec%60)
	}
	return fmt.Sprintf("%d:%02d", sec/60, sec%60)
}

// measure writes the amount of in, scaled, in metric or US units.
func measure(in Ingredient, scale float64, us bool) string {
	v := in.Amount * scale
	if g, ok := grams[in.Unit]; ok {
		v *= g
		switch {
		case us && v >= 453.592:
			return fraction(v/453.592) + " lb"
		case us:
			return fraction(v/28.3495) + " oz"
		case v >= 1000:
			return trim(v/1000) + " kg"
		}
		return trim(round(v)) + " g"
	}
	if ml, ok := litres[in.Unit]; ok {
		v *= ml
		small := in.Unit == "tsp" || in.Unit == "tbsp"
		switch {
		case (us || small) && v < 14.7868:
			return fraction(v/4.92892) + " tsp"
		case (us || small) && v < 59:
			return fraction(v/14.7868) + " tbsp"
		case us:
			return fraction(v/236.588) + " cup"
		case v >= 1000:
			return trim(v/1000) + " l"
		}
		return trim(round(v)) + " ml"
	}
	unit := ""
	if in.Unit != "" {
		unit = " " + in.Unit
	}
	return fraction(v) + unit
}

// round rounds grams or milliliters as a cook would weigh them.
func round(v float64) float64 {
	switch {
	case v >= 100:
		return math.Round(v/5) * 5
	case v >= 10:
		return math.Round(v)
	}
	return math.Round(v*10) / 10
}

func trim(v float64) string { return strconv.FormatFloat(math.Round(v*100)/100, 'f', -1, 64) }

var vulgar = map[int]string{1: "⅛", 2: "¼", 3: "⅓", 4: "½", 5: "⅔", 6: "¾", 7: "⅞"}

// fraction writes v to the nearest eighth or third, as 1½.
func fraction(v float64) string {
	whole := math.Floor(v)
	rest := v - whole
	best, part := math.Inf(1), 0
	for k, f := range []float64{0, 1. / 8, 1. / 4, 1. / 3, 1. / 2, 2. / 3, 3. / 4, 7. / 8, 1} {
		if d := math.Abs(rest - f); d < best {
			best, part = d, k
		}
	}
	if part == 8 {
		whole, part = whole+1, 0
	}
	switch {
	case part == 0 && whole == 0:
		return vulgar[1]
	case part == 0:
		return strconv.Itoa(int(whole))
	case whole == 0:
		return vulgar[part]
	}
	return strconv.Itoa(int(whole)) + vulgar[part]
}

// oven writes the degrees in text in °F for US units.
func oven(text string, us bool) string {
	if !us {
		return text
	}
	return heat.ReplaceAllStringFunc(text, func(m string) string {
		n, _ := strconv.Atoi(heat.FindStringSubmatch(m)[1])
		return strconv.Itoa(int(math.Round((float64(n)*9/5+32)/25)*25)) + "°F"
	})
}
