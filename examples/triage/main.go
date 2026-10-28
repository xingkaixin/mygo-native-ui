// Triage sorts a support inbox: each ticket in turn, labeled with a click
// or the keys 1 to 5.
//
//	go run ./examples/triage
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/triage"
)

type ticket struct {
	from, subject, body string
}

var tickets = []ticket{
	{"Dana Whitfield", "Export to CSV drops the last row", "When I export a report with more than 100 rows, the CSV ends one row short. The on-screen table has all of them."},
	{"Marcus Lee", "Dark mode for the dashboard?", "Our team works late and the dashboard is very bright. Is a dark theme on the roadmap?"},
	{"Priya Raman", "Charged twice this month", "My card shows two charges of $49 on the 3rd. Could you refund the duplicate?"},
	{"Tom Becker", "How do I invite a guest?", "I want to share one project with a contractor without giving access to everything else."},
	{"WIN-BIG Promotions", "You have been selected!!!", "Claim your prize today. Click the link below to verify your account details."},
	{"Lena Ortiz", "App crashes when I rotate my phone", "On the iOS app, rotating to landscape on the timeline screen closes the app every time."},
	{"Hiro Tanaka", "Keyboard shortcuts list", "Is there a page listing every shortcut? I keep finding new ones by accident."},
	{"Chloe Martin", "Annual plan discount", "If we switch from monthly to annual billing, how much do we save, and is it prorated?"},
	{"Sam Okafor", "Webhooks for status changes", "We would love a webhook when a task moves between columns so we can post to our chat."},
	{"Ada Novak", "Search ignores accented names", "Searching for 'Jose' does not find 'José'. It used to before last week's update."},
	{"Ruth Klein", "Invoice needs our VAT number", "Our accounting team needs the VAT ID on invoices. Where can I add it?"},
	{"Felix Braun", "Bulk edit due dates", "Could we select several tasks and move their due dates at once?"},
}

var labels = []string{"Bug", "Feature request", "Question", "Billing", "Spam"}

type app struct {
	queue   triage.State
	answers []int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(t.Space(8)).Background(t.Background).Children(func() {
		triage.View(c, &a.queue, labels, &a.answers, func(i int) {
			tk := tickets[i]
			ui.Text(c, tk.from).TextColor(t.TextMuted).FontSize(t.FontSize - 1)
			ui.Text(c, tk.subject).FontSize(t.FontSize + 4).Bold()
			ui.Text(c, tk.body).LineHeight(1.5).Margin(t.Space(2), 0, 0, 0)
		}).Grow(1)
	})
}

func main() {
	a := &app{answers: make([]int, len(tickets))}
	for i := range a.answers {
		a.answers[i] = -1
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Triage",
			Width:   760,
			Height:  480,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
