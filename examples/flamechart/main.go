// Flamechart shows two seconds of a made-up notes server handling
// requests on eight connections, traced in the Trace Event Format, or the
// trace in the file given, as Chrome's tracing and Perfetto save one.
// Scroll with Command held to zoom, drag to pan, and choose a call to see
// it on the right.
//
//	go run ./examples/flamechart [trace.json]
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/flamechart"
)

type app struct {
	name  string
	trace *flamechart.Trace
	chart flamechart.State
	query string
	split float32
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Gap(t.Space(3)).Padding(t.Space(2.5), t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, a.name).Bold().SingleLine().Grow(1).Shrink(1)
			ui.SearchField(c, &a.query).Placeholder("Search calls").Width(t.Space(64))
		})
		ui.Divider(c)
		ui.Split(c, &a.split, func() {
			flamechart.View(c, &a.chart, a.trace, a.query).Fill()
		}, func() {
			a.details(c)
		}).Grow(1)
	})
}

// details shows the call chosen: its thread, when it ran, how long, and
// its arguments.
func (a *app) details(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).Gap(t.Space(3)).Padding(t.Space(4)).Children(func() {
			th, call, ok := a.chart.Chosen()
			if !ok {
				ui.Text(c, "Choose a call to see its timing").TextColor(t.TextMuted)
				return
			}
			ui.Text(c, call.Name).Bold().FontSize(t.FontSize + 2)
			rows := [][2]string{
				{"Thread", th.Name},
				{"Category", call.Category},
				{"Started", call.Start.Round(time.Microsecond).String()},
				{"Took", call.Duration.String()},
				{"Self", call.Self.String()},
				{"Depth", fmt.Sprint(call.Depth)},
			}
			keys := make([]string, 0, len(call.Args))
			for k := range call.Args {
				keys = append(keys, k)
			}
			slices.Sort(keys)
			for _, k := range keys {
				rows = append(rows, [2]string{k, fmt.Sprint(call.Args[k])})
			}
			for i, r := range rows {
				ui.Row(c.Key(i)).Gap(t.Space(3)).Children(func() {
					ui.Text(c, r[0]).TextColor(t.TextMuted).Width(t.Space(24)).Shrink(0)
					ui.Text(c, r[1]).FontFeatures("tnum").Selectable().Shrink(1)
				})
			}
		})
	})
}

func main() {
	data, name := server(), "notes-server trace"
	if len(os.Args) > 1 {
		var err error
		if data, err = os.ReadFile(os.Args[1]); err != nil {
			log.Fatal(err)
		}
		name = filepath.Base(os.Args[1])
	}
	trace, err := flamechart.Parse(bytes.NewReader(data))
	if err != nil {
		log.Fatal(err)
	}
	a := &app{name: fmt.Sprintf("%s — %d calls", name, trace.Calls()), trace: trace, split: 900}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Flame Chart",
			Width:   1240,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// event is an event of the Trace Event Format, its times in microseconds.
type event struct {
	Name string         `json:"name"`
	Cat  string         `json:"cat,omitempty"`
	Ph   string         `json:"ph"`
	Ts   float64        `json:"ts"`
	Dur  float64        `json:"dur,omitempty"`
	Pid  int            `json:"pid"`
	Tid  int            `json:"tid"`
	Args map[string]any `json:"args,omitempty"`
}

// tracer writes the events of a thread.
type tracer struct {
	rnd    *rand.Rand
	tid    int
	events *[]event
}

// span returns a time between lo and hi microseconds.
func (g *tracer) span(lo, hi float64) float64 { return lo + g.rnd.Float64()*(hi-lo) }

// call traces a call from at, which runs a little, then body, which
// returns when it is done, then a little more, and returns when it ended.
func (g *tracer) call(name, cat string, at float64, args map[string]any, body func(at float64) float64) float64 {
	end := at + g.span(0.5, 3)
	if body != nil {
		end = body(end)
	}
	end += g.span(0.5, 3)
	*g.events = append(*g.events, event{Name: name, Cat: cat, Ph: "X", Ts: at, Dur: end - at, Pid: 1, Tid: g.tid, Args: args})
	return end
}

// leaf traces a call from at that takes between lo and hi microseconds.
func (g *tracer) leaf(name, cat string, at, lo, hi float64) float64 {
	end := at + g.span(lo, hi)
	*g.events = append(*g.events, event{Name: name, Cat: cat, Ph: "X", Ts: at, Dur: end - at, Pid: 1, Tid: g.tid})
	return end
}

// times runs body n times, one after another, from at.
func times(n int, at float64, body func(at float64) float64) float64 {
	for range n {
		at = body(at)
	}
	return at
}

// server returns two seconds of a notes server traced: connections taking
// requests, each read, authenticated, served from the database or the
// cache, rendered and written, and the collector marking alongside.
func server() []byte {
	rnd := rand.New(rand.NewPCG(7, 11))
	var events []event
	thread := func(tid int, name string) *tracer {
		events = append(events, event{Name: "thread_name", Ph: "M", Pid: 1, Tid: tid, Args: map[string]any{"name": name}})
		return &tracer{rnd: rnd, tid: tid, events: &events}
	}
	const length = 2e6
	boot := thread(1, "main")
	accept := thread(2, "http.Serve")
	conns := make([]*tracer, 8)
	for i := range conns {
		conns[i] = thread(10+i, fmt.Sprintf("conn 10.0.4.%d", 17+i*3))
	}
	gc := thread(30, "gcBgMarkWorker")

	boot.call("main.main", "runtime", 0, nil, func(at float64) float64 {
		at = boot.leaf("config.Load", "runtime", at, 800, 1200)
		at = boot.call("sql.Open", "db", at, map[string]any{"driver": "pgx"}, func(at float64) float64 {
			return boot.leaf("pgx.Connect", "db", at, 4000, 6000)
		})
		return boot.leaf("signal.Wait", "runtime", at, length-at-20, length-at-10)
	})

	routes := []string{"GET /api/notes", "GET /api/notes/:id", "POST /api/notes", "GET /api/search"}
	accept.call("http.(*Server).Serve", "http", 7000, nil, func(at float64) float64 {
		for _, g := range conns {
			at = accept.leaf("net.(*TCPListener).Accept", "http", at+accept.span(200, 900), 40, 120)
			at = accept.leaf("runtime.newproc", "runtime", at, 3, 8)
			for at := at + 20; at < length-60000; {
				at = g.request(routes[rnd.IntN(len(routes))], at) + g.span(4000, 22000)
			}
		}
		return accept.leaf("net.(*TCPListener).Accept", "http", at+20, length-at-200, length-at-100)
	})

	at := 150000.0
	for at < length-100000 {
		begin := at
		events = append(events, event{Name: "GC mark", Cat: "gc", Ph: "B", Ts: begin, Pid: 1, Tid: gc.tid})
		at = times(rnd.IntN(30)+20, at+5, func(at float64) float64 {
			return gc.call("runtime.gcDrain", "gc", at, nil, func(at float64) float64 {
				return times(rnd.IntN(20)+5, at, func(at float64) float64 {
					return gc.leaf("runtime.scanobject", "gc", at, 2, 30)
				})
			})
		})
		events = append(events, event{Name: "GC mark", Ph: "E", Ts: at + 5, Pid: 1, Tid: gc.tid, Args: map[string]any{"heap": fmt.Sprintf("%d MB", 40+rnd.IntN(30))}})
		at += gc.span(200000, 400000)
	}

	data, err := json.Marshal(map[string]any{"traceEvents": events, "displayTimeUnit": "ms"})
	if err != nil {
		log.Fatal(err)
	}
	return data
}

// request traces a request to route from at on g's connection, and
// returns when the response was written.
func (g *tracer) request(route string, at float64) float64 {
	return g.call("net/http.(*conn).serve", "http", at, map[string]any{"route": route}, func(at float64) float64 {
		at = g.leaf("http.readRequest", "http", at, 20, 80)
		at = g.call("server.(*Router).ServeHTTP", "http", at, nil, func(at float64) float64 {
			return g.call("middleware.Logger", "http", at, nil, func(at float64) float64 {
				at = g.call("middleware.Auth", "auth", at, nil, func(at float64) float64 {
					at = g.call("redis.GET", "cache", at, map[string]any{"key": "session:9c1e"}, func(at float64) float64 {
						return g.leaf("redis.(*Conn).Read", "cache", at, 150, 400)
					})
					return g.call("jwt.Verify", "auth", at, nil, func(at float64) float64 {
						return g.leaf("crypto/ecdsa.Verify", "auth", at, 60, 120)
					})
				})
				at = g.handle(route, at)
				return g.leaf("http.(*response).finishRequest", "http", at, 10, 40)
			})
		})
		return g.leaf("bufio.(*Writer).Flush", "http", at, 20, 90)
	})
}

// handle traces the handler of route.
func (g *tracer) handle(route string, at float64) float64 {
	query := func(at float64, sql string, rows int) float64 {
		return g.call("sql.(*DB).QueryContext", "db", at, map[string]any{"db.statement": sql}, func(at float64) float64 {
			at = g.leaf("sql.(*DB).conn", "db", at, 5, 40)
			return g.call("pgx.(*Conn).Query", "db", at, nil, func(at float64) float64 {
				at = g.leaf("pgx.(*Conn).sendQuery", "db", at, 30, 80)
				at += g.span(800, 6000)
				return times(rows, at, func(at float64) float64 {
					return g.leaf("pgx.(*Conn).readMessage", "db", at, 3, 25)
				})
			})
		})
	}
	encode := func(at float64, n int) float64 {
		return g.call("json.(*Encoder).Encode", "encode", at, nil, func(at float64) float64 {
			return times(n, at, func(at float64) float64 {
				at = g.leaf("json.encodeString", "encode", at, 1, 6)
				if g.rnd.IntN(8) == 0 {
					at = g.leaf("runtime.mallocgc", "runtime", at, 1, 4)
				}
				return at
			})
		})
	}
	render := func(at float64) float64 {
		return g.call("notes.renderPreview", "render", at, nil, func(at float64) float64 {
			at = g.call("markdown.Parse", "render", at, nil, func(at float64) float64 {
				return times(g.rnd.IntN(12)+4, at, func(at float64) float64 {
					return g.leaf("markdown.parseInline", "render", at, 2, 18)
				})
			})
			return g.leaf("html.Sanitize", "render", at, 10, 60)
		})
	}
	switch route {
	case "GET /api/notes":
		return g.call("notes.List", "handler", at, map[string]any{"limit": 50}, func(at float64) float64 {
			at = g.leaf("redis.GET", "cache", at, 120, 300)
			at = query(at, "SELECT id, title, updated_at FROM notes WHERE owner = $1 LIMIT 50", 50)
			at = times(g.rnd.IntN(10)+5, at, render)
			return encode(at, 60)
		})
	case "GET /api/notes/:id":
		return g.call("notes.Get", "handler", at, map[string]any{"id": g.rnd.IntN(9000) + 1000}, func(at float64) float64 {
			at = query(at, "SELECT * FROM notes WHERE id = $1", 1)
			at = render(at)
			return encode(at, 20)
		})
	case "POST /api/notes":
		return g.call("notes.Create", "handler", at, nil, func(at float64) float64 {
			at = g.leaf("json.(*Decoder).Decode", "encode", at, 30, 120)
			at = query(at, "INSERT INTO notes (owner, title, body) VALUES ($1, $2, $3) RETURNING id", 1)
			at = g.leaf("redis.DEL", "cache", at, 100, 250)
			return encode(at, 8)
		})
	}
	return g.call("search.Query", "handler", at, map[string]any{"q": "meeting"}, func(at float64) float64 {
		at = query(at, "SELECT id, title FROM notes WHERE body_tsv @@ plainto_tsquery($1)", 80)
		at = g.call("search.rank", "handler", at, nil, func(at float64) float64 {
			return times(40, at, func(at float64) float64 {
				return g.leaf("search.score", "handler", at, 1, 8)
			})
		})
		return encode(at, 100)
	})
}
