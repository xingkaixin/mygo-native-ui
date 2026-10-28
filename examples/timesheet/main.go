// Timesheet keeps a made-up week of hours across five projects, and an
// empty sheet for each week before or after it.
//
//	go run ./examples/timesheet
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/timesheet"
)

var projects = []timesheet.Row{
	{Project: "Atlas", Task: "Checkout redesign"},
	{Project: "Atlas", Task: "Code review"},
	{Project: "Beacon", Task: "Search indexing"},
	{Project: "Internal", Task: "Meetings"},
	{Project: "Internal", Task: "Hiring"},
}

type app struct {
	monday time.Time
	weeks  map[time.Time][]timesheet.Row
	sheet  timesheet.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	rows, ok := a.weeks[a.monday]
	if !ok {
		rows = append([]timesheet.Row(nil), projects...)
	}
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(5)).Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			if ui.Button(c, "‹").Clicked() {
				a.monday = a.monday.AddDate(0, 0, -7)
			}
			if ui.Button(c, "›").Clicked() {
				a.monday = a.monday.AddDate(0, 0, 7)
			}
			sunday := a.monday.AddDate(0, 0, 6)
			ui.Text(c, a.monday.Format("2 Jan")+" – "+sunday.Format("2 Jan 2006")).FontSize(t.FontSize + 3).FontWeight(700).MarginX(t.Space(2))
		})
		timesheet.View(c, &a.sheet, a.monday, &rows, 40)
	})
	a.weeks[a.monday] = rows
}

func main() {
	now := time.Now()
	monday := time.Date(now.Year(), now.Month(), now.Day()-(int(now.Weekday())+6)%7, 0, 0, 0, 0, now.Location())
	week := append([]timesheet.Row(nil), projects...)
	week[0].Hours = [7]float64{4, 5.5, 3}
	week[1].Hours = [7]float64{1, 0.75, 1.5}
	week[2].Hours = [7]float64{2, 1, 2.5}
	week[3].Hours = [7]float64{1, 0.5, 1}
	week[4].Hours = [7]float64{0, 0, 0.5}
	a := &app{monday: monday, weeks: map[time.Time][]timesheet.Row{monday: week}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Timesheet",
			Width:   1000,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
