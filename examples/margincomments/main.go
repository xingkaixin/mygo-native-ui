// Margincomments is a design doc under review: teammates' comments stand
// in the margin by the words they are about, some answered. Click a
// tinted phrase or a card to bring its thread forward and reply, select
// words to comment on them yourself, and resolve a thread to put it away.
//
//	go run ./examples/margincomments
package main

import (
	"log"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/margincomments"
)

const doc = `Fieldnotes saves every edit to the server the moment it is made. That works well at a desk, but our field teams spend whole days in basements, plant rooms and rural sites with no signal, and today the app shows a spinner and refuses to open a note it cannot fetch. This document proposes letting the app read and write notes offline, and reconcile them once the connection returns.
Goals. Every note a user opened in the last 30 days opens at once without a network. Edits made offline are never lost, and they reach the server within a minute of reconnecting. Conflicts are rare enough that most people never see one.
Storage. Each device keeps a local SQLite database of the notes, their attachments up to 20 MB each, and an outbox of operations not yet sent. We estimate 150 MB for a typical field engineer, which fits on every device we support.
Sync model. Instead of sending whole notes, the client records small operations, such as inserting text, deleting a range or setting a field, each stamped with a hybrid logical clock. The server applies them in clock order and passes them on to the user's other devices. Text is a sequence CRDT, so edits to the same paragraph on two devices merge without asking anyone.
Conflicts. Two edits to the same structured field, such as a status or a due date, cannot merge. The later clock wins, and the losing value stays in the note's history, with a banner telling both authors what changed.
Rollout. We ship read-only offline access first, behind a flag for the field operations team, then offline editing for 10% of users in March and for everyone by the end of April. Success means fewer than 0.1% of syncs need a manual repair.`

// me is the user of the app, who writes new comments and replies.
const me = "Jordan Lee"

type app struct {
	threads []margincomments.Thread
	margin  margincomments.State
}

// on returns the runes of doc that phrase takes, where it first shows.
func on(phrase string) (start, end int) {
	i := strings.Index(doc, phrase)
	if i < 0 {
		log.Fatalf("%q is not in the document", phrase)
	}
	start = utf8.RuneCountInString(doc[:i])
	return start, start + utf8.RuneCountInString(phrase)
}

// thread returns a thread on phrase by who, written ago before now.
func thread(phrase, who string, ago time.Duration, text string, replies ...margincomments.Comment) margincomments.Thread {
	start, end := on(phrase)
	return margincomments.Thread{
		Start: start, End: end,
		Comment: margincomments.Comment{Author: who, Time: time.Now().Add(-ago), Text: text},
		Replies: replies,
	}
}

// reply returns a reply by who, written ago before now.
func reply(who string, ago time.Duration, text string) margincomments.Comment {
	return margincomments.Comment{Author: who, Time: time.Now().Add(-ago), Text: text}
}

func threads() []margincomments.Thread {
	resolved := thread("Rollout", "Maya Chen", 26*time.Hour, "Can we name who flips the flag back if sync breaks in the field?",
		reply("Priya Raman", 25*time.Hour, "Added the on-call rotation to the rollout plan."))
	resolved.Resolved = true
	return []margincomments.Thread{
		thread("shows a spinner and refuses to open a note it cannot fetch", "Lena Fischer", 50*time.Hour,
			"This is our top support complaint: 41 tickets last month mention it.",
			reply("Priya Raman", 30*time.Hour, "Thanks, I'll link the ticket dashboard in the appendix.")),
		thread("within a minute of reconnecting", "Omar Haddad", 5*time.Hour,
			"Is a minute realistic on a flaky 3G link with a long outbox? Maybe state it as a p95.",
			reply("Priya Raman", 4*time.Hour, "Fair. p95 under 60 seconds, measured from the first request that succeeds."),
			reply("Omar Haddad", 3*time.Hour, "Works for me.")),
		thread("150 MB", "Sam Okafor", 22*time.Hour,
			"Older Android phones in the field often have under 2 GB free. Can we cap it and evict the notes opened least recently?"),
		thread("hybrid logical clock", "Maya Chen", 2*time.Hour,
			"The audit log service already stamps events with HLCs. Worth reusing its library rather than writing another.",
			reply("Omar Haddad", 90*time.Minute, "Agreed, it has been solid in production for a year.")),
		thread("The later clock wins", "Lena Fischer", 40*time.Minute,
			"Last writer wins will surprise people on status fields. Could we ask the second author instead?"),
		resolved,
	}
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	open := 0
	for _, th := range a.threads {
		if !th.Resolved {
			open++
		}
	}
	ui.Scroll(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).MaxWidth(t.Space(260)).Padding(t.Space(10), t.Space(12), t.Space(24)).Gap(t.Space(8)).Children(func() {
			ui.Column(c).Gap(t.Space(1.5)).Children(func() {
				ui.Text(c, "Offline editing for Fieldnotes").Bold().FontSize(t.FontSize + 10)
				ui.Text(c, "Design doc · Draft 3 · Priya Raman · "+strconv.Itoa(open)+" open comments, "+
					strconv.Itoa(len(a.threads)-open)+" resolved").TextColor(t.TextMuted)
			})
			ev := margincomments.View(c, &a.margin, doc, a.threads, me)
			if start, end, text, ok := ev.Added(); ok {
				a.threads = append(a.threads, margincomments.Thread{Start: start, End: end,
					Comment: margincomments.Comment{Author: me, Time: time.Now(), Text: text}})
			}
			if i, text, ok := ev.Replied(); ok {
				a.threads[i].Replies = append(a.threads[i].Replies, reply(me, 0, text))
			}
			if i, ok := ev.Resolved(); ok {
				a.threads[i].Resolved = true
			}
		})
	})
}

func main() {
	a := &app{threads: threads()}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Margin comments",
			Width:   1100,
			Height:  800,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
