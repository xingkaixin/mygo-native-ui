// Package lightbox shows photos over the whole window, one at a time, as
// a gallery's lightbox: the photo on a dark backdrop, fitted to the room
// there is, its caption and where it is in the set, buttons to the
// previous and next ones, and a strip of thumbnails to jump to any. The
// photo zooms and pans as zoomview.View does; a new photo starts fitted.
//
//	if thumb.Clicked() {
//		app.open, app.index = true, i
//	}
//	lightbox.Show(c, &app.lightbox, &app.open, &app.index, app.photos)
package lightbox

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/zoomview"
)

// Photo is a picture the lightbox shows, with its caption.
type Photo struct {
	Image   *ui.Bitmap
	Caption string
}

// State is what a lightbox keeps: how its photo shows, and which photo
// that is. Keep it in the app's state, one for each lightbox.
type State struct {
	zoom  zoomview.State
	shown int
}

// Show shows photos[*index] over the window while *open. Left and Right,
// the buttons beside the photo and the thumbnails change *index; Escape
// and the close button set *open to false. Cmd+= and Cmd+- zoom in and
// out, and Cmd+0 fits the photo. While it shows it keeps the keyboard,
// and what is behind it takes neither the pointer nor the window's
// shortcuts. It returns the lightbox, or a zero Element while it does
// not show.
func Show(c *ui.Context, s *State, open *bool, index *int, photos []Photo) ui.Element {
	if !*open || len(photos) == 0 {
		return ui.Element{}
	}
	i := max(0, min(*index, len(photos)-1))
	*index = i
	moved := s.shown != i+1
	if moved {
		s.shown, s.zoom = i+1, zoomview.State{}
	}
	t := c.Theme()
	light := ui.RGB(255, 255, 255)
	ww, _ := c.Size()
	photo := photos[i]
	var back ui.Element
	ui.Overlay(c, func() {
		back = ui.Column(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).Modal().
			Background(ui.RGBA(10, 10, 12, 0.97)).Role(ui.RoleDialog).Label(photo.Caption)
		back.Opacity(back.Keyframes("in", []ui.Keyframe{{Value: 0}, {At: 150 * time.Millisecond, Value: 1}}))
		for _, k := range []struct {
			mods ui.Modifiers
			key  ui.Key
			do   func()
		}{
			{0, ui.KeyEscape, func() { *open = false }},
			{0, ui.KeyLeft, func() { *index = max(0, i-1) }},
			{0, ui.KeyRight, func() { *index = min(len(photos)-1, i+1) }},
			{ui.Cmd, ui.KeyEqual, s.zoom.ZoomIn},
			{ui.Cmd, ui.KeyMinus, s.zoom.ZoomOut},
			{ui.Cmd, ui.Key0, s.zoom.Fit},
		} {
			if back.OverlayShortcut(k.mods, k.key) {
				k.do()
			}
		}
		back.Children(func() {
			ui.Row(c).Padding(t.Space(3), t.Space(4)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, fmt.Sprintf("%d / %d", i+1, len(photos))).TextColor(light.Alpha(0.6)).FontFeatures("tnum").Grow(1)
				if button(c, "✕", "Close", false).Clicked() {
					*open = false
				}
			})
			ui.Row(c).Grow(1).AlignItems(ui.Stretch).Padding(0, t.Space(2)).Gap(t.Space(2)).Children(func() {
				if button(c, "‹", "Previous", i == 0).Clicked() {
					*index = i - 1
				}
				w, h := photo.Image.Size()
				zoomview.View(c.Key(i), &s.zoom, float32(w), float32(h), func() {
					ui.Image(c, photo.Image).Fill()
				}).Grow(1).Label(photo.Caption)
				if button(c, "›", "Next", i == len(photos)-1).Clicked() {
					*index = i + 1
				}
			})
			ui.Text(c, photo.Caption).TextColor(light.Alpha(0.85)).TextAlign(ui.Center).Padding(t.Space(3), t.Space(4), t.Space(2), t.Space(4))
			ui.ScrollHorizontal(c).Padding(0, 0, t.Space(4), 0).Children(func() {
				ui.Row(c).Gap(t.Space(1.5)).Padding(t.Space(1), t.Space(4)).Justify(ui.Center).MinWidth(ww).Children(func() {
					for j, p := range photos {
						th := ui.Box(c.Key(j)).Size(64, 44).Radius(4).Clip().Cursor(ui.CursorPointer).Label(p.Caption)
						if j == i {
							th.Border(2, light)
						} else if th.Hovered() {
							th.Opacity(0.9)
						} else {
							th.Opacity(0.5)
						}
						if th.Clicked() {
							*index = j
						}
						if j == i && moved {
							th.ScrollIntoView()
						}
						th.Children(func() {
							ui.Image(c, p.Image).Fit(ui.Cover).Fill()
						})
					}
				})
			})
		})
	})
	return back
}

// button creates a round button over the backdrop, showing text, named
// label, dim while disabled.
func button(c *ui.Context, text, label string, disabled bool) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Key(label).Size(t.Space(10), t.Space(10)).Radius(t.Space(5)).Label(label).
		Justify(ui.Center).AlignItems(ui.Center).AlignSelf(ui.Center).Disabled(disabled)
	color := ui.RGB(255, 255, 255)
	if disabled {
		color = color.Alpha(0.25)
	} else if b.Hovered() {
		b.Background(ui.RGBA(255, 255, 255, 0.12))
	}
	b.Children(func() {
		ui.Text(c, text).FontSize(t.FontSize + 8).TextColor(color)
	})
	return b
}
