// Response sends requests to a small API served in the process and shows
// what comes back: a JSON list of users that sets cookies, an HTML page,
// a 404 with a JSON error, a slow 500, and a PNG.
//
//	go run ./examples/response
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/response"
)

var routes = []string{"/users", "/about", "/users/99", "/report", "/logo.png"}

func serve() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "c2Vzc2lvbi00Mg", Path: "/", HttpOnly: true, MaxAge: 3600})
		http.SetCookie(w, &http.Cookie{Name: "theme", Value: "dark", Path: "/"})
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "private, max-age=60")
		w.Header().Set("X-Request-Id", "req_7f3a9c2e")
		io.WriteString(w, `{"users":[{"id":1,"name":"Ada Lovelace","email":"ada@example.com","roles":["admin","editor"],"active":true},`+
			`{"id":2,"name":"Alan Turing","email":"alan@example.com","roles":["editor"],"active":true},`+
			`{"id":3,"name":"Grace Hopper","email":"grace@example.com","roles":[],"active":false,"lastSeen":null}],`+
			`"page":1,"total":3}`)
	})
	mux.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, "<!doctype html>\n<html>\n  <head>\n    <title>Kettle API</title>\n  </head>\n"+
			"  <body>\n    <h1>Kettle API</h1>\n    <p>Version <code>2.4.0</code>, serving since 2019.</p>\n  </body>\n</html>\n")
	})
	mux.HandleFunc("/users/99", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, `{"type":"about:blank","title":"Not Found","status":404,"detail":"No user has the id 99."}`)
	})
	mux.HandleFunc("/report", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(600 * time.Millisecond)
		http.Error(w, "upstream timed out", http.StatusInternalServerError)
	})
	mux.HandleFunc("/logo.png", func(w http.ResponseWriter, r *http.Request) {
		img := image.NewRGBA(image.Rect(0, 0, 64, 64))
		for i := range img.Pix {
			img.Pix[i] = uint8(i * 7)
		}
		img.Set(0, 0, color.White)
		var buf bytes.Buffer
		png.Encode(&buf, img)
		w.Header().Set("Content-Type", "image/png")
		w.Write(buf.Bytes())
	})
	return httptest.NewServer(mux)
}

// result is what came back for a request.
type result struct {
	res    *http.Response
	body   []byte
	timing *response.Timing
}

type app struct {
	win     *mygo.Window
	base    string
	route   string
	sending bool
	err     string
	last    *result
	shown   response.State
}

func (a *app) send() {
	a.sending, a.err = true, ""
	url := a.base + a.route
	go func() {
		r, err := fetch(url)
		a.win.Update(func() {
			a.sending = false
			if err != nil {
				a.err = err.Error()
				return
			}
			a.last = r
		})
	}()
}

func fetch(url string) (*result, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	timing := &response.Timing{}
	req = req.WithContext(timing.Trace(req.Context()))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	timing.Done()
	return &result{res, body, timing}, nil
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(16).Gap(12).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "GET").Bold().TextColor(t.Success)
			ui.Select(c, &a.route, routes).Label("Path").Grow(1)
			send := ui.PrimaryButton(c, "Send").Disabled(a.sending)
			if send.Clicked() {
				a.send()
			}
		})
		switch {
		case a.err != "":
			ui.Text(c, a.err).TextColor(t.Danger)
		case a.last != nil:
			ui.Column(c).Grow(1).Radius(t.Radius).Border(1, t.Border).Clip().Children(func() {
				response.View(c, &a.shown, a.last.res, a.last.body, a.last.timing).Grow(1)
			})
		case a.sending:
			ui.Spinner(c).AlignSelf(ui.Center)
		}
	})
}

func main() {
	srv := serve()
	defer srv.Close()
	a := &app{base: srv.URL, route: routes[0]}
	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:   "Response",
			Width:   820,
			Height:  560,
			Content: ui.View(a.view),
		})
		a.send()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
