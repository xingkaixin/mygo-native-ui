// Package contextmeter shows how much of a model's context window a
// conversation fills, as AI chat apps show it by the prompt: a small ring
// and the share used, amber from 80% and red from 95%, and while the
// pointer rests on it, a card of the tokens, by kind, and the cost.
//
//	contextmeter.View(c, contextmeter.Usage{
//		Used: 45_200, Window: 200_000,
//		Input: 38_000, Output: 4_100, Cached: 3_100, Cost: 0.18,
//	})
package contextmeter

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/ring"
)

// Usage is what a conversation uses of a model's context window.
type Usage struct {
	// Used is the tokens the context holds, of Window, the most it holds.
	Used, Window int
	// Input, Output and Cached are the tokens by kind, read from the cache
	// for Cached; the card leaves out those 0.
	Input, Output, Cached int
	// Cost is in dollars; the card leaves it out at 0.
	Cost float64
}

// warning and critical are the shares of the window from which the meter
// turns amber and red.
const (
	warning  = 0.8
	critical = 0.95
)

// View creates the meter of u: a ring and the share used, "23%", and the
// card on hover, or as it gets the keyboard focus.
func View(c *ui.Context, u Usage) ui.Element {
	t := c.Theme()
	f := 0.0
	if u.Window > 0 {
		f = float64(u.Used) / float64(u.Window)
	}
	color := t.TextMuted
	switch {
	case f >= critical:
		color = t.Danger
	case f >= warning:
		color = t.Warning
	}
	summary := fmt.Sprintf("%s of %s tokens", tokens(u.Used), tokens(u.Window))
	meter := ui.Row(c).Gap(t.Space(1.5)).Padding(t.Space(1), t.Space(2)).Radius(t.Space(1.5)).AlignItems(ui.Center).
		Focusable().Label("Context").Description(summary)
	if meter.Hovered() {
		meter.Background(t.Text.Alpha(0.06))
	}
	meter.Children(func() {
		ring.Progress(c, f, color).Size(t.Space(3.5), t.Space(3.5)).Label("Context used")
		ui.Text(c, percent(f)).FontSize(t.FontSize - 1).TextColor(color).FontFeatures("tnum")
	})
	ui.TooltipBase(c, meter, func(tip ui.Element) {
		tip.Width(t.Space(60)).Padding(t.Space(3)).Radius(t.Space(2.5)).Background(t.Background).
			Border(1, t.Border).Shadow(0, 4, 16, 0, ui.RGB(0, 0, 0).Alpha(0.12))
		tip.Children(func() { card(c, u, f, summary) })
	})
	return meter
}

// card builds the card of u, f of its window used.
func card(c *ui.Context, u Usage, f float64, summary string) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(2)).Children(func() {
		ui.Row(c).Gap(t.Space(2)).Children(func() {
			ui.Text(c, percent(f)).Bold().FontFeatures("tnum").Grow(1)
			ui.Text(c, summary).TextColor(t.TextMuted).FontFeatures("tnum")
		})
		ui.Meter(c, float64(u.Used), 0, float64(u.Window), &ui.MeterLevels{
			Warning: warning * float64(u.Window), Critical: critical * float64(u.Window),
		}).Label("Context used")
		rows := []struct {
			name string
			n    int
		}{{"Input", u.Input}, {"Output", u.Output}, {"Cached", u.Cached}}
		for _, r := range rows {
			if r.n == 0 {
				continue
			}
			line(c, r.name, tokens(r.n), false)
		}
		if u.Cost > 0 {
			ui.Divider(c)
			line(c, "Total cost", fmt.Sprintf("$%.2f", u.Cost), true)
		}
	})
}

// line shows name, muted, and value at the end of a row.
func line(c *ui.Context, name, value string, bold bool) {
	t := c.Theme()
	ui.Row(c).Children(func() {
		ui.Text(c, name).TextColor(t.TextMuted).Grow(1)
		v := ui.Text(c, value).FontFeatures("tnum")
		if bold {
			v.Bold()
		}
	})
}

// percent formats f as "23%".
func percent(f float64) string { return fmt.Sprintf("%.0f%%", 100*f) }

// tokens formats n as "820", "4.1k", "45k" or "1.2M".
func tokens(n int) string {
	switch {
	case n < 1_000:
		return fmt.Sprint(n)
	case n < 10_000:
		return strings.TrimSuffix(fmt.Sprintf("%.1f", float64(n)/1e3), ".0") + "k"
	case n < 999_500:
		return fmt.Sprintf("%.0fk", float64(n)/1e3)
	}
	return strings.TrimSuffix(fmt.Sprintf("%.1f", float64(n)/1e6), ".0") + "M"
}
