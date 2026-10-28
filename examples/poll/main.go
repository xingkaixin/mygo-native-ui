// Poll shows three polls posted in a team's channel: one to vote on with
// a click, one where several options may be ticked, and one closed. Vote,
// and retract the vote to choose again.
//
//	go run ./examples/poll
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/poll"
)

type post struct {
	author, time string
	card         poll.State
	poll         poll.Poll
}

type app struct {
	posts []*post
	log   string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(4)).Children(func() {
		ui.Text(c, "# offsite-planning").FontSize(t.FontSize + 6).Bold()
		ui.Row(c).Gap(t.Space(6)).AlignItems(ui.Start).Children(func() {
			for col := range 2 {
				ui.Column(c.Key(col)).Gap(t.Space(5)).Width(t.Space(96)).Children(func() {
					for i, p := range a.posts {
						if i%2 != col {
							continue
						}
						ui.Column(c.Key(i)).Gap(t.Space(1.5)).Children(func() {
							ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Baseline).Children(func() {
								ui.Text(c, p.author).Bold().FontSize(t.FontSize - 1)
								ui.Text(c, p.time).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
							})
							ev := poll.View(c, &p.card, &p.poll)
							switch {
							case ev.Voted:
								a.log = "Sent your vote on “" + p.poll.Question + "”"
							case ev.Retracted:
								a.log = "Took back your vote on “" + p.poll.Question + "”"
							}
						})
					}
				})
			}
		})
		ui.Spacer(c)
		ui.Text(c, a.log).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
	})
}

func main() {
	a := &app{posts: []*post{
		{author: "Maya Chen", time: "10:42", poll: poll.Poll{
			Question: "Where should we hold the spring offsite?",
			Options:  []poll.Option{{Text: "Lisbon", Votes: 9}, {Text: "Kyoto", Votes: 12}, {Text: "Mexico City", Votes: 5}, {Text: "Stay home, ship more", Votes: 2}},
		}},
		{author: "Jonas Weber", time: "11:05", poll: poll.Poll{
			Question: "Which sessions would you join?",
			Multiple: true,
			Options:  []poll.Option{{Text: "Roadmap review", Votes: 14}, {Text: "Design critique", Votes: 9}, {Text: "On-call retro", Votes: 6}, {Text: "Hackathon", Votes: 17}},
		}},
		{author: "Priya Nair", time: "Yesterday", poll: poll.Poll{
			Question: "Offsite dates?",
			Closed:   true,
			Mine:     []int{1},
			Options:  []poll.Option{{Text: "April 14–16", Votes: 8}, {Text: "April 21–23", Votes: 15}, {Text: "May 5–7", Votes: 6}},
		}},
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Poll",
			Width:   900,
			Height:  700,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
