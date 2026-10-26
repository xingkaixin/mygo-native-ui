// Tzpicker tells the time in a city: choose one, or find it by its name,
// country or zone, and read the time there and how far it is from yours.
//
//	go run ./examples/tzpicker
package main

import (
	"fmt"
	"log"
	"time"
	_ "time/tzdata"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/tzpicker"
)

type app struct {
	city tzpicker.City
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	locale := c.Region().FormatLocale(c.Locale())
	now := c.Now()
	c.After(now.Truncate(time.Second).Add(time.Second).Sub(now))
	there := now.In(a.city.Location())
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(6)).Children(func() {
		tzpicker.View(c, &a.city).Width(t.Space(72))
		ui.Column(c).Gap(t.Space(1)).Children(func() {
			ui.Text(c, i18n.FormatDate(locale, there, i18n.DateOptions{Skeleton: "jms"})).FontSize(56).Bold().FontFeatures("tnum")
			ui.Text(c, i18n.FormatDate(locale, there, i18n.DateOptions{Skeleton: "yMMMMEEEEd"})).FontSize(t.FontSize + 2)
			ui.Text(c, apart(now, a.city.Location())).TextColor(t.TextMuted)
		})
	})
}

// apart says how far loc's time is from the user's at now.
func apart(now time.Time, loc *time.Location) string {
	_, here := now.Zone()
	_, there := now.In(loc).Zone()
	d := time.Duration(there-here) * time.Second
	switch {
	case d == 0:
		return "Same time as you"
	case d > 0:
		return fmt.Sprintf("%s ahead of you", hours(d))
	default:
		return fmt.Sprintf("%s behind you", hours(-d))
	}
}

func hours(d time.Duration) string {
	h, m := int(d.Hours()), int(d.Minutes())%60
	unit := "hours"
	if h == 1 {
		unit = "hour"
	}
	switch {
	case h == 0:
		return fmt.Sprintf("%d minutes", m)
	case m == 0:
		return fmt.Sprintf("%d %s", h, unit)
	}
	return fmt.Sprintf("%d %s %d minutes", h, unit, m)
}

func main() {
	a := &app{city: tzpicker.City{Name: "Tokyo", Country: "Japan", Zone: "Asia/Tokyo"}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Time Zone Picker",
			Width:   640,
			Height:  480,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
