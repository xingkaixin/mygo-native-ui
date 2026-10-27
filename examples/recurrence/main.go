// Recurrence sets how a team's design review repeats: choose the
// frequency, the days and when it ends, and read the next times and the
// RRULE below. Paste an RRULE into the field at the bottom to load it.
//
//	go run ./examples/recurrence
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/recurrence"
)

type app struct {
	start time.Time
	rule  recurrence.Rule
	typed string
	wrong string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Padding(t.Space(6)).Gap(t.Space(5)).Children(func() {
			ui.Column(c).Gap(t.Space(1)).Children(func() {
				ui.Text(c, "Design review").FontSize(t.FontSize + 6).Bold()
				ui.Text(c, a.start.Format("Monday, January 2 2006, 15:04")+" – 16:00").TextColor(t.TextMuted)
			})
			recurrence.View(c, &a.rule, a.start)
			ui.Divider(c)
			field := ui.TextInput(c, &a.typed).Font("monospace").Placeholder("Paste an RRULE, as FREQ=MONTHLY;BYDAY=-1FR").Label("RRULE")
			if a.wrong != "" {
				field.Error(a.wrong)
				ui.Text(c, a.wrong).FontSize(t.FontSize - 1).TextColor(t.Danger)
			}
			if field.Submitted() {
				r, err := recurrence.Parse(a.typed)
				a.wrong = ""
				if err != nil {
					a.wrong = err.Error()
				} else {
					a.rule, a.typed = r, ""
				}
			}
		})
	})
}

func main() {
	start := time.Date(2026, time.November, 5, 15, 0, 0, 0, time.Local)
	rule := recurrence.Rule{Freq: recurrence.Weekly, Interval: 2, Days: []time.Weekday{time.Monday, time.Thursday}, Count: 12}
	a := &app{start: start, rule: rule}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Recurrence",
			Width:   640,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
