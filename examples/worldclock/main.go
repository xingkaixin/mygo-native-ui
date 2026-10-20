// Worldclock plans a call across nine cities, from San Francisco: click or
// drag across the hours, or press the arrows, to choose its time. The
// buttons go to the days before and after, and back to now, and the time
// chosen is written out below for each city, ready to copy.
//
//	go run ./examples/worldclock
package main

import (
	"log"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/worldclock"
)

type app struct {
	places []worldclock.Place
	at     time.Time
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	locale := c.Region().FormatLocale(c.Locale())
	home := a.places[0].Loc
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(5)).Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.FormatDate(locale, a.at.In(home), i18n.DateOptions{Skeleton: "yMMMMEEEEd"})).Bold().FontSize(t.FontSize + 4).Grow(1)
			if ui.Button(c, "‹").Label("Previous day").Clicked() {
				a.at = a.at.In(home).AddDate(0, 0, -1)
			}
			if ui.Button(c, "Now").Clicked() {
				a.at = c.Now().Truncate(15 * time.Minute)
			}
			if ui.Button(c, "›").Label("Next day").Clicked() {
				a.at = a.at.In(home).AddDate(0, 0, 1)
			}
		})
		worldclock.View(c, &a.at, a.places)
		var lines []string
		for _, p := range a.places {
			tm := a.at.In(p.Loc)
			lines = append(lines, p.Name+" — "+i18n.FormatDate(locale, tm, i18n.DateOptions{Skeleton: "MMMEdjm"})+" "+tm.Format("MST"))
		}
		ui.Column(c).Gap(t.Space(2)).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "In each city").Bold().Grow(1)
				if ui.Button(c, "Copy").Clicked() {
					c.WriteClipboard(strings.Join(lines, "\n"))
					c.Toast("Copied the times")
				}
			})
			ui.Column(c).Selectable().Padding(t.Space(3), t.Space(4)).Gap(t.Space(1)).Radius(t.Space(2)).Background(t.Surface).Children(func() {
				for _, line := range lines {
					ui.Text(c, line).FontFeatures("tnum")
				}
			})
		})
	})
}

func main() {
	a := &app{at: time.Now().Truncate(15 * time.Minute)}
	for _, p := range []struct{ name, zone string }{
		{"San Francisco", "America/Los_Angeles"},
		{"New York", "America/New_York"},
		{"London", "Europe/London"},
		{"Berlin", "Europe/Berlin"},
		{"Kolkata", "Asia/Kolkata"},
		{"Kathmandu", "Asia/Kathmandu"},
		{"Shanghai", "Asia/Shanghai"},
		{"Tokyo", "Asia/Tokyo"},
		{"Sydney", "Australia/Sydney"},
	} {
		loc, err := time.LoadLocation(p.zone)
		if err != nil {
			log.Fatal(err)
		}
		a.places = append(a.places, worldclock.Place{Name: p.name, Loc: loc})
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "World clock",
			Width:   1280,
			Height:  860,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
