// Currency shows an expense form for a trip: an amount in the currency
// paid, and the expenses so far, each in its own currency. Type or paste
// an amount, choose its currency, and add it.
//
//	go run ./examples/currency
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/currency"
)

type expense struct {
	what   string
	amount int64
	code   string
}

type app struct {
	field    currency.State
	amount   int64
	code     string
	what     string
	expenses []expense
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(5)).Children(func() {
		ui.Text(c, "Lisbon offsite").FontSize(t.FontSize + 8).Bold()
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.End).Children(func() {
			ui.Column(c).Gap(t.Space(1.5)).Width(t.Space(64)).Children(func() {
				ui.Text(c, "Amount").FontSize(t.FontSize - 1).TextColor(t.TextMuted)
				currency.Field(c, &a.field, &a.amount, &a.code)
			})
			ui.Column(c).Gap(t.Space(1.5)).Grow(1).Children(func() {
				ui.Text(c, "For").FontSize(t.FontSize - 1).TextColor(t.TextMuted)
				ui.TextInput(c, &a.what).Placeholder("Dinner, taxi, tickets")
			})
			add := ui.Button(c, "Add")
			if add.Clicked() && a.amount > 0 {
				what := a.what
				if what == "" {
					what = "Expense"
				}
				a.expenses = append([]expense{{what, a.amount, a.code}}, a.expenses...)
				a.amount, a.what = 0, ""
			}
		})
		ui.Column(c).Gap(t.Space(0.5)).Children(func() {
			for i, e := range a.expenses {
				ui.Row(c.Key(i)).Padding(t.Space(2), 0).Gap(t.Space(3)).AlignItems(ui.Baseline).Children(func() {
					ui.Text(c, e.what).Grow(1)
					ui.Text(c, e.code).TextColor(t.TextMuted).FontSize(t.FontSize - 1)
					ui.Text(c, currency.Format(e.amount, e.code)).FontFeatures("tnum").TextAlign(ui.End).Width(t.Space(30))
				}).BorderWidth(0, 0, 1, 0).BorderColor(t.Border)
			}
		})
	})
}

func main() {
	a := &app{code: "EUR", expenses: []expense{
		{"Hotel, 3 nights", 64200, "EUR"},
		{"Flights from Tokyo", 184500, "JPY"},
		{"Team dinner", 31875, "EUR"},
		{"Conference passes", 120000, "USD"},
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Currency",
			Width:   760,
			Height:  520,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
