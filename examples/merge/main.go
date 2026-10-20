// Merge resolves the conflicts git left in a file: the file given as its
// argument, which Save writes back once each conflict is resolved, or a
// Go file of its own with three conflicts, one with its base as diff3
// writes it, whose result Copy copies.
//
//	go run ./examples/merge [file]
package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"github.com/xingkaixin/mygo-native-ui/ui/merge"
)

// app is the state the window shows: the file, its text with the
// conflicts, the view's state, and the text last saved or copied.
type app struct {
	path  string
	name  string
	text  string
	merge merge.State
	done  string
	err   string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Padding(t.Space(2), t.Space(3)).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, a.name).Bold().SingleLine().Grow(1).Shrink(1)
			text, ok := a.merge.Resolved()
			label, note := "Copy", "Copied"
			if a.path != "" {
				label, note = "Save", "Saved"
			}
			switch {
			case a.err != "":
				ui.Text(c, a.err).FontSize(t.FontSize - 1).TextColor(t.Danger)
			case ok && text == a.done:
				ui.Text(c, note).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
			}
			if ui.PrimaryButton(c, label).Disabled(!ok).Clicked() {
				a.finish(c, text)
			}
		})
		ui.Divider(c)
		merge.View(c, &a.merge, a.text, editor.LanguageFor(a.name)).Grow(1)
	})
}

// finish writes text to the file, or copies it without one.
func (a *app) finish(c *ui.Context, text string) {
	a.err = ""
	if a.path == "" {
		c.WriteClipboard(text)
		a.done = text
		return
	}
	if err := os.WriteFile(a.path, []byte(text), 0o644); err != nil {
		a.err = err.Error()
		return
	}
	a.done = text
}

func main() {
	a := &app{name: "fetch.go", text: sample}
	if len(os.Args) > 1 {
		data, err := os.ReadFile(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		a.path, a.name, a.text = os.Args[1], filepath.Base(os.Args[1]), string(data)
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Merge",
			Width:   1120,
			Height:  780,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// sample is a file as git leaves it after merging feature/logging into
// a branch that changed the same lines.
const sample = `// Package fetch gets documents over HTTP, trying again when a server
// fails for a while.
package fetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
<<<<<<< HEAD
	"strconv"
	"time"
=======
	"log/slog"
	"time"
>>>>>>> feature/logging
)

// Client gets documents, trying each request a few times.
type Client struct {
	HTTP    *http.Client
	Retries int
}

// ErrGone is returned for documents the server no longer has.
var ErrGone = errors.New("fetch: document gone")

// New returns a client that tries each request three times.
func New() *Client {
	return &Client{HTTP: &http.Client{Timeout: 10 * time.Second}, Retries: 3}
}

// backoff returns how long to wait after try n failed with resp.
func backoff(n int, resp *http.Response) time.Duration {
<<<<<<< HEAD
	if s := resp.Header.Get("Retry-After"); s != "" {
		if secs, err := strconv.Atoi(s); err == nil {
			return time.Duration(secs) * time.Second
		}
	}
	return time.Duration(n+1) * 500 * time.Millisecond
||||||| 3f2a9c1
	return time.Duration(n+1) * 500 * time.Millisecond
=======
	return min(time.Duration(1<<n)*250*time.Millisecond, 8*time.Second)
>>>>>>> feature/logging
}

// Get gets the document at url, trying again after server errors.
func (c *Client) Get(ctx context.Context, url string) ([]byte, error) {
	var last error
	for n := range c.Retries {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		resp, err := c.HTTP.Do(req)
		if err != nil {
			last = err
			continue
		}
		body, err := read(resp)
		switch {
		case err == nil:
			return body, nil
		case resp.StatusCode == http.StatusGone:
			return nil, ErrGone
		case resp.StatusCode < 500:
			return nil, err
		}
		last = err
<<<<<<< HEAD
		select {
		case <-time.After(backoff(n, resp)):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
=======
		slog.Warn("fetch: trying again", "url", url, "status", resp.StatusCode)
		time.Sleep(backoff(n, resp))
>>>>>>> feature/logging
	}
	return nil, fmt.Errorf("fetch %s: %w", url, last)
}

// read reads the body of resp and closes it, or returns an error of its
// status when it is not 200 OK.
func read(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}
`
