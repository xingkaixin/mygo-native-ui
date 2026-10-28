// Package swatches finds the colors a picture is made of, as palette
// tools pull one from a photo: the picture with a ring where each color
// is found, a bar of how much of the picture each color takes, and a
// swatch of each, most of the picture first, with its hex and share. The
// pointer on a swatch, or its ring, dims the rest of the picture to show
// where the color is; a click on a swatch copies its hex.
//
//	swatches.View(c, &app.palette, app.photo, 6)
//
// Extract finds the colors without showing them.
package swatches

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand/v2"
	"slices"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
)

// Swatch is a color of a picture, the share of the picture near it, and
// a pixel of the picture nearest it.
type Swatch struct {
	Color ui.Color
	Share float64
	At    image.Point
}

// Hex writes c as "#3e8ed0".
func Hex(c ui.Color) string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }

// lab is a color in Oklab.
type lab [3]float64

func toLab(r, g, b uint8) lab {
	lin := func(v uint8) float64 {
		f := float64(v) / 255
		if f <= 0.04045 {
			return f / 12.92
		}
		return math.Pow((f+0.055)/1.055, 2.4)
	}
	R, G, B := lin(r), lin(g), lin(b)
	l := math.Cbrt(0.4122214708*R + 0.5363325363*G + 0.0514459929*B)
	m := math.Cbrt(0.2119034982*R + 0.6806995451*G + 0.1073969566*B)
	s := math.Cbrt(0.0883024619*R + 0.2817188376*G + 0.6299787005*B)
	return lab{
		0.2104542553*l + 0.7936177850*m - 0.0040720468*s,
		1.9779984951*l - 2.4285922050*m + 0.4505937099*s,
		0.0259040371*l + 0.7827717662*m - 0.8086757660*s,
	}
}

func (c lab) rgb() ui.Color {
	l := c[0] + 0.3963377774*c[1] + 0.2158037573*c[2]
	m := c[0] - 0.1055613458*c[1] - 0.0638541728*c[2]
	s := c[0] - 0.0894841775*c[1] - 1.2914855480*c[2]
	l, m, s = l*l*l, m*m*m, s*s*s
	gam := func(v float64) uint8 {
		v = max(0, min(1, v))
		if v <= 0.0031308 {
			v *= 12.92
		} else {
			v = 1.055*math.Pow(v, 1/2.4) - 0.055
		}
		return uint8(math.Round(v * 255))
	}
	return ui.RGB(
		gam(4.0767416621*l-3.3077115913*m+0.2309699292*s),
		gam(-1.2684380046*l+2.6097574011*m-0.3413193965*s),
		gam(-0.0041960863*l-0.7034186147*m+1.7076147010*s),
	)
}

func dist(a, b lab) float64 {
	d0, d1, d2 := a[0]-b[0], a[1]-b[1], a[2]-b[2]
	return d0*d0 + d1*d1 + d2*d2
}

// sample is a pixel of the picture, sampled on a grid: where it is, its
// color, and the swatch it goes to.
type sample struct {
	at    image.Point
	color lab
	group int
}

// grid is how many samples a side of the picture gives at most.
const grid = 160

// samples returns the picture's pixels on a grid at most grid by grid,
// and the grid's size, leaving out pixels mostly transparent.
func samples(img image.Image) ([]sample, int, int) {
	b := img.Bounds()
	step := max(1, max(b.Dx(), b.Dy())/grid)
	gw, gh := (b.Dx()+step-1)/step, (b.Dy()+step-1)/step
	out := make([]sample, 0, gw*gh)
	for y := b.Min.Y; y < b.Max.Y; y += step {
		for x := b.Min.X; x < b.Max.X; x += step {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			g := -1
			if c.A < 128 {
				g = -2
			}
			out = append(out, sample{at: image.Pt(x, y), color: toLab(c.R, c.G, c.B), group: g})
		}
	}
	return out, gw, gh
}

// cluster groups the samples into n by k-means in Oklab, seeded as
// k-means++ does, and returns the swatches most of the picture first.
func cluster(ss []sample, n int) []Swatch {
	rnd := rand.New(rand.NewPCG(1, 2))
	var live []int
	for i, s := range ss {
		if s.group != -2 {
			live = append(live, i)
		}
	}
	if len(live) == 0 || n <= 0 {
		return nil
	}
	n = min(n, len(live))
	centers := []lab{ss[live[rnd.IntN(len(live))]].color}
	near := make([]float64, len(live))
	for len(centers) < n {
		sum := 0.0
		for k, i := range live {
			d := math.Inf(1)
			for _, c := range centers {
				d = min(d, dist(ss[i].color, c))
			}
			near[k] = d
			sum += d
		}
		if sum == 0 {
			break
		}
		pick := rnd.Float64() * sum
		k := 0
		for ; k < len(live)-1 && pick > near[k]; k++ {
			pick -= near[k]
		}
		centers = append(centers, ss[live[k]].color)
	}
	sums := make([]lab, len(centers))
	counts := make([]int, len(centers))
	for range 16 {
		clear(sums)
		clear(counts)
		for _, i := range live {
			best, bd := 0, math.Inf(1)
			for g, c := range centers {
				if d := dist(ss[i].color, c); d < bd {
					best, bd = g, d
				}
			}
			ss[i].group = best
			counts[best]++
			for j := range 3 {
				sums[best][j] += ss[i].color[j]
			}
		}
		for g := range centers {
			if counts[g] > 0 {
				for j := range 3 {
					centers[g][j] = sums[g][j] / float64(counts[g])
				}
			}
		}
	}
	out := make([]Swatch, len(centers))
	nearest := make([]float64, len(centers))
	for g := range nearest {
		nearest[g] = math.Inf(1)
	}
	// Of the samples nearest a center, as along a band of sky, the one
	// nearest the middle of the picture marks it, away from the edges.
	var mid image.Point
	for _, i := range live {
		mid = mid.Add(ss[i].at)
	}
	mid = mid.Div(len(live))
	off := func(p image.Point) int { d := p.Sub(mid); return d.X*d.X + d.Y*d.Y }
	for _, i := range live {
		g := ss[i].group
		d := dist(ss[i].color, centers[g])
		if d < nearest[g]-1e-6 || d < nearest[g]+1e-6 && off(ss[i].at) < off(out[g].At) {
			nearest[g], out[g].At = min(d, nearest[g]), ss[i].at
		}
	}
	for g, c := range centers {
		out[g].Color, out[g].Share = c.rgb(), float64(counts[g])/float64(len(live))
	}
	order := make([]int, len(out))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int {
		switch {
		case out[a].Share > out[b].Share:
			return -1
		case out[a].Share < out[b].Share:
			return 1
		}
		return 0
	})
	rank := make([]int, len(out))
	sorted := make([]Swatch, len(out))
	for r, g := range order {
		sorted[r], rank[g] = out[g], r
	}
	for _, i := range live {
		ss[i].group = rank[ss[i].group]
	}
	return slices.DeleteFunc(sorted, func(s Swatch) bool { return s.Share == 0 })
}

// Extract returns the n colors most of img is made of, most first.
func Extract(img image.Image, n int) []Swatch {
	ss, _, _ := samples(img)
	return cluster(ss, n)
}

// State is what a view keeps: the picture's samples and swatches, the
// picture shown, the masks of the swatches pointed at, and the swatch
// copied last. Keep it in the app's state, one for each view.
type State struct {
	img      image.Image
	n        int
	swatches []Swatch
	samples  []sample
	gw, gh   int
	pic      *ui.Bitmap
	masks    map[int]*ui.Bitmap
	copied   int
	copiedAt time.Time
}

func (s *State) prepare(img image.Image, n int) {
	s.img, s.n = img, n
	s.samples, s.gw, s.gh = samples(img)
	s.swatches = cluster(s.samples, n)
	s.pic = ui.NewBitmap(img)
	s.masks = map[int]*ui.Bitmap{}
	s.copied = -1
}

// mask returns a layer over the picture dimming what is not swatch g.
func (s *State) mask(g int) *ui.Bitmap {
	if m, ok := s.masks[g]; ok {
		return m
	}
	img := image.NewNRGBA(image.Rect(0, 0, s.gw, s.gh))
	for i, sm := range s.samples {
		if sm.group != g {
			img.Pix[i*4+3] = 170
		}
	}
	m := ui.NewBitmap(img)
	s.masks[g] = m
	return m
}

// View creates the picture img with its n colors, their bar and their
// swatches.
func View(c *ui.Context, s *State, img image.Image, n int) ui.Element {
	t := c.Theme()
	if s.img != img || s.n != n || s.pic == nil {
		s.prepare(img, n)
	}
	hot := -1
	b := img.Bounds()
	iw, ih := float32(b.Dx()), float32(b.Dy())
	root := ui.Column(c).Gap(t.Space(4))
	root.Children(func() {
		pic := ui.Box(c).FillWidth().AspectRatio(iw / ih).Radius(t.Space(2)).Clip().Role(ui.RoleImage).Label("Picture")
		pr := pic.LayoutBounds()
		k := pr.W / iw
		ring := t.Space(2.5)
		if x, y, ok := pic.PointerPosition(); ok && pic.Hovered() {
			for i, sw := range s.swatches {
				dx, dy := x-(float32(sw.At.X-b.Min.X)+0.5)*k, y-(float32(sw.At.Y-b.Min.Y)+0.5)*k
				if dx*dx+dy*dy <= (ring+3)*(ring+3) {
					hot = i
				}
			}
		}

		bar := ui.Row(c).Height(t.Space(2)).Radius(t.Space(1)).Clip()
		bar.Children(func() {
			for i, sw := range s.swatches {
				seg := ui.Box(c.Key(i)).Grow(float32(sw.Share)).Basis(0).FillHeight().Background(sw.Color)
				if seg.Hovered() {
					hot = i
				}
			}
		})

		ui.Grid(c).Columns(min(len(s.swatches), 4)).Gap(t.Space(3)).Children(func() {
			for i, sw := range s.swatches {
				hex := Hex(sw.Color)
				tile := ui.ButtonBase(c.Key(i)).Column().Radius(t.Space(2)).Border(1, t.Border).Clip().
					Label(hex).Description(strconv.Itoa(int(math.Round(sw.Share*100))) + "%")
				if tile.Hovered() || tile.FocusVisible() {
					hot = i
				}
				if tile.Clicked() {
					c.WriteClipboard(hex)
					s.copied, s.copiedAt = i, c.Now()
				}
				label := hex
				if i == s.copied {
					if left := 1200*time.Millisecond - c.Now().Sub(s.copiedAt); left > 0 {
						label = "Copied"
						c.After(left)
					}
				}
				tile.Children(func() {
					ui.Box(c).FillWidth().Height(t.Space(16)).Background(sw.Color)
					ui.Row(c).FillWidth().Padding(t.Space(2), t.Space(3)).AlignItems(ui.Center).Children(func() {
						ui.Text(c, label).Font("monospace").FontSize(t.FontSize - 1).Grow(1)
						ui.Text(c, strconv.Itoa(int(math.Round(sw.Share*100)))+"%").FontSize(t.FontSize - 1).
							FontFeatures("tnum").TextColor(t.TextMuted)
					})
				})
			}
		})

		var dim *ui.Bitmap
		if hot >= 0 {
			dim = s.mask(hot)
		}
		shown, swatches := s.pic, s.swatches
		pic.Draw(func(p *ui.Painter, r ui.Rect) {
			p.Image(shown, r, ui.FillBox)
			if dim != nil {
				p.Image(dim, r, ui.FillBox)
			}
			k := r.W / iw
			for i, sw := range swatches {
				x, y := r.X+(float32(sw.At.X-b.Min.X)+0.5)*k, r.Y+(float32(sw.At.Y-b.Min.Y)+0.5)*k
				rad := ring
				if i == hot {
					rad *= 1.4
				}
				var dot ui.Path
				dot.Circle(x, y, rad)
				p.Shadow(ui.Rect{X: x - rad, Y: y - rad, W: 2 * rad, H: 2 * rad}, rad, 0, 1, 4, 0, ui.RGBA(0, 0, 0, 0.35))
				p.FillPath(&dot, sw.Color)
				p.StrokePath(&dot, 2, ui.RGB(255, 255, 255))
			}
		})
	})
	return root
}
