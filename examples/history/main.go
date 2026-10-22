// History shows a chat app's sidebar of past conversations beside the
// one open: 300 of them over the last year, two pinned and one whose
// reply is still coming. A click or the arrows open one, the "…" button
// or a right-click renames, pins or deletes it, and the field searches
// titles and the first message of each. New chat adds one, whose reply
// takes a few seconds.
//
//	go run ./examples/history
package main

import (
	"log"
	"math"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/history"
)

// chat is a conversation: what the sidebar shows, its first message, and
// when its reply is through.
type chat struct {
	history.Chat
	prompt string
	done   time.Time
}

// app is the state the window shows: the conversations, the ID of the one
// open, and the sidebar's state.
type app struct {
	chats   []chat
	list    []history.Chat
	active  string
	history history.State
	next    int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	now := c.Now()
	a.list = a.list[:0]
	for i := range a.chats {
		ch := &a.chats[i]
		ch.Streaming = now.Before(ch.done)
		if ch.Streaming {
			c.After(ch.done.Sub(now))
		}
		a.list = append(a.list, ch.Chat)
	}
	ui.Row(c).Fill().AlignItems(ui.Stretch).Background(t.Background).Children(func() {
		ui.Column(c).Width(272).BorderEndWidth(1).BorderColor(t.Border).Children(func() {
			ev := history.View(c, &a.history, a.list, a.active, a.snippet)
			if i, ok := ev.Opened(); ok {
				a.active = a.chats[i].ID
			}
			if i, title, ok := ev.Renamed(); ok {
				a.chats[i].Title = title
			}
			if i, pinned, ok := ev.Pinned(); ok {
				a.chats[i].Pinned = pinned
			}
			if i, ok := ev.Deleted(); ok {
				if a.chats[i].ID == a.active {
					a.active = ""
				}
				a.chats = slices.Delete(a.chats, i, i+1)
			}
			if ev.New() {
				a.add(now)
			}
		})
		a.pane(c)
	})
}

// pane stands in for the conversation open.
func (a *app) pane(c *ui.Context) {
	t := c.Theme()
	i := slices.IndexFunc(a.chats, func(ch chat) bool { return ch.ID == a.active })
	ui.Column(c).Grow(1).Shrink(1).Padding(24, 32).Gap(12).Children(func() {
		if i < 0 {
			ui.Column(c).Grow(1).Center().Children(func() {
				ui.Text(c, "Choose a chat, or start a new one").TextColor(t.TextMuted)
			})
			return
		}
		ch := a.chats[i]
		ui.Text(c, ch.Title).FontSize(t.FontSize + 6).Bold()
		ui.Text(c, ch.Updated.Format("Monday, January 2, 2006 15:04")).TextColor(t.TextMuted)
		ui.Text(c, ch.prompt).Padding(10, 14).Radius(12).Background(t.Surface).MaxWidth(520)
		if ch.Streaming {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Spinner(c)
				ui.Text(c, "Writing a reply…").TextColor(t.TextMuted)
			})
			return
		}
		ui.Text(c, "The conversation's messages would show here.").TextColor(t.TextMuted)
	})
}

// snippet returns the words around query in the first message of chat
// i, or "" where it is not there.
func (a *app) snippet(i int, query string) string {
	text := a.chats[i].prompt
	at := strings.Index(strings.ToLower(text), strings.ToLower(query))
	if at < 0 {
		return ""
	}
	from := strings.LastIndexByte(text[:max(at-20, 0)], ' ') + 1
	if from > 0 {
		return "…" + text[from:]
	}
	return text
}

// add adds a new conversation and opens it.
func (a *app) add(now time.Time) {
	a.next++
	id := "new-" + strconv.Itoa(a.next)
	a.chats = append(a.chats, chat{
		Chat:   history.Chat{ID: id, Title: "New chat", Updated: now},
		prompt: "Hello! What can you help me with today?",
		done:   now.Add(4 * time.Second),
	})
	a.active = id
}

func main() {
	a := &app{chats: generate(time.Now())}
	a.active = a.chats[0].ID
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "History",
			Width:   1000,
			Height:  680,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// generate returns 300 conversations over the year before now, more of
// them recent, newest first: the first still replying, and two pinned.
func generate(now time.Time) []chat {
	r := rand.New(rand.NewPCG(7, 11))
	chats := make([]chat, 300)
	for i := range chats {
		topic := topics[r.IntN(len(topics))]
		ask := asks[r.IntN(len(asks))]
		age := time.Duration(365*24*math.Pow(float64(i)/float64(len(chats)), 1.5)*float64(time.Hour)) +
			time.Duration(r.IntN(90))*time.Minute
		chats[i] = chat{
			Chat: history.Chat{
				ID:      "chat-" + strconv.Itoa(i),
				Title:   strings.Replace(ask[0], "%", topic, 1),
				Updated: now.Add(-age),
			},
			prompt: strings.Replace(ask[1], "%", topic, 1),
		}
	}
	chats[0].done = now.Add(10 * time.Second)
	chats[40].Pinned = true
	chats[120].Pinned = true
	return chats
}

var topics = []string{
	"Go generics", "a sourdough starter", "the Kubernetes ingress", "React server components",
	"a trip to Kyoto", "SQLite full-text search", "my cover letter", "Rust lifetimes",
	"a birthday party", "the quarterly budget", "Postgres indexes", "a marathon plan",
	"WebGPU shaders", "a garden layout", "the onboarding email", "TLS certificates",
	"a podcast outline", "Python asyncio", "the team offsite", "CSS grid",
}

var asks = [][2]string{
	{"Questions about %", "I have a few questions about % that I couldn't find answered anywhere."},
	{"Debugging %", "Something is wrong with % and I've spent the afternoon on it. Where should I start?"},
	{"Ideas for %", "Can you give me some ideas for %? I'd like a few options to compare."},
	{"Explain %", "Explain % to me as if I knew the basics but not the details."},
	{"Plan for %", "Help me put together a plan for %, step by step, with rough timings."},
	{"Review of %", "Could you review % and tell me what you would change and why?"},
	{"Summary of %", "Write a short summary of % that I can share with the team."},
}
