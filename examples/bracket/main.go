// Bracket follows a sixteen-team cup through its rounds: the first two
// are played, one semifinal is over and the other live. Point at a team
// to see its way, click a match for its details, and Next Result plays
// the rest one match at a time.
//
//	go run ./examples/bracket
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/bracket"
)

var teams = []string{
	"Harbor City", "Westbrook", "Northgate", "Pine Valley", "Riverside", "Ashford", "Kingsport", "Millbrook",
	"Lakewood", "Stonebridge", "Fairhaven", "Redcliff", "Oakmont", "Bayview", "Eastfield", "Glenwood",
}

// seeds pairs the first seed with the last, as cups draw.
var seeds = []int{1, 16, 8, 9, 5, 12, 4, 13, 3, 14, 6, 11, 7, 10, 2, 15}

// scores are the results of the matches played, in order, the goals of
// the top side and the bottom.
var scores = [][2]int{
	{3, 0}, {1, 2}, {2, 1}, {0, 1}, {4, 2}, {1, 0}, {2, 2}, {0, 3},
	{2, 1}, {1, 3}, {3, 1}, {0, 2},
	{2, 0}, {1, 1},
	{3, 2},
}

type app struct {
	played int
	rounds []bracket.Round
	chosen [2]int
}

// build makes the rounds with the first n matches played, the one after
// them live.
func build(n int) []bracket.Round {
	names := []string{"Round of 16", "Quarterfinals", "Semifinals", "Final"}
	times := []string{"", "", "Sat 18:00", "Sun 20:00"}
	var rounds []bracket.Round
	var prev []bracket.Side
	for _, s := range seeds {
		prev = append(prev, bracket.Side{Name: teams[s-1], Seed: s})
	}
	k := 0
	for r, name := range names {
		var round bracket.Round
		round.Name = name
		var winners []bracket.Side
		for m := 0; m < len(prev)/2; m++ {
			top, bottom := prev[2*m], prev[2*m+1]
			top.Score, top.Won, bottom.Score, bottom.Won = "", false, "", false
			mt := bracket.Match{Top: top, Bottom: bottom, Time: times[r]}
			var w bracket.Side
			switch {
			case k < n:
				g := scores[k]
				mt.Top.Score, mt.Bottom.Score = fmt.Sprint(g[0]), fmt.Sprint(g[1])
				if g[0] == g[1] {
					// A draw goes to penalties, which the top side wins.
					mt.Top.Score += " (5)"
					mt.Bottom.Score += " (4)"
				}
				if g[0] >= g[1] {
					mt.Top.Won, w = true, mt.Top
				} else {
					mt.Bottom.Won, w = true, mt.Bottom
				}
			case k == n && top.Name != "" && bottom.Name != "":
				mt.Live = true
				mt.Top.Score, mt.Bottom.Score = "0", "0"
			}
			w.Score, w.Won = "", false
			winners = append(winners, w)
			round.Matches = append(round.Matches, mt)
			k++
		}
		rounds = append(rounds, round)
		prev = winners
	}
	return rounds
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	a.rounds = build(a.played)
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "City Cup").FontSize(t.FontSize + 8).Bold().Grow(1)
			if ui.Button(c, "Next Result").Disabled(a.played >= len(scores)).Clicked() {
				a.played++
			}
			if ui.Button(c, "Reset").Clicked() {
				a.played = 13
			}
		})
		ui.ScrollBoth(c).Grow(1).Children(func() {
			if ev := bracket.View(c, a.rounds); ev.Round >= 0 {
				a.chosen = [2]int{ev.Round, ev.Match}
			}
		})
		mt := a.rounds[a.chosen[0]].Matches[a.chosen[1]]
		detail := fmt.Sprintf("%s, match %d: %s v %s", a.rounds[a.chosen[0]].Name, a.chosen[1]+1, name(mt.Top), name(mt.Bottom))
		switch {
		case mt.Live:
			detail += ", being played"
		case mt.Top.Score != "":
			detail += fmt.Sprintf(", %s–%s", mt.Top.Score, mt.Bottom.Score)
		case mt.Time != "":
			detail += ", " + mt.Time
		}
		ui.Text(c, detail).TextColor(t.TextMuted)
	})
}

func name(s bracket.Side) string {
	if s.Name == "" {
		return "TBD"
	}
	return s.Name
}

func main() {
	a := &app{played: 13, chosen: [2]int{2, 1}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Bracket",
			Width:   1220,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
