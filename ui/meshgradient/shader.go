package meshgradient

import "github.com/egoist/mygo/ui/shader"

// meshShader fills a rectangle with a mesh: it finds the patch of four
// points a pixel is in and where in it, then blends the points' colors
// around there along Catmull-Rom splines, in Oklab, as smooth across the
// patches' edges as within them.
var meshShader = shader.MustCompile(`package main

// Cols and Rows are how many points the mesh has across and down, P their
// places in DIPs from the corner, row by row, and C their colors in
// Oklab.
var Cols int
var Rows int
var P [25]vec2
var C [25]vec3

func cross2(a, b vec2) float {
	return a.x*b.y - a.y*b.x
}

// inside returns where p is in the patch a, b, c, d, from 0 to 1 across
// and down, as Inigo Quilez inverts a bilinear patch.
func inside(p, a, b, c, d vec2) vec2 {
	e := b - a
	f := d - a
	g := a - b + c - d
	h := p - a
	k2 := cross2(g, f)
	k1 := cross2(e, f) + cross2(h, g)
	k0 := cross2(h, e)
	v := 0.0
	if abs(k2) < 0.0001 {
		v = -k0 / k1
	} else {
		w := k1*k1 - 4*k0*k2
		if w < 0 {
			return vec2(-1)
		}
		w = sqrt(w)
		v = (-k1 - w) / (2 * k2)
		if v < -0.001 || v > 1.001 {
			v = (-k1 + w) / (2 * k2)
		}
	}
	den := e + g*v
	u := 0.0
	if abs(den.x) > abs(den.y) {
		u = (h.x - f.x*v) / den.x
	} else {
		u = (h.y - f.y*v) / den.y
	}
	return vec2(u, v)
}

func color(i, j int) vec3 {
	if i < 0 {
		i = 0
	}
	if i > Cols-1 {
		i = Cols - 1
	}
	if j < 0 {
		j = 0
	}
	if j > Rows-1 {
		j = Rows - 1
	}
	return C[j*Cols+i]
}

// weight returns the Catmull-Rom weight of the k-th of four points, u of
// the way from the second to the third.
func weight(u float, k int) float {
	u2 := u * u
	u3 := u2 * u
	if k == 0 {
		return -0.5*u3 + u2 - 0.5*u
	}
	if k == 1 {
		return 1.5*u3 - 2.5*u2 + 1
	}
	if k == 2 {
		return -1.5*u3 + 2*u2 + 0.5*u
	}
	return 0.5*u3 - 0.5*u2
}

func gamma(x float) float {
	if x <= 0.0031308 {
		return 12.92 * x
	}
	return 1.055*pow(x, 1/2.4) - 0.055
}

func srgb(lab vec3) vec3 {
	l := lab.x + 0.3963377774*lab.y + 0.2158037573*lab.z
	m := lab.x - 0.1055613458*lab.y - 0.0638541728*lab.z
	s := lab.x - 0.0894841775*lab.y - 1.2914855480*lab.z
	l = l * l * l
	m = m * m * m
	s = s * s * s
	r := 4.0767416621*l - 3.3077115913*m + 0.2309699292*s
	g := -1.2684380046*l + 2.6097574011*m - 0.3413193965*s
	b := -0.0041960863*l - 0.7034186147*m + 1.7076147010*s
	return clamp(vec3(gamma(r), gamma(g), gamma(b)), 0, 1)
}

func Fragment(dst vec4, pos vec2) vec4 {
	st := vec2(-1)
	for j := 0; j < 4; j++ {
		for i := 0; i < 4; i++ {
			if i < Cols-1 && j < Rows-1 && st.x < 0 {
				uv := inside(pos, P[j*Cols+i], P[j*Cols+i+1], P[(j+1)*Cols+i+1], P[(j+1)*Cols+i])
				if uv.x >= -0.001 && uv.x <= 1.001 && uv.y >= -0.001 && uv.y <= 1.001 {
					st = vec2(float(i), float(j)) + clamp(uv, 0, 1)
				}
			}
		}
	}
	if st.x < 0 {
		return vec4(0)
	}
	i0 := int(floor(st.x))
	j0 := int(floor(st.y))
	if i0 > Cols-2 {
		i0 = Cols - 2
	}
	if j0 > Rows-2 {
		j0 = Rows - 2
	}
	u := st.x - float(i0)
	v := st.y - float(j0)
	lab := vec3(0)
	for dj := 0; dj < 4; dj++ {
		row := vec3(0)
		for di := 0; di < 4; di++ {
			row += color(i0-1+di, j0-1+dj) * weight(u, di)
		}
		lab += row * weight(v, dj)
	}
	return vec4(srgb(lab), 1)
}
`)

var (
	uCols, uRows = meshShader.Uniform("Cols"), meshShader.Uniform("Rows")
	uP, uC       = meshShader.Uniform("P"), meshShader.Uniform("C")
)
