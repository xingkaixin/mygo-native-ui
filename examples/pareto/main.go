// Pareto ranks the reasons for a quarter's 1,200 support tickets: four
// of ten make up 80% of them. Point at a bar, or press Tab and the
// arrows, to read its share.
//
//	go run ./examples/pareto
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/pareto"
)

var tickets = []pareto.Cause{
	{Name: "Billing", Count: 118},
	{Name: "Password reset", Count: 412},
	{Name: "Sync conflict", Count: 61},
	{Name: "Slow export", Count: 236},
	{Name: "Login loop", Count: 197},
	{Name: "Missing invoice", Count: 44},
	{Name: "Plugin crash", Count: 38},
	{Name: "Wrong timezone", Count: 52},
	{Name: "Sharing", Count: 27},
	{Name: "Other", Count: 15},
}

func view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(4)).Children(func() {
		ui.Text(c, "Support tickets, third quarter").FontSize(t.FontSize + 4).Bold()
		pareto.View(c, tickets).Grow(1)
	})
}

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Pareto",
			Width:   900,
			Height:  560,
			Content: ui.View(view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
