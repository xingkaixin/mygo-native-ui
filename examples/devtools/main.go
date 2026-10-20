// Devtools is a developer's toolbox: a regular expression tester, a cron
// schedule field, and the activity of a Git repository by day over the
// last year, its commits of a day chosen listed below it.
//
//	go run ./examples/devtools [repository]
package main

import (
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/cron"
	"github.com/xingkaixin/mygo-native-ui/ui/heatmap"
	"github.com/xingkaixin/mygo-native-ui/ui/regex"
)

// commit is a commit of the repository: when it was made, its hash and
// its subject.
type commit struct {
	at      time.Time
	hash    string
	subject string
}

type app struct {
	tab      int
	tester   regex.State
	pattern  string
	sample   string
	schedule string
	repo     string
	commits  []commit
	counts   map[heatmap.Date]int
	activity heatmap.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Padding(t.Space(2), t.Space(3)).Children(func() {
			ui.Segmented(c, &a.tab, "Regex", "Cron", "Activity")
		})
		ui.Divider(c)
		ui.Column(c).Grow(1).Padding(t.Space(5)).Children(func() {
			switch a.tab {
			case 0:
				regex.View(c, &a.tester, &a.pattern, &a.sample).Grow(1)
			case 1:
				cron.Field(c, &a.schedule).MaxWidth(t.Space(160))
			case 2:
				a.history(c)
			}
		})
	})
}

// history shows the repository's commits by day, and those of the day
// chosen.
func (a *app) history(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(4)).Grow(1).Children(func() {
		ui.Text(c, a.repo).TextColor(t.TextMuted).FontSize(t.FontSize - 1)
		heatmap.View(c, &a.activity, a.counts, c.Now(), "commit")
		day, ok := a.activity.Chosen()
		if !ok || a.counts[day] == 0 {
			return
		}
		ui.Divider(c)
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Gap(t.Space(1.5)).Children(func() {
				for i, cm := range a.commits {
					if heatmap.DateOf(cm.at) != day {
						continue
					}
					ui.Row(c.Key(i)).Gap(t.Space(3)).Children(func() {
						ui.Text(c, cm.at.Format("15:04")).FontFeatures("tnum").TextColor(t.TextMuted)
						ui.Text(c, cm.hash).Font("monospace").TextColor(t.TextMuted)
						ui.Text(c, cm.subject).SingleLine().Shrink(1)
					})
				}
			})
		})
	})
}

// load reads the commits of the repository in dir.
func load(dir string) ([]commit, error) {
	out, err := exec.Command("git", "-C", dir, "log", "--all", "--format=%at%x1f%h%x1f%s").Output()
	if err != nil {
		return nil, err
	}
	var commits []commit
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		f := strings.Split(line, "\x1f")
		if len(f) < 3 {
			continue
		}
		at, _ := strconv.ParseInt(f[0], 10, 64)
		commits = append(commits, commit{time.Unix(at, 0), f[1], f[2]})
	}
	return commits, nil
}

func main() {
	dir := "."
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	commits, err := load(dir)
	if err != nil {
		log.Fatal(err)
	}
	a := &app{
		pattern:  `(?i)(?P<user>[\w.+-]+)@(?P<domain>[\w-]+(?:\.[\w-]+)+)`,
		sample:   "Write to ada@example.com or Grace.Hopper@navy.mil.\nNot an address: me@localhost\nBob+news@mail.co.uk replied.",
		schedule: "*/15 9-17 * * MON-FRI",
		repo:     dir,
		commits:  commits,
		counts:   map[heatmap.Date]int{},
	}
	if abs, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output(); err == nil {
		a.repo = strings.TrimSpace(string(abs))
	}
	for _, cm := range commits {
		a.counts[heatmap.DateOf(cm.at)]++
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Devtools",
			Width:   900,
			Height:  620,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
