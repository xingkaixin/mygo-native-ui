// Agenda lists the next two weeks of a made-up week planner, made around
// today: standups, a meeting going on now, a conference over three days,
// a holiday, lunches and an evening flight. Click an event to show it
// below.
//
//	go run ./examples/agenda
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/agenda"
)

var (
	work     = ui.Hex("#3b82f6")
	personal = ui.Hex("#22c55e")
	travel   = ui.Hex("#f59e0b")
	holiday  = ui.Hex("#ef4444")
)

func events(now time.Time) []agenda.Event {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	at := func(d, h, m int) time.Time {
		return today.AddDate(0, 0, d).Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute)
	}
	ev := func(title, where string, d, h, m, minutes int, col ui.Color) agenda.Event {
		return agenda.Event{Title: title, Location: where, Start: at(d, h, m), End: at(d, h, m).Add(time.Duration(minutes) * time.Minute), Color: col}
	}
	es := []agenda.Event{
		{Title: "Design review", Location: "Room 4B", Start: now.Add(-20 * time.Minute), End: now.Add(40 * time.Minute), Color: work},
		ev("Lunch with Ada", "Noodle Bar", 0, 12, 30, 60, personal),
		ev("Ship 2.4 release notes", "", 0, 17, 0, 30, work),
		{Title: "GopherCon", Location: "Berlin", Start: at(3, 0, 0), End: at(6, 0, 0), Color: travel},
		ev("Flight to Berlin", "Gate B12", 2, 20, 45, 150, travel),
		{Title: "Founders' Day", Start: at(8, 0, 0), End: at(9, 0, 0), Color: holiday},
		ev("Dentist", "Smile Clinic", 9, 8, 15, 45, personal),
		ev("Quarterly planning", "Main hall", 10, 13, 0, 180, work),
		ev("Movie night", "Downtown Cinema", 11, 19, 30, 150, personal),
	}
	for d := range 14 {
		if wd := today.AddDate(0, 0, d).Weekday(); wd != time.Saturday && wd != time.Sunday {
			es = append(es, ev("Standup", "Video call", d, 9, 30, 15, work))
		}
	}
	return es
}

type app struct {
	events []agenda.Event
	list   agenda.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	if a.events == nil {
		a.events = events(c.Now())
	}
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		agenda.View(c, &a.list, c.Now(), 14, a.events).Grow(1)
		ui.Divider(c)
		ui.Row(c).Padding(t.Space(3), t.Space(4)).Gap(t.Space(2)).Children(func() {
			i := a.list.Chosen()
			if i < 0 {
				ui.Text(c, "Click an event").TextColor(t.TextMuted)
				return
			}
			e := a.events[i]
			ui.Text(c, e.Title).Bold()
			ui.Text(c, e.Start.Format("Mon Jan 2, 15:04")+" – "+e.End.Format("Mon Jan 2, 15:04")).TextColor(t.TextMuted)
		})
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Agenda",
			Width:   480,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
