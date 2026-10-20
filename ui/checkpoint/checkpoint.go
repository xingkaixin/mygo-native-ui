// Package checkpoint marks a point in a chat or an agent's transcript to
// go back to, as Cursor and Claude Code keep one after each turn: a thin
// rule with a flag, "Checkpoint", its time and what it covers, "3 files
// changed", and while the pointer is over it, or the keyboard focus in
// it, a button restoring it. Restore asks first, in place, with Restore
// and Cancel; Escape cancels.
//
//	if checkpoint.View(c, turn.at, "3 files changed").Restored() {
//		app.turns = app.turns[:i+1]
//		app.undoFiles(i)
//	}
//
// The app keeps the turns and what they changed, and drops those after
// the checkpoint in the frame the user restores it.
package checkpoint

import (
	"time"

	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
)

// Events are what the user did to a checkpoint in a frame. The zero
// Events report nothing.
type Events struct {
	restored bool
}

// Restored reports that the user chose to restore the checkpoint, for the
// app to drop what came after it.
func (e Events) Restored() bool { return e.restored }

// View creates the checkpoint made at at, covering summary, which may be
// "".
func View(c *ui.Context, at time.Time, summary string) Events {
	t := c.Theme()
	var ev Events
	small := t.FontSize - 1
	box := ui.Column(c).Gap(t.Space(1.5)).Role(ui.RoleGroup).Label("Checkpoint")
	asking := ui.Local(box, "checkpoint.asking", func() bool { return false })
	box.Children(func() {
		// As high as the button, so that the line keeps its height as the
		// button comes and goes.
		line := ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).MinHeight(t.Space(7))
		show := (line.Hovered() || line.FocusWithin()) && !*asking
		line.Children(func() {
			ui.Box(c).Size(t.Space(3.5), t.Space(3.5)).Draw(func(p *ui.Painter, r ui.Rect) { flag(p, r, t.TextMuted) })
			ui.Text(c, "Checkpoint").FontSize(small).FontWeight(600).TextColor(t.TextMuted)
			r := c.Region()
			when := i18n.FormatDate(r.FormatLocale(c.Locale()), at, i18n.DateOptions{Time: i18n.ShortStyle, HourCycle: r.HourCycle})
			if summary != "" {
				when += " · " + summary
			}
			ui.Text(c, when).FontSize(small).TextColor(t.TextMuted).FontFeatures("tnum").SingleLine().Shrink(1)
			ui.Box(c).Height(1).Grow(1).Background(t.Border)
			b := ui.ButtonBase(c).Padding(t.Space(1), t.Space(2)).Radius(t.Space(1.5)).Label("Restore checkpoint")
			if b.Hovered() {
				b.Background(t.Text.Alpha(0.06))
			}
			// Hidden, the button leaves the row, for the rule to run to its
			// end, as the card below does, and still takes the focus.
			if !show {
				b.Opacity(0).Absolute().Right(0).Top(0)
			}
			if b.Clicked() {
				*asking = true
			}
			b.Children(func() {
				ui.Text(c, "Restore").FontSize(small).TextColor(t.TextMuted)
			})
		})
		if !*asking {
			return
		}
		card := ui.Row(c).Gap(t.Space(2)).Padding(t.Space(2), t.Space(2), t.Space(2), t.Space(3)).AlignItems(ui.Center).
			Radius(t.Space(2)).Border(1, t.Border).Background(t.Surface).Role(ui.RoleAlertDialog).Label("Restore to here?")
		if card.Shortcut(0, ui.KeyEscape) {
			*asking = false
		}
		card.Children(func() {
			ui.Text(c, "Restore to here? Later messages and file changes will be undone.").FontSize(small).Grow(1).Shrink(1)
			// Cancel takes the focus, as it does in the system's alerts about
			// what cannot be undone, so that Enter does not undo by accident.
			if ui.Button(c, "Cancel").AutoFocus().Clicked() {
				*asking = false
			}
			restore := ui.ButtonBase(c).Padding(t.Space(1.5), t.Space(3.5)).Radius(t.Radius).Background(t.Danger).Label("Restore")
			if restore.Hovered() {
				restore.Opacity(0.9)
			}
			if restore.Clicked() {
				*asking, ev.restored = false, true
			}
			restore.Children(func() {
				ui.Text(c, "Restore").TextColor(ui.RGB(255, 255, 255))
			})
		})
	})
	return ev
}

// flag draws a flag on a pole in the middle of r.
func flag(p *ui.Painter, r ui.Rect, color ui.Color) {
	x, y, w, h := r.X+r.W*0.2, r.Y+r.H*0.1, r.W*0.65, r.H*0.8
	p.StrokePolyline([][2]float32{{x, y + h}, {x, y}}, 1.5, color)
	var cloth ui.Path
	cloth.MoveTo(x, y).LineTo(x+w, y+h*0.25).LineTo(x, y+h*0.5).Close()
	p.FillPath(&cloth, color)
}
