// Package keypoints labels the poses of people in a picture, as the
// keypoint datasets of pose estimators are made and pose editors set out
// a figure: each person a skeleton of named joints joined by bones, the
// person's left side in one color and right in another, over the picture.
//
// Dragging a joint moves it, and dragging a bone moves the whole person.
// A click chooses a joint, and its name shows by it. V turns the joint
// chosen between seen and hidden behind something, a hidden joint hollow
// and its bones dashed; Delete takes it out, and a click on the picture
// puts back the first joint missing, or the one chosen in the list beside
// the picture. F swaps the person's left and right, N adds a person in
// the skeleton's rest pose, and Escape lets the joint go. Changed reports
// an edit.
//
//	keypoints.View(c, &app.labeler, app.photo, keypoints.COCO, &app.poses).Grow(1)
package keypoints

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Visibility is whether a joint is labeled and seen, numbered as COCO's
// keypoints number it.
type Visibility uint8

const (
	Missing Visibility = iota
	Hidden
	Visible
)

// Joint is where a joint is, in the picture's pixels, and whether it is
// seen.
type Joint struct {
	X, Y float32
	V    Visibility
}

// Skeleton is the joints a person has and the bones between them. Names
// starting "Left " and "Right " are on those sides of the body; Rest is
// where each joint stands in a person one unit tall, facing the viewer, x
// from the middle and y down from the top of the head.
type Skeleton struct {
	Names []string
	Bones [][2]int
	Rest  [][2]float32
}

// COCO is the 17 keypoints of the COCO dataset.
var COCO = Skeleton{
	Names: []string{"Nose", "Left eye", "Right eye", "Left ear", "Right ear",
		"Left shoulder", "Right shoulder", "Left elbow", "Right elbow", "Left wrist", "Right wrist",
		"Left hip", "Right hip", "Left knee", "Right knee", "Left ankle", "Right ankle"},
	Bones: [][2]int{{15, 13}, {13, 11}, {16, 14}, {14, 12}, {11, 12}, {5, 11}, {6, 12}, {5, 6},
		{5, 7}, {6, 8}, {7, 9}, {8, 10}, {1, 2}, {0, 1}, {0, 2}, {1, 3}, {2, 4}, {3, 5}, {4, 6}},
	Rest: [][2]float32{{0, 0.06}, {0.025, 0.045}, {-0.025, 0.045}, {0.05, 0.055}, {-0.05, 0.055},
		{0.11, 0.19}, {-0.11, 0.19}, {0.15, 0.33}, {-0.15, 0.33}, {0.17, 0.46}, {-0.17, 0.46},
		{0.07, 0.5}, {-0.07, 0.5}, {0.08, 0.72}, {-0.08, 0.72}, {0.08, 0.94}, {-0.08, 0.94}},
}

// side returns -1 for a joint on the left of the body, 1 on the right and
// 0 in the middle.
func (sk Skeleton) side(i int) int {
	switch {
	case strings.HasPrefix(sk.Names[i], "Left "):
		return -1
	case strings.HasPrefix(sk.Names[i], "Right "):
		return 1
	}
	return 0
}

// mirror returns the joint on the other side of the body from i, or i.
func (sk Skeleton) mirror(i int) int {
	name := sk.Names[i]
	switch {
	case strings.HasPrefix(name, "Left "):
		name = "Right " + name[5:]
	case strings.HasPrefix(name, "Right "):
		name = "Left " + name[6:]
	}
	if j := slices.Index(sk.Names, name); j >= 0 {
		return j
	}
	return i
}

// Pose returns a person in the skeleton's rest pose, height pixels tall,
// the top of the head at x, y.
func (sk Skeleton) Pose(x, y, height float32) []Joint {
	p := make([]Joint, len(sk.Names))
	for i, at := range sk.Rest {
		p[i] = Joint{x + at[0]*height, y + at[1]*height, Visible}
	}
	return p
}

var (
	leftColor   = ui.Hex("#339AF0")
	rightColor  = ui.Hex("#FF922B")
	middleColor = ui.Hex("#51CF66")
)

func (sk Skeleton) color(sides ...int) ui.Color {
	sum := 0
	for _, s := range sides {
		sum += s
	}
	switch {
	case sum == -len(sides):
		return leftColor
	case sum == len(sides):
		return rightColor
	}
	return middleColor
}

type mode uint8

const (
	idle mode = iota
	moveJoint
	movePose
)

// State is what a labeler keeps: the person and joint chosen, the joint
// to place next, and what a drag is doing. Keep it in the app's state, one
// for each labeler.
type State struct {
	pose, joint int
	chosen      bool
	// next is the joint a click places, plus 1, so that the zero State
	// places the first one missing.
	next int
	mode mode
	from [2]float32
	was  []Joint
	down bool
}

// View creates the labeler of *poses, each a joint for each of sk's, in
// pic.
func View(c *ui.Context, s *State, pic *ui.Bitmap, sk Skeleton, poses *[][]Joint) ui.Element {
	t := c.Theme()
	iw, ih := pic.Size()
	changed := false
	ps := *poses
	if s.pose >= len(ps) {
		s.pose, s.chosen, s.next = max(0, len(ps)-1), false, 0
	}
	// place is the joint a click on the picture places: the one chosen in
	// the list, or the first missing.
	place := -1
	if s.pose < len(ps) {
		pose := ps[s.pose]
		if s.next > 0 && s.next <= len(pose) && pose[s.next-1].V == Missing {
			place = s.next - 1
		} else {
			place = slices.IndexFunc(pose, func(j Joint) bool { return j.V == Missing })
		}
	}
	add := func() {
		h := float32(ih) * 0.7
		*poses = append(*poses, sk.Pose(float32(iw)/2, float32(ih)*0.15, h))
		s.pose, s.joint, s.chosen, s.next = len(*poses)-1, 0, false, 0
		changed = true
	}
	root := ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Stretch)
	root.Children(func() {
		canvas := ui.Box(c).Grow(1).MinHeight(t.Space(40)).Radius(t.Space(2)).Background(t.Surface).Clip().
			Focusable().FocusRing(false).AutoFocus().Role(ui.RoleImage).
			Label(fmt.Sprintf("Picture, %d people", len(ps)))
		b := canvas.Bounds()
		k := min(b.W/float32(iw), b.H/float32(ih))
		ox, oy := (b.W-float32(iw)*k)/2, (b.H-float32(ih)*k)/2
		lx, ly, over := canvas.PointerPosition()
		px, py := (lx-ox)/k, (ly-oy)/k
		inside := px >= 0 && py >= 0 && px <= float32(iw) && py <= float32(ih)
		hitPose, hitJoint := hit(*poses, sk, px, py, 9/k)
		hitBone := -1
		if hitJoint < 0 {
			hitBone = bone(*poses, sk, px, py, 5/k)
		}
		switch {
		case s.mode == moveJoint || hitJoint >= 0:
			canvas.Cursor(ui.CursorMove)
		case s.mode == movePose || hitBone >= 0:
			canvas.Cursor(ui.CursorGrab)
		case place >= 0 && inside:
			canvas.Cursor(ui.CursorCrosshair)
		}
		pressed := canvas.Pressed() && k > 0
		if pressed && !s.down {
			switch {
			case hitJoint >= 0:
				s.pose, s.joint, s.chosen, s.mode = hitPose, hitJoint, true, moveJoint
			case hitBone >= 0:
				s.pose, s.chosen, s.mode = hitBone, false, movePose
			case place >= 0 && inside:
				(*poses)[s.pose][place] = Joint{px, py, Visible}
				s.joint, s.chosen, s.next, s.mode = place, true, 0, moveJoint
				changed = true
			default:
				s.chosen = false
			}
			s.from = [2]float32{px, py}
			if s.pose < len(*poses) {
				s.was = slices.Clone((*poses)[s.pose])
			}
		}
		if _, _, dragged := canvas.Dragged(); pressed && dragged && s.pose < len(*poses) {
			cx, cy := max(0, min(float32(iw), px)), max(0, min(float32(ih), py))
			pose := (*poses)[s.pose]
			switch s.mode {
			case moveJoint:
				pose[s.joint].X, pose[s.joint].Y = cx, cy
				changed = true
			case movePose:
				for i, j := range s.was {
					pose[i].X, pose[i].Y = j.X+cx-s.from[0], j.Y+cy-s.from[1]
				}
				changed = true
			}
		}
		if !pressed {
			s.mode = idle
		}
		s.down = pressed

		if s.chosen && s.pose < len(*poses) {
			j := &(*poses)[s.pose][s.joint]
			switch {
			case canvas.Shortcut(0, ui.KeyV) && j.V != Missing:
				j.V = Visible + Hidden - j.V
				changed = true
			case canvas.Shortcut(0, ui.KeyDelete), canvas.Shortcut(0, ui.KeyBackspace):
				j.V = Missing
				s.chosen, s.next = false, s.joint+1
				changed = true
			case canvas.Shortcut(0, ui.KeyEscape):
				s.chosen = false
			}
		}
		if canvas.Shortcut(0, ui.KeyF) && s.pose < len(*poses) {
			pose := (*poses)[s.pose]
			flipped := make([]Joint, len(pose))
			for i := range pose {
				flipped[sk.mirror(i)] = pose[i]
			}
			copy(pose, flipped)
			if s.chosen {
				s.joint = sk.mirror(s.joint)
			}
			changed = true
		}
		if canvas.Shortcut(0, ui.KeyN) {
			add()
		}

		shown, pose, joint, chosen := *poses, s.pose, s.joint, s.chosen
		hover := -1
		if over && s.mode == idle && hitPose >= 0 {
			hover = hitJoint
		}
		canvas.Draw(func(p *ui.Painter, r ui.Rect) {
			k := min(r.W/float32(iw), r.H/float32(ih))
			pr := ui.Rect{X: r.X + (r.W-float32(iw)*k)/2, Y: r.Y + (r.H-float32(ih)*k)/2, W: float32(iw) * k, H: float32(ih) * k}
			p.Image(pic, pr, ui.FillBox)
			at := func(j Joint) (float32, float32) { return pr.X + j.X*k, pr.Y + j.Y*k }
			for pi, js := range shown {
				alpha := float32(1)
				if pi != pose {
					alpha = 0.55
				}
				for _, bn := range sk.Bones {
					a, b := js[bn[0]], js[bn[1]]
					if a.V == Missing || b.V == Missing {
						continue
					}
					x0, y0 := at(a)
					x1, y1 := at(b)
					var line ui.Path
					line.MoveTo(x0, y0)
					line.LineTo(x1, y1)
					line.Cap(ui.RoundCap)
					if a.V == Hidden || b.V == Hidden {
						line.Dash(0, 6, 5)
					}
					p.StrokePath(&line, 3, sk.color(sk.side(bn[0]), sk.side(bn[1])).Alpha(alpha))
				}
				for i, j := range js {
					if j.V == Missing {
						continue
					}
					x, y := at(j)
					col := sk.color(sk.side(i)).Alpha(alpha)
					var d ui.Path
					d.Circle(x, y, 5)
					if j.V == Hidden {
						p.FillPath(&d, ui.RGB(255, 255, 255).Alpha(alpha))
						p.StrokePath(&d, 2, col)
					} else {
						p.FillPath(&d, col)
						p.StrokePath(&d, 1.5, ui.RGB(255, 255, 255).Alpha(alpha))
					}
					if pi == pose && (chosen && i == joint || i == hover) {
						var ring ui.Path
						ring.Circle(x, y, 9)
						p.StrokePath(&ring, 2, ui.RGB(255, 255, 255))
						tag(p, t, x+12, y-10, sk.Names[i])
					}
				}
			}
		})

		panel(c, s, sk, poses, place, add, &changed)
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// panel lists the joints of the person chosen, and adds and removes
// people.
func panel(c *ui.Context, s *State, sk Skeleton, poses *[][]Joint, place int, add func(), changed *bool) {
	t := c.Theme()
	ps := *poses
	ui.Column(c).Width(t.Space(52)).Gap(t.Space(2)).Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			if len(ps) > 1 {
				ui.Button(c, "‹").Label("Previous person").Disabled(s.pose == 0).OnClick(func() { s.pose, s.chosen, s.next = s.pose-1, false, 0 })
			}
			title := "No one"
			if len(ps) > 0 {
				title = fmt.Sprintf("Person %d of %d", s.pose+1, len(ps))
			}
			ui.Text(c, title).Bold().Grow(1)
			if len(ps) > 1 {
				ui.Button(c, "›").Label("Next person").Disabled(s.pose == len(ps)-1).OnClick(func() { s.pose, s.chosen, s.next = s.pose+1, false, 0 })
			}
		})
		ui.Row(c).Gap(t.Space(2)).Children(func() {
			if ui.Button(c, "Add person").Tooltip("N").Grow(1).Clicked() {
				add()
			}
			if ui.Button(c, "Remove").Disabled(len(ps) == 0).Grow(1).Clicked() {
				*poses = slices.Delete(ps, s.pose, s.pose+1)
				s.pose, s.chosen, s.next = max(0, s.pose-1), false, 0
				*changed = true
			}
		})
		if len(ps) == 0 {
			return
		}
		pose := ps[s.pose]
		n := 0
		for _, j := range pose {
			if j.V != Missing {
				n++
			}
		}
		hint := fmt.Sprintf("%d of %d joints", n, len(pose))
		if place >= 0 {
			hint = "Click the picture to place " + strings.ToLower(sk.Names[place][:1]) + sk.Names[place][1:]
		}
		ui.Text(c, hint).TextColor(t.TextMuted).FontSize(t.FontSize - 1)
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Gap(1).Role(ui.RoleList).Label("Joints").Children(func() {
				for i, j := range pose {
					row := ui.ButtonBase(c.Key(i)).Row().Gap(t.Space(2)).AlignItems(ui.Center).
						Padding(t.Space(1), t.Space(2)).Radius(t.Space(1.5)).Label(sk.Names[i]).
						Value([...]string{"missing", "hidden", "visible"}[j.V])
					on := s.chosen && s.joint == i || i == place
					switch {
					case on:
						row.Background(t.Selection)
					case row.Hovered():
						row.Background(t.SurfaceHover)
					}
					if row.Clicked() {
						if j.V == Missing {
							s.next, s.chosen = i+1, false
						} else {
							s.joint, s.chosen = i, true
						}
					}
					col := sk.color(sk.side(i))
					v := j.V
					row.Children(func() {
						ui.Box(c).Size(10, 10).Draw(func(p *ui.Painter, r ui.Rect) {
							var d ui.Path
							d.Circle(r.X+5, r.Y+5, 4)
							switch v {
							case Visible:
								p.FillPath(&d, col)
							case Hidden:
								p.StrokePath(&d, 1.5, col)
							default:
								d.Dash(0, 2, 2)
								p.StrokePath(&d, 1, t.TextMuted)
							}
						})
						text := ui.Text(c, sk.Names[i]).Grow(1)
						if v == Missing {
							text.TextColor(t.TextMuted)
						}
						if v == Hidden {
							ui.Text(c, "hidden").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
						}
					})
				}
			})
		})
	})
}

// tag draws a joint's name in a dark pill at x, y.
func tag(p *ui.Painter, t *ui.Theme, x, y float32, name string) {
	span := ui.Span{Text: name, Size: t.FontSize - 2, Weight: 600, Color: ui.RGB(255, 255, 255)}
	w, h := p.MeasureText(0, span)
	p.Fill(ui.Rect{X: x, Y: y - 2, W: w + 10, H: h + 4}, ui.RGBA(0, 0, 0, 0.72), (h+4)/2)
	p.RichText(x+5, y, 0, span)
}

// hit returns the person and joint within d of x, y, the nearest, or -1s.
func hit(poses [][]Joint, sk Skeleton, x, y, d float32) (int, int) {
	pose, joint := -1, -1
	best := d * d
	for pi, js := range poses {
		for i, j := range js {
			if j.V == Missing {
				continue
			}
			if dd := (j.X-x)*(j.X-x) + (j.Y-y)*(j.Y-y); dd <= best {
				best, pose, joint = dd, pi, i
			}
		}
	}
	return pose, joint
}

// bone returns the person with a bone within d of x, y, or -1.
func bone(poses [][]Joint, sk Skeleton, x, y, d float32) int {
	for pi := len(poses) - 1; pi >= 0; pi-- {
		js := poses[pi]
		for _, b := range sk.Bones {
			p, q := js[b[0]], js[b[1]]
			if p.V == Missing || q.V == Missing {
				continue
			}
			if segment(x, y, p.X, p.Y, q.X, q.Y) <= d {
				return pi
			}
		}
	}
	return -1
}

// segment returns how far x, y is from the segment from x0, y0 to x1, y1.
func segment(x, y, x0, y0, x1, y1 float32) float32 {
	dx, dy := x1-x0, y1-y0
	l := dx*dx + dy*dy
	u := float32(0)
	if l > 0 {
		u = max(0, min(1, ((x-x0)*dx+(y-y0)*dy)/l))
	}
	ex, ey := x0+u*dx-x, y0+u*dy-y
	return float32(math.Sqrt(float64(ex*ex + ey*ey)))
}
