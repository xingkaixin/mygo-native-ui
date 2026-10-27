// Wheelpicker sets an alarm's date and time on wheels: drag or scroll a
// wheel, or click a choice above or below the band. The days follow the
// month and year, and the hours and minutes go round.
//
//	go run ./examples/wheelpicker
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/wheelpicker"
)

const firstYear = 2000

var (
	months  = numbers(1, 12, func(i int) string { return time.Month(i).String() })
	years   = numbers(firstYear, firstYear+50, func(i int) string { return fmt.Sprint(i) })
	hours   = numbers(1, 12, func(i int) string { return fmt.Sprint(i) })
	minutes = numbers(0, 59, func(i int) string { return fmt.Sprintf("%02d", i) })
	halves  = []string{"AM", "PM"}
)

func numbers(from, to int, f func(int) string) []string {
	s := make([]string, 0, to-from+1)
	for i := from; i <= to; i++ {
		s = append(s, f(i))
	}
	return s
}

type app struct {
	month, day, year   int
	hour, minute, half int
	changed            string
}

func (a *app) when() time.Time {
	h := a.hour + 1
	if h == 12 {
		h = 0
	}
	return time.Date(firstYear+a.year, time.Month(a.month+1), a.day+1, h+12*a.half, a.minute, 0, 0, time.Local)
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	last := time.Date(firstYear+a.year, time.Month(a.month+2), 0, 0, 0, 0, 0, time.UTC).Day()
	days := numbers(1, last, func(i int) string { return fmt.Sprint(i) })

	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(6)).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		ui.Text(c, a.when().Format("Monday, January 2, 2006 at 3:04 PM")).FontSize(t.FontSize + 3).Bold()
		ui.Row(c).Gap(t.Space(6)).Children(func() {
			wheelpicker.View(c, func() {
				changed := wheelpicker.Wheel(c, &a.month, months).Label("Month").Width(t.Space(32)).Changed()
				changed = wheelpicker.Wheel(c, &a.day, days).Label("Day").Changed() || changed
				changed = wheelpicker.Wheel(c, &a.year, years).Label("Year").Changed() || changed
				if changed {
					a.changed = "Date set"
				}
			})
			wheelpicker.View(c, func() {
				changed := wheelpicker.Cycle(c, &a.hour, hours).Label("Hour").Changed()
				changed = wheelpicker.Cycle(c, &a.minute, minutes).Label("Minute").Changed() || changed
				changed = wheelpicker.Wheel(c, &a.half, halves).Label("AM or PM").Changed() || changed
				if changed {
					a.changed = "Time set"
				}
			})
		})
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, a.changed).FontSize(t.FontSize - 1).TextColor(t.TextMuted).Width(t.Space(20))
			if ui.Button(c, "Now").Clicked() {
				a.set(time.Now())
				a.changed = ""
			}
		})
	})
}

func (a *app) set(now time.Time) {
	a.month, a.day, a.year = int(now.Month())-1, now.Day()-1, now.Year()-firstYear
	h := now.Hour()
	a.half, a.minute = h/12, now.Minute()
	a.hour = (h + 11) % 12
}

func main() {
	a := &app{}
	a.set(time.Now())
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Wheelpicker",
			Width:   760,
			Height:  420,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
