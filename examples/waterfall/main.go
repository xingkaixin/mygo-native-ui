// Waterfall shows two runs as developer tools show them: a page loading,
// read from a HAR file as browsers save one, each request with its DNS
// lookup, connection, wait and download, and a request to a notes server
// traced, each span of its work nested under the one it was part of.
// Drag across the time scale to zoom in; choose a row to see it on the
// right.
//
//	go run ./examples/waterfall [file.har]
package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/waterfall"
)

//go:embed page.har
var page []byte

type app struct {
	view    int
	split   float32
	network []*waterfall.Span
	trace   []*waterfall.Span
	attrs   map[*waterfall.Span][][2]string
	views   [2]waterfall.State
}

func (a *app) render(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Padding(t.Space(2), t.Space(3)).Children(func() {
			ui.Segmented(c, &a.view, "Network", "Trace")
		})
		ui.Divider(c)
		ui.Split(c, &a.split, func() {
			s := &a.views[a.view]
			if a.view == 0 {
				waterfall.View(c.Key("network"), s, waterfall.HARColumns, a.network).Fill()
			} else {
				waterfall.View(c.Key("trace"), s, []string{"Service"}, a.trace).Fill()
			}
		}, func() {
			a.details(c)
		}).Grow(1)
	})
}

// details shows the span chosen: when it started, how long it took and
// its phases, and the attributes of a span of the trace.
func (a *app) details(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).Gap(t.Space(3)).Padding(t.Space(4)).Children(func() {
			sp, ok := a.views[a.view].Chosen()
			if !ok {
				ui.Text(c, "Choose a row to see its timing").TextColor(t.TextMuted)
				return
			}
			ui.Text(c, sp.Name).Bold().FontSize(t.FontSize + 2)
			rows := [][2]string{
				{"Started", sp.Start.Round(time.Millisecond).String()},
				{"Took", (sp.End() - sp.Start).Round(time.Millisecond).String()},
			}
			if len(sp.Phases) > 1 {
				for _, p := range sp.Phases {
					rows = append(rows, [2]string{p.Name, p.Duration.Round(100 * time.Microsecond).String()})
				}
			}
			rows = append(rows, a.attrs[sp]...)
			for i, r := range rows {
				ui.Row(c.Key(i)).Gap(t.Space(3)).Children(func() {
					ui.Text(c, r[0]).TextColor(t.TextMuted).Width(t.Space(28)).Shrink(0)
					ui.Text(c, r[1]).FontFeatures("tnum").Selectable().Shrink(1)
				})
			}
		})
	})
}

// span returns a span of the trace of service, its phase named after it,
// from start for d, with its attributes.
func (a *app) span(name, service string, start, d time.Duration, attrs [][2]string, children ...*waterfall.Span) *waterfall.Span {
	sp := &waterfall.Span{Name: name, Info: []string{service}, Start: start,
		Phases: []waterfall.Phase{{Name: service, Duration: d}}, Children: children}
	a.attrs[sp] = attrs
	return sp
}

// traced returns a request to a notes server, traced.
func (a *app) traced() []*waterfall.Span {
	ms := func(v float64) time.Duration { return time.Duration(v * float64(time.Millisecond)) }
	sql := func(q string) [][2]string { return [][2]string{{"db.system", "postgresql"}, {"db.statement", q}} }
	return []*waterfall.Span{
		a.span("GET /api/notes", "gateway", 0, ms(148.2), [][2]string{{"http.method", "GET"}, {"http.status_code", "200"}, {"net.peer.ip", "10.0.4.17"}},
			a.span("authenticate", "auth", ms(1.4), ms(12.6), [][2]string{{"user.id", "u_51f2"}},
				a.span("GET session", "redis", ms(2.1), ms(1.8), [][2]string{{"db.system", "redis"}, {"db.statement", "GET session:9c1e"}}),
				a.span("verify token", "auth", ms(4.3), ms(9.2), nil),
			),
			a.span("notes.List", "notes", ms(15.2), ms(118.4), [][2]string{{"notes.limit", "50"}},
				a.span("GET notes:u_51f2", "redis", ms(16.0), ms(1.1), [][2]string{{"db.system", "redis"}, {"cache.hit", "false"}}),
				a.span("SELECT notes", "postgres", ms(17.9), ms(64.3), sql("SELECT id, title, updated_at FROM notes WHERE owner = $1 ORDER BY updated_at DESC LIMIT 50")),
				a.span("SELECT tags", "postgres", ms(83.0), ms(21.7), sql("SELECT note_id, tag FROM note_tags WHERE note_id = ANY($1)")),
				a.span("render previews", "notes", ms(105.4), ms(24.8), [][2]string{{"notes.count", "50"}},
					a.span("markdown", "notes", ms(105.9), ms(18.2), nil),
					a.span("sanitize", "notes", ms(124.3), ms(5.3), nil),
				),
				a.span("SET notes:u_51f2", "redis", ms(130.6), ms(2.4), [][2]string{{"db.system", "redis"}, {"cache.ttl", "60s"}}),
			),
			a.span("encode JSON", "gateway", ms(134.1), ms(9.6), [][2]string{{"http.response_content_length", "38420"}}),
		),
	}
}

func main() {
	data := page
	if len(os.Args) > 1 {
		var err error
		if data, err = os.ReadFile(os.Args[1]); err != nil {
			log.Fatal(err)
		}
	}
	network, err := waterfall.ParseHAR(bytes.NewReader(data))
	if err != nil {
		log.Fatal(err)
	}
	a := &app{split: 860, network: network, attrs: map[*waterfall.Span][][2]string{}}
	a.trace = a.traced()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   fmt.Sprintf("Waterfall — %d requests", len(network)),
			Width:   1180,
			Height:  640,
			Content: ui.View(a.render),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
