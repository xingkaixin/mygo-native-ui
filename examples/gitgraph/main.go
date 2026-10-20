// Gitgraph shows the history of the Git repository given, or of the one
// the current directory is in, as a graph of its branches; the commit
// chosen shows its message and the files it changed.
//
//	go run ./examples/gitgraph [repository]
package main

import (
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/gitgraph"
)

type app struct {
	dir     string
	commits []gitgraph.Commit
	graph   gitgraph.State
	split   float32
	// shown is the hash of the commit whose message and files show.
	shown   string
	message string
	files   []string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	commit, ok := a.graph.Chosen()
	if ok && commit.Hash != a.shown {
		a.shown = commit.Hash
		a.message = git(a.dir, "show", "-s", "--format=%B", commit.Hash)
		a.files = strings.Split(strings.TrimSpace(git(a.dir, "show", "--name-status", "--format=", commit.Hash)), "\n")
	}
	ui.Split(c, &a.split, func() {
		gitgraph.View(c, &a.graph, a.commits).Fill()
	}, func() {
		ui.Scroll(c).Fill().Background(t.Surface).Children(func() {
			ui.Column(c).Gap(t.Space(3)).Padding(t.Space(4)).Children(func() {
				if !ok {
					ui.Text(c, "Choose a commit to see its message and files.").TextColor(t.TextMuted)
					return
				}
				ui.Text(c, strings.TrimSpace(a.message)).Selectable()
				ui.Column(c).Gap(t.Space(1)).Children(func() {
					meta(c, "Author", commit.Author)
					meta(c, "Date", commit.Time.Format("Mon Jan 2 15:04:05 2006"))
					meta(c, "Commit", commit.Hash)
					meta(c, "Parents", strings.Join(commit.Parents, "\n"))
				})
				ui.Divider(c)
				for i, f := range a.files {
					ui.Text(c.Key(i), f).Font("monospace").FontSize(t.FontSize - 1).SingleLine()
				}
			})
		})
	}).Fill().Background(t.Background)
}

// meta shows name and value in a row.
func meta(c *ui.Context, name, value string) {
	t := c.Theme()
	ui.Row(c).Gap(t.Space(3)).Children(func() {
		ui.Text(c, name).TextColor(t.TextMuted).Width(t.Space(16))
		ui.Text(c, value).Font("monospace").FontSize(t.FontSize - 1).Selectable().Shrink(1)
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

func main() {
	dir := "."
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	a := &app{dir: dir, split: 780}
	a.commits = gitgraph.Parse(git(dir, "log", "--all", "--topo-order", "--format="+gitgraph.Format))
	if len(a.commits) == 0 {
		log.Fatalf("no commits in %s", dir)
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Git Graph",
			Width:   1180,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
