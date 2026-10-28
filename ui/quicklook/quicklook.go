// Package quicklook previews files over the window, as Quick Look does
// when Space is pressed on a file: a picture fitted to the panel with its
// size in pixels, a text file with numbered lines, and anything else as
// its kind's icon with its name, size and date. The title bar names the
// file and its place among the files, with buttons to go back and forth,
// as the arrow keys do, and to open the file in its app. Space, Escape
// and the close button close the panel.
//
//	if c.Shortcut(0, ui.KeySpace) {
//		app.looking = true
//	}
//	ev := quicklook.View(c, &app.look, &app.looking, app.files, &app.selected)
//	if ev.Open {
//		open(app.files[app.selected])
//	}
package quicklook

import (
	"bytes"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// File is a file to preview: its name, when it was last changed, and
// what it holds.
type File struct {
	Name     string
	Modified time.Time
	Data     []byte
}

// State is what a panel keeps: the pictures decoded so far, by name.
// Keep it in the app's state, one for each panel.
type State struct {
	pictures map[string]*ui.Bitmap
}

// Events is a panel, and whether Open was clicked in it in the frame.
type Events struct {
	ui.Element
	Open bool
}

func icon(body string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` + body + `</svg>`))
}

var (
	backIcon    = icon(`<path d="M15 6l-6 6 6 6"/>`)
	forwardIcon = icon(`<path d="M9 6l6 6-6 6"/>`)
	closeIcon   = icon(`<path d="M7 7l10 10M17 7L7 17"/>`)
)

// picture returns f decoded as a picture, nil when it is not one.
func (s *State) picture(f File) *ui.Bitmap {
	if s.pictures == nil {
		s.pictures = map[string]*ui.Bitmap{}
	}
	key := f.Name + f.Modified.String()
	if b, ok := s.pictures[key]; ok {
		return b
	}
	b, err := ui.DecodeBitmap(f.Data)
	if err != nil {
		b = nil
	}
	s.pictures[key] = b
	return b
}

// text reports whether f reads as text: valid UTF-8 with no NUL bytes.
func text(f File) bool {
	head := f.Data[:min(len(f.Data), 8192)]
	return !bytes.ContainsRune(head, 0) && utf8.Valid(f.Data)
}

// View creates the panel over the window while *open, previewing
// files[*index].
func View(c *ui.Context, s *State, open *bool, files []File, index *int) Events {
	t := c.Theme()
	var ev Events
	if len(files) == 0 {
		*open = false
	}
	*index = max(0, min(*index, len(files)-1))
	ev.Element = ui.DialogBase(c, open, func(backdrop, panel ui.Element) {
		f := files[*index]
		pic := s.picture(f)
		backdrop.Background(ui.RGBA(0, 0, 0, 0.25))
		panel.Column().Size(t.Space(170), t.Space(125)).Radius(t.Space(3)).Background(t.Background).Clip().
			Shadow(0, 16, 48, 0, ui.RGBA(0, 0, 0, 0.35)).Label("Quick Look: " + f.Name)
		step := func(d int) {
			*index = (*index + d + len(files)) % len(files)
		}
		switch {
		case panel.Shortcut(0, ui.KeyLeft), panel.Shortcut(0, ui.KeyUp):
			step(-1)
		case panel.Shortcut(0, ui.KeyRight), panel.Shortcut(0, ui.KeyDown):
			step(1)
		case panel.Shortcut(0, ui.KeySpace):
			*open = false
		}
		ui.Row(c).Gap(t.Space(1)).Padding(t.Space(2), t.Space(3)).AlignItems(ui.Center).Background(t.Surface).Children(func() {
			if button(c, closeIcon, "Close").Clicked() {
				*open = false
			}
			if button(c, backIcon, "Previous").Disabled(len(files) < 2).Clicked() {
				step(-1)
			}
			if button(c, forwardIcon, "Next").Disabled(len(files) < 2).Clicked() {
				step(1)
			}
			ui.Column(c).Grow(1).AlignItems(ui.Center).Children(func() {
				ui.Text(c, f.Name).Bold().SingleLine().Ellipsis("…")
				ui.Text(c, fmt.Sprintf("%d of %d", *index+1, len(files))).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
			})
			if ui.Button(c, "Open").Clicked() {
				ev.Open = true
			}
		})
		ui.Divider(c)
		ui.Column(c).Grow(1).Basis(0).Children(func() {
			switch {
			case pic != nil:
				w, h := pic.Size()
				ui.Image(c, pic).Grow(1).Basis(0).FillWidth().Fit(ui.ScaleDown).Margin(t.Space(4))
				footer(c, fmt.Sprintf("%s image · %d × %d · %s", kind(f.Name), w, h, size(len(f.Data))), f.Modified)
			case text(f):
				lines(c, string(f.Data))
				footer(c, fmt.Sprintf("%s · %s", kind(f.Name), size(len(f.Data))), f.Modified)
			default:
				other(c, f)
			}
		})
	})
	return ev
}

func button(c *ui.Context, ic *ui.SVG, label string) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c.Key(label)).Size(t.Space(7), t.Space(7)).Radius(t.Space(1.5)).Center().Label(label).Tooltip(label)
	if b.Hovered() && !b.IsDisabled() {
		b.Background(t.SurfaceHover)
	}
	col := t.Text
	if b.IsDisabled() {
		col = t.TextMuted.Alpha(0.5)
	}
	b.Children(func() { ui.Icon(c, ic).Size(t.Space(4), t.Space(4)).TextColor(col) })
	return b
}

// lines creates text with its lines numbered, its first few thousand.
func lines(c *ui.Context, s string) {
	t := c.Theme()
	all := strings.Split(strings.TrimRight(s, "\n"), "\n")
	shown := all[:min(len(all), 3000)]
	digits := len(strconv.Itoa(len(shown)))
	var nums strings.Builder
	for i := range shown {
		fmt.Fprintf(&nums, "%*d\n", digits, i+1)
	}
	ui.ScrollBoth(c).Grow(1).Basis(0).Children(func() {
		ui.Row(c).Gap(t.Space(4)).Padding(t.Space(3), t.Space(4)).Children(func() {
			ui.Text(c, strings.TrimRight(nums.String(), "\n")).Font("monospace").FontSize(t.FontSize - 1).TextColor(t.TextMuted).TextAlign(ui.End).Unselectable()
			ui.Text(c, strings.Join(shown, "\n")).Font("monospace").FontSize(t.FontSize - 1).NoWrap().Selectable()
		})
	})
}

// other creates the icon of a file that has no preview, and its details.
func other(c *ui.Context, f File) {
	t := c.Theme()
	ui.Column(c).Grow(1).Gap(t.Space(3)).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		ext := strings.ToUpper(strings.TrimPrefix(path.Ext(f.Name), "."))
		ui.Box(c).Size(t.Space(24), t.Space(30)).Draw(func(p *ui.Painter, r ui.Rect) {
			fold := r.W * 0.28
			var page ui.Path
			page.MoveTo(r.X, r.Y).LineTo(r.X+r.W-fold, r.Y).LineTo(r.X+r.W, r.Y+fold).LineTo(r.X+r.W, r.Y+r.H).LineTo(r.X, r.Y+r.H).Close()
			p.FillPath(&page, t.Surface)
			p.StrokePath(&page, 1.5, t.Border)
			var corner ui.Path
			corner.MoveTo(r.X+r.W-fold, r.Y).LineTo(r.X+r.W-fold, r.Y+fold).LineTo(r.X+r.W, r.Y+fold)
			p.StrokePath(&corner, 1.5, t.Border)
			if ext != "" {
				label := ui.Span{Text: ext, Size: r.W * 0.2, Weight: 700, Color: t.Accent}
				w, _ := p.MeasureText(0, label)
				p.RichText(r.X+(r.W-w)/2, r.Y+r.H*0.55, 0, label)
			}
		})
		ui.Text(c, f.Name).Bold().FontSize(t.FontSize + 2)
		ui.Text(c, kind(f.Name)+" · "+size(len(f.Data))).TextColor(t.TextMuted)
		ui.Text(c, "Modified "+f.Modified.Format("Jan 2, 2006 at 3:04 PM")).TextColor(t.TextMuted)
	})
}

func footer(c *ui.Context, info string, modified time.Time) {
	t := c.Theme()
	ui.Row(c).Padding(t.Space(2), t.Space(4)).Background(t.Surface).Children(func() {
		ui.Text(c, info).FontSize(t.FontSize - 1).TextColor(t.TextMuted).Grow(1)
		ui.Text(c, "Modified "+modified.Format("Jan 2, 2006")).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
	})
}

// kind names a file's kind by its extension, as "PNG" or "Go source".
func kind(name string) string {
	ext := strings.ToLower(strings.TrimPrefix(path.Ext(name), "."))
	if k, ok := map[string]string{"go": "Go source", "md": "Markdown text", "txt": "Plain text", "json": "JSON", "zip": "ZIP archive", "pdf": "PDF document"}[ext]; ok {
		return k
	}
	if ext == "" {
		return "Document"
	}
	return strings.ToUpper(ext)
}

func size(n int) string {
	switch {
	case n < 1000:
		return fmt.Sprintf("%d bytes", n)
	case n < 1000*1000:
		return fmt.Sprintf("%.0f KB", float64(n)/1000)
	}
	return fmt.Sprintf("%.1f MB", float64(n)/1e6)
}
