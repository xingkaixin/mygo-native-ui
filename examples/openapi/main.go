// Openapi browses the reference of a made-up Notes API, read from its
// OpenAPI document: its operations by tag in the sidebar, the parameters,
// request body and responses of the one chosen, and Try it, which sends
// requests to a server in the app that keeps notes in memory. The other
// server the document names, production, is made up too, and fails.
//
//	go run ./examples/openapi
package main

import (
	"bytes"
	"cmp"
	_ "embed"
	"encoding/json"
	"io"
	"log"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/openapi"
)

//go:embed notes.json
var spec []byte

// app is the document the window shows, the view's state, and the server
// in the app, which relative URLs of the document's servers are on.
type app struct {
	doc    *openapi.Document
	api    openapi.State
	base   *url.URL
	client *http.Client
	win    *mygo.Window
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		if req, ok := openapi.View(c, &a.api, a.doc).Sent(); ok {
			go a.send(req)
		}
	})
}

// send makes req and gives the view its response.
func (a *app) send(req openapi.Request) {
	resp := a.do(req)
	a.win.Update(func() { a.api.Respond(req, resp) })
}

func (a *app) do(req openapi.Request) openapi.Response {
	u, err := a.base.Parse(req.URL)
	if err != nil {
		return openapi.Response{Err: err}
	}
	r, err := http.NewRequest(req.Method, u.String(), bytes.NewReader(req.Body))
	if err != nil {
		return openapi.Response{Err: err}
	}
	r.Header = req.Header
	start := time.Now()
	res, err := a.client.Do(r)
	if err != nil {
		return openapi.Response{Err: err, Time: time.Since(start)}
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	return openapi.Response{Status: res.StatusCode, Time: time.Since(start), Header: res.Header, Body: body, Err: err}
}

// person is a user as a note's author shows them.
type person struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Plan  string `json:"plan"`
}

type note struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body,omitempty"`
	Color     string    `json:"color"`
	Tags      []string  `json:"tags"`
	Pinned    bool      `json:"pinned"`
	Author    person    `json:"author"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// store is the notes the server keeps, of the one user it knows.
type store struct {
	mu    sync.Mutex
	notes []*note
	next  int
}

var ada = person{ID: "us_ada", Name: "Ada Park", Email: "ada@notes.example", Plan: "pro"}

func newStore() *store {
	day := func(d, h int) time.Time { return time.Date(2026, 10, d, h, 0, 0, 0, time.UTC) }
	return &store{next: 5, notes: []*note{
		{ID: "nt_8c1d", Title: "Launch checklist", Body: "- [x] Write the release notes\n- [ ] Tell support", Color: "yellow",
			Tags: []string{"work", "launch"}, Pinned: true, Author: ada, CreatedAt: day(2, 9), UpdatedAt: day(20, 16)},
		{ID: "nt_1f07", Title: "Groceries", Body: "- oat milk\n- lemons\n- rye bread", Color: "green",
			Tags: []string{"home"}, Author: ada, CreatedAt: day(18, 8), UpdatedAt: day(21, 18)},
		{ID: "nt_5a92", Title: "Talk ideas", Body: "Why our sync engine stopped using timestamps.", Color: "blue",
			Tags: []string{"ideas", "work"}, Author: ada, CreatedAt: day(9, 14), UpdatedAt: day(9, 15)},
		{ID: "nt_c3e4", Title: "Books to read", Body: "The Dispossessed; Piranesi; A Psalm for the Wild-Built", Color: "pink",
			Tags: []string{"ideas"}, Author: ada, CreatedAt: day(1, 21), UpdatedAt: day(12, 22)},
	}}
}

// api serves the Notes API, as its document describes it.
func (s *store) api() http.Handler {
	mux := http.NewServeMux()
	find := func(w http.ResponseWriter, r *http.Request) *note {
		id := r.PathValue("noteId")
		if i := slices.IndexFunc(s.notes, func(n *note) bool { return n.ID == id }); i >= 0 {
			return s.notes[i]
		}
		fail(w, http.StatusNotFound, "not_found", "No note has the ID "+id+".")
		return nil
	}
	list := func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		words := strings.Fields(strings.ToLower(q.Get("q")))
		found := []*note{}
		for _, n := range s.notes {
			text := strings.ToLower(n.Title + " " + n.Body)
			if !slices.ContainsFunc(words, func(w string) bool { return !strings.Contains(text, w) }) &&
				(q.Get("tag") == "" || slices.Contains(n.Tags, q.Get("tag"))) {
				found = append(found, n)
			}
		}
		by := map[string]func(a, b *note) int{
			"updated": func(a, b *note) int { return b.UpdatedAt.Compare(a.UpdatedAt) },
			"created": func(a, b *note) int { return b.CreatedAt.Compare(a.CreatedAt) },
			"title":   func(a, b *note) int { return strings.Compare(a.Title, b.Title) },
		}[cmp.Or(q.Get("sort"), "updated")]
		if by == nil {
			fail(w, http.StatusBadRequest, "bad_request", "sort must be updated, created or title.")
			return
		}
		slices.SortFunc(found, func(a, b *note) int {
			if a.Pinned != b.Pinned {
				if a.Pinned {
					return -1
				}
				return 1
			}
			return by(a, b)
		})
		total := len(found)
		if limit, err := strconv.Atoi(cmp.Or(q.Get("limit"), "20")); err != nil || limit < 1 || limit > 100 {
			fail(w, http.StatusBadRequest, "bad_request", "limit must be a number from 1 to 100.")
			return
		} else if limit < total {
			found = found[:limit]
		}
		reply(w, http.StatusOK, struct {
			Notes []*note `json:"notes"`
			Total int     `json:"total"`
			Next  *string `json:"next"`
		}{found, total, nil})
	}
	handle := func(pattern string, h http.HandlerFunc) {
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			s.mu.Lock()
			defer s.mu.Unlock()
			h(w, r)
		})
	}
	handle("GET /v1/notes", list)
	handle("GET /v1/notes/search", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Deprecation", "true")
		list(w, r)
	})
	handle("POST /v1/notes", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Title, Body, Color string
			Tags               []string
		}
		in.Tags = []string{}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || strings.TrimSpace(in.Title) == "" {
			fail(w, http.StatusBadRequest, "bad_request", "A note needs a title.")
			return
		}
		now := time.Now().UTC().Truncate(time.Second)
		n := &note{ID: "nt_" + strconv.FormatInt(int64(0x9000+s.next), 16), Title: in.Title, Body: in.Body,
			Color: cmp.Or(in.Color, "yellow"), Tags: in.Tags, Author: ada, CreatedAt: now, UpdatedAt: now}
		s.next++
		s.notes = append(s.notes, n)
		reply(w, http.StatusCreated, n)
	})
	handle("GET /v1/notes/{noteId}", func(w http.ResponseWriter, r *http.Request) {
		if n := find(w, r); n != nil {
			reply(w, http.StatusOK, n)
		}
	})
	handle("PATCH /v1/notes/{noteId}", func(w http.ResponseWriter, r *http.Request) {
		n := find(w, r)
		if n == nil {
			return
		}
		var in struct {
			Title, Body, Color *string
			Pinned             *bool
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			fail(w, http.StatusBadRequest, "bad_request", "The body is not JSON: "+err.Error())
			return
		}
		if in.Title != nil {
			n.Title = *in.Title
		}
		if in.Body != nil {
			n.Body = *in.Body
		}
		if in.Color != nil {
			n.Color = *in.Color
		}
		if in.Pinned != nil {
			n.Pinned = *in.Pinned
		}
		n.UpdatedAt = time.Now().UTC().Truncate(time.Second)
		reply(w, http.StatusOK, n)
	})
	handle("DELETE /v1/notes/{noteId}", func(w http.ResponseWriter, r *http.Request) {
		if n := find(w, r); n != nil {
			s.notes = slices.DeleteFunc(s.notes, func(m *note) bool { return m == n })
			w.WriteHeader(http.StatusNoContent)
		}
	})
	handle("PUT /v1/notes/{noteId}/tags", func(w http.ResponseWriter, r *http.Request) {
		n := find(w, r)
		if n == nil {
			return
		}
		tags := []string{}
		if err := json.NewDecoder(r.Body).Decode(&tags); err != nil {
			fail(w, http.StatusBadRequest, "bad_request", "The body must be a list of tags.")
			return
		}
		n.Tags = tags
		reply(w, http.StatusOK, n)
	})
	handle("GET /v1/tags", func(w http.ResponseWriter, r *http.Request) {
		count := map[string]int{}
		for _, n := range s.notes {
			for _, t := range n.Tags {
				count[t]++
			}
		}
		tags := []map[string]any{}
		for _, name := range slices.Sorted(maps.Keys(count)) {
			tags = append(tags, map[string]any{"name": name, "notes": count[name]})
		}
		reply(w, http.StatusOK, tags)
	})
	handle("DELETE /v1/tags/{name}", func(w http.ResponseWriter, r *http.Request) {
		name, had := r.PathValue("name"), false
		for _, n := range s.notes {
			had = had || slices.Contains(n.Tags, name)
			n.Tags = slices.DeleteFunc(n.Tags, func(t string) bool { return t == name })
		}
		if !had {
			fail(w, http.StatusNotFound, "not_found", "No note has the tag "+name+".")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	handle("GET /v1/users/me", func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("session"); err != nil || c.Value != "s_demo" {
			fail(w, http.StatusUnauthorized, "unauthorized", "Sign in again: the session is s_demo.")
			return
		}
		pinned := []*note{}
		for _, n := range s.notes {
			if n.Pinned {
				pinned = append(pinned, n)
			}
		}
		reply(w, http.StatusOK, struct {
			person
			Pinned []*note `json:"pinned"`
		}{ada, pinned})
	})
	return mux
}

func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, status int, code, message string) {
	reply(w, status, map[string]string{"code": code, "message": message})
}

func main() {
	srv := httptest.NewServer(newStore().api())
	defer srv.Close()
	doc, err := openapi.Parse(spec)
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range doc.Problems() {
		log.Print(p)
	}
	base, _ := url.Parse(srv.URL)
	a := &app{doc: doc, base: base, client: &http.Client{Timeout: 10 * time.Second}}
	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:   "Notes API",
			Width:   1440,
			Height:  860,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
