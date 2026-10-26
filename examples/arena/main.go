// Arena runs rounds of a blind comparison: read the prompt and the two
// answers, vote with the buttons or the keys 1 to 4, see which models
// they were, and go on to the next round. The tally counts the wins.
//
//	go run ./examples/arena
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/arena"
)

type round struct {
	prompt  string
	models  [2]string
	answers [2]string
	vote    arena.Vote
}

type app struct {
	rounds []round
	at     int
	wins   map[string]int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	r := &a.rounds[a.at]
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(5)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, fmt.Sprintf("Round %d of %d", a.at+1, len(a.rounds))).TextColor(t.TextMuted).Grow(1)
			for _, m := range []string{"Atlas", "Borealis", "Cirrus"} {
				ui.Text(c, fmt.Sprintf("%s %d", m, a.wins[m])).FontFeatures("tnum").TextColor(t.TextMuted)
			}
		})
		ui.Text(c, r.prompt).FontSize(t.FontSize + 3).Bold()
		ui.Scroll(c).Grow(1).Children(func() {
			if arena.View(c.Key(a.at), &r.vote, r.models, func() {
				ui.Text(c, r.answers[0]).Selectable()
			}, func() {
				ui.Text(c, r.answers[1]).Selectable()
			}).Changed() {
				switch r.vote {
				case arena.A:
					a.wins[r.models[0]]++
				case arena.B:
					a.wins[r.models[1]]++
				}
			}
		})
		if r.vote != arena.None && a.at+1 < len(a.rounds) {
			ui.Row(c).Justify(ui.End).Children(func() {
				if ui.PrimaryButton(c, "Next round").AutoFocus().Clicked() {
					a.at++
				}
			})
		}
	})
}

func main() {
	a := &app{
		wins: map[string]int{},
		rounds: []round{
			{
				prompt: "Explain what a mutex is to someone new to programming.",
				models: [2]string{"Borealis", "Atlas"},
				answers: [2]string{
					"A mutex is a lock for data that several parts of a program use at once. Before a part touches the data, it takes the lock; when it is done, it gives it back. While one part holds it, the others wait their turn, so no two change the data at the same moment and leave it half-written.",
					"Mutex stands for mutual exclusion. It is a synchronization primitive that guarantees at most one thread executes a critical section at a time, typically implemented with atomic compare-and-swap operations and a wait queue managed by the scheduler.",
				},
			},
			{
				prompt: "Write a subject line for an email moving Thursday's team meeting to Friday.",
				models: [2]string{"Cirrus", "Borealis"},
				answers: [2]string{
					"Team meeting moved: Thursday → Friday, same time",
					"Important update regarding our upcoming weekly team synchronization meeting schedule",
				},
			},
			{
				prompt: "Is 1,001 a prime number?",
				models: [2]string{"Atlas", "Cirrus"},
				answers: [2]string{
					"No. 1,001 = 7 × 11 × 13.",
					"No. 1,001 is divisible by 7: 7 × 143 = 1,001, and 143 = 11 × 13.",
				},
			},
		},
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Arena",
			Width:   900,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
