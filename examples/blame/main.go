// Blame shows who last changed each line of a file, from git blame, by
// default README.md of this repository. The commit chosen
// shows below, with how many of the file's lines are its.
//
//	go run ./examples/blame [file]
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/blame"
)

// app is the state the window shows: the file, its lines and commits, and
// the commit chosen, of which shown is the one counted.
type app struct {
	path    string
	lines   []blame.Line
	commits int
	blame   blame.State
	chosen  string
	shown   string
	count   int
	commit  *blame.Commit
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Padding(t.Space(2), t.Space(3)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, a.path).Bold().SingleLine().Shrink(1)
			ui.Textf(c, "%d lines · %d commits", len(a.lines), a.commits).TextColor(t.TextMuted)
		})
		ui.Divider(c)
		blame.View(c, &a.blame, a.path, a.lines, &a.chosen).Grow(1)
		if a.chosen != a.shown {
			a.shown, a.count, a.commit = a.chosen, 0, nil
			for _, l := range a.lines {
				if l.Commit.Hash == a.chosen {
					a.count, a.commit = a.count+1, l.Commit
				}
			}
		}
		ui.Divider(c)
		ui.Row(c).Padding(t.Space(1.5), t.Space(3)).Gap(t.Space(2)).FontSize(t.FontSize - 1).Children(func() {
			if a.commit == nil {
				ui.Text(c, "Click a line to choose its commit.").TextColor(t.TextMuted)
				return
			}
			ui.Text(c, a.commit.Hash[:7]).Font("monospace").TextColor(t.Accent)
			ui.Text(c, a.commit.Summary).SingleLine().Shrink(1)
			ui.Spacer(c)
			ui.Textf(c, "%s · %d of %d lines", a.commit.Author, a.count, len(a.lines)).TextColor(t.TextMuted).SingleLine()
		})
	})
}

// git runs git with args in dir and returns what it printed.
func git(dir string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		log.Printf("git %s: %v", strings.Join(args, " "), err)
	}
	return string(out)
}

// messages fills in the messages of the commits of lines, which git blame
// leaves out, with one git show.
func messages(dir string, lines []blame.Line) int {
	commits := map[string]*blame.Commit{}
	for _, l := range lines {
		commits[l.Commit.Hash] = l.Commit
	}
	args := []string{"show", "-s", "--format=%H%x00%B%x00"}
	for h := range commits {
		// Lines not committed yet have a hash of zeros, which git show
		// does not know.
		if strings.Trim(h, "0") != "" {
			args = append(args, h)
		}
	}
	if len(args) > 3 {
		f := strings.Split(git(dir, args...), "\x00")
		for i := 0; i+1 < len(f); i += 2 {
			if cm := commits[strings.TrimSpace(f[i])]; cm != nil {
				cm.Message = f[i+1]
			}
		}
	}
	return len(commits)
}

func main() {
	path := "README.md"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	dir, name := filepath.Split(path)
	if dir == "" {
		dir = "."
	}
	a := &app{path: path}
	a.lines = blame.Parse(git(dir, "blame", "--porcelain", "--", name))
	if len(a.lines) == 0 {
		log.Fatalf("no lines to blame in %s", path)
	}
	a.commits = messages(dir, a.lines)
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   fmt.Sprintf("Blame — %s", filepath.Base(path)),
			Width:   1180,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
