// Promptvars edits a prompt that summarizes a support ticket for the
// engineer on call: the template with its four variables, three of them
// filled in from a ticket, the tone still without a value, and the
// prompt they make.
//
//	go run ./examples/promptvars
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/promptvars"
)

const template = `You are a support lead. Summarize the ticket below for the engineer on call.

Customer: {{customer_name}}
Product area: {{product}}

<ticket>
{{ticket}}
</ticket>

Write three bullet points: what the customer reported, what they already tried, and what they need next. Keep it under 80 words and use a {{tone}} tone.`

type app struct {
	template string
	values   map[string]string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(5)).Children(func() {
		ui.Column(c).Gap(t.Space(1)).Children(func() {
			ui.Text(c, "Summarize a support ticket").Bold().FontSize(t.FontSize + 4)
			ui.Text(c, "Prompt for the on-call digest, sent to the model with each new ticket.").TextColor(t.TextMuted)
		})
		promptvars.View(c, &a.template, &a.values).Grow(1)
	})
}

func main() {
	a := &app{
		template: template,
		values: map[string]string{
			"customer_name": "Priya Raman, Northwind Logistics",
			"product":       "Billing · invoice exports",
			"ticket": `Since Monday our monthly invoice export to CSV stops at 1,000 rows. We have 3,412 invoices for September.

I tried exporting from Chrome and Safari, and splitting the date range into weeks, which works but takes an hour each month. Our accountant needs the full file by Friday for the quarterly close.`,
		},
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Prompt variables",
			Width:   1200,
			Height:  780,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
