// Dag runs a CI pipeline, as CI services show one: lint and the builds
// for three systems first, the tests and the package after the builds
// they need, then the deploys, each job starting once those it needs
// pass. The UI tests fail, so production is skipped; Run again starts
// over. Choose a job to see what it needs and how long it took.
//
//	go run ./examples/dag
package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/dag"
)

// step is a job of the pipeline: its name, the steps it needs, how long
// it runs, and whether it fails.
type step struct {
	name  string
	needs []int
	takes time.Duration
	fails bool
}

var steps = []step{
	{name: "lint", takes: 4 * time.Second},
	{name: "build (linux)", takes: 6 * time.Second},
	{name: "build (macos)", takes: 9 * time.Second},
	{name: "build (windows)", takes: 8 * time.Second},
	{name: "unit tests", needs: []int{1}, takes: 5 * time.Second},
	{name: "integration tests", needs: []int{0, 1}, takes: 7 * time.Second},
	{name: "ui tests (macos)", needs: []int{2}, takes: 6 * time.Second, fails: true},
	{name: "package", needs: []int{1, 2, 3}, takes: 3 * time.Second},
	{name: "deploy staging", needs: []int{4, 5, 7}, takes: 4 * time.Second},
	{name: "smoke tests", needs: []int{8}, takes: 3 * time.Second},
	{name: "deploy production", needs: []int{6, 9}, takes: 4 * time.Second},
}

type app struct {
	started  time.Time
	pipeline dag.State
}

// jobs returns the jobs of the pipeline as they are at now: a step starts
// once those it needs pass, takes its time, and is skipped once one of
// them fails or is skipped.
func (a *app) jobs(now time.Time) []dag.Job {
	elapsed := now.Sub(a.started)
	jobs := make([]dag.Job, len(steps))
	ends := make([]time.Duration, len(steps))
	for i, st := range steps {
		j := dag.Job{Name: st.name, Needs: st.needs}
		var start time.Duration
		skipped, waiting := false, false
		for _, k := range st.needs {
			switch jobs[k].Status {
			case dag.Failed, dag.Skipped:
				skipped = true
			case dag.Queued, dag.Running:
				waiting = true
			}
			start = max(start, ends[k])
		}
		ends[i] = start + st.takes
		switch {
		case skipped:
			j.Status, j.Detail = dag.Skipped, "Skipped"
		case waiting || elapsed < start:
			j.Status, j.Detail = dag.Queued, "Waiting"
		case elapsed < ends[i]:
			j.Status, j.Detail = dag.Running, clock(elapsed-start)
		case st.fails:
			j.Status, j.Detail = dag.Failed, "Failed in "+clock(st.takes)
		default:
			j.Status, j.Detail = dag.Passed, clock(st.takes)
		}
		jobs[i] = j
	}
	return jobs
}

// clock formats d of the pipeline, a second of which shows as ten.
func clock(d time.Duration) string {
	s := int(d.Seconds() * 10)
	return fmt.Sprintf("%dm %02ds", s/60, s%60)
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	jobs := a.jobs(c.Now())
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Gap(t.Space(3)).Padding(t.Space(3), t.Space(4)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "CI · main · 4f2a9c1").Bold().Grow(1)
			if ui.Button(c, "Run again").Clicked() {
				a.started = c.Now()
			}
		})
		ui.Divider(c)
		dag.View(c, &a.pipeline, jobs).Grow(1).Background(t.Surface)
		ui.Divider(c)
		ui.Row(c).Height(t.Space(12)).Gap(t.Space(4)).Padding(0, t.Space(4)).AlignItems(ui.Center).Children(func() {
			i, ok := a.pipeline.Chosen()
			if !ok {
				ui.Text(c, "Choose a job").TextColor(t.TextMuted)
				return
			}
			ui.Text(c, jobs[i].Name).Bold()
			ui.Text(c, jobs[i].Detail).FontFeatures("tnum").TextColor(t.TextMuted)
			var needs []string
			for _, k := range jobs[i].Needs {
				needs = append(needs, jobs[k].Name)
			}
			if len(needs) > 0 {
				ui.Text(c, "Needs "+strings.Join(needs, ", ")).TextColor(t.TextMuted).Shrink(1)
			}
		})
	})
}

func main() {
	a := &app{started: time.Now()}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Pipeline",
			Width:   1180,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
