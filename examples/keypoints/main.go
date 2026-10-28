// Keypoints labels the poses of two people: the first comes labeled, the
// second has its waving arm and a leg still to place. The labels show
// below as COCO's annotations take them, x, y and v for each joint.
//
//	go run ./examples/keypoints
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/keypoints"
)

const w, h = 900, 600

type app struct {
	photo   *ui.Bitmap
	poses   [][]keypoints.Joint
	labeler keypoints.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(5)).Gap(t.Space(3)).Children(func() {
		keypoints.View(c, &a.labeler, a.photo, keypoints.COCO, &a.poses).Grow(1)
		var lines []string
		for _, p := range a.poses {
			var nums []string
			for _, j := range p {
				nums = append(nums, fmt.Sprintf("%.0f,%.0f,%d", j.X, j.Y, j.V))
			}
			lines = append(lines, "["+strings.Join(nums, " ")+"]")
		}
		ui.Text(c, strings.Join(lines, "\n")).Font("monospace").FontSize(t.FontSize - 3).TextColor(t.TextMuted).
			Height(t.Space(16)).Selectable()
	})
}

// standing and waving are the two people: the second waves with their
// right arm and steps out with their left leg.
var (
	standing = keypoints.COCO.Pose(280, 100, 440)
	waving   = func() []keypoints.Joint {
		p := keypoints.COCO.Pose(620, 110, 420)
		p[8] = keypoints.Joint{X: 620 - 0.2*420, Y: 110 + 0.13*420, V: keypoints.Visible}
		p[10] = keypoints.Joint{X: 620 - 0.16*420, Y: 110 - 0.02*420, V: keypoints.Visible}
		p[13] = keypoints.Joint{X: 620 + 0.15*420, Y: 110 + 0.7*420, V: keypoints.Visible}
		p[15] = keypoints.Joint{X: 620 + 0.2*420, Y: 110 + 0.93*420, V: keypoints.Visible}
		return p
	}()
)

// scene draws a room with the two people in it.
func scene(c *ui.Context) {
	ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
		p.Fill(ui.Rect{W: w, H: 470}, ui.Hex("#E9E4DA"), 0)
		p.Fill(ui.Rect{Y: 470, W: w, H: 130}, ui.Hex("#B89B7A"), 0)
		p.Fill(ui.Rect{X: 380, Y: 70, W: 140, H: 180}, ui.Hex("#BFD8EA"), 4)
		p.Stroke(ui.Rect{X: 380, Y: 70, W: 140, H: 180}, ui.Hex("#FFFFFF"), 4, 6)
		person := func(js []keypoints.Joint, shirt, trousers ui.Color) {
			limb := func(a, b int, width float32, col ui.Color) {
				var l ui.Path
				l.MoveTo(js[a].X, js[a].Y)
				l.LineTo(js[b].X, js[b].Y)
				l.Cap(ui.RoundCap)
				p.StrokePath(&l, width, col)
			}
			skin := ui.Hex("#D9A27E")
			limb(11, 13, 30, trousers)
			limb(13, 15, 26, trousers)
			limb(12, 14, 30, trousers)
			limb(14, 16, 26, trousers)
			var torso ui.Path
			torso.MoveTo(js[5].X+8, js[5].Y-6)
			torso.LineTo(js[6].X-8, js[6].Y-6)
			torso.LineTo(js[12].X-12, js[12].Y+10)
			torso.LineTo(js[11].X+12, js[11].Y+10)
			torso.Close()
			p.FillPath(&torso, shirt)
			limb(5, 7, 22, shirt)
			limb(6, 8, 22, shirt)
			limb(7, 9, 18, skin)
			limb(8, 10, 18, skin)
			nx, ny := js[0].X, js[0].Y
			var neck, head ui.Path
			neck.MoveTo(nx, ny+20)
			neck.LineTo((js[5].X+js[6].X)/2, js[5].Y-4)
			p.StrokePath(neck.Cap(ui.RoundCap), 18, skin)
			head.Circle(nx, ny-4, 30)
			p.FillPath(&head, skin)
			var hair ui.Path
			hair.MoveTo(nx-31, ny-6)
			hair.CubeTo(nx-30, ny-46, nx+30, ny-46, nx+31, ny-6)
			hair.QuadTo(nx, ny-22, nx-31, ny-6)
			p.FillPath(&hair, ui.Hex("#3B2A20"))
		}
		person(standing, ui.Hex("#4C6EF5"), ui.Hex("#343A40"))
		person(waving, ui.Hex("#E8590C"), ui.Hex("#2B8A3E"))
	})
}

func main() {
	second := append([]keypoints.Joint(nil), waving...)
	for _, i := range []int{8, 10, 13, 15} {
		second[i] = keypoints.Joint{}
	}
	second[4].V = keypoints.Hidden
	a := &app{
		photo: ui.NewBitmap(ui.Render(scene, w, h, 1)),
		poses: [][]keypoints.Joint{append([]keypoints.Joint(nil), standing...), second},
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Keypoints",
			Width:   1120,
			Height:  800,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
