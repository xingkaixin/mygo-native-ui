// Gantt plans a release on a chart of its tasks, as project planners
// do. Drag a bar to move its task and its ends to change its days; choose
// a task to see when it runs and set how much of it is done.
//
//	go run ./examples/gantt
package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/gantt"
)

type app struct {
	tasks []gantt.Task
	chart gantt.State
	today time.Time
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Padding(t.Space(3), t.Space(4)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Release 2.0").Bold().FontSize(t.FontSize + 4).Grow(1)
			ui.Text(c, "Drag a bar to move it, its ends to change its days").FontSize(t.FontSize - 1).TextColor(t.TextMuted)
		})
		ui.Divider(c)
		ui.Scroll(c).Grow(1).Children(func() {
			gantt.View(c, &a.chart, &a.tasks, a.today)
		})
		ui.Divider(c)
		a.details(c)
	})
}

// details shows the task chosen: when it runs, what it waits for, and how
// much of it is done.
func (a *app) details(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Height(t.Space(16)).Gap(t.Space(6)).Padding(t.Space(3), t.Space(4)).AlignItems(ui.Center).Children(func() {
		i, ok := a.chart.Chosen()
		if !ok {
			ui.Text(c, "Choose a task").TextColor(t.TextMuted)
			return
		}
		tk := &a.tasks[i]
		ui.Column(c).Gap(t.Space(1)).Width(t.Space(80)).Children(func() {
			ui.Text(c, tk.Name).Bold()
			when := tk.Start.Format("Mon, Jan 2")
			if tk.Days > 1 {
				when += " – " + tk.Start.AddDate(0, 0, tk.Days-1).Format("Mon, Jan 2")
			}
			ui.Text(c, when).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
		})
		var after []string
		for _, k := range tk.After {
			after = append(after, a.tasks[k].Name)
		}
		if len(after) > 0 {
			ui.Text(c, "After "+strings.Join(after, ", ")).FontSize(t.FontSize - 1).TextColor(t.TextMuted).Shrink(1)
		}
		ui.Box(c).Grow(1)
		if tk.Days > 0 {
			done := float64(tk.Done)
			if ui.Slider(c, &done, 0, 1).Label("Done").Width(t.Space(48)).Changed() {
				tk.Done = float32(done)
			}
			ui.Text(c, fmt.Sprintf("%d%% done", int(tk.Done*100+0.5))).FontFeatures("tnum").Width(t.Space(20))
		}
	})
}

// plan returns the tasks of the release, around the week of today.
func plan(today time.Time) []gantt.Task {
	monday := today.AddDate(0, 0, -(int(today.Weekday())+6)%7)
	on := func(k int) time.Time { return monday.AddDate(0, 0, k) }
	return []gantt.Task{
		{Name: "Design review", Start: on(-7), Days: 3, Done: 1},
		{Name: "Sync engine", Start: on(-4), Days: 9, Done: 0.7, After: []int{0}},
		{Name: "Offline editing", Start: on(0), Days: 8, Done: 0.35, After: []int{0}},
		{Name: "Settings screen", Start: on(2), Days: 4, Done: 0.5},
		{Name: "Feature freeze", Start: on(9), After: []int{1, 2}},
		{Name: "Beta testing", Start: on(9), Days: 5, After: []int{4}},
		{Name: "Fix beta issues", Start: on(14), Days: 5, After: []int{5}},
		{Name: "Release notes", Start: on(15), Days: 3, After: []int{4}},
		{Name: "Ship", Start: on(21), After: []int{6, 7}},
	}
}

func main() {
	y, m, d := time.Now().Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
	a := &app{today: today, tasks: plan(today)}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Gantt",
			Width:   1180,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
