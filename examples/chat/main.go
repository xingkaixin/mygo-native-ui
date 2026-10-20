// Chat plays a chat with a model that reads the web, as ChatGPT and
// Perplexity show one: each reply with citations by its paragraphs and
// the sources it used below it, and the actions of each message: Copy,
// Edit for the user's, which sends it again as another version, and Retry
// for the model's, which writes another reply. The arrows go between the
// versions, each with the conversation that came after it. A checkpoint
// follows each reply, and restoring it drops the conversation after it.
//
//	go run ./examples/chat
package main

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"

	"github.com/xingkaixin/mygo-native-ui/ui/checkpoint"
	"github.com/xingkaixin/mygo-native-ui/ui/message"
	"github.com/xingkaixin/mygo-native-ui/ui/promptinput"
	"github.com/xingkaixin/mygo-native-ui/ui/shimmer"
	"github.com/xingkaixin/mygo-native-ui/ui/sources"
)

// turn is a message of the user's and its versions, of which at shows.
type turn struct {
	versions []*version
	at       int
}

// version is a version of a message, the replies to it, of which at
// shows.
type version struct {
	prompt  string
	replies []*reply
	at      int
}

// reply is a reply of the model's, words of its answer written so far,
// when it was through, and the turn that came after it, nil until the
// user sends one.
type reply struct {
	answer      *answer
	words       int
	due         time.Time
	at          time.Time
	sourcesOpen bool
	next        *turn
}

func (r *reply) writing() bool { return r.words < r.answer.words() }

type app struct {
	root    *turn
	list    ui.ListState
	prompt  promptinput.State
	writing *reply
}

// thread returns the turns showing, from the first, following the
// versions and replies showing.
func (a *app) thread() []*turn {
	var turns []*turn
	for t := a.root; t != nil; {
		turns = append(turns, t)
		v := t.versions[t.at]
		t = v.replies[v.at].next
	}
	return turns
}

// ask returns a version of prompt, its reply being written.
func (a *app) ask(prompt string) *version {
	v := &version{prompt: prompt}
	a.retry(v)
	return v
}

// retry adds a reply to v, being written, and shows it.
func (a *app) retry(v *version) {
	if a.writing != nil {
		a.writing.words = a.writing.answer.words()
	}
	r := &reply{answer: answerFor(v.prompt, len(v.replies))}
	v.replies = append(v.replies, r)
	v.at = len(v.replies) - 1
	a.writing = r
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	a.advance(c)
	turns := a.thread()
	busy := a.writing != nil && a.writing.writing()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.List(c, &a.list, len(turns), func(i int) { a.show(c, turns[i], i) }).Grow(1).Padding(t.Space(6), t.Space(8), 0)
		ui.Box(c).Padding(t.Space(2), t.Space(8), t.Space(5)).Children(func() {
			promptinput.View(c, &a.prompt, "Ask anything", busy, nil)
		})
	})
	if m, ok := a.prompt.Sent(); ok {
		last := turns[len(turns)-1]
		v := last.versions[last.at]
		v.replies[v.at].next = &turn{versions: []*version{a.ask(m.Text)}}
		a.list.ScrollToEnd()
		c.Invalidate()
	}
	if a.prompt.Stopped() && busy {
		a.writing.answer = a.writing.answer.cut(a.writing.words)
	}
}

// show shows t, turn i, the version of it showing and its reply, and
// once the reply is through, a checkpoint.
func (a *app) show(c *ui.Context, t *turn, i int) {
	th := c.Theme()
	v := t.versions[t.at]
	r := v.replies[v.at]
	ui.Column(c).Gap(th.Space(3)).Padding(0, 0, th.Space(8)).Children(func() {
		ev := message.User(c, &t.at, len(t.versions), v.prompt)
		if text, ok := ev.Edited(); ok {
			t.versions = append(t.versions, a.ask(text))
			t.at = len(t.versions) - 1
			c.Invalidate()
		}
		ev = message.Reply(c, &v.at, len(v.replies), r.answer.text(), r.writing(), func() {
			a.body(c, r)
		})
		if ev.Retried() {
			a.retry(v)
			c.Invalidate()
		}
		if r.writing() {
			return
		}
		if r.at.IsZero() {
			r.at = c.Now()
		}
		if checkpoint.View(c, r.at, strconv.Itoa(2*(i+1))+" messages").Restored() {
			r.next = nil
			// Only the last reply of a conversation can be still coming: one
			// that is was dropped, or is in a version not showing. Finish it,
			// so that the field does not wait on it.
			if a.writing != nil {
				a.writing.words = a.writing.answer.words()
			}
			c.Invalidate()
		}
	})
}

// body shows the paragraphs of r written so far, each with its citation,
// and once written, its sources.
func (a *app) body(c *ui.Context, r *reply) {
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
	v := a.ask("How do iterators work in Go 1.23?")
	a.writing.words = a.writing.answer.words()
	a.root = &turn{versions: []*version{v}}
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
