// Cascader fills in a delivery address as a shop's checkout does: the
// province, city and district from embedded regions, those not delivered
// to disabled; and a file's category, any level of it, whose subcategories
// load as they open, after a short wait.
//
//	go run ./examples/cascader
package main

import (
	"log"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/cascader"
)

// app is the state the window shows: the region and category chosen, the
// categories loaded so far, and those loaded in the background, not yet
// put among them.
type app struct {
	region, category []string
	categories       []cascader.Option
	loaded           chan loaded
}

type loaded struct {
	path []string
	kids []cascader.Option
}

func leaves(names ...string) []cascader.Option {
	opts := make([]cascader.Option, len(names))
	for i, n := range names {
		opts[i] = cascader.Option{Name: n}
	}
	return opts
}

func place(name string, kids ...cascader.Option) cascader.Option {
	return cascader.Option{Name: name, Children: kids}
}

var regions = []cascader.Option{
	place("北京", place("北京市", leaves("东城区", "西城区", "朝阳区", "海淀区", "丰台区")...)),
	place("上海", place("上海市", leaves("黄浦区", "徐汇区", "静安区", "浦东新区")...)),
	place("浙江",
		place("杭州", leaves("上城区", "拱墅区", "西湖区", "滨江区", "余杭区")...),
		place("宁波", leaves("海曙区", "江北区", "鄞州区", "北仑区")...),
		place("温州", leaves("鹿城区", "龙湾区", "瓯海区")...),
	),
	place("江苏",
		place("南京", leaves("玄武区", "秦淮区", "建邺区", "鼓楼区")...),
		place("苏州", leaves("姑苏区", "吴中区", "相城区", "吴江区")...),
	),
	place("广东",
		place("广州", leaves("越秀区", "海珠区", "天河区", "白云区")...),
		place("深圳", leaves("福田区", "罗湖区", "南山区", "宝安区")...),
		place("珠海", append(leaves("香洲区", "金湾区"), cascader.Option{Name: "斗门区", Disabled: true})...),
	),
	{Name: "四川", Disabled: true, Children: []cascader.Option{place("成都", leaves("锦江区", "青羊区", "武侯区")...)}},
}

// catalog holds the subcategories of each category, by its path joined
// with "/", as a server would send them.
var catalog = map[string][]string{
	"":      {"文档", "图片", "音频", "视频", "其他"},
	"文档":    {"合同", "报告", "表格", "演示文稿"},
	"文档/合同": {"采购合同", "劳动合同", "租赁合同"},
	"文档/报告": {"周报", "月报", "年报"},
	"图片":    {"照片", "截图", "设计稿"},
	"图片/照片": {"旅行", "家庭", "活动"},
	"音频":    {"音乐", "录音", "播客"},
	"视频":    {"电影", "剧集", "录屏"},
}

// fetch returns the subcategories of the category at path, each Lazy if
// it has some of its own.
func fetch(path []string) []cascader.Option {
	key := strings.Join(path, "/")
	var kids []cascader.Option
	for _, n := range catalog[key] {
		_, more := catalog[strings.TrimPrefix(key+"/"+n, "/")]
		kids = append(kids, cascader.Option{Name: n, Lazy: more})
	}
	return kids
}

// put gives the category at path its subcategories.
func put(opts []cascader.Option, path []string, kids []cascader.Option) {
	for i := range opts {
		if opts[i].Name != path[0] {
			continue
		}
		if len(path) > 1 {
			put(opts[i].Children, path[1:], kids)
			return
		}
		opts[i].Children, opts[i].Lazy = kids, len(kids) > 0
	}
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	for drained := false; !drained; {
		select {
		case l := <-a.loaded:
			put(a.categories, l.path, l.kids)
		default:
			drained = true
		}
	}
	load := func(path []string) {
		go func() {
			time.Sleep(800 * time.Millisecond)
			a.loaded <- loaded{path, fetch(path)}
			c.Invalidate()
		}()
	}
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(5)).Children(func() {
		ui.Text(c, "收货信息").Bold().FontSize(t.FontSize + 6)
		field(c, "所在地区", "四川和斗门区暂不配送", func() {
			cascader.Field(c, &a.region, "省 / 市 / 区", regions, false, nil).Width(t.Space(90))
		})
		field(c, "文件分类", "可选任意一级，子分类展开时加载", func() {
			cascader.Field(c, &a.category, "选择分类", a.categories, true, load).Width(t.Space(90))
		})
		chosen := "未选择"
		if len(a.region) > 0 {
			chosen = strings.Join(a.region, "")
		}
		ui.Text(c, "寄往："+chosen).TextColor(t.TextMuted)
	})
}

// field builds a labeled field with a hint below it.
func field(c *ui.Context, label, hint string, build func()) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(1.5)).Children(func() {
		ui.Text(c, label).FontWeight(600)
		build()
		ui.Text(c, hint).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
	})
}

func main() {
	a := &app{categories: fetch(nil), loaded: make(chan loaded, 8)}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Cascader",
			Width:   720,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
