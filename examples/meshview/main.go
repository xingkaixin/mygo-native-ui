// Meshview shows four models built from their geometry: a gear read from
// the OBJ file it writes, a torus knot, a bracket read from the binary
// STL file it writes with z up, and a sphere. Pass an OBJ or STL file to
// show it in their place. Drag to turn a model, scroll to zoom, and click
// a face of the cube to look at that side.
//
//	go run ./examples/meshview [file.obj|file.stl]
package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/meshview"
)

type vec = [3]float64

// gearOBJ writes a spur gear of 18 teeth around a hole as an OBJ file.
func gearOBJ() string {
	const teeth, depth = 18, 0.6
	var outer []vec
	for i := range teeth * 4 {
		a := float64(i) * 2 * math.Pi / (teeth * 4)
		r := 3.0
		if i%4 == 1 || i%4 == 2 {
			r = 3.5
		}
		outer = append(outer, vec{r * math.Cos(a), r * math.Sin(a)})
	}
	var b strings.Builder
	n := len(outer)
	for _, z := range []float64{depth / 2, -depth / 2} {
		for _, p := range outer {
			fmt.Fprintf(&b, "v %.4f %.4f %.4f\n", p[0], p[1], z)
		}
		for _, p := range outer {
			a := math.Atan2(p[1], p[0])
			fmt.Fprintf(&b, "v %.4f %.4f %.4f\n", math.Cos(a), math.Sin(a), z)
		}
	}
	// Vertices 1 to n are the front's rim, n+1 to 2n its hole, then the
	// back's.
	v := func(side, ring, i int) int { return side*2*n + ring*n + i%n + 1 }
	for i := range n {
		fmt.Fprintf(&b, "f %d %d %d %d\n", v(0, 1, i), v(0, 0, i), v(0, 0, i+1), v(0, 1, i+1))
		fmt.Fprintf(&b, "f %d %d %d %d\n", v(1, 1, i+1), v(1, 0, i+1), v(1, 0, i), v(1, 1, i))
		fmt.Fprintf(&b, "f %d %d %d %d\n", v(0, 0, i), v(1, 0, i), v(1, 0, i+1), v(0, 0, i+1))
		fmt.Fprintf(&b, "f %d %d %d %d\n", v(0, 1, i+1), v(1, 1, i+1), v(1, 1, i), v(0, 1, i))
	}
	return b.String()
}

func gear() meshview.Mesh {
	m, err := meshview.ParseOBJ(gearOBJ())
	if err != nil {
		log.Fatal(err)
	}
	return m
}

// tube returns a tube of radius r around the closed curve at, in segs
// rings of sides corners.
func tube(at func(t float64) vec, r float64, segs, sides int) meshview.Mesh {
	var m meshview.Mesh
	const e = 1e-4
	for i := range segs {
		t := float64(i) / float64(segs) * 2 * math.Pi
		p, a, b := at(t), at(t-e), at(t+e)
		tan := norm(sub(b, a))
		bin := norm(cross(tan, sub(add(a, b), scale(p, 2))))
		nrm := cross(bin, tan)
		for j := range sides {
			s := float64(j) / float64(sides) * 2 * math.Pi
			m.Vertices = append(m.Vertices, add(p, add(scale(nrm, r*math.Cos(s)), scale(bin, r*math.Sin(s)))))
		}
	}
	for i := range segs {
		for j := range sides {
			a, b := i*sides+j, i*sides+(j+1)%sides
			c, d := (i+1)%segs*sides+(j+1)%sides, (i+1)%segs*sides+j
			m.Faces = append(m.Faces, [3]int{a, b, c}, [3]int{a, c, d})
		}
	}
	return m
}

func knot() meshview.Mesh {
	return tube(func(t float64) vec {
		r := 2 + math.Cos(3*t)
		return vec{r * math.Cos(2*t), -math.Sin(3 * t), r * math.Sin(2*t)}
	}, 0.42, 240, 16)
}

// bracketSTL writes an L-shaped bracket with a gusset as a binary STL
// file, z up as CAD apps write them.
func bracketSTL() []byte {
	var tris [][3]vec
	quad := func(a, b, c, d vec) { tris = append(tris, [3]vec{a, b, c}, [3]vec{a, c, d}) }
	box := func(x0, y0, z0, x1, y1, z1 float64) {
		p := func(i int) vec {
			return vec{[]float64{x0, x1}[i&1], []float64{y0, y1}[i>>1&1], []float64{z0, z1}[i>>2&1]}
		}
		quad(p(0), p(2), p(3), p(1))
		quad(p(4), p(5), p(7), p(6))
		quad(p(0), p(1), p(5), p(4))
		quad(p(2), p(6), p(7), p(3))
		quad(p(0), p(4), p(6), p(2))
		quad(p(1), p(3), p(7), p(5))
	}
	box(0, 0, 0, 6, 4, 0.5)
	box(0, 0, 0, 0.5, 4, 5)
	for _, y := range []float64{0.4, 3.2} {
		a, b, c := vec{0.5, y, 0.5}, vec{4, y, 0.5}, vec{0.5, y, 3.5}
		d, e, f := add(a, vec{0, 0.4}), add(b, vec{0, 0.4}), add(c, vec{0, 0.4})
		tris = append(tris, [3]vec{a, c, b}, [3]vec{d, e, f})
		quad(a, b, e, d)
		quad(b, c, f, e)
		quad(c, a, d, f)
	}
	out := make([]byte, 84, 84+50*len(tris))
	binary.LittleEndian.PutUint32(out[80:], uint32(len(tris)))
	for _, t := range tris {
		rec := make([]byte, 50)
		for v, p := range t {
			for k := range 3 {
				binary.LittleEndian.PutUint32(rec[12+12*v+4*k:], math.Float32bits(float32(p[k])))
			}
		}
		out = append(out, rec...)
	}
	return out
}

func bracket() meshview.Mesh {
	m, err := meshview.ParseSTL(bracketSTL())
	if err != nil {
		log.Fatal(err)
	}
	return m.ZUp()
}

// sphere returns an icosahedron split three times over, pushed out to a
// sphere.
func sphere() meshview.Mesh {
	t := (1 + math.Sqrt(5)) / 2
	m := meshview.Mesh{
		Vertices: []vec{{-1, t, 0}, {1, t, 0}, {-1, -t, 0}, {1, -t, 0}, {0, -1, t}, {0, 1, t}, {0, -1, -t}, {0, 1, -t}, {t, 0, -1}, {t, 0, 1}, {-t, 0, -1}, {-t, 0, 1}},
		Faces:    [][3]int{{0, 11, 5}, {0, 5, 1}, {0, 1, 7}, {0, 7, 10}, {0, 10, 11}, {1, 5, 9}, {5, 11, 4}, {11, 10, 2}, {10, 7, 6}, {7, 1, 8}, {3, 9, 4}, {3, 4, 2}, {3, 2, 6}, {3, 6, 8}, {3, 8, 9}, {4, 9, 5}, {2, 4, 11}, {6, 2, 10}, {8, 6, 7}, {9, 8, 1}},
	}
	for range 3 {
		mid := map[[2]int]int{}
		half := func(a, b int) int {
			k := [2]int{min(a, b), max(a, b)}
			if i, ok := mid[k]; ok {
				return i
			}
			mid[k] = len(m.Vertices)
			m.Vertices = append(m.Vertices, scale(add(m.Vertices[a], m.Vertices[b]), 0.5))
			return mid[k]
		}
		var faces [][3]int
		for _, f := range m.Faces {
			a, b, c := half(f[0], f[1]), half(f[1], f[2]), half(f[2], f[0])
			faces = append(faces, [3]int{f[0], a, c}, [3]int{f[1], b, a}, [3]int{f[2], c, b}, [3]int{a, b, c})
		}
		m.Faces = faces
	}
	for i, v := range m.Vertices {
		m.Vertices[i] = norm(v)
	}
	return m
}

func sub(a, b vec) vec           { return vec{a[0] - b[0], a[1] - b[1], a[2] - b[2]} }
func add(a, b vec) vec           { return vec{a[0] + b[0], a[1] + b[1], a[2] + b[2]} }
func scale(a vec, k float64) vec { return vec{a[0] * k, a[1] * k, a[2] * k} }
func norm(a vec) vec             { return scale(a, 1/math.Sqrt(a[0]*a[0]+a[1]*a[1]+a[2]*a[2])) }
func cross(a, b vec) vec {
	return vec{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
}

type model struct {
	name string
	m    meshview.Mesh
}

type app struct {
	models []model
	chosen string
	style  int
	viewer meshview.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	var m meshview.Mesh
	names := make([]string, len(a.models))
	for i, x := range a.models {
		names[i] = x.name
		if x.name == a.chosen {
			m = x.m
		}
	}
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(3)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Select(c, &a.chosen, names).Width(t.Space(48))
			ui.Segmented(c, &a.style, "Shaded", "Edges", "Wireframe").Label("Style")
			ui.Text(c, fmt.Sprintf("%d vertices, %d triangles", len(m.Vertices), len(m.Faces))).TextColor(t.TextMuted).FontFeatures("tnum")
		})
		a.viewer.Style = meshview.Style(a.style)
		meshview.View(c, &a.viewer, m).Grow(1)
	})
}

func main() {
	a := &app{models: []model{{"Gear", gear()}, {"Torus knot", knot()}, {"Bracket", bracket()}, {"Sphere", sphere()}}, chosen: "Gear"}
	if len(os.Args) > 1 {
		data, err := os.ReadFile(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		var m meshview.Mesh
		if strings.EqualFold(filepath.Ext(os.Args[1]), ".obj") {
			m, err = meshview.ParseOBJ(string(data))
		} else {
			m, err = meshview.ParseSTL(data)
			m = m.ZUp()
		}
		if err != nil {
			log.Fatal(err)
		}
		a.models = []model{{filepath.Base(os.Args[1]), m}}
		a.chosen = a.models[0].name
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Meshview",
			Width:   820,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
