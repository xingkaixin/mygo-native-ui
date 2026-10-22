// Vertical sets text in vertical lines, as Chinese and Japanese books
// are: a note on an autumn walk by a lake in Chinese, and one on a rainy
// evening in Japanese, with brackets, dashes, ellipses, a Latin word and
// dates among them. The control at the top picks the text and the slider
// sizes it; the columns follow the window's height, and those that do not
// fit scroll sideways from the right.
//
//	go run ./examples/vertical
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/vertical"
)

const chinese = `湖畔秋行
　　2026年10月22日，午后无风，我沿着湖岸慢慢走。水面平得像一块旧玻璃，倒映着岸边的柳树、远处的塔，还有几朵不肯散去的云。
　　路边的小摊上摆着一本旧书，书名是《湖上》。摊主说：「这是三十年前印的，只剩这一本了，你要吗？」我翻了几页，纸已经发黄，字却还清楚。
　　走到桥头，天色暗了下来——灯一盏一盏亮起，水里的灯也跟着亮起……我想起早上写的那段 Go 代码，用 mygo 画一个窗口，窗口里也有一片小小的湖。
　　回家的路上，风终于起了。（后记：这一天走了大约 12 公里。）`

const japanese = `雨の夕方
　2026年10月22日、夕方から雨が降り出した。駅前の本屋で『雨の街』という古い本を見つけ、しばらく立ち読みをした。
　店主は「もう誰も読まないよ」と笑ったが、ページをめくるたびに、知らない町の匂いがした——濡れた石畳、遠くの汽笛、小さな灯り……。
　帰ってから、Go で書いた小さなアプリを開いた。mygo の窓の中では、文字が右から左へ、上から下へと並んでいる。（雨は 3 時間ほどで止んだ。）`

type app struct {
	text  int
	size  float64
	views [2]vertical.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Gap(t.Space(3)).Padding(t.Space(3), t.Space(6)).AlignItems(ui.Center).Children(func() {
			ui.Segmented(c, &a.text, "Chinese", "Japanese").Label("Text")
			ui.Box(c).Grow(1)
			ui.Text(c, "Size")
			ui.Slider(c, &a.size, 14, 36).Label("Size").Width(t.Space(48))
			ui.Text(c, fmt.Sprintf("%.0f", a.size)).TextColor(t.TextMuted).FontFeatures("tnum").Width(t.Space(6))
		})
		ui.Divider(c)
		texts := [2]string{chinese, japanese}
		ui.Column(c).Grow(1).Padding(t.Space(8), t.Space(6)).Children(func() {
			vertical.View(c, &a.views[a.text], texts[a.text], float32(a.size)).Grow(1)
		})
	})
}

func main() {
	a := &app{size: 22}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Vertical",
			Width:   900,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
