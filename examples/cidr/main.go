// Cidr plans the subnets of a cloud network: Divide and Join the subnets,
// click the bar to choose one, and type another network to start again.
// The red band is a range the office network already uses.
//
//	go run ./examples/cidr
package main

import (
	"log"
	"net/netip"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/cidr"
)

type app struct {
	plan    cidr.Plan
	planner cidr.State
	typed   string
	wrong   bool
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(5)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Network").Bold()
			field := ui.TextInput(c, &a.typed).Width(t.Space(60)).Font("monospace").Label("Network")
			if a.wrong {
				field.Error("Type a network, as 10.0.0.0/16 or 2001:db8::/48")
			}
			if field.Submitted() {
				p, err := netip.ParsePrefix(a.typed)
				a.wrong = err != nil
				if err == nil {
					a.plan.Network, a.plan.Subnets = p.Masked(), nil
					a.typed = a.plan.Network.String()
				}
			}
			ui.Spacer(c)
			ui.Text(c, "Office network 10.20.80.0/20").FontSize(t.FontSize - 1).TextColor(t.Danger)
		})
		cidr.View(c, &a.planner, &a.plan).Grow(1)
	})
}

func main() {
	p := netip.MustParsePrefix
	a := &app{
		plan: cidr.Plan{
			Network: p("10.20.0.0/16"),
			Subnets: []netip.Prefix{p("10.20.0.0/18"), p("10.20.64.0/19"), p("10.20.96.0/19"), p("10.20.128.0/17")},
			Taken:   []netip.Prefix{p("10.20.80.0/20")},
		},
		typed: "10.20.0.0/16",
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Cidr",
			Width:   980,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
