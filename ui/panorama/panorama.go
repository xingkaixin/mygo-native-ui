// Package panorama shows a 360° photo, as panorama viewers and virtual
// tours do: the view from the middle of an equirectangular picture, two
// times as wide as tall, drawn by a shader as a camera sees it. Dragging
// looks around, going on for a moment once let go; the wheel and a pinch
// zoom in and out between 30° and 100° across; and the arrows turn while
// the view has the focus. Hotspots pin labels to places in the picture,
// and a click on one reports it, as a door to the next room. A compass
// shows where the view faces.
//
//	ev := panorama.View(c, &app.view, app.photo, app.doors)
//	if ev.Chosen >= 0 {
//		app.enter(app.doors[ev.Chosen])
//	}
package panorama

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/shader"
)

// Hotspot is a label pinned to a place of the picture: Yaw degrees right
// of its middle and Pitch degrees above its horizon.
type Hotspot struct {
	Yaw, Pitch float64
	Label      string
}

// Events are what happened in the view since the last frame.
type Events struct {
	ui.Element
	// Chosen is the hotspot clicked, or -1.
	Chosen int
}

// State is where a view looks: the yaw and pitch it faces, in degrees,
// and how wide it sees, with how fast it turns once let go. Keep it in
// the app's state, one for each view.
type State struct {
	yaw, pitch, fov float64
	// vyaw, vpitch are how fast the view turns, in degrees a second, as
	// of at.
	vyaw, vpitch float64
	at           time.Time
}

// Look faces yaw degrees right of the picture's middle and pitch degrees
// above its horizon.
func (s *State) Look(yaw, pitch float64) {
	s.vyaw, s.vpitch = 0, 0
	s.turn(yaw-s.yaw, pitch-s.pitch)
}

// Yaw returns the yaw the view faces, in degrees from -180 to 180.
func (s *State) Yaw() float64 { return s.yaw }

const (
	minFov, maxFov, startFov = 30, 100, 75
	// glide is how long, in seconds, a view let go takes to turn e times
	// slower.
	glide = 0.35
)

func (s *State) turn(dyaw, dpitch float64) {
	s.yaw = math.Remainder(s.yaw+dyaw, 360)
	s.pitch = max(-89, min(89, s.pitch+dpitch))
}

func (s *State) coast(now time.Time) bool {
	d := math.Exp(-now.Sub(s.at).Seconds() / glide)
	s.at = now
	s.turn(s.vyaw*glide*(1-d), s.vpitch*glide*(1-d))
	s.vyaw, s.vpitch = s.vyaw*d, s.vpitch*d
	if math.Hypot(s.vyaw, s.vpitch) < 1 {
		s.vyaw, s.vpitch = 0, 0
	}
	return s.vyaw != 0 || s.vpitch != 0
}

// sphere draws the view: for each pixel the ray through it, turned by
// the pitch and yaw, and the picture's pixel at its longitude and
// latitude.
var sphere = shader.MustCompile(`package main

var Yaw float
var Pitch float

// Focal is how far the eye is from the view, in DIPs.
var Focal float

func Fragment(dst vec4, pos vec2) vec4 {
	p := pos - rectSize()/2
	d := normalize(vec3(p.x, -p.y, Focal))
	cp := cos(Pitch)
	sp := sin(Pitch)
	d = vec3(d.x, d.y*cp+d.z*sp, -d.y*sp+d.z*cp)
	cy := cos(Yaw)
	sy := sin(Yaw)
	d = vec3(d.x*cy+d.z*sy, d.y, -d.x*sy+d.z*cy)
	lon := atan2(d.x, d.z)
	lat := asin(clamp(d.y, -1, 1))
	size := image0Size()
	uv := vec2((lon/6.2831853+0.5)*size.x, (0.5-lat/3.1415927)*size.y)
	// The pixels at the seam blend with nothing beyond the picture's
	// edges; half a pixel in, they blend with its own.
	uv = clamp(uv, vec2(0.5), size-0.5)
	return image0Sample(uv)
}
`)

var uYaw, uPitch, uFocal = sphere.Uniform("Yaw"), sphere.Uniform("Pitch"), sphere.Uniform("Focal")

// focal returns how far the eye is from a view h tall seeing fov degrees
// up and down.
func focal(h float32, fov float64) float32 {
	return h / 2 / float32(math.Tan(fov*math.Pi/360))
}

// project returns where yaw and pitch, in degrees, show in a view of
// size w by h, and whether they are in front.
func (s *State) project(yaw, pitch float64, w, h float32) (float32, float32, bool) {
	ly, lp := yaw*math.Pi/180, pitch*math.Pi/180
	v := [3]float64{math.Cos(lp) * math.Sin(ly), math.Sin(lp), math.Cos(lp) * math.Cos(ly)}
	sy, cy := math.Sincos(-s.yaw * math.Pi / 180)
	v = [3]float64{v[0]*cy + v[2]*sy, v[1], -v[0]*sy + v[2]*cy}
	sp, cp := math.Sincos(-s.pitch * math.Pi / 180)
	v = [3]float64{v[0], v[1]*cp + v[2]*sp, -v[1]*sp + v[2]*cp}
	if v[2] <= 0.01 {
		return 0, 0, false
	}
	f := float64(focal(h, s.fov))
	return w/2 + float32(v[0]/v[2]*f), h/2 - float32(v[1]/v[2]*f), true
}

// View creates the view of photo, an equirectangular picture, with the
// hotspots on it. Give it a size, or grow it in its parent.
func View(c *ui.Context, s *State, photo *ui.Bitmap, spots []Hotspot) Events {
	t := c.Theme()
	if s.fov == 0 {
		s.fov = startFov
	}
	ev := Events{Chosen: -1}
	view := ui.Box(c).Clip().Focusable().Background(ui.RGB(0, 0, 0)).Cursor(ui.CursorGrab).
		Role(ui.RoleImage).Label("Panorama").Value(compassPoint(s.yaw))
	b := view.LayoutBounds()
	// A DIP dragged turns the view as far as it moves the picture under
	// the pointer.
	k := s.fov / float64(max(1, b.H))
	if dx, dy, ok := view.Dragged(); ok {
		s.turn(-float64(dx)*k, float64(dy)*k)
		view.Cursor(ui.CursorGrabbing)
	}
	if view.Pressed() {
		s.vyaw, s.vpitch = 0, 0
	}
	now := c.Now()
	if vx, vy, ok := view.DragEnded(); ok && !c.Preferences().ReduceMotion {
		s.vyaw, s.vpitch, s.at = -float64(vx)*k, float64(vy)*k, now
	}
	if (s.vyaw != 0 || s.vpitch != 0) && s.coast(now) {
		c.AnimationFrame()
	}
	view.HandleInput(func(in ui.InputEvent) bool {
		switch in.Kind {
		case ui.InputScroll:
			s.fov = max(minFov, min(maxFov, s.fov*math.Exp2(float64(in.DY)/400)))
		case ui.InputZoom:
			s.fov = max(minFov, min(maxFov, s.fov/float64(max(0.01, in.Scale))))
		default:
			return false
		}
		return true
	})
	for _, key := range []struct {
		key        ui.Key
		yaw, pitch float64
	}{{ui.KeyLeft, -10, 0}, {ui.KeyRight, 10, 0}, {ui.KeyUp, 0, 10}, {ui.KeyDown, 0, -10}} {
		if view.Shortcut(0, key.key) {
			s.turn(key.yaw, key.pitch)
		}
	}
	if view.Shortcut(0, ui.KeyEqual) {
		s.fov = max(minFov, s.fov/1.25)
	}
	if view.Shortcut(0, ui.KeyMinus) {
		s.fov = min(maxFov, s.fov*1.25)
	}
	yaw, pitch, fov := s.yaw, s.pitch, s.fov
	view.Children(func() {
		for i, h := range spots {
			x, y, front := s.project(h.Yaw, h.Pitch, b.W, b.H)
			if !front {
				continue
			}
			pin := ui.ButtonBase(c.Key(i)).Absolute().Left(x).Top(y).Translate(-t.Space(3), -t.Space(3)).
				Label(h.Label).Tooltip(h.Label)
			if pin.Clicked() {
				ev.Chosen = i
			}
			pin.Row().Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				ring := ui.Box(c).Size(t.Space(6), t.Space(6)).Radius(t.Space(3)).Border(2, ui.RGB(255, 255, 255)).
					Background(ui.RGBA(0, 0, 0, 0.35)).Shadow(0, 1, 4, 0, ui.RGBA(0, 0, 0, 0.5))
				if pin.Hovered() {
					ring.Background(t.Accent)
				}
				ui.Text(c, h.Label).TextColor(ui.RGB(255, 255, 255)).Padding(t.Space(0.5), t.Space(2)).
					Radius(t.Space(1)).Background(ui.RGBA(0, 0, 0, 0.55)).SingleLine()
			})
		}
		ui.Box(c).Absolute().Left(t.Space(3)).Bottom(t.Space(3)).Size(t.Space(12), t.Space(12)).PassThrough().
			Draw(func(p *ui.Painter, r ui.Rect) { compass(p, r, yaw, fov) })
	})
	view.Draw(func(p *ui.Painter, r ui.Rect) {
		if photo == nil {
			return
		}
		v := sphere.NewValues().Float(uYaw, float32(yaw*math.Pi/180)).Float(uPitch, float32(pitch*math.Pi/180)).
			Float(uFocal, focal(r.H, fov))
		sphere.Draw(p, r, shader.Options{Values: v, Images: [4]*ui.Bitmap{photo}})
	})
	ev.Element = view
	return ev
}

// compass draws a dial in r with the picture's middle at its top and a
// wedge as wide as the view, turned to where it faces.
func compass(p *ui.Painter, r ui.Rect, yaw, fov float64) {
	cx, cy, rad := r.X+r.W/2, r.Y+r.H/2, r.W/2
	var disc ui.Path
	p.FillPath(disc.Circle(cx, cy, rad), ui.RGBA(0, 0, 0, 0.45))
	var wedge ui.Path
	wedge.MoveTo(cx, cy)
	for i := 0; i <= 12; i++ {
		a := (yaw - fov/2 + fov*float64(i)/12 - 90) * math.Pi / 180
		wedge.LineTo(cx+rad*0.85*float32(math.Cos(a)), cy+rad*0.85*float32(math.Sin(a)))
	}
	p.FillPath(wedge.Close(), ui.RGBA(255, 255, 255, 0.55))
	p.Fill(ui.Rect{X: cx - 1, Y: r.Y + 2, W: 2, H: rad * 0.3}, ui.RGB(255, 90, 70), 1)
}

// compassPoint names the way yaw faces, as "north-east", the picture's
// middle north.
func compassPoint(yaw float64) string {
	points := []string{"north", "north-east", "east", "south-east", "south", "south-west", "west", "north-west"}
	return "Facing " + points[int(math.Round(math.Mod(yaw+360, 360)/45))%8]
}
