// Mailthread shows a made-up thread of six messages planning a team's
// offsite, the last two unread.
//
//	go run ./examples/mailthread
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/mailthread"
)

func thread(now time.Time) []mailthread.Message {
	day := func(back int, h, m int) time.Time {
		d := now.AddDate(0, 0, -back)
		return time.Date(d.Year(), d.Month(), d.Day(), h, m, 0, 0, d.Location())
	}
	return []mailthread.Message{
		{From: "Priya Raman", To: []string{"Design team"}, Date: day(6, 9, 12), Body: `Hi all,

The offsite is booked for the second week of next month at the lake house. Two days, Thursday and Friday.

Could everyone send me one topic they want on the agenda by Monday?

Thanks,
Priya`},
		{From: "Marco Ruiz", To: []string{"Priya Raman"}, Cc: []string{"Design team"}, Date: day(5, 14, 40), Body: `I'd like half a day on the design system: what we keep, what we retire, who owns it.

On Thursday, Priya Raman wrote:
> Could everyone send me one topic they want on the agenda by Monday?`},
		{From: "Hana Sato", To: []string{"Priya Raman"}, Date: day(4, 10, 5), Body: `Research readout from the onboarding study, 45 minutes. I'll bring the clips.

On Thursday, Priya Raman wrote:
> Could everyone send me one topic they want on the agenda by Monday?`, Attached: []string{"Onboarding study.pdf"}},
		{From: "Dev Okafor", To: []string{"Priya Raman"}, Date: day(3, 16, 22), Body: `Not a topic, but could we keep Friday afternoon free? Half the team has a long drive back.`},
		{From: "Priya Raman", To: []string{"Design team"}, Date: day(0, 8, 47), Unread: true, Body: `Here's the draft agenda. Design system Thursday morning, research readout after lunch, and Friday wraps at one.

Shout if anything clashes.`, Attached: []string{"Offsite agenda.pdf", "Lake house directions.png"}},
		{From: "Marco Ruiz", To: []string{"Design team"}, Date: day(0, 9, 30), Unread: true, Body: `Looks great. I'll book the minibus for Thursday at eight.

On Wednesday, Priya Raman wrote:
> Here's the draft agenda. Design system Thursday morning, research readout after lunch, and Friday wraps at one.
>
> Shout if anything clashes.`},
	}
}

type app struct {
	msgs   []mailthread.Message
	thread mailthread.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Padding(t.Space(6)).Children(func() {
			mailthread.View(c, &a.thread, "Team offsite: agenda", a.msgs)
		})
	})
}

func main() {
	a := &app{msgs: thread(time.Now())}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Mail thread",
			Width:   760,
			Height:  820,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
