// Notifications shows a made-up project tool's inbox: mentions, a review
// asked for with Approve and Decline, a failed build, a deploy and
// invitations, from the last few days. Click one to open it, hover it to
// mark it read or dismiss it, and choose Unread to see what is left.
//
//	go run ./examples/notifications
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/notifications"
)

func inbox(now time.Time) []notifications.Item {
	ago := func(d time.Duration) time.Time { return now.Add(-d) }
	return []notifications.Item{
		{Source: "Ada Lovelace", Title: "Ada mentioned you in Engine notes", Body: "@you can we move the bearing tolerances to the appendix? The table is getting long.", Time: ago(4 * time.Minute)},
		{Source: "Alan Turing", Title: "Review requested: Decode faster with lookup tables", Body: "#482 changes 6 files: +214 −87", Time: ago(38 * time.Minute), Actions: []string{"Approve", "Decline"}},
		{Source: "CI", Title: "Build failed on main", Body: "go test ./... failed in ui/parser: TestNested timed out after 10m.", Time: ago(2 * time.Hour)},
		{Source: "Grace Hopper", Title: "Grace invited you to Compiler guild", Body: "Weekly on Thursdays at 4 PM.", Time: ago(26 * time.Hour), Actions: []string{"Accept", "Ignore"}},
		{Source: "Deploys", Title: "Production deployed v2.4.0", Body: "Rolled out to 3 regions in 6 minutes.", Time: ago(29 * time.Hour), Read: true},
		{Source: "Edsger Dijkstra", Title: "Edsger replied to your comment", Body: "Goto considered harmful, as I said. Use a loop.", Time: ago(3 * 24 * time.Hour), Read: true},
		{Source: "Billing", Title: "Your invoice for September is ready", Time: ago(9 * 24 * time.Hour), Read: true},
	}
}

type app struct {
	items  []notifications.Item
	inbox  notifications.State
	status string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	if a.items == nil {
		a.items = inbox(c.Now())
	}
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ev := notifications.View(c, &a.inbox, &a.items)
		if i, ok := ev.Opened(); ok {
			a.status = "Opened: " + a.items[i].Title
		}
		if i, action, ok := ev.Action(); ok {
			a.status = action + ": " + a.items[i].Title
			a.items[i].Read, a.items[i].Actions = true, nil
		}
		ui.Divider(c)
		ui.Text(c, a.status).TextColor(t.TextMuted).SingleLine().Padding(t.Space(2), t.Space(4)).Height(t.Space(9))
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Notifications",
			Width:   440,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
