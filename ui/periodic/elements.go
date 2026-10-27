package periodic

import "strings"

// Category is the kind of an element, which its cell's color shows.
type Category int

const (
	AlkaliMetal Category = iota
	AlkalineEarthMetal
	Lanthanide
	Actinide
	TransitionMetal
	PostTransitionMetal
	Metalloid
	Nonmetal
	Halogen
	NobleGas
)

var categoryNames = []string{
	"Alkali metal", "Alkaline earth metal", "Lanthanide", "Actinide", "Transition metal",
	"Post-transition metal", "Metalloid", "Nonmetal", "Halogen", "Noble gas",
}

func (c Category) String() string { return categoryNames[c] }

// Element is a chemical element: its atomic number, symbol and name, its
// standard atomic weight, or in brackets the mass number of its longest
// lived isotope when it has none, and its kind.
type Element struct {
	Number       int
	Symbol, Name string
	Mass         string
	Category     Category
}

// Period returns the row of the table e is in, from 1 to 7.
func (e Element) Period() int {
	p, _, _ := place(e.Number)
	return p
}

// Group returns the column of the table e is in, from 1 to 18, or 0 for
// the lanthanides and actinides, which show in rows of their own.
func (e Element) Group() int {
	_, g, _ := place(e.Number)
	return g
}

// starts are the atomic numbers the periods start at.
var starts = []int{1, 3, 11, 19, 37, 55, 87}

// place returns the period and group of element z, and for the
// lanthanides and actinides, whose group is 0, the place in their row,
// from 0 to 14.
func place(z int) (period, group, f int) {
	for period = len(starts); z < starts[period-1]; period-- {
	}
	off := z - starts[period-1]
	switch {
	case period == 1 && z == 1:
		return 1, 1, 0
	case period == 1:
		return 1, 18, 0
	case off < 2:
		return period, off + 1, 0
	case period <= 3:
		return period, off + 11, 0
	case period <= 5:
		return period, off + 1, 0
	case off <= 16:
		return period, 0, off - 2
	}
	return period, off - 13, 0
}

func category(z, group, period int) Category {
	switch z {
	case 1, 6, 7, 8, 15, 16, 34:
		return Nonmetal
	case 5, 14, 32, 33, 51, 52:
		return Metalloid
	case 9, 17, 35, 53, 85, 117:
		return Halogen
	}
	switch {
	case group == 0 && period == 6:
		return Lanthanide
	case group == 0:
		return Actinide
	case group == 1:
		return AlkaliMetal
	case group == 2:
		return AlkalineEarthMetal
	case group == 18:
		return NobleGas
	case group <= 12:
		return TransitionMetal
	}
	return PostTransitionMetal
}

// Elements are the 118 elements, by atomic number from 1.
var Elements = func() []Element {
	var out []Element
	for i, line := range strings.Split(strings.TrimSpace(table), "\n") {
		f := strings.Fields(line)
		z := i + 1
		p, g, _ := place(z)
		out = append(out, Element{Number: z, Symbol: f[0], Name: f[1], Mass: f[2], Category: category(z, g, p)})
	}
	return out
}()

const table = `
H Hydrogen 1.008
He Helium 4.0026
Li Lithium 6.94
Be Beryllium 9.0122
B Boron 10.81
C Carbon 12.011
N Nitrogen 14.007
O Oxygen 15.999
F Fluorine 18.998
Ne Neon 20.180
Na Sodium 22.990
Mg Magnesium 24.305
Al Aluminium 26.982
Si Silicon 28.085
P Phosphorus 30.974
S Sulfur 32.06
Cl Chlorine 35.45
Ar Argon 39.95
K Potassium 39.098
Ca Calcium 40.078
Sc Scandium 44.956
Ti Titanium 47.867
V Vanadium 50.942
Cr Chromium 51.996
Mn Manganese 54.938
Fe Iron 55.845
Co Cobalt 58.933
Ni Nickel 58.693
Cu Copper 63.546
Zn Zinc 65.38
Ga Gallium 69.723
Ge Germanium 72.630
As Arsenic 74.922
Se Selenium 78.971
Br Bromine 79.904
Kr Krypton 83.798
Rb Rubidium 85.468
Sr Strontium 87.62
Y Yttrium 88.906
Zr Zirconium 91.224
Nb Niobium 92.906
Mo Molybdenum 95.95
Tc Technetium [97]
Ru Ruthenium 101.07
Rh Rhodium 102.91
Pd Palladium 106.42
Ag Silver 107.87
Cd Cadmium 112.41
In Indium 114.82
Sn Tin 118.71
Sb Antimony 121.76
Te Tellurium 127.60
I Iodine 126.90
Xe Xenon 131.29
Cs Caesium 132.91
Ba Barium 137.33
La Lanthanum 138.91
Ce Cerium 140.12
Pr Praseodymium 140.91
Nd Neodymium 144.24
Pm Promethium [145]
Sm Samarium 150.36
Eu Europium 151.96
Gd Gadolinium 157.25
Tb Terbium 158.93
Dy Dysprosium 162.50
Ho Holmium 164.93
Er Erbium 167.26
Tm Thulium 168.93
Yb Ytterbium 173.05
Lu Lutetium 174.97
Hf Hafnium 178.49
Ta Tantalum 180.95
W Tungsten 183.84
Re Rhenium 186.21
Os Osmium 190.23
Ir Iridium 192.22
Pt Platinum 195.08
Au Gold 196.97
Hg Mercury 200.59
Tl Thallium 204.38
Pb Lead 207.2
Bi Bismuth 208.98
Po Polonium [209]
At Astatine [210]
Rn Radon [222]
Fr Francium [223]
Ra Radium [226]
Ac Actinium [227]
Th Thorium 232.04
Pa Protactinium 231.04
U Uranium 238.03
Np Neptunium [237]
Pu Plutonium [244]
Am Americium [243]
Cm Curium [247]
Bk Berkelium [247]
Cf Californium [251]
Es Einsteinium [252]
Fm Fermium [257]
Md Mendelevium [258]
No Nobelium [259]
Lr Lawrencium [266]
Rf Rutherfordium [267]
Db Dubnium [268]
Sg Seaborgium [269]
Bh Bohrium [270]
Hs Hassium [269]
Mt Meitnerium [278]
Ds Darmstadtium [281]
Rg Roentgenium [282]
Cn Copernicium [285]
Nh Nihonium [286]
Fl Flerovium [289]
Mc Moscovium [290]
Lv Livermorium [293]
Ts Tennessine [294]
Og Oganesson [294]
`
