// Package mailthread shows an email conversation, as mail apps thread
// one: its subject, then each message in order, the newest and the unread
// ones open and the rest folded to a line of the sender, the first words
// and the date. In a long thread the folded middle collapses further into
// a single line that counts its messages. An open message shows who sent
// it to whom, its text, its attachments, and the text it quotes folded
// behind a ••• button. Clicking a message's header folds or opens it.
//
//	mailthread.View(c, &app.thread, "Offsite agenda", messages)
package mailthread

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// Message is a message of the thread.
type Message struct {
	From     string
	To, Cc   []string
	Date     time.Time
	Body     string
	Attached []string
	Unread   bool
}

// State is what a thread keeps: which messages and quotes the user
// opened or folded. Keep it in the app's state, one for each thread.
type State struct {
	open   map[int]bool
	quotes map[int]bool
	middle bool
}

// View creates the thread of msgs, oldest first, under subject.
func View(c *ui.Context, s *State, subject string, msgs []Message) ui.Element {
	t := c.Theme()
	if s.open == nil {
		s.open, s.quotes = map[int]bool{}, map[int]bool{}
	}
	isOpen := func(i int) bool {
		if o, ok := s.open[i]; ok {
			return o
		}
		return i == len(msgs)-1 || msgs[i].Unread
	}
	// A thread of more than four messages folds all but the first and the
	// last two, unless one of them is open.
	hidden := func(i int) bool {
		return len(msgs) > 4 && !s.middle && i > 0 && i < len(msgs)-2 && !isOpen(i)
	}
	now := c.Now()
	root := ui.Column(c).Gap(t.Space(3)).Role(ui.RoleGroup).Label(subject)
	root.Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Baseline).MarginY(t.Space(1)).Children(func() {
			ui.Text(c, subject).FontSize(t.FontSize + 6).FontWeight(700).Grow(1)
			ui.Text(c, fmt.Sprintf("%d messages", len(msgs))).TextColor(t.TextMuted).FontFeatures("tnum")
		})
		ui.Column(c).Border(1, t.Border).Radius(t.Radius+4).Clip().Dividers(1, t.Border).Children(func() {
			folded := 0
			for i := range msgs {
				if hidden(i) {
					folded++
				}
			}
			for i, m := range msgs {
				if hidden(i) {
					if i == 1 {
						fold := ui.Row(c.Key("fold")).Padding(t.Space(2.5)).Justify(ui.Center).Cursor(ui.CursorPointer).Background(t.Surface)
						fold.Children(func() {
							ui.Text(c, fmt.Sprintf("%d more messages", folded)).FontSize(t.FontSize - 1).FontWeight(600).TextColor(t.Accent)
						})
						if fold.Clicked() {
							s.middle = true
						}
					}
					continue
				}
				ui.Column(c.Key(i)).Children(func() {
					open := isOpen(i)
					head := ui.Row(c).Gap(t.Space(3)).Padding(t.Space(3)).AlignItems(ui.Center).Cursor(ui.CursorPointer)
					head.Children(func() {
						ui.Avatar(c, m.From, nil)
						ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(t.Space(0.5)).Children(func() {
							name := ui.Text(c, m.From).FontWeight(600).MaxLines(1).Ellipsis("…")
							if m.Unread {
								name.FontWeight(700)
							}
							if open {
								ui.Text(c, recipients(m)).FontSize(t.FontSize - 1).TextColor(t.TextMuted).MaxLines(1).Ellipsis("…")
							} else {
								ui.Text(c, snippet(m.Body)).FontSize(t.FontSize - 1).TextColor(t.TextMuted).MaxLines(1).Ellipsis("…")
							}
						})
						if len(m.Attached) > 0 && !open {
							ui.Icon(c, clip).Size(t.Space(4), t.Space(4)).TextColor(t.TextMuted)
						}
						if m.Unread {
							ui.Box(c).Size(t.Space(2), t.Space(2)).Radius(t.Space(1)).Background(t.Accent)
						}
						ui.Text(c, when(m.Date, now)).FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum")
					})
					if head.Clicked() {
						s.open[i] = !open
					}
					if !open {
						return
					}
					own, quoted := split(m.Body)
					ui.Column(c).Gap(t.Space(3)).Padding(0, t.Space(3), t.Space(4), t.Space(3)+t.Space(8)+t.Space(3)).Children(func() {
						for k, para := range paragraphs(own) {
							ui.Text(c.Key(k), para).Selectable().LineHeight(1.45)
						}
						if len(m.Attached) > 0 {
							ui.Row(c).Wrap().Gap(t.Space(2)).Children(func() {
								for _, name := range m.Attached {
									ui.Row(c.Key(name)).Gap(t.Space(1.5)).Padding(t.Space(1.5), t.Space(2.5)).AlignItems(ui.Center).
										Border(1, t.Border).Radius(t.Radius).Children(func() {
										ui.Icon(c, file).Size(t.Space(4), t.Space(4)).TextColor(t.TextMuted)
										ui.Text(c, name).FontSize(t.FontSize - 1)
									})
								}
							})
						}
						if quoted != "" {
							btn := ui.Box(c).AlignSelf(ui.Start).Padding(0, t.Space(2)).Radius(t.Radius).Background(t.Surface).
								Border(1, t.Border).Cursor(ui.CursorPointer).Tooltip("Show quoted text")
							btn.Children(func() {
								ui.Text(c, "•••").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
							})
							if btn.Clicked() {
								s.quotes[i] = !s.quotes[i]
							}
							if s.quotes[i] {
								ui.Text(c, quoted).FontSize(t.FontSize-1).TextColor(t.TextMuted).LineHeight(1.4).Selectable().
									PaddingStart(t.Space(3)).BorderWidth(0, 0, 0, 2).BorderColor(t.Border)
							}
						}
					})
				})
			}
		})
	})
	return root
}

func icon(body string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` + body + `</svg>`))
}

var (
	clip = icon(`<path d="m21.44 11.05-9.19 9.19a6 6 0 0 1-8.49-8.49l8.57-8.57A4 4 0 1 1 18 8.84l-8.59 8.57a2 2 0 0 1-2.83-2.83l8.49-8.48"/>`)
	file = icon(`<path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/><path d="M14 2v6h6"/>`)
)

// recipients writes whom a message went to: "to Ana, Ben, cc Cy".
func recipients(m Message) string {
	s := "to " + strings.Join(m.To, ", ")
	if len(m.Cc) > 0 {
		s += ", cc " + strings.Join(m.Cc, ", ")
	}
	return s
}

// split parts a message's own text from the text it quotes: lines from
// one ending "wrote:" or starting ">" on.
func split(body string) (own, quoted string) {
	lines := strings.Split(strings.TrimSpace(body), "\n")
	for i, l := range lines {
		l = strings.TrimSpace(l)
		if strings.HasSuffix(l, "wrote:") || strings.HasPrefix(l, ">") {
			var q []string
			for _, ql := range lines[i:] {
				q = append(q, strings.TrimPrefix(strings.TrimPrefix(ql, ">"), " "))
			}
			return strings.TrimSpace(strings.Join(lines[:i], "\n")), strings.TrimSpace(strings.Join(q, "\n"))
		}
	}
	return strings.TrimSpace(body), ""
}

func paragraphs(s string) []string {
	var out []string
	for _, p := range strings.Split(s, "\n\n") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// snippet returns a message's first words, without what it quotes.
func snippet(body string) string {
	own, _ := split(body)
	return strings.Join(strings.Fields(own), " ")
}

// when writes a date as mail lists do: the time today, the day and month
// this year, and the year too before that.
func when(d, now time.Time) string {
	switch {
	case d.Year() == now.Year() && d.YearDay() == now.YearDay():
		return d.Format("15:04")
	case d.Year() == now.Year():
		return d.Format("2 Jan, 15:04")
	}
	return d.Format("2 Jan 2006")
}
