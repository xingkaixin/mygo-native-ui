// Request edits requests to a small API served in the process, which
// echoes back what it gets, and shows the response under the editor.
// Type a query in the URL to fill the parameters, or turn a parameter off
// to drop it from the URL.
//
//	go run ./examples/request
package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/kvtable"
	"github.com/xingkaixin/mygo-native-ui/ui/request"
	"github.com/xingkaixin/mygo-native-ui/ui/response"
)

func serve() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"method":  r.Method,
			"path":    r.URL.Path,
			"query":   r.URL.Query(),
			"headers": r.Header,
			"body":    string(body),
		})
	}))
}

type app struct {
	win     *mygo.Window
	req     request.Request
	edit    request.State
	sending bool
	err     string
	res     *http.Response
	body    []byte
	timing  *response.Timing
	shown   response.State
}

func (a *app) send() {
	timing := &response.Timing{}
	req, err := a.req.HTTP(timing.Trace(context.Background()))
	if err != nil {
		a.err = err.Error()
		return
	}
	a.sending, a.err = true, ""
	go func() {
		res, err := http.DefaultClient.Do(req)
		var body []byte
		if err == nil {
			body, err = io.ReadAll(res.Body)
			res.Body.Close()
			timing.Done()
		}
		a.win.Update(func() {
			a.sending = false
			if err != nil {
				a.err = err.Error()
				return
			}
			a.res, a.body, a.timing = res, body, timing
		})
	}()
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(16).Gap(12).Children(func() {
		ui.Column(c).Height(300).Children(func() {
			if request.View(c, &a.edit, &a.req).Sent() && !a.sending {
				a.send()
			}
		})
		switch {
		case a.err != "":
			ui.Text(c, a.err).TextColor(t.Danger)
		case a.res != nil:
			ui.Column(c).Grow(1).Radius(t.Radius).Border(1, t.Border).Clip().Children(func() {
				response.View(c, &a.shown, a.res, a.body, a.timing).Grow(1)
			})
		case a.sending:
			ui.Spinner(c).AlignSelf(ui.Center)
		}
	})
}

func main() {
	srv := serve()
	defer srv.Close()
	a := &app{req: request.Request{
		Method: "POST",
		URL:    srv.URL + "/v1/orders?status=open&limit=20",
		Headers: []kvtable.Pair{
			{Key: "Accept", Value: "application/json", On: true},
			{Key: "Authorization", Value: "Bearer {{token}}", On: true},
			{Key: "X-Debug", Value: "1"},
		},
		Kind: request.JSON,
		Body: `{"customer": 42, "items": [{"sku": "KT-100", "qty": 2}]}`,
	}}
	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:   "Request",
			Width:   820,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
