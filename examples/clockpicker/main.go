// Clockpicker sets an alarm on a 12-hour clock and a meeting on a 24-hour
// one.
//
//	go run ./examples/clockpicker
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/clockpicker"
)

type app struct {
	alarm, meeting   time.Time
	alarmP, meetingP clockpicker.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Gap(t.Space(14)).AlignItems(ui.Center).Justify(ui.Center).Background(t.Background).Children(func() {
		for _, p := range []struct {
			title string
			state *clockpicker.State
			at    *time.Time
		}{{"Alarm", &a.alarmP, &a.alarm}, {"Meeting", &a.meetingP, &a.meeting}} {
			ui.Column(c.Key(p.title)).Gap(t.Space(4)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, p.title).Bold().TextColor(t.TextMuted)
				clockpicker.View(c, p.state, p.at)
				ui.Text(c, p.at.Format("Mon Jan 2, 3:04 PM")).FontFeatures("tnum").TextColor(t.TextMuted)
			})
		}
	})
}

func main() {
	day := time.Date(2026, 11, 5, 0, 0, 0, 0, time.Local)
	a := &app{alarm: day.Add(6*time.Hour + 45*time.Minute), meeting: day.Add(14*time.Hour + 30*time.Minute)}
	a.meetingP.Clock24 = true
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Clock picker",
			Width:   880,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
