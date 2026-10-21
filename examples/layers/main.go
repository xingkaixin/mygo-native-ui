// Layers edits a landing page as a design tool does: the layers panel on
// the left and the canvas drawing the layers on the right, the front-most
// on top. Dragging layers in the panel restacks them on the canvas, the
// eye hides them, and the selection shows as outlines; a click on the
// canvas selects the layer under it, a group as a whole, and Shift-click
// adds it or takes it away.
//
//	go run ./examples/layers
package main

import (
	"fmt"
	"log"
	"math"
	"slices"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/layers"
)

// shape is where a layer is on the canvas, and how it draws: a frame and
// a rectangle as rounded boxes of fill, an ellipse within its box, text
// at its top left, an image as a picture of hills and a path as a star.
type shape struct {
	x, y, w, h float32
	fill       ui.Color
	radius     float32
	text       string
	size       float32
	weight     int
}

type app struct {
	doc    []layers.Layer
	panel  layers.State
	shapes map[string]shape
	status string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Width(t.Space(66)).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, "Layers").Bold().Padding(t.Space(3), t.Space(3), t.Space(2))
			ui.Column(c).Grow(1).Children(func() {
				a.events(layers.View(c, &a.panel, &a.doc))
			})
			ui.Divider(c)
			ui.Text(c, a.status).FontSize(t.FontSize-2).TextColor(t.TextMuted).SingleLine().Padding(t.Space(2), t.Space(3))
		})
		ui.Divider(c).Vertical()
		a.canvas(c)
	})
}

// events tells what the user last did in the panel.
func (a *app) events(ev layers.Events) {
	if ids, ok := ev.Moved(); ok {
		a.status = "Moved " + a.names(ids)
	}
	if id, ok := ev.Renamed(); ok {
		a.status = "Renamed " + a.names([]string{id})
	}
	if id, ok := ev.Toggled(); ok {
		a.status = "Changed " + a.names([]string{id})
	}
	if ids, ok := ev.Deleted(); ok {
		a.status = fmt.Sprintf("Deleted %d layers", len(ids))
		for _, id := range ids {
			delete(a.shapes, id)
		}
	}
	if ev.SelectionChanged() && !ev.Changed() {
		a.status = fmt.Sprintf("%d selected", len(a.panel.Selected()))
	}
}

// names returns the names of the layers of ids.
func (a *app) names(ids []string) string {
	var names []string
	for _, id := range ids {
		if l := lookup(a.doc, id); l != nil {
			names = append(names, l.Name)
		}
	}
	return strings.Join(names, ", ")
}

// canvas draws the layers, back to front, and the selection's outlines;
// a click selects the layer under it.
func (a *app) canvas(c *ui.Context) {
	t := c.Theme()
	box := ui.Box(c).Grow(1).AlignSelf(ui.Stretch).Background(t.Surface).Clip().KeepFocus().Label("Canvas")
	if box.Clicked() {
		x, y, _ := box.PointerPosition()
		id := a.hit(a.doc, x, y)
		chosen := a.panel.Selected()
		switch {
		case box.ClickModifiers()&ui.Shift == 0 && id == "":
			a.panel.Select()
		case box.ClickModifiers()&ui.Shift == 0:
			a.panel.Select(id)
		case id == "":
		case slices.Contains(chosen, id):
			a.panel.Select(slices.DeleteFunc(slices.Clone(chosen), func(s string) bool { return s == id })...)
		default:
			a.panel.Select(append(slices.Clone(chosen), id)...)
		}
	}
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		a.paint(p, a.doc, r.X, r.Y)
		for _, id := range a.panel.Selected() {
			if b, ok := a.bounds(lookup(a.doc, id)); ok {
				p.Stroke(ui.Rect{X: r.X + b.X - 1, Y: r.Y + b.Y - 1, W: b.W + 2, H: b.H + 2}, t.Accent, 0, 1.5)
			}
		}
	})
}

// paint draws ls, the back-most first, those of frames within them, with
// the canvas's origin at x, y.
func (a *app) paint(p *ui.Painter, ls []layers.Layer, x, y float32) {
	for i := len(ls) - 1; i >= 0; i-- {
		l := &ls[i]
		if l.Hidden {
			continue
		}
		s := a.shapes[l.ID]
		r := ui.Rect{X: x + s.x, Y: y + s.y, W: s.w, H: s.h}
		switch l.Kind {
		case layers.Frame:
			p.Shadow(r, s.radius, 0, 2, 12, 0, ui.RGB(0, 0, 0).Alpha(0.12))
			p.Fill(r, s.fill, s.radius)
			p.Clip(r, s.radius, func() { a.paint(p, l.Children, x, y) })
		case layers.Group:
			a.paint(p, l.Children, x, y)
		case layers.Rectangle:
			p.Fill(r, s.fill, s.radius)
		case layers.Ellipse:
			p.Fill(r, s.fill, min(r.W, r.H)/2)
		case layers.Text:
			p.RichText(r.X, r.Y, r.W, ui.Span{Text: s.text, Size: s.size, Weight: s.weight, Color: s.fill})
		case layers.Image:
			p.Clip(r, s.radius, func() {
				p.FillGradient(r, ui.LinearGradient{Angle: 180, Stops: []ui.GradientStop{
					{Color: ui.Hex("#9ad0f5"), At: 0}, {Color: ui.Hex("#e6f3fb"), At: 1},
				}}, 0)
				p.Fill(ui.Rect{X: r.X - r.W*0.2, Y: r.Y + r.H*0.55, W: r.W * 0.9, H: r.H}, ui.Hex("#5fa37a"), r.W*0.45)
				p.Fill(ui.Rect{X: r.X + r.W*0.35, Y: r.Y + r.H*0.7, W: r.W, H: r.H}, ui.Hex("#3f7d5c"), r.W*0.5)
			})
		case layers.Path:
			p.FillPath(star(r), s.fill)
		}
	}
}

// star returns a five-pointed star within r.
func star(r ui.Rect) *ui.Path {
	var path ui.Path
	cx, cy, out := r.X+r.W/2, r.Y+r.H/2, min(r.W, r.H)/2
	for i := range 10 {
		k := out
		if i%2 == 1 {
			k = out * 0.45
		}
		s, c := sincos(float32(i)*36 - 90)
		if i == 0 {
			path.MoveTo(cx+k*c, cy+k*s)
		} else {
			path.LineTo(cx+k*c, cy+k*s)
		}
	}
	return path.Close()
}

// sincos returns the sine and cosine of deg degrees.
func sincos(deg float32) (float32, float32) {
	s, c := math.Sincos(float64(deg) * math.Pi / 180)
	return float32(s), float32(c)
}

// hit returns the layer of ls at x, y, the front-most: a group as a whole,
// in a frame the layer there or else the frame. Hidden and locked layers
// let the click through.
func (a *app) hit(ls []layers.Layer, x, y float32) string {
	for i := range ls {
		l := &ls[i]
		if l.Hidden {
			continue
		}
		switch b, _ := a.bounds(l); {
		case !(ui.Rect{X: b.X, Y: b.Y, W: b.W, H: b.H}).Contains(x, y):
		case l.Kind == layers.Group:
			if !l.Locked && a.hit(l.Children, x, y) != "" {
				return l.ID
			}
		case l.Kind == layers.Frame:
			if id := a.hit(l.Children, x, y); id != "" {
				return id
			}
			if !l.Locked {
				return l.ID
			}
		case !l.Locked:
			return l.ID
		}
	}
	return ""
}

// bounds returns the box of l on the canvas: a group's around the layers
// in it.
func (a *app) bounds(l *layers.Layer) (ui.Rect, bool) {
	if l == nil {
		return ui.Rect{}, false
	}
	if l.Kind != layers.Group {
		s := a.shapes[l.ID]
		return ui.Rect{X: s.x, Y: s.y, W: s.w, H: s.h}, true
	}
	var box ui.Rect
	found := false
	for i := range l.Children {
		b, ok := a.bounds(&l.Children[i])
		switch {
		case !ok:
		case !found:
			box, found = b, true
		default:
			x0, y0 := min(box.X, b.X), min(box.Y, b.Y)
			x1, y1 := max(box.X+box.W, b.X+b.W), max(box.Y+box.H, b.Y+b.H)
			box = ui.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
		}
	}
	return box, found
}

// lookup returns the layer id of ls, or nil.
func lookup(ls []layers.Layer, id string) *layers.Layer {
	for i := range ls {
		if ls[i].ID == id {
			return &ls[i]
		}
		if l := lookup(ls[i].Children, id); l != nil {
			return l
		}
	}
	return nil
}

func main() {
	a := &app{status: "Drag layers to restack them", shapes: map[string]shape{}}
	n := 0
	add := func(name string, kind layers.Kind, s shape, kids ...layers.Layer) layers.Layer {
		n++
		id := fmt.Sprint(n)
		a.shapes[id] = s
		return layers.Layer{ID: id, Name: name, Kind: kind, Children: kids}
	}
	ink, accent, white := ui.Hex("#1d1d2b"), ui.Hex("#5b5bf0"), ui.Hex("#ffffff")
	a.doc = []layers.Layer{
		add("Hero", layers.Frame, shape{x: 48, y: 48, w: 600, h: 380, fill: ui.Hex("#fbfaf7"), radius: 14},
			add("Headline", layers.Text, shape{x: 88, y: 104, w: 300, h: 80, text: "Ship designs\nfaster", size: 34, weight: 700, fill: ink}),
			add("Subhead", layers.Text, shape{x: 88, y: 196, w: 280, h: 44, text: "Frames, groups and layers that stay in sync with the canvas.", size: 14, fill: ui.Hex("#6b6b7b")}),
			add("Button", layers.Group, shape{},
				add("Label", layers.Text, shape{x: 112, y: 280, w: 100, h: 20, text: "Get started", size: 14, weight: 600, fill: white}),
				add("Fill", layers.Rectangle, shape{x: 88, y: 268, w: 140, h: 44, fill: accent, radius: 10}),
			),
			add("Illustration", layers.Group, shape{},
				add("Star", layers.Path, shape{x: 520, y: 92, w: 64, h: 64, fill: ui.Hex("#ffc93c")}),
				add("Sun", layers.Ellipse, shape{x: 430, y: 150, w: 120, h: 120, fill: ui.Hex("#ff8a5b")}),
				add("Photo", layers.Image, shape{x: 400, y: 210, w: 210, h: 170, radius: 12}),
			),
			add("Grid", layers.Rectangle, shape{x: 48, y: 48, w: 600, h: 380, fill: accent.Alpha(0.06)}),
		),
		add("Card", layers.Frame, shape{x: 688, y: 48, w: 240, h: 300, fill: white, radius: 14},
			add("Title", layers.Text, shape{x: 708, y: 224, w: 200, h: 24, text: "Weekly review", size: 17, weight: 650, fill: ink}),
			add("Caption", layers.Text, shape{x: 708, y: 252, w: 200, h: 40, text: "12 frames changed", size: 13, fill: ui.Hex("#6b6b7b")}),
			add("Badge", layers.Ellipse, shape{x: 888, y: 64, w: 24, h: 24, fill: ui.Hex("#2fbf71")}),
			add("Cover", layers.Image, shape{x: 688, y: 48, w: 240, h: 160}),
		),
	}
	a.doc[0].Children[4].Hidden = true
	a.doc[1].Children[3].Locked = true
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Layers",
			Width:   1240,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
