// Package colorblind shows any part of an app as people with a color
// vision deficiency see it, as Sim Daltonism and the browsers' developer
// tools do: red-blind (protan), green-blind (deutan) and blue-blind
// (tritan) vision after Machado, Oliveira and Fernandes' model, and no
// color at all (achromatopsia). Severity runs from 0, as most people see,
// to 1, the full deficiency; between, the colors mix with those of the
// full one. What it wraps still takes clicks, focus and keys as before.
//
//	colorblind.Simulate(c, colorblind.Deutan, 1, func() {
//		app.dashboard(c)
//	})
//
// The content renders into a layer on the GPU each frame it changes, as
// any element with an effect does.
package colorblind

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/shader"
)

// Vision is a kind of color vision.
type Vision int

const (
	// Normal is the vision of most people, and leaves colors as they are.
	Normal Vision = iota
	// Protan is red-blind vision: reds dark and close to greens.
	Protan
	// Deutan is green-blind vision, the most common: reds and greens alike.
	Deutan
	// Tritan is blue-blind vision: blues close to greens, yellows to pinks.
	Tritan
	// Achroma sees no color, only how light each is.
	Achroma
)

// Visions are the kinds of vision, in order.
var Visions = []Vision{Normal, Protan, Deutan, Tritan, Achroma}

// String returns the vision's name, as "Deuteranopia".
func (v Vision) String() string {
	switch v {
	case Protan:
		return "Protanopia"
	case Deutan:
		return "Deuteranopia"
	case Tritan:
		return "Tritanopia"
	case Achroma:
		return "Achromatopsia"
	}
	return "Normal"
}

// matrices turn linear RGB into what each vision sees of it, Machado et
// al.'s for severity 1, rows first.
var matrices = map[Vision][9]float32{
	Protan: {
		0.152286, 1.052583, -0.204868,
		0.114503, 0.786281, 0.099216,
		-0.003882, -0.048116, 1.051998,
	},
	Deutan: {
		0.367322, 0.860646, -0.227968,
		0.280085, 0.672501, 0.047413,
		-0.011820, 0.042940, 0.968881,
	},
	Tritan: {
		1.255528, -0.076749, -0.178779,
		-0.078411, 0.930809, 0.147602,
		0.004733, 0.691367, 0.303900,
	},
	// Each row is the luminance, the same for red, green and blue.
	Achroma: {
		0.2126, 0.7152, 0.0722,
		0.2126, 0.7152, 0.0722,
		0.2126, 0.7152, 0.0722,
	},
}

var simulate = shader.MustCompile(`package main

var Red vec3
var Green vec3
var Blue vec3

func linear(c vec3) vec3 {
	return mix(c/12.92, pow((c+0.055)/1.055, vec3(2.4)), step(vec3(0.04045), c))
}

func encode(c vec3) vec3 {
	return mix(c*12.92, 1.055*pow(c, vec3(1/2.4))-0.055, step(vec3(0.0031308), c))
}

func ColorEffect(pos vec2, color vec4) vec4 {
	if color.a == 0 {
		return color
	}
	c := linear(color.rgb / color.a)
	c = clamp(vec3(dot(Red, c), dot(Green, c), dot(Blue, c)), vec3(0), vec3(1))
	return vec4(encode(c)*color.a, color.a)
}
`)

// Simulate creates a box of what build creates, its colors as vision v
// sees them at severity from 0 to 1.
func Simulate(c *ui.Context, v Vision, severity float32, build func()) ui.Element {
	box := ui.Column(c)
	m, ok := matrices[v]
	severity = max(0, min(1, severity))
	if ok && severity > 0 {
		var rows [3][3]float32
		for i := range 3 {
			for j := range 3 {
				id := float32(0)
				if i == j {
					id = 1
				}
				rows[i][j] = id + (m[3*i+j]-id)*severity
			}
		}
		box.Effect(simulate.Effect(shader.Options{Uniforms: map[string]any{
			"Red": rows[0][:], "Green": rows[1][:], "Blue": rows[2][:],
		}}))
	}
	return box.Children(build)
}
