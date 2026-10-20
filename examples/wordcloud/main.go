// Wordcloud counts the words of a repository's commit messages, this one
// unless given another, and lists the commits of the word chosen.
//
//	go run ./examples/wordcloud [repository]
package main

import (
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"unicode"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/wordcloud"
)

var stop = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`the and for with from into that this its are was were be been
		has have had not but all any can will when then than them they their there here also only
		via per out off over more most some such use used uses using one two new old`) {
		stop[w] = true
	}
}

// words returns the words of subject, three letters or more and not stop
// words; the scope of "feat(scope): Subject" is one too.
func words(subject string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.FieldsFunc(strings.ToLower(subject), func(r rune) bool { return !unicode.IsLetter(r) }) {
		if len(w) >= 3 && !stop[w] {
			out[w] = true
		}
	}
	return out
}

// count returns the words of subjects with how many subjects have each.
func count(subjects []string) []wordcloud.Word {
	n := map[string]int{}
	for _, s := range subjects {
		for w := range words(s) {
			n[w]++
		}
	}
	var out []wordcloud.Word
	for w, k := range n {
		out = append(out, wordcloud.Word{Text: w, Count: k})
	}
	return out
}

type app struct {
	repo     string
	subjects []string
	words    []wordcloud.Word
	cloud    wordcloud.State
	word     string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
		ui.Column(c).Gap(t.Space(1)).Children(func() {
			ui.Text(c, "Commit words").Bold().FontSize(t.FontSize + 6)
			ui.Text(c, strconv.Itoa(len(a.subjects))+" commits of "+a.repo).TextColor(t.TextMuted)
		})
		wordcloud.View(c, &a.cloud, a.words, &a.word).AlignSelf(ui.Stretch).Grow(1).
			Background(t.Surface).Border(1, t.Border).Radius(t.Space(2))
		ui.Column(c).Gap(t.Space(1)).Height(t.Space(36)).Children(func() {
			if a.word == "" {
				ui.Text(c, "Click a word to list its commits").TextColor(t.TextMuted)
				return
			}
			var hits []string
			for _, s := range a.subjects {
				if words(s)[a.word] {
					hits = append(hits, s)
				}
			}
			ui.Text(c, strconv.Itoa(len(hits))+" commits with “"+a.word+"”").Bold()
			for _, s := range hits[:min(len(hits), 6)] {
				ui.Text(c, s).FontSize(t.FontSize - 1).Font("monospace").TextColor(t.TextMuted).SingleLine()
			}
		})
	})
}

func main() {
	a := &app{repo: "."}
	if len(os.Args) > 1 {
		a.repo = os.Args[1]
	}
	out, err := exec.Command("git", "-C", a.repo, "log", "--format=%s").Output()
	if err != nil {
		log.Fatal(err)
	}
	a.subjects = strings.Split(strings.TrimSpace(string(out)), "\n")
	a.words = count(a.subjects)
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Word cloud",
			Width:   860,
			Height:  680,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
