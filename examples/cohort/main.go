// Cohort shows twelve weeks of sign-ups and how many came back in the
// weeks after: point at a share for its counts, and click one to read it
// below. The later cohorts keep a little better, after an onboarding
// change in week 7.
//
//	go run ./examples/cohort
package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/cohort"
)

func cohorts() []cohort.Cohort {
	start := time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC)
	const weeks = 12
	out := make([]cohort.Cohort, weeks)
	for w := range weeks {
		size := 900 + (w*137)%600
		lift := 0.0
		if w >= 6 {
			lift = 0.06
		}
		kept := []int{size}
		for p := 1; p < weeks-w; p++ {
			share := 0.38*math.Exp(-float64(p)/2.5) + 0.17 + lift + 0.02*math.Sin(float64(w*3+p))
			kept = append(kept, int(float64(size)*share))
		}
		out[w] = cohort.Cohort{Name: start.AddDate(0, 0, 7*w).Format("Jan 2"), Size: size, Kept: kept}
	}
	return out
}

type app struct {
	table   cohort.State
	cohorts []cohort.Cohort
	note    string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(5)).Children(func() {
		ui.Text(c, "Weekly retention").FontSize(t.FontSize + 8).Bold()
		ui.ScrollHorizontal(c).Children(func() {
			ev := cohort.View(c, &a.table, a.cohorts, "Week")
			switch {
			case ev.Row == -1:
				a.note = fmt.Sprintf("All users, week %d", ev.Period)
			case ev.Row >= 0:
				co := a.cohorts[ev.Row]
				a.note = fmt.Sprintf("Users who signed up the week of %s: %d of %d came back in week %d", co.Name, co.Kept[ev.Period], co.Size, ev.Period)
			}
		})
		note := a.note
		if note == "" {
			note = "Click a share to read it here"
		}
		ui.Text(c, note).TextColor(t.TextMuted)
	})
}

func main() {
	a := &app{cohorts: cohorts()}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Cohort",
			Width:   1160,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
