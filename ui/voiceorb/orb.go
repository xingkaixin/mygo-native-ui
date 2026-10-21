package voiceorb

import "github.com/egoist/mygo/ui/shader"

// orbShader draws the orb in a square: a sphere of light whose surface
// flows through three colors, drawn from noise that drifts, turns and
// swirls, with a pale rim and a soft halo around.
var orbShader = shader.MustCompile(`package main

// A and B are the colors of the body, C of the light in it.
var A vec3
var B vec3
var C vec3

// Scale is the radius, of the radius at rest, and Ripple how far the
// outline goes out and in, of the radius.
var Scale float
var Ripple float

// Flow is how far the surface drifted, looping at 32 as the noise does;
// Spin how far it turned and Wave how far its ripples went, in radians.
var Flow float
var Spin float
var Wave float

// Twist is how much further the middle turns than the rim, in radians,
// Pulse how strong the waves going out across it are, Energy how bright it
// is, from 0 to 1, and Glow how strong its halo.
var Twist float
var Pulse float
var Energy float
var Glow float

// Still paints a gradient in place of the surface that flows.
var Still bool

// rest is the radius at rest, of the square's width.
const rest = 0.32

// pcg returns a hash of v, PCG's.
func pcg(v uint) uint {
	s := v*747796405 + 2891336453
	w := ((s >> ((s >> 28) + 4)) ^ s) * 277803737
	return (w >> 22) ^ w
}

// slope returns a direction for the corner c of the lattice, the same
// every 32 along z, so that the surface drifting along z loops.
func slope(c vec3) vec3 {
	h := pcg(uint(int(c.x)+4096) + pcg(uint(int(c.y)+4096)+pcg(uint(int(mod(c.z, 32))))))
	return vec3(float(h&1023), float((h>>10)&1023), float((h>>20)&1023))/511.5 - 1
}

// corner returns the noise at f of the corner o of the cell at i.
func corner(i, f, o vec3) float {
	return dot(slope(i+o), f-o)
}

// noise returns gradient noise at p, from about 0 to 1.
func noise(p vec3) float {
	i := floor(p)
	f := p - i
	u := f * f * f * (f*(f*6-vec3(15)) + vec3(10))
	x00 := mix(corner(i, f, vec3(0, 0, 0)), corner(i, f, vec3(1, 0, 0)), u.x)
	x10 := mix(corner(i, f, vec3(0, 1, 0)), corner(i, f, vec3(1, 1, 0)), u.x)
	x01 := mix(corner(i, f, vec3(0, 0, 1)), corner(i, f, vec3(1, 0, 1)), u.x)
	x11 := mix(corner(i, f, vec3(0, 1, 1)), corner(i, f, vec3(1, 1, 1)), u.x)
	return 0.5 + 0.7*mix(mix(x00, x10, u.y), mix(x01, x11, u.y), u.z)
}

// fbm returns three octaves of noise at p, from 0 to 1. Each octave
// doubles z as it does x and y, so that the sum still loops along z.
func fbm(p vec3) float {
	v := 0.0
	a := 0.5
	for i := 0; i < 3; i++ {
		v += a * noise(p)
		p = vec3(p.xy*2+vec2(5.3, 1.7), p.z*2)
		a *= 0.5
	}
	return v / 0.875
}

// Fragment returns the color at pos: the halo, the sphere, or both at its
// outline, which covers as much of the pixel as it does.
func Fragment(dst vec4, pos vec2) vec4 {
	size := rectSize()
	unit := size.x * rest
	uv := (pos - size/2) / unit
	r := length(uv)
	edge := 0.5 / rest
	if r >= edge {
		return vec4(0)
	}
	a := atan2(uv.y, uv.x)
	shape := 0.5*sin(3*a+Wave) + 0.3*sin(5*a-2*Wave+1.7) + 0.2*sin(7*a+3*Wave+4.1)
	outline := Scale * (1 + Ripple*shape)
	cover := clamp((outline-r)*unit*pixelScale()+0.5, 0, 1)

	// The halo, gone before the edge of the square.
	gap := max(r-outline, 0)
	h := Glow * (0.45 + 0.55*Energy) * exp(-gap/(0.1+0.14*Energy)) * smoothstep(edge, edge*0.75, r)
	halo := vec4(mix(mix(A, B, 0.5+0.5*sin(a+Spin)), C, 0.25)*h, h)
	if cover <= 0 {
		return halo
	}

	q := uv / outline
	d := min(length(q), 1)
	z := sqrt(1 - d*d)
	col := vec3(0)
	if Still {
		g := clamp(0.5+0.55*dot(q, vec2(0.6, 0.8)), 0, 1)
		col = mix(B, A, g)
		col = mix(col, C, 0.7*smoothstep(0.55, 0, g))
	} else {
		turn := Spin + Twist*(1-d)*(1-d)
		s := vec2(cos(turn)*q.x-sin(turn)*q.y, sin(turn)*q.x+cos(turn)*q.y)
		p := vec3(s*1.1, z*1.1+Flow)
		warp := vec2(noise(p), noise(p+vec3(3.7, 8.1, 0))) - 0.5
		p = vec3(p.xy+1.4*warp, p.z)
		k := fbm(p)
		m := fbm(vec3(p.xy*1.3+vec2(9.2, 4.4), p.z))
		col = mix(A, B, smoothstep(0.3, 0.7, k))
		col = mix(col, C, 0.8*smoothstep(0.45, 0.75, m))
		// Waves going out from the middle.
		col *= 1 + Pulse*0.22*sin(d*12-2*Wave)*z
	}

	// Lit from the top left, with a pale rim and light within.
	n := vec3(q, z)
	l := normalize(vec3(-0.45, -0.6, 0.66))
	col *= 0.78 + 0.32*max(dot(n, l), 0)
	col = mix(col, mix(C, vec3(1), 0.35), 0.45*pow(1-z, 3))
	col += vec3(0.3 * pow(max(dot(n, normalize(l+vec3(0, 0, 1))), 0), 28))
	col += (0.04 + 0.12*Energy) * C * z
	return vec4(min(col, vec3(1)), 1)*cover + halo*(1-cover)
}
`)

// The shader's uniforms, looked up once.
var (
	uA, uB, uC          = orbShader.Uniform("A"), orbShader.Uniform("B"), orbShader.Uniform("C")
	uScale, uRipple     = orbShader.Uniform("Scale"), orbShader.Uniform("Ripple")
	uFlow, uSpin, uWave = orbShader.Uniform("Flow"), orbShader.Uniform("Spin"), orbShader.Uniform("Wave")
	uTwist, uPulse      = orbShader.Uniform("Twist"), orbShader.Uniform("Pulse")
	uEnergy, uGlow      = orbShader.Uniform("Energy"), orbShader.Uniform("Glow")
	uStill              = orbShader.Uniform("Still")
)
