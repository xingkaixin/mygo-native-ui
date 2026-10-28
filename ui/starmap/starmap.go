// Package starmap shows the sky over a place at a time, as planisphere
// and planetarium apps do: the sky above the horizon as a disc, the
// zenith at its middle, north at the top and east on the left, as it looks
// lying on one's back with one's head to the north. The brightest stars
// show as dots, larger the brighter and tinted by their color, joined in
// the figures of their constellations, the constellations named and the
// brightest stars too. The sky darkens through twilight as the sun goes
// down, and the sun shows while it is up.
//
// Pointing at a star names it, with how bright it is and where it stands.
// Dragging across the sky turns it through the night, as the hours do,
// and ← and → step ten minutes, Shift an hour, moving *at.
//
//	starmap.View(c, &app.sky, starmap.Place{Lat: 31.23, Lon: 121.47}, &app.at).Grow(1)
package starmap

import (
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// Place is where on the earth, in degrees: north and east are positive.
type Place struct {
	Lat, Lon float64
}

type star struct {
	name string
	ra   float64 // hours
	dec  float64 // degrees
	mag  float64
	bv   float64 // color index, blue below 0 and red above 1
}

// stars are the brightest stars and those that draw the figures, at
// J2000; their names are the proper ones, or the Bayer letter where the
// star has none in common use.
var stars = []star{
	{"Sirius", 6.752, -16.716, -1.46, 0}, {"Canopus", 6.399, -52.696, -0.74, 0.15},
	{"Rigil Kentaurus", 14.660, -60.834, -0.27, 0.71}, {"Arcturus", 14.261, 19.182, -0.05, 1.23},
	{"Vega", 18.616, 38.784, 0.03, 0}, {"Capella", 5.278, 45.998, 0.08, 0.8},
	{"Rigel", 5.242, -8.202, 0.13, -0.03}, {"Procyon", 7.655, 5.225, 0.34, 0.42},
	{"Achernar", 1.629, -57.237, 0.46, -0.16}, {"Betelgeuse", 5.919, 7.407, 0.5, 1.85},
	{"Hadar", 14.064, -60.373, 0.61, -0.23}, {"Altair", 19.846, 8.868, 0.76, 0.22},
	{"Acrux", 12.443, -63.099, 0.77, -0.24}, {"Aldebaran", 4.599, 16.509, 0.85, 1.54},
	{"Spica", 13.420, -11.161, 0.97, -0.23}, {"Antares", 16.490, -26.432, 1.06, 1.83},
	{"Pollux", 7.755, 28.026, 1.14, 1.0}, {"Fomalhaut", 22.961, -29.622, 1.16, 0.09},
	{"Deneb", 20.690, 45.280, 1.25, 0.09}, {"Mimosa", 12.795, -59.689, 1.25, -0.23},
	{"Regulus", 10.140, 11.967, 1.35, -0.11}, {"Adhara", 6.977, -28.972, 1.5, -0.21},
	{"Castor", 7.577, 31.888, 1.58, 0.03}, {"Shaula", 17.560, -37.104, 1.62, -0.22},
	{"Gacrux", 12.519, -57.113, 1.64, 1.6}, {"Bellatrix", 5.419, 6.350, 1.64, -0.22},
	{"Elnath", 5.438, 28.608, 1.65, -0.13}, {"Alnilam", 5.604, -1.202, 1.69, -0.18},
	{"Alnitak", 5.679, -1.943, 1.77, -0.21}, {"Alioth", 12.900, 55.960, 1.77, -0.02},
	{"Dubhe", 11.062, 61.751, 1.79, 1.07}, {"Mirfak", 3.405, 49.861, 1.79, 0.48},
	{"Kaus Australis", 18.403, -34.385, 1.79, -0.03}, {"Wezen", 7.140, -26.393, 1.83, 0.68},
	{"Alkaid", 13.792, 49.313, 1.86, -0.19}, {"Sargas", 17.622, -42.998, 1.86, 0.4},
	{"Menkalinan", 5.992, 44.948, 1.9, 0.03}, {"Alhena", 6.629, 16.399, 1.93, 0},
	{"Mirzam", 6.378, -17.956, 1.98, -0.24}, {"Polaris", 2.530, 89.264, 1.98, 0.6},
	{"Alphard", 9.460, -8.659, 1.98, 1.44}, {"Hamal", 2.119, 23.463, 2.0, 1.15},
	{"Deneb Kaitos", 0.727, -17.987, 2.04, 1.02}, {"Nunki", 18.921, -26.297, 2.05, -0.13},
	{"Mirach", 1.162, 35.621, 2.05, 1.58}, {"Alpheratz", 0.140, 29.091, 2.06, -0.11},
	{"Saiph", 5.796, -9.670, 2.07, -0.17}, {"Kochab", 14.845, 74.156, 2.08, 1.47},
	{"Rasalhague", 17.582, 12.560, 2.08, 0.15}, {"Algieba", 10.333, 19.842, 2.08, 1.15},
	{"Algol", 3.136, 40.956, 2.1, -0.05}, {"Almach", 2.065, 42.330, 2.1, 1.37},
	{"Denebola", 11.818, 14.572, 2.11, 0.09}, {"Sadr", 20.370, 40.257, 2.23, 0.67},
	{"Mintaka", 5.533, -0.299, 2.23, -0.22}, {"Schedar", 0.675, 56.537, 2.24, 1.17},
	{"Alphecca", 15.578, 26.715, 2.23, -0.02}, {"Eltanin", 17.943, 51.489, 2.23, 1.52},
	{"Mizar", 13.399, 54.925, 2.27, 0.02}, {"Caph", 0.153, 59.150, 2.28, 0.34},
	{"Larawag", 16.836, -34.293, 2.29, 1.15}, {"Dschubba", 16.006, -22.622, 2.32, -0.12},
	{"Merak", 11.031, 56.382, 2.37, -0.02}, {"Izar", 14.750, 27.074, 2.37, 0.97},
	{"Enif", 21.737, 9.875, 2.38, 1.52}, {"Kappa Scorpii", 17.708, -39.030, 2.39, -0.22},
	{"Scheat", 23.063, 28.083, 2.42, 1.67}, {"Phecda", 11.897, 53.695, 2.44, 0},
	{"Aludra", 7.402, -29.303, 2.45, -0.08}, {"Alderamin", 21.310, 62.585, 2.45, 0.22},
	{"Gamma Cassiopeiae", 0.945, 60.717, 2.47, -0.15}, {"Gienah", 20.770, 33.970, 2.48, 1.03},
	{"Markab", 23.079, 15.205, 2.48, -0.04}, {"Menkar", 3.038, 4.090, 2.53, 1.64},
	{"Zosma", 11.235, 20.524, 2.56, 0.12}, {"Acrab", 16.091, -19.805, 2.62, -0.07},
	{"Ascella", 19.044, -29.880, 2.6, 0.08}, {"Ruchbah", 1.430, 60.235, 2.68, 0.13},
	{"Muphrid", 13.911, 18.398, 2.68, 0.58}, {"Hassaleh", 4.950, 33.166, 2.69, 1.53},
	{"Kaus Media", 18.350, -29.828, 2.7, 1.38}, {"Theta Aurigae", 5.995, 37.213, 2.6, -0.08},
	{"Tarazed", 19.771, 10.613, 2.72, 1.52}, {"Delta Crucis", 12.253, -58.749, 2.79, -0.19},
	{"Kaus Borealis", 18.466, -25.422, 2.81, 1.04}, {"Tau Scorpii", 16.598, -28.216, 2.82, -0.25},
	{"Algenib", 0.221, 15.184, 2.83, -0.23}, {"Alcyone", 3.791, 24.105, 2.87, -0.09},
	{"Tejat", 6.383, 22.514, 2.87, 1.64}, {"Pi Scorpii", 15.981, -26.114, 2.89, -0.19},
	{"Sigma Scorpii", 16.353, -25.593, 2.89, 0.13}, {"Gomeisa", 7.453, 8.289, 2.9, -0.1},
	{"Mebsuta", 6.732, 25.131, 2.98, 1.4}, {"Ras Elased", 9.764, 23.774, 2.98, 0.81},
	{"Zeta Aquilae", 19.090, 13.863, 2.99, 0.01}, {"Alnasl", 18.097, -30.424, 2.99, 1.0},
	{"Iota Scorpii", 17.793, -40.127, 3.0, 0.51}, {"Mu Scorpii", 16.864, -38.048, 3.0, -0.2},
	{"Tianguan", 5.628, 21.143, 3.0, -0.19}, {"Epsilon Aurigae", 5.033, 43.823, 3.0, 0.54},
	{"Seginus", 14.535, 38.308, 3.03, 0.19}, {"Albireo", 19.512, 27.960, 3.05, 1.13},
	{"Pherkad", 15.346, 71.834, 3.05, 0.05}, {"Phi Sagittarii", 18.761, -26.991, 3.17, -0.11},
	{"Theta Aquilae", 20.188, -0.821, 3.2, -0.07}, {"Sulafat", 18.982, 32.690, 3.25, -0.05},
	{"Tau Sagittarii", 19.116, -27.670, 3.3, 1.19}, {"Meissa", 5.585, 9.934, 3.33, -0.18},
	{"Chertan", 11.237, 15.430, 3.33, 0}, {"Eta Scorpii", 17.203, -43.239, 3.33, 0.41},
	{"Delta Aquilae", 19.425, 3.115, 3.36, 0.32}, {"Segin", 1.907, 63.670, 3.37, -0.15},
	{"Lambda Tauri", 4.011, 12.490, 3.4, -0.12}, {"Adhafera", 10.278, 23.417, 3.43, 0.31},
	{"Lambda Aquilae", 19.104, -4.883, 3.43, -0.09}, {"Delta Bootis", 15.258, 33.315, 3.47, 0.95},
	{"Sheliak", 18.835, 33.363, 3.5, 0}, {"Nekkar", 15.032, 40.391, 3.5, 0.96},
	{"Eta Leonis", 10.122, 16.763, 3.5, -0.03}, {"Wasat", 7.335, 21.982, 3.53, 0.37},
	{"Ain", 4.477, 19.180, 3.53, 1.01}, {"Rho Bootis", 14.531, 30.371, 3.58, 1.3},
	{"Prima Hyadum", 4.330, 15.628, 3.65, 0.99}, {"Zeta Scorpii", 16.910, -42.362, 3.6, 1.37},
	{"Alshain", 19.922, 6.407, 3.71, 0.86}, {"Mu Leonis", 9.879, 26.007, 3.88, 1.22},
	{"Epsilon Ursae Minoris", 16.766, 82.037, 4.21, 0.89}, {"Zeta Ursae Minoris", 15.734, 77.794, 4.29, 0.04},
	{"Zeta Lyrae", 18.746, 37.605, 4.3, 0.19}, {"Delta Lyrae", 18.909, 36.899, 4.3, 1.68},
	{"Yildun", 17.537, 86.586, 4.36, 0.02}, {"Eta Ursae Minoris", 16.292, 75.755, 4.95, 0.38},
	{"Delta Cygni", 19.750, 45.131, 2.87, -0.03}, {"Megrez", 12.257, 57.033, 3.31, 0.08},
}

type figure struct {
	name  string
	lines [][]string
}

// figures are constellations drawn as chains of stars.
var figures = []figure{
	{"Orion", [][]string{{"Meissa", "Betelgeuse", "Alnitak", "Saiph"}, {"Meissa", "Bellatrix", "Mintaka", "Rigel"},
		{"Betelgeuse", "Bellatrix"}, {"Alnitak", "Alnilam", "Mintaka"}}},
	{"Taurus", [][]string{{"Elnath", "Ain", "Prima Hyadum", "Aldebaran", "Tianguan"}, {"Prima Hyadum", "Lambda Tauri"}}},
	{"Gemini", [][]string{{"Castor", "Mebsuta", "Tejat"}, {"Pollux", "Wasat", "Alhena"}, {"Castor", "Pollux"}}},
	{"Canis Major", [][]string{{"Mirzam", "Sirius", "Wezen", "Adhara"}, {"Wezen", "Aludra"}}},
	{"Canis Minor", [][]string{{"Procyon", "Gomeisa"}}},
	{"Leo", [][]string{{"Regulus", "Eta Leonis", "Algieba", "Adhafera", "Mu Leonis", "Ras Elased"},
		{"Algieba", "Zosma", "Denebola", "Chertan", "Regulus"}, {"Zosma", "Chertan"}}},
	{"Ursa Major", [][]string{{"Dubhe", "Merak", "Phecda", "Megrez", "Dubhe"}, {"Megrez", "Alioth", "Mizar", "Alkaid"}}},
	{"Ursa Minor", [][]string{{"Polaris", "Yildun", "Epsilon Ursae Minoris", "Zeta Ursae Minoris", "Kochab", "Pherkad", "Eta Ursae Minoris", "Zeta Ursae Minoris"}}},
	{"Cassiopeia", [][]string{{"Caph", "Schedar", "Gamma Cassiopeiae", "Ruchbah", "Segin"}}},
	{"Cygnus", [][]string{{"Deneb", "Sadr", "Albireo"}, {"Gienah", "Sadr", "Delta Cygni"}}},
	{"Lyra", [][]string{{"Vega", "Zeta Lyrae", "Sheliak", "Sulafat", "Delta Lyrae", "Zeta Lyrae"}}},
	{"Aquila", [][]string{{"Tarazed", "Altair", "Alshain", "Theta Aquilae"}, {"Altair", "Delta Aquilae", "Lambda Aquilae"}, {"Delta Aquilae", "Zeta Aquilae"}}},
	{"Scorpius", [][]string{{"Acrab", "Dschubba", "Pi Scorpii"}, {"Dschubba", "Sigma Scorpii", "Antares", "Tau Scorpii", "Larawag",
		"Mu Scorpii", "Zeta Scorpii", "Eta Scorpii", "Sargas", "Iota Scorpii", "Kappa Scorpii", "Shaula"}}},
	{"Sagittarius", [][]string{{"Alnasl", "Kaus Media", "Kaus Australis", "Alnasl"}, {"Kaus Media", "Kaus Borealis", "Phi Sagittarii", "Kaus Media"},
		{"Phi Sagittarii", "Nunki", "Tau Sagittarii", "Ascella", "Phi Sagittarii"}, {"Ascella", "Kaus Australis"}}},
	{"Boötes", [][]string{{"Arcturus", "Izar", "Delta Bootis", "Nekkar", "Seginus", "Rho Bootis", "Arcturus"}, {"Arcturus", "Muphrid"}}},
	{"Auriga", [][]string{{"Capella", "Menkalinan", "Theta Aurigae", "Elnath", "Hassaleh", "Epsilon Aurigae", "Capella"}}},
	{"Pegasus", [][]string{{"Alpheratz", "Scheat", "Markab", "Algenib", "Alpheratz"}, {"Markab", "Enif"}}},
	{"Andromeda", [][]string{{"Alpheratz", "Mirach", "Almach"}}},
	{"Crux", [][]string{{"Acrux", "Gacrux"}, {"Mimosa", "Delta Crucis"}}},
	{"Centaurus", [][]string{{"Rigil Kentaurus", "Hadar"}}},
}

// index finds a star by its name.
var index = func() map[string]int {
	m := map[string]int{}
	for i, s := range stars {
		m[s.name] = i
	}
	return m
}()

const rad = math.Pi / 180

// horizontal returns where a star at ra hours and dec degrees stands seen
// from p at t: its altitude above the horizon and its azimuth east of
// north, in degrees.
func horizontal(p Place, t time.Time, ra, dec float64) (alt, az float64) {
	d := float64(t.Unix())/86400 + 2440587.5 - 2451545
	lst := math.Mod(18.697374558+24.06570982441908*d+p.Lon/15, 24)
	ha := (lst - ra) * 15 * rad
	lat, de := p.Lat*rad, dec*rad
	sinAlt := math.Sin(de)*math.Sin(lat) + math.Cos(de)*math.Cos(lat)*math.Cos(ha)
	alt = math.Asin(sinAlt)
	az = math.Atan2(-math.Sin(ha)*math.Cos(de), math.Cos(lat)*math.Sin(de)-math.Sin(lat)*math.Cos(de)*math.Cos(ha))
	return alt / rad, math.Mod(az/rad+360, 360)
}

// sun returns the sun's right ascension in hours and declination in
// degrees at t, to a few hundredths of a degree.
func sun(t time.Time) (float64, float64) {
	d := float64(t.Unix())/86400 + 2440587.5 - 2451545
	g := (357.529 + 0.98560028*d) * rad
	q := 280.459 + 0.98564736*d
	l := (q + 1.915*math.Sin(g) + 0.020*math.Sin(2*g)) * rad
	e := (23.439 - 0.00000036*d) * rad
	ra := math.Atan2(math.Cos(e)*math.Sin(l), math.Cos(l)) / rad / 15
	return math.Mod(ra+24, 24), math.Asin(math.Sin(e)*math.Sin(l)) / rad
}

// tint returns the color of a star of color index bv.
func tint(bv float64) ui.Color {
	switch {
	case bv < 0:
		return ui.Hex("#B5CCFF")
	case bv < 0.4:
		return ui.Hex("#F2F5FF")
	case bv < 0.8:
		return ui.Hex("#FFF4D6")
	case bv < 1.3:
		return ui.Hex("#FFD8A0")
	}
	return ui.Hex("#FFB27A")
}

// State is what a star map keeps: how far a drag has turned the sky. Keep
// it in the app's state, one for each map.
type State struct {
	drag float32
}

// View creates the sky over p at *at.
func View(c *ui.Context, s *State, p Place, at *time.Time) ui.Element {
	t := c.Theme()
	when := *at
	e := ui.Box(c).MinHeight(t.Space(60)).Focusable().FocusRing(false).Cursor(ui.CursorGrab).Role(ui.RoleImage).
		Label("Sky at " + when.Format("15:04, Jan 2"))
	step := func(d time.Duration) {
		*at = at.Add(d)
		e.SetChanged()
	}
	switch {
	case e.Shortcut(0, ui.KeyLeft):
		step(-10 * time.Minute)
	case e.Shortcut(0, ui.KeyRight):
		step(10 * time.Minute)
	case e.Shortcut(ui.Shift, ui.KeyLeft):
		step(-time.Hour)
	case e.Shortcut(ui.Shift, ui.KeyRight):
		step(time.Hour)
	}
	if dx, _, ok := e.Dragged(); ok {
		e.Cursor(ui.CursorGrabbing)
		// A sidereal day is the turn of the sky; a drag across the disc
		// turns it about a quarter.
		s.drag += dx
		if m := math.Trunc(float64(s.drag)); m != 0 {
			step(time.Duration(m) * 30 * time.Second)
			s.drag -= float32(m)
		}
	}
	type spot struct {
		x, y, alt, az float64 // x and y from the middle, the horizon at 1
	}
	project := func(ra, dec float64) spot {
		alt, az := horizontal(p, when, ra, dec)
		r := math.Tan((90 - alt) * rad / 2)
		return spot{-r * math.Sin(az*rad), -r * math.Cos(az*rad), alt, az}
	}
	spots := make([]spot, len(stars))
	for i, st := range stars {
		spots[i] = project(st.ra, st.dec)
	}
	sra, sdec := sun(when)
	sunAt := project(sra, sdec)
	px, py, over := e.PointerPosition()
	b := e.Bounds()
	size := min(b.W, b.H) / 2 * 0.9
	cx, cy := b.W/2, b.H/2
	hover := -1
	if over && size > 0 {
		best := float32(10 * 10)
		for i, sp := range spots {
			if sp.alt < 0 {
				continue
			}
			dx, dy := cx+float32(sp.x)*size-px, cy+float32(sp.y)*size-py
			if d := dx*dx + dy*dy; d < best {
				best, hover = d, i
			}
		}
	}
	e.Draw(func(pt *ui.Painter, r ui.Rect) {
		R := min(r.W, r.H) / 2 * 0.9
		ox, oy := r.X+r.W/2, r.Y+r.H/2
		at := func(sp spot) (float32, float32) { return ox + float32(sp.x)*R, oy + float32(sp.y)*R }
		night, dusk, day := ui.Hex("#0B1026"), ui.Hex("#2B3F73"), ui.Hex("#6FA8DC")
		sky := night
		switch {
		case sunAt.alt > 0:
			sky = dusk.Mix(day, float32(min(1, sunAt.alt/10)))
		case sunAt.alt > -18:
			sky = night.Mix(dusk, float32(1+sunAt.alt/18))
		}
		var disc ui.Path
		disc.Circle(ox, oy, R)
		pt.FillPath(&disc, sky)
		light := float32(max(0, min(1, (sunAt.alt+12)/12)))
		for _, alt := range []float64{30, 60} {
			var ring ui.Path
			ring.Circle(ox, oy, R*float32(math.Tan((90-alt)*rad/2)))
			pt.StrokePath(ring.Dash(0, 2, 4), 1, ui.RGBA(255, 255, 255, 0.12))
		}
		pt.Clip(ui.Rect{X: ox - R, Y: oy - R, W: 2 * R, H: 2 * R}, R, func() {
			for _, f := range figures {
				for _, chain := range f.lines {
					var line ui.Path
					up := false
					for k, name := range chain {
						sp := spots[index[name]]
						x, y := at(sp)
						if k == 0 {
							line.MoveTo(x, y)
						} else {
							line.LineTo(x, y)
						}
						up = up || sp.alt > -5
					}
					if up {
						pt.StrokePath(&line, 1, ui.RGBA(140, 180, 255, 0.35*(1-light)))
					}
				}
			}
			for i, st := range stars {
				sp := spots[i]
				if sp.alt < -2 {
					continue
				}
				x, y := at(sp)
				size := float32(math.Max(0.8, 3.6-0.75*st.mag))
				col := tint(st.bv).Alpha(max(0.15, 1-light*0.9))
				if st.mag < 1 {
					var glow ui.Path
					glow.Circle(x, y, size*2.4)
					pt.FillPath(&glow, col.Alpha(0.12*(1-light)))
				}
				var d ui.Path
				d.Circle(x, y, size)
				pt.FillPath(&d, col)
				if st.mag < 1.4 && i != hover && light < 0.95 {
					pt.Text(x+size+3, y-6, st.name, t.FontSize-3, ui.RGBA(255, 255, 255, 0.55*(1-light)))
				}
			}
			for _, f := range figures {
				var sx, sy, n float64
				for _, chain := range f.lines {
					for _, name := range chain {
						sp := spots[index[name]]
						sx, sy, n = sx+sp.x, sy+sp.y, n+1
					}
				}
				mx, my := ox+float32(sx/n)*R, oy+float32(sy/n)*R
				// A span's zero alpha would take the theme's text color.
				if math.Hypot(sx/n, sy/n) > 0.95 || light > 0.95 {
					continue
				}
				span := ui.Span{Text: f.name, Size: t.FontSize - 2, Color: ui.RGBA(170, 200, 255, 0.6*(1-light)), LetterSpacing: 1}
				w, _ := pt.MeasureText(0, span)
				pt.RichText(mx-w/2, my+8, 0, span)
			}
			if sunAt.alt > -1 {
				x, y := at(sunAt)
				var glow, d ui.Path
				glow.Circle(x, y, 22)
				pt.FillPath(&glow, ui.RGBA(255, 214, 102, 0.25))
				d.Circle(x, y, 9)
				pt.FillPath(&d, ui.Hex("#FFD666"))
			}
		})
		pt.StrokePath(&disc, 1.5, t.Border)
		for i, label := range []string{"N", "E", "S", "W"} {
			a := float64(i) * 90 * rad
			x, y := ox-float32(math.Sin(a))*(R+12), oy-float32(math.Cos(a))*(R+12)
			span := ui.Span{Text: label, Size: t.FontSize - 1, Weight: 700, Color: t.TextMuted}
			w, h := pt.MeasureText(0, span)
			pt.RichText(x-w/2, y-h/2, 0, span)
		}
		if hover >= 0 {
			st, sp := stars[hover], spots[hover]
			x, y := at(sp)
			var ring ui.Path
			ring.Circle(x, y, 7)
			pt.StrokePath(&ring, 1.5, ui.RGB(255, 255, 255))
			title := ui.Span{Text: st.name, Size: t.FontSize - 1, Weight: 700, Color: ui.RGB(255, 255, 255)}
			info := ui.Span{Text: fmt.Sprintf("\nmagnitude %.1f · %.0f° up, %s", st.mag, sp.alt, compass(sp.az)),
				Size: t.FontSize - 2, Color: ui.RGBA(255, 255, 255, 0.75)}
			w, h := pt.MeasureText(0, title, info)
			bx, by := x+12, y-h-12
			if bx+w+16 > r.X+r.W {
				bx = x - w - 28
			}
			if by < r.Y {
				by = y + 12
			}
			pt.Fill(ui.Rect{X: bx, Y: by, W: w + 16, H: h + 10}, ui.RGBA(10, 14, 30, 0.88), 6)
			pt.RichText(bx+8, by+5, 0, title, info)
		}
	})
	return e
}

// compass names the direction of an azimuth to the nearest of sixteen.
func compass(az float64) string {
	names := []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
	return names[int(math.Round(az/22.5))%16]
}
