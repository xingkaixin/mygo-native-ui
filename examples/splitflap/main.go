// Splitflap is the departure board of an airport: every few seconds the
// first flight moves on from boarding to departed and leaves the board,
// the flights below move up a row, and a clock on a drum of digits ticks
// above them.
//
//	go run ./examples/splitflap
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/splitflap"
)

type flight struct {
	time, to, code, gate string
}

var schedule = []flight{
	{"08:05", "TOKYO HND", "NH 922", "B12"},
	{"08:20", "SINGAPORE", "SQ 831", "C3"},
	{"08:35", "LONDON LHR", "BA 168", "D7"},
	{"08:50", "SEOUL ICN", "KE 892", "A4"},
	{"09:10", "SYDNEY", "QF 130", "C9"},
	{"09:25", "FRANKFURT", "LH 729", "D2"},
	{"09:40", "SAN FRANCISCO", "UA 858", "B6"},
	{"10:05", "DUBAI", "EK 303", "E1"},
	{"10:30", "PARIS CDG", "AF 111", "D11"},
	{"10:45", "BANGKOK", "TG 665", "C14"},
}

// remarks are what the first flight shows as it goes; the others show
// the first.
var remarks = []string{"ON TIME", "BOARDING", "FINAL CALL", "DEPARTED"}

type app struct {
	next  int
	stage int
	last  time.Time
}

const rows = 6

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	now := c.Now()
	if a.last.IsZero() {
		a.last = now
	}
	if now.Sub(a.last) >= 4*time.Second {
		a.last = now
		a.stage++
		if a.stage == len(remarks) {
			a.stage, a.next = 0, (a.next+1)%len(schedule)
		}
	}
	c.After(time.Second - time.Duration(now.Nanosecond()))

	label := func(s string, n int) {
		ui.Text(c, s).FontSize(t.FontSize - 2).Bold().TextColor(ui.RGB(150, 150, 150)).Width(float32(n)*29.5 - 2.5)
	}
	ui.Column(c).Fill().Background(ui.RGB(14, 14, 16)).Padding(t.Space(8)).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Gap(t.Space(6)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "DEPARTURES").FontSize(28).Bold().TextColor(ui.RGB(250, 204, 21))
			splitflap.Row(c, now.Format("15:04:05"), 8, splitflap.Options{Height: 48, Drum: " 0123456789:"})
		})
		ui.Row(c).Gap(t.Space(4)).Children(func() {
			label("TIME", 5)
			label("DESTINATION", 13)
			label("FLIGHT", 6)
			label("GATE", 3)
			label("REMARKS", 10)
		})
		for i := range rows {
			f := schedule[(a.next+i)%len(schedule)]
			remark := remarks[0]
			if i == 0 {
				remark = remarks[a.stage]
			}
			ui.Row(c.Key(i)).Gap(t.Space(4)).Children(func() {
				o := splitflap.Options{Height: 40}
				splitflap.Row(c, f.time, 5, o)
				splitflap.Row(c, f.to, 13, o)
				splitflap.Row(c, f.code, 6, o)
				splitflap.Row(c, f.gate, 3, o)
				splitflap.Row(c, remark, 10, o)
			})
		}
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Departures",
			Width:   1200,
			Height:  470,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
