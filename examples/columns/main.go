// Columns browses a directory as Finder's column view does: the directory
// given as its argument, or the home directory. A file chosen shows its
// size, its date and, for text, its first lines; a double click or Return
// opens it in its app.
//
//	go run ./examples/columns [dir]
package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/columns"
)

var (
	folderIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="#5aa9f7" d="M2 6a2 2 0 0 1 2-2h5l2 2h9a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2z"/><path fill="#7dbcfa" d="M2 9h20v9a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2z"/></svg>`))
	fileIcon   = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="#ffffff" stroke="#a1a1aa" stroke-width="1.2" d="M6 2.5h8l4.5 4.5v13a1.5 1.5 0 0 1-1.5 1.5H6A1.5 1.5 0 0 1 4.5 20V4A1.5 1.5 0 0 1 6 2.5z"/><path fill="none" stroke="#a1a1aa" stroke-width="1.2" d="M14 2.5V7h4.5"/></svg>`))
)

// app is the state the window shows: the root, the entries of the
// directories read and which are directories, and the view's state.
type app struct {
	root    string
	entries map[string][]string
	dirs    map[string]bool
	browse  columns.State[string]
}

// children returns the paths in dir, read once, without hidden ones,
// sorted by name as Finder sorts them; none for a file.
func (a *app) children(dir string) []string {
	if kids, ok := a.entries[dir]; ok {
		return kids
	}
	var kids []string
	if es, err := os.ReadDir(dir); err == nil {
		for _, e := range es {
			if !strings.HasPrefix(e.Name(), ".") {
				p := filepath.Join(dir, e.Name())
				kids = append(kids, p)
				a.dirs[p] = e.IsDir()
			}
		}
	}
	slices.SortFunc(kids, func(x, y string) int {
		return strings.Compare(strings.ToLower(filepath.Base(x)), strings.ToLower(filepath.Base(y)))
	})
	a.entries[dir] = kids
	return kids
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		path := a.browse.Path()
		where := a.root
		if len(path) > 0 {
			where = path[len(path)-1]
		}
		ui.Text(c, where).FontSize(12).TextColor(t.TextMuted).SingleLine().Padding(8, 12)
		ui.Divider(c)
		columns.View(c, &a.browse, a.children(a.root), a.children, func(p string) {
			icon := fileIcon
			if a.dirs[p] {
				icon = folderIcon
			}
			ui.Image(c, icon).Size(16, 16)
			ui.Text(c, filepath.Base(p)).SingleLine()
		}, func(p string) { preview(c, p) }).Grow(1)
		if p, ok := a.browse.Opened(); ok {
			c.OpenURL((&url.URL{Scheme: "file", Path: p}).String())
		}
	})
}

// preview builds what the column past the last shows of the file at p:
// its name, kind, size and date, and the first lines of text.
func preview(c *ui.Context, p string) {
	t := c.Theme()
	info, err := os.Stat(p)
	if err != nil {
		ui.Text(c, err.Error()).TextColor(t.Danger)
		return
	}
	kind := "Folder"
	if !info.IsDir() {
		kind = strings.TrimSpace(strings.ToUpper(strings.TrimPrefix(filepath.Ext(p), "."))+" file") + " · " + size(info.Size())
	}
	ui.Text(c, filepath.Base(p)).Bold().FontSize(15)
	ui.Text(c, kind).FontSize(12).TextColor(t.TextMuted)
	ui.Text(c, "Modified "+info.ModTime().Format("Jan 2, 2006 at 15:04")).FontSize(12).TextColor(t.TextMuted)
	if text, ok := head(p, 20); ok {
		ui.Text(c, text).Font("monospace").FontSize(11).Padding(8).Radius(6).Background(t.Surface).Margin(8, 0, 0, 0)
	}
}

// head returns the first lines of the file at p, if it holds text: UTF-8
// without NULs.
func head(p string, lines int) (string, bool) {
	f, err := os.Open(p)
	if err != nil {
		return "", false
	}
	defer f.Close()
	buf := make([]byte, 4096)
	n, _ := io.ReadFull(f, buf)
	buf = buf[:n]
	if n == 0 || bytes.IndexByte(buf, 0) >= 0 {
		return "", false
	}
	// A rune cut at the end of the buffer is not a sign of binary.
	for len(buf) > 0 && !utf8.Valid(buf) && len(buf) > n-4 {
		buf = buf[:len(buf)-1]
	}
	if !utf8.Valid(buf) {
		return "", false
	}
	all := strings.Split(string(buf), "\n")
	return strings.Join(all[:min(lines, len(all))], "\n"), true
}

// size returns n bytes as Finder writes them, in powers of 1000.
func size(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d bytes", n)
	}
	v, unit := float64(n), ""
	for _, u := range []string{"KB", "MB", "GB", "TB"} {
		v, unit = v/1000, u
		if v < 1000 {
			break
		}
	}
	return fmt.Sprintf("%.1f %s", v, unit)
}

func main() {
	root, err := os.UserHomeDir()
	if len(os.Args) > 1 {
		root, err = filepath.Abs(os.Args[1])
	}
	if err != nil {
		log.Fatal(err)
	}
	a := &app{root: root, entries: map[string][]string{}, dirs: map[string]bool{}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   filepath.Base(root),
			Width:   900,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
