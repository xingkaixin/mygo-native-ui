// Weekview is a calendar's week, as macOS Calendar shows it: drag events
// to move them, and their bottom edges to change when they end. The
// buttons go to the weeks before and after, and back to this one.
//
//	go run ./examples/weekview
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/weekview"
)

// app is the events, the week shown from its Monday, and the view's
// state.
type app struct {
	events []weekview.Event
	monday time.Time
	week   weekview.State
	moves  int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Padding(8, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			sunday := a.monday.AddDate(0, 0, 6)
			ui.Text(c, a.monday.Format("January 2")+" – "+sunday.Format("January 2, 2006")).Bold().Grow(1)
			if ui.Button(c, "‹").Label("Previous week").Clicked() {
				a.monday = a.monday.AddDate(0, 0, -7)
			}
			if ui.Button(c, "Today").Clicked() {
				a.monday = monday(c.Now())
			}
			if ui.Button(c, "›").Label("Next week").Clicked() {
				a.monday = a.monday.AddDate(0, 0, 7)
			}
		})
		ui.Divider(c)
		if weekview.View(c, &a.week, a.monday, 7, a.events).Grow(1).Changed() {
			a.moves++
		}
		ui.Divider(c)
		status := fmt.Sprintf("%d events · %d moved", len(a.events), a.moves)
		if i := a.week.Chosen(); i >= 0 {
			e := a.events[i]
			status = fmt.Sprintf("%s · %s, %s – %s", e.Title, e.Start.Format("Monday"), e.Start.Format("15:04"), e.End.Format("15:04"))
		}
		ui.Text(c, status).FontSize(12).TextColor(t.TextMuted).Padding(6, 12)
	})
}

// monday returns the start of the Monday of the week of tm.
func monday(tm time.Time) time.Time {
	day := time.Date(tm.Year(), tm.Month(), tm.Day(), 0, 0, 0, 0, tm.Location())
	return day.AddDate(0, 0, -(int(day.Weekday())+6)%7)
}

func main() {
	a := &app{monday: monday(time.Now())}
	purple, green, orange, red := ui.RGB(175, 82, 222), ui.RGB(52, 199, 89), ui.RGB(255, 149, 0), ui.RGB(255, 59, 48)
	event := func(title string, day, hour, minute, length int, color ui.Color) {
		start := a.monday.AddDate(0, 0, day).Add(time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute)
		a.events = append(a.events, weekview.Event{Title: title, Start: start, End: start.Add(time.Duration(length) * time.Minute), Color: color})
	}
	for day := range 5 {
		event("Standup", day, 9, 30, 15, ui.Color{})
		event("Lunch", day, 12, 0, 60, orange)
	}
	event("Design review", 0, 10, 0, 90, purple)
	event("1:1 with Sam", 0, 10, 30, 30, green)
	event("Roadmap planning", 1, 14, 0, 120, ui.Color{})
	event("Interview", 1, 15, 0, 60, red)
	event("Focus time", 2, 13, 30, 150, green)
	event("Release", 3, 16, 0, 45, red)
	event("Retro", 4, 15, 0, 60, purple)
	event("Climbing", 5, 10, 0, 120, orange)
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Calendar",
			Width:   1000,
			Height:  700,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
