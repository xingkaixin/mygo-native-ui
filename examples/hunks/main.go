// Hunks reviews an agent's edit to a Go file hunk by hunk, with Accept and
// Reject, or with Stage and Discard as a git client stages it, and shows
// beside it the patch the decisions so far make, which git apply --cached
// takes and Copy copies.
//
//	go run ./examples/hunks
package main

import (
	"log"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/hunks"
)

// app is the state the window shows: the view's state, the decisions on
// the hunks, which words the buttons use, and the width of the hunks' side.
type app struct {
	hunks     hunks.State
	decisions []hunks.Decision
	words     int
	split     float32
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	labels := hunks.Review
	if a.words == 1 {
		labels = hunks.Stage
	}
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Padding(t.Space(2), t.Space(3)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "session/store.go").Bold().SingleLine()
			ui.Text(c, "edited by the agent").TextColor(t.TextMuted).SingleLine().Grow(1).Shrink(1)
			ui.Segmented(c, &a.words, "Review", "Stage").Label("Buttons")
		})
		ui.Divider(c)
		ui.Split(c, &a.split, func() {
			hunks.View(c, &a.hunks, diff, &a.decisions, labels).Fill().AutoFocus()
		}, func() {
			patch(c, hunks.Patch(diff, a.decisions), labels)
		}).Grow(1)
	})
}

// patch shows p, its lines colored as a diff's, and Copy.
func patch(c *ui.Context, p string, labels hunks.Labels) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Surface).Children(func() {
		ui.Row(c).Padding(t.Space(2), t.Space(3)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Patch").Bold()
			ui.Text(c, "for git apply --cached").TextColor(t.TextMuted).SingleLine().Grow(1).Shrink(1)
			if ui.Button(c, "Copy").Disabled(p == "").Clicked() {
				c.WriteClipboard(p)
			}
		})
		ui.Divider(c)
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Padding(t.Space(2), t.Space(3)).Children(func() {
				if p == "" {
					ui.Text(c, "Nothing "+strings.ToLower(labels.Accepted)+" yet.").TextColor(t.TextMuted)
					return
				}
				for i, l := range strings.Split(strings.TrimSuffix(p, "\n"), "\n") {
					color := t.TextMuted
					switch {
					case strings.HasPrefix(l, "@@"):
						color = t.Accent
					case strings.HasPrefix(l, "+++"), strings.HasPrefix(l, "---"):
					case strings.HasPrefix(l, "+"):
						color = t.Success
					case strings.HasPrefix(l, "-"):
						color = t.Danger
					case strings.HasPrefix(l, " "):
						color = t.Text
					}
					ui.Text(c.Key(i), strings.ReplaceAll(l, "\t", "    ")).Font("monospace").FontSize(t.FontSize - 1).
						LineHeight(1.5).TextColor(color).NoWrap()
				}
			})
		})
	})
}

func main() {
	a := &app{split: 780}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Hunks",
			Width:   1280,
			Height:  820,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// diff is git diff of an agent's edit to a store of sessions: Put takes a
// context, Get drops an expired session with an error of its own, Delete
// reports whether it held the session, and Sweep counts what it drops.
const diff = `diff --git a/store.go b/store.go
index f9e56ad..11e4c92 100644
--- a/store.go
+++ b/store.go
@@ -3,6 +3,7 @@
 package session
 
 import (
+	"context"
 	"errors"
 	"sync"
 	"time"
@@ -11,6 +12,9 @@ import (
 // ErrNotFound is returned for sessions the store does not hold.
 var ErrNotFound = errors.New("session: not found")
 
+// ErrExpired is returned for sessions past their expiry.
+var ErrExpired = errors.New("session: expired")
+
 // Session is a signed-in user's session.
 type Session struct {
 	ID      string
@@ -30,13 +34,17 @@ func New(ttl time.Duration) *Store {
 	return &Store{sessions: map[string]Session{}, ttl: ttl}
 }
 
-// Put adds a session for user, lasting the store's ttl from now.
-func (s *Store) Put(id string, user int64) Session {
+// Put adds a session for user, lasting the store's ttl from now, unless
+// ctx is done.
+func (s *Store) Put(ctx context.Context, id string, user int64) (Session, error) {
+	if err := ctx.Err(); err != nil {
+		return Session{}, err
+	}
 	s.mu.Lock()
 	defer s.mu.Unlock()
 	sess := Session{ID: id, UserID: user, Expires: time.Now().Add(s.ttl)}
 	s.sessions[id] = sess
-	return sess
+	return sess, nil
 }
 
 // Len returns how many sessions the store holds, expired ones too.
@@ -55,7 +63,8 @@ func (s *Store) Get(id string) (Session, error) {
 		return Session{}, ErrNotFound
 	}
 	if time.Now().After(sess.Expires) {
-		return Session{}, ErrNotFound
+		delete(s.sessions, id)
+		return Session{}, ErrExpired
 	}
 	return sess, nil
 }
@@ -75,11 +84,13 @@ func (s *Store) Users() []int64 {
 	return users
 }
 
-// Delete removes the session id.
-func (s *Store) Delete(id string) {
+// Delete removes the session id, reporting whether the store held it.
+func (s *Store) Delete(id string) bool {
 	s.mu.Lock()
 	defer s.mu.Unlock()
+	_, ok := s.sessions[id]
 	delete(s.sessions, id)
+	return ok
 }
 
 // DeleteUser removes every session of user.
@@ -93,13 +104,16 @@ func (s *Store) DeleteUser(user int64) {
 	}
 }
 
-// Sweep drops the sessions that expired.
-func (s *Store) Sweep() {
+// Sweep drops the sessions that expired and returns how many it dropped.
+func (s *Store) Sweep() int {
 	s.mu.Lock()
 	defer s.mu.Unlock()
+	now, n := time.Now(), 0
 	for id, sess := range s.sessions {
-		if time.Now().After(sess.Expires) {
+		if now.After(sess.Expires) {
 			delete(s.sessions, id)
+			n++
 		}
 	}
+	return n
 }
`
