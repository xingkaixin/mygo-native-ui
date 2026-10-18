// Tabs shows document tabs above an editor's text: twelve documents, more
// than the window's width holds, two of them with changes not saved. A
// click shows a document; its close button or the middle button closes
// it, and dragging moves it to another place in the bar.
//
//	go run ./examples/tabs
package main

import (
	"log"
	"slices"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/tabs"
)

// doc is an open document.
type doc struct {
	name     string
	text     string
	modified bool
}

// app is the state the window shows: the documents open, in the order of
// their tabs, and the index of the one shown.
type app struct {
	docs     []doc
	selected int
}

// view shows the tabs over the text of the document chosen. The bar
// reports a tab closed or moved, which the view removes or moves in the
// list at once: the bar has set the index for the list as it leaves it.
func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		bar := tabs.Bar(c, &a.selected, len(a.docs), func(i int) tabs.Tab {
			d := a.docs[i]
			return tabs.Tab{Title: d.name, Modified: d.modified, Key: d.name}
		})
		if i, ok := bar.Closed(); ok {
			a.docs = slices.Delete(a.docs, i, i+1)
		}
		if from, to, ok := bar.Moved(); ok {
			d := a.docs[from]
			a.docs = slices.Insert(slices.Delete(a.docs, from, from+1), to, d)
		}
		if len(a.docs) == 0 {
			ui.Column(c).Grow(1).Center().Children(func() {
				ui.Text(c, "No open documents").TextColor(t.TextMuted)
			})
			return
		}
		ui.Scroll(c).Grow(1).Padding(16, 24).Children(func() {
			ui.Text(c, a.docs[a.selected].text).Font("monospace").LineHeight(1.5).Selectable()
		})
	})
}

func main() {
	a := &app{docs: documents}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Tabs",
			Width:   960,
			Height:  600,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

var documents = []doc{
	{name: "main.go", modified: true, text: `package main

import (
    "log"
    "net/http"
)

func main() {
    cfg, err := loadConfig("config.yaml")
    if err != nil {
        log.Fatal(err)
    }
    srv := newServer(cfg)
    log.Printf("listening on %s", cfg.Addr)
    log.Fatal(http.ListenAndServe(cfg.Addr, srv))
}`},
	{name: "server.go", text: `package main

import "net/http"

type server struct {
    cfg   config
    store *store
    mux   *http.ServeMux
}

func newServer(cfg config) *server {
    s := &server{cfg: cfg, store: newStore(), mux: http.NewServeMux()}
    s.routes()
    return s
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    s.mux.ServeHTTP(w, r)
}`},
	{name: "routes.go", text: `package main

func (s *server) routes() {
    s.mux.HandleFunc("GET /notes", s.listNotes)
    s.mux.HandleFunc("POST /notes", s.createNote)
    s.mux.HandleFunc("GET /notes/{id}", s.getNote)
    s.mux.HandleFunc("DELETE /notes/{id}", s.deleteNote)
}`},
	{name: "handlers.go", modified: true, text: `package main

import (
    "encoding/json"
    "net/http"
)

func (s *server) listNotes(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, s.store.all())
}

func (s *server) createNote(w http.ResponseWriter, r *http.Request) {
    var n note
    if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    writeJSON(w, s.store.add(n))
}`},
	{name: "store.go", text: `package main

import "sync"

type note struct {
    ID   int    ` + "`json:\"id\"`" + `
    Text string ` + "`json:\"text\"`" + `
}

type store struct {
    mu    sync.Mutex
    next  int
    notes map[int]note
}

func newStore() *store { return &store{notes: map[int]note{}} }`},
	{name: "store_test.go", text: `package main

import "testing"

func TestAdd(t *testing.T) {
    s := newStore()
    n := s.add(note{Text: "milk"})
    if got := s.all(); len(got) != 1 || got[0] != n {
        t.Fatalf("all() = %v", got)
    }
}`},
	{name: "config.go", text: `package main

type config struct {
    Addr    string
    DataDir string
}

func loadConfig(path string) (config, error) {
    return config{Addr: ":8080", DataDir: "data"}, nil
}`},
	{name: "middleware.go", text: `package main

import (
    "log"
    "net/http"
    "time"
)

func logged(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
    })
}`},
	{name: "README.md", text: `# Notes

A small server that keeps notes in memory.

    go run .

Then list them with curl localhost:8080/notes.`},
	{name: "go.mod", text: `module example.com/notes

go 1.24`},
	{name: "Makefile", text: `run:
    go run .

test:
    go test ./...`},
	{name: "config.yaml", text: `addr: ":8080"
data_dir: data`},
}
