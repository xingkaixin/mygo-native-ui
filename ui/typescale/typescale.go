// Package typescale previews a modular type scale, as design systems set
// their font sizes: a base size and a ratio, each step the one below times
// the ratio. A row for each step, from the largest down, names it, gives
// its size in pixels and rems, and sets a line of sample text at it, the
// base step marked. The stepper sets the base, the menu the ratio by its
// musical name, and the field the sample. The steps as CSS custom
// properties show under them, with a button to copy them.
//
//	typescale.View(c, &app.preview, &app.scale)
package typescale

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// Scale is a type scale: the base size in pixels, the ratio between
// steps, and how many steps it has below and above the base.
type Scale struct {
	Base         float64
	Ratio        float64
	Below, Above int
}

// Size returns the size of step n in pixels, 0 the base.
func (s Scale) Size(n int) float64 { return s.Base * math.Pow(s.Ratio, float64(n)) }

// Name returns the name of step n, as Tailwind names them: base, sm and
// xs below it, lg, xl, 2xl and so on above.
func Name(n int) string {
	switch {
	case n == 0:
		return "base"
	case n == 1:
		return "lg"
	case n == 2:
		return "xl"
	case n > 2:
		return fmt.Sprintf("%dxl", n-1)
	case n == -1:
		return "sm"
	case n == -2:
		return "xs"
	}
	return fmt.Sprintf("%dxs", -n-1)
}

// CSS returns the steps as custom properties in rems of 16 pixels.
func (s Scale) CSS() string {
	var b strings.Builder
	b.WriteString(":root {\n")
	for n := s.Above; n >= -s.Below; n-- {
		fmt.Fprintf(&b, "  --text-%s: %srem;\n", Name(n), trim(s.Size(n)/16, 3))
	}
	b.WriteString("}")
	return b.String()
}

// Ratios are the ratios of the menu, by name.
var Ratios = []struct {
	Name  string
	Ratio float64
}{
	{"Minor second", 1.067}, {"Major second", 1.125}, {"Minor third", 1.2}, {"Major third", 1.25},
	{"Perfect fourth", 1.333}, {"Augmented fourth", 1.414}, {"Perfect fifth", 1.5}, {"Golden ratio", 1.618},
}

// State is what a preview keeps: the sample and when the CSS was copied.
// Keep it in the app's state, one for each preview.
type State struct {
	sample string
	set    bool
	copied time.Time
}

// View creates the preview of *scale.
func View(c *ui.Context, s *State, scale *Scale) ui.Element {
	t := c.Theme()
	if !s.set {
		s.sample, s.set = "Sphinx of black quartz, judge my vow", true
	}
	root := ui.Column(c).Gap(t.Space(5))
	changed := false
	root.Children(func() {
		ui.Row(c).Gap(t.Space(5)).AlignItems(ui.Center).Wrap().Children(func() {
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Base").TextColor(t.TextMuted)
				ui.Text(c, trim(scale.Base, 0)+" px").Bold().FontFeatures("tnum").MinWidth(t.Space(11)).TextAlign(ui.End)
				if ui.Stepper(c, &scale.Base, 10, 24, 1).Label("Base size").Changed() {
					changed = true
				}
			})
			names := make([]string, len(Ratios))
			name := ""
			for i, r := range Ratios {
				names[i] = fmt.Sprintf("%s · %g", r.Name, r.Ratio)
				if r.Ratio == scale.Ratio {
					name = names[i]
				}
			}
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Ratio").TextColor(t.TextMuted)
				if ui.Select(c, &name, names).Label("Ratio").Changed() {
					for i, n := range names {
						if n == name {
							scale.Ratio = Ratios[i].Ratio
						}
					}
					changed = true
				}
			})
			ui.TextInput(c, &s.sample).Grow(1).MinWidth(t.Space(50)).Label("Sample text")
		})
		ui.Column(c).FillWidth().Children(func() {
			for n := scale.Above; n >= -scale.Below; n-- {
				size := scale.Size(n)
				row := ui.Row(c.Key(n)).FillWidth().Gap(t.Space(4)).AlignItems(ui.Center).Padding(t.Space(2), t.Space(3)).
					Radius(t.Space(1.5)).Label(Name(n)).Value(trim(size, 2) + " pixels")
				if n == 0 {
					row.Background(t.Accent.Alpha(0.08))
				}
				row.Children(func() {
					ui.Column(c).Width(t.Space(32)).Gap(t.Space(0.5)).Children(func() {
						ui.Text(c, Name(n)).Font("monospace").Bold()
						ui.Text(c, trim(size, 2)+" px · "+trim(size/16, 3)+" rem").FontSize(t.FontSize - 2).FontFeatures("tnum").TextColor(t.TextMuted)
					})
					ui.Text(c, s.sample).FontSize(float32(size)).Grow(1).Ellipsis("…").MaxLines(1)
				})
				ui.Box(c.Key(fmt.Sprint("rule", n))).FillWidth().Height(1).Background(t.Border.Alpha(0.6))
			}
		})
		css := scale.CSS()
		ui.Row(c).Gap(t.Space(3)).Padding(t.Space(3)).Radius(t.Space(2)).Background(t.Surface).AlignItems(ui.Start).Children(func() {
			ui.Text(c, css).Font("monospace").FontSize(t.FontSize - 1).Selectable().Grow(1)
			label := "Copy"
			if left := 1200*time.Millisecond - c.Now().Sub(s.copied); left > 0 {
				label = "Copied"
				c.After(left)
			}
			if ui.Button(c, label).Clicked() {
				c.WriteClipboard(css)
				s.copied = c.Now()
			}
		})
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// trim writes v to at most places decimals, without trailing zeros.
func trim(v float64, places int) string {
	s := fmt.Sprintf("%.*f", places, v)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	return s
}
