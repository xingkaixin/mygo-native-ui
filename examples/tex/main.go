// TeX typesets a formula as it is typed, under the field it is typed in,
// with what is wrong with it named in red: start from the quadratic
// formula, Maxwell's equations or another sample. Below, a Markdown note
// sets its $$ blocks the same way, through markdown's Math option.
//
//	go run ./examples/tex
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"

	"github.com/xingkaixin/mygo-native-ui/ui/tex"
)

var samples = []struct {
	name, tex string
}{
	{"Quadratic formula", `x = \frac{-b \pm \sqrt{b^2 - 4ac}}{2a}`},
	{"Euler's identity", `e^{i\pi} + 1 = 0`},
	{"Gaussian integral", `\int_{-\infty}^{\infty} e^{-x^2} \, dx = \sqrt{\pi}`},
	{"Matrix", `A = \begin{pmatrix}
  a_{11} & a_{12} & \cdots & a_{1n} \\
  a_{21} & a_{22} & \cdots & a_{2n} \\
  \vdots & \vdots & \ddots & \vdots \\
  a_{m1} & a_{m2} & \cdots & a_{mn}
\end{pmatrix}`},
	{"Cases", `|x| = \begin{cases}
  x & \text{if } x \geq 0 \\
  -x & \text{otherwise}
\end{cases}`},
	{"Maxwell's equations", `\begin{aligned}
  \nabla \cdot \mathbf{E} &= \frac{\rho}{\varepsilon_0} \\
  \nabla \cdot \mathbf{B} &= 0 \\
  \nabla \times \mathbf{E} &= -\frac{\partial \mathbf{B}}{\partial t} \\
  \nabla \times \mathbf{B} &= \mu_0 \mathbf{J} + \mu_0 \varepsilon_0 \frac{\partial \mathbf{E}}{\partial t}
\end{aligned}`},
	{"Continued fraction", `\varphi = 1 + \cfrac{1}{1 + \cfrac{1}{1 + \cfrac{1}{1 + \cdots}}}`},
	{"Sum", `\sum_{n=1}^{\infty} \frac{1}{n^2} = \frac{\pi^2}{6}`},
}

const note = `The normal distribution of mean μ and standard deviation σ has this
density, which the Gaussian integral makes add up to one:

$$
f(x) = \frac{1}{\sigma \sqrt{2\pi}} \exp\left( -\frac{(x - \mu)^2}{2\sigma^2} \right)
$$

Its mean and variance follow from it the same way.`

type app struct {
	tex string
	md  markdown.Options
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Gap(t.Space(3)).Padding(t.Space(5)).Children(func() {
			ui.Row(c).Wrap().Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Start from").FontSize(t.FontSize - 1).TextColor(t.TextMuted)
				for i, s := range samples {
					b := ui.Button(c.Key(i), s.name)
					if a.tex == s.tex {
						b.Border(1.5, t.Accent)
					}
					if b.Clicked() {
						a.tex = s.tex
					}
				}
			})
			ui.TextArea(c, &a.tex).Font("monospace").Label("TeX").Height(t.Space(30))
			ui.Column(c).Justify(ui.Center).MinHeight(t.Space(36)).Padding(t.Space(5)).Radius(t.Radius).Border(1, t.Border).Children(func() {
				tex.View(c, a.tex)
			})
			if err := tex.Err(a.tex); err != nil {
				ui.Text(c, err.Error()).TextColor(t.Danger)
			}
			ui.Divider(c).Margin(t.Space(3), 0)
			ui.Text(c, "In Markdown").Bold()
			markdown.View(c, note, &a.md)
		})
	})
}

func main() {
	a := &app{tex: samples[0].tex}
	a.md.Math = func(c *ui.Context, src string) { tex.View(c, src) }
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "TeX",
			Width:   960,
			Height:  860,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
