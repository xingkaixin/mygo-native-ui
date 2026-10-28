// Scheduler books six meeting rooms from 8 to 20 o'clock. Drag along an
// empty stretch of a room to book it, drag a booking to move it, or drag
// its ends to change its times. Two bookings in Orion overlap, in red.
// Choose a booking to rename it, or press Delete to remove it.
//
//	go run ./examples/scheduler
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/scheduler"
)

var rooms = []string{"Andromeda", "Orion", "Lyra", "Cygnus", "Phone booth 1", "Phone booth 2"}

type app struct {
	sched    scheduler.State
	bookings []scheduler.Booking
}

func at(h, m int) time.Duration { return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute }

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
		ui.Text(c, "Wednesday, October 28").FontSize(t.FontSize + 4).Bold()
		scheduler.View(c, &a.sched, &a.bookings, rooms, 8, 20)
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			i := a.sched.Chosen()
			if i < 0 {
				ui.Text(c, "Choose a booking to rename it.").TextColor(t.TextMuted)
				return
			}
			b := &a.bookings[i]
			ui.Text(c, rooms[b.Resource]).Bold()
			ui.TextInput(c, &b.Title).Placeholder("Title").Width(t.Space(70))
		})
	})
}

func main() {
	a := &app{bookings: []scheduler.Booking{
		{Title: "Design review", Resource: 0, Start: at(9, 0), End: at(10, 30)},
		{Title: "All hands", Resource: 0, Start: at(13, 0), End: at(14, 0)},
		{Title: "Planning", Resource: 1, Start: at(10, 0), End: at(11, 30)},
		{Title: "Interview", Resource: 1, Start: at(11, 0), End: at(12, 0)},
		{Title: "1:1", Resource: 1, Start: at(15, 30), End: at(16, 0)},
		{Title: "Workshop", Resource: 2, Start: at(9, 30), End: at(12, 30)},
		{Title: "Retro", Resource: 3, Start: at(16, 0), End: at(17, 0)},
		{Title: "Call with Lisbon", Resource: 4, Start: at(8, 30), End: at(9, 15)},
		{Title: "Vendor call", Resource: 5, Start: at(14, 0), End: at(14, 45)},
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Scheduler",
			Width:   960,
			Height:  520,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
