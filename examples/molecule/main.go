// Molecule shows five molecules built from their geometry: methane,
// water read from an XYZ file, benzene with its alternating double bonds, cyclohexane as a
// chair, and buckminsterfullerene. Drag to turn one, scroll to zoom, and
// click two or three atoms to measure a bond or an angle.
//
//	go run ./examples/molecule
package main

import (
	"fmt"
	"log"
	"math"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/molecule"
)

type atom = molecule.Atom

func methane() molecule.Molecule {
	atoms := []atom{{Element: "C"}}
	d := 1.09 / math.Sqrt(3)
	for _, v := range [][3]float64{{1, 1, 1}, {1, -1, -1}, {-1, 1, -1}, {-1, -1, 1}} {
		atoms = append(atoms, atom{Element: "H", X: v[0] * d, Y: v[1] * d, Z: v[2] * d})
	}
	return molecule.Molecule{Atoms: atoms, Bonds: molecule.Connect(atoms)}
}

const waterXYZ = `3
Water, 104.5° between bonds of 0.96 Å
O   0.0000   0.0000  0.0000
H   0.7591  -0.5877  0.0000
H  -0.7591  -0.5877  0.0000
`

func water() molecule.Molecule {
	m, err := molecule.ParseXYZ(waterXYZ)
	if err != nil {
		log.Fatal(err)
	}
	return m
}

func benzene() molecule.Molecule {
	var m molecule.Molecule
	for i := range 6 {
		a := float64(i) * math.Pi / 3
		m.Atoms = append(m.Atoms, atom{Element: "C", X: 1.39 * math.Cos(a), Y: 1.39 * math.Sin(a)})
	}
	for i := range 6 {
		a := float64(i) * math.Pi / 3
		m.Atoms = append(m.Atoms, atom{Element: "H", X: 2.48 * math.Cos(a), Y: 2.48 * math.Sin(a)})
		m.Bonds = append(m.Bonds, molecule.Bond{A: i, B: (i + 1) % 6, Order: 1 + (i+1)%2}, molecule.Bond{A: i, B: 6 + i, Order: 1})
	}
	return m
}

func cyclohexane() molecule.Molecule {
	var atoms []atom
	for i := range 6 {
		a := float64(i) * math.Pi / 3
		up := 1 - 2*float64(i%2)
		x, y, z := 1.44*math.Cos(a), 1.44*math.Sin(a), 0.25*up
		atoms = append(atoms,
			atom{Element: "C", X: x, Y: y, Z: z},
			atom{Element: "H", X: x, Y: y, Z: z + 1.09*up},
			atom{Element: "H", X: x + 1.03*math.Cos(a), Y: y + 1.03*math.Sin(a), Z: z - 0.36*up},
		)
	}
	return molecule.Molecule{Atoms: atoms, Bonds: molecule.Connect(atoms)}
}

// fullerene returns C60: the corners of a truncated icosahedron, its
// bonds between two hexagons double.
func fullerene() molecule.Molecule {
	phi := (1 + math.Sqrt(5)) / 2
	var atoms []atom
	for _, base := range [][3]float64{{0, 1, 3 * phi}, {1, 2 + phi, 2 * phi}, {phi, 2, 2*phi + 1}} {
		for signs := range 8 {
			if base[0] == 0 && signs&1 != 0 {
				continue
			}
			v := base
			for i := range 3 {
				if signs&(1<<i) != 0 {
					v[i] = -v[i]
				}
			}
			for shift := range 3 {
				// The edges are 2 long; C60's bonds are about 1.42 Å.
				atoms = append(atoms, atom{Element: "C", X: v[shift%3] * 0.71, Y: v[(shift+1)%3] * 0.71, Z: v[(shift+2)%3] * 0.71})
			}
		}
	}
	bonds := molecule.Connect(atoms)
	near := make([][]int, len(atoms))
	for _, b := range bonds {
		near[b.A] = append(near[b.A], b.B)
		near[b.B] = append(near[b.B], b.A)
	}
	angle := func(o, a, b int) float64 {
		u := [3]float64{atoms[a].X - atoms[o].X, atoms[a].Y - atoms[o].Y, atoms[a].Z - atoms[o].Z}
		v := [3]float64{atoms[b].X - atoms[o].X, atoms[b].Y - atoms[o].Y, atoms[b].Z - atoms[o].Z}
		return math.Acos((u[0]*v[0] + u[1]*v[1] + u[2]*v[2]) / (math.Hypot(math.Hypot(u[0], u[1]), u[2]) * math.Hypot(math.Hypot(v[0], v[1]), v[2])))
	}
	for i := range bonds {
		b := &bonds[i]
		// The two other bonds of an atom on a pentagon meet at 108°; the
		// bond left out of the pentagon joins two hexagons.
		var others []int
		for _, n := range near[b.A] {
			if n != b.B {
				others = append(others, n)
			}
		}
		if len(others) == 2 && math.Abs(angle(b.A, others[0], others[1])*180/math.Pi-108) < 3 {
			b.Order = 2
		}
	}
	return molecule.Molecule{Atoms: atoms, Bonds: bonds}
}

var molecules = []struct {
	name string
	m    molecule.Molecule
}{
	{"Methane", methane()},
	{"Water", water()},
	{"Benzene", benzene()},
	{"Cyclohexane", cyclohexane()},
	{"Buckminsterfullerene", fullerene()},
}

type app struct {
	chosen string
	viewer molecule.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	var m molecule.Molecule
	names := make([]string, len(molecules))
	for i, x := range molecules {
		names[i] = x.name
		if x.name == a.chosen {
			m = x.m
		}
	}
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(3)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Select(c, &a.chosen, names).Width(t.Space(56))
			ui.Text(c, fmt.Sprintf("%d atoms, %d bonds", len(m.Atoms), len(m.Bonds))).TextColor(t.TextMuted).FontFeatures("tnum")
		})
		molecule.View(c, &a.viewer, m).Grow(1)
	})
}

func main() {
	a := &app{chosen: "Benzene"}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Molecule",
			Width:   760,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
