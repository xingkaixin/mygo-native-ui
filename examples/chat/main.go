// Chat plays a chat with a model that reads the web, as ChatGPT and
// Perplexity show one: each reply with citations by its paragraphs and
// the sources it used below it, and the actions of each message: Copy,
// Edit for the user's, which sends it again as a branch of the
// conversation, and Retry for the model's, which writes another version
// of the reply. The arrows go between the versions, each with the
// conversation that came after it, and Branches in the toolbar opens a
// map of them all, where a click shows a message. A checkpoint follows
// each reply, and restoring it drops the conversation after it.
//
//	go run ./examples/chat
package main

import (
	"log"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"

	"github.com/xingkaixin/mygo-native-ui/ui/branches"
	"github.com/xingkaixin/mygo-native-ui/ui/checkpoint"
	"github.com/xingkaixin/mygo-native-ui/ui/message"
	"github.com/xingkaixin/mygo-native-ui/ui/promptinput"
	"github.com/xingkaixin/mygo-native-ui/ui/shimmer"
	"github.com/xingkaixin/mygo-native-ui/ui/sources"
)

// msg is a message of the conversation: the user's prompt, or a reply of
// the model's, words of its answer written so far, and when it was
// through.
type msg struct {
	prompt      string
	answer      *answer
	words       int
	due         time.Time
	at          time.Time
	sourcesOpen bool
}

func (m *msg) writing() bool { return m.words < m.answer.words() }

type app struct {
	tree     branches.Tree[msg]
	list     ui.ListState
	prompt   promptinput.State
	writing  *msg
	branches bool
}

// ask adds prompt after message parent, -1 for the first, as the last of
// its versions, and a reply to it, being written.
func (a *app) ask(parent int, prompt string) {
	a.reply(a.tree.Add(a.tree.Add(parent, msg{prompt: prompt}), msg{}), prompt)
}

// reply has the model write reply id to prompt, worded as the version it
// is.
func (a *app) reply(id int, prompt string) {
	a.finish()
	_, n := a.tree.Version(id)
	m := a.tree.Message(id)
	m.answer = answerFor(prompt, n-1)
	a.writing = m
}

// finish writes the rest of the reply being written at once, for another
// to start.
func (a *app) finish() {
	if a.writing != nil {
		a.writing.words = a.writing.answer.words()
	}
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	a.advance(c)
	path := a.tree.Path()
	busy := a.writing != nil && a.writing.writing()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		var button ui.Element
		ui.Toolbar(c, func() {
			ui.Spacer(c)
			button = ui.Button(c, "Branches").Expanded(a.branches)
			if button.Clicked() {
				a.branches = !a.branches
			}
		}).Label("Conversation").Padding(t.Space(2), t.Space(3))
		ui.Popover(c, button, &a.branches, func() {
			ui.ScrollBoth(c).MaxWidth(t.Space(100)).MaxHeight(t.Space(120)).Children(func() {
				if id, ok := branches.Map(c, &a.tree, label).Shown(); ok {
					a.list.ScrollTo(slices.Index(a.tree.Path(), id)/2, ui.Start)
				}
			})
		})
		ui.Divider(c)
		ui.List(c, &a.list, len(path)/2, func(i int) { a.show(c, path[2*i], path[2*i+1], i) }).Grow(1).Padding(t.Space(6), t.Space(8), 0)
		ui.Box(c).Padding(t.Space(2), t.Space(8), t.Space(5)).Children(func() {
			promptinput.View(c, &a.prompt, "Ask anything", busy, nil)
		})
	})
	if m, ok := a.prompt.Sent(); ok {
		a.ask(path[len(path)-1], m.Text)
		a.list.ScrollToEnd()
		c.Invalidate()
	}
	if a.prompt.Stopped() && busy {
		a.writing.answer = a.writing.answer.cut(a.writing.words)
	}
}

// show shows turn i, the user's message u and the reply r to it, and
// once the reply is through, a checkpoint.
func (a *app) show(c *ui.Context, u, r, i int) {
	th := c.Theme()
	um, rm := a.tree.Message(u), a.tree.Message(r)
	ui.Column(c).Gap(th.Space(3)).Padding(0, 0, th.Space(8)).Children(func() {
		at, n := a.tree.Version(u)
		if text, ok := message.User(c, at, n, um.prompt).Edited(); ok {
			a.reply(a.tree.Add(a.tree.Branch(u, msg{prompt: text}), msg{}), text)
			c.Invalidate()
		}
		at, n = a.tree.Version(r)
		ev := message.Reply(c, at, n, rm.answer.text(), rm.writing(), func() {
			a.body(c, rm)
		})
		if ev.Retried() {
			a.reply(a.tree.Branch(r, msg{}), um.prompt)
			c.Invalidate()
		}
		if rm.writing() {
			return
		}
		if rm.at.IsZero() {
			rm.at = c.Now()
		}
		if checkpoint.View(c, rm.at, strconv.Itoa(2*(i+1))+" messages").Restored() {
			a.tree.Cut(r)
			// Only the last reply of a conversation can be still coming: one
			// that is was dropped, or is in a version not showing. Finish it,
			// so that the field does not wait on it.
			a.finish()
			c.Invalidate()
		}
	})
}

// label names m on the map of branches: the user's prompt, or the first
// sentence of the reply.
func label(m *msg) string {
	if m.answer == nil {
		return m.prompt
	}
	if len(m.answer.paras) == 0 {
		return "Stopped"
	}
	s := strings.ReplaceAll(m.answer.paras[0].text, "`", "")
	if i := strings.Index(s, ". "); i >= 0 {
		s = s[:i+1]
	}
	return s
}

// body shows the paragraphs of r written so far, each with its citation,
// and once written, its sources.
func (a *app) body(c *ui.Context, r *msg) {
	t := c.Theme()
	ans := r.answer
	ui.Column(c).Gap(t.Space(3)).Children(func() {
		if r.words == 0 && r.writing() {
			shimmer.Text(c, "Reading the web…")
			return
		}
		left := r.words
		for i, p := range ans.paras {
			if left <= 0 {
				break
			}
			words := strings.SplitAfter(p.text, " ")
			ui.Row(c.Key(i)).Wrap().AlignItems(ui.Center).Children(func() {
				if left < len(words) {
					markdown.Stream(c, strings.Join(words[:left], ""), nil).Shrink(1)
					return
				}
				markdown.View(c, p.text, nil).Shrink(1)
				if len(p.cites) > 0 {
					var cited []sources.Source
					for _, k := range p.cites {
						cited = append(cited, ans.sources[k])
					}
					sources.Citation(c, cited)
				}
			})
			left -= len(words)
		}
		if !r.writing() && len(ans.sources) > 0 {
			sources.List(c, &r.sourcesOpen, ans.sources)
		}
	})
}

// advance writes the reply being written a word further when due, as a
// model streams it, and asks for a frame when the next word is.
func (a *app) advance(c *ui.Context) {
	r := a.writing
	if r == nil || !r.writing() {
		return
	}
	now := c.Now()
	if r.due.IsZero() {
		r.due = now.Add(1200 * time.Millisecond)
	}
	if !now.Before(r.due) {
		r.words++
		r.due = now.Add(35 * time.Millisecond)
	}
	c.After(r.due.Sub(now))
}

func main() {
	a := &app{prompt: promptinput.State{Text: "How do I log structured data?"}}
	const first = "How do iterators work in Go 1.23?"
	a.ask(-1, first)
	path := a.tree.Path()
	a.reply(a.tree.Branch(path[1], msg{}), first)
	a.finish()
	a.list.FollowEnd = true
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Chat",
			Width:   860,
			Height:  780,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
