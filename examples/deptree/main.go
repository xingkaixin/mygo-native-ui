// Deptree shows the dependencies of a small Node.js API, an Express
// server with a Postgres client, logging, validation, tokens and a
// socket, as npm installs them: debug comes in at two versions and ms at
// three, and several packages are shared. Search for a package to see where it
// comes in, and choose one to see why it is installed.
//
//	go run ./examples/deptree
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/deptree"
)

func pkg(name, version string, size int64, deps ...int) deptree.Package {
	return deptree.Package{Name: name, Version: version, Size: size, Deps: deps}
}

var (
	packages = []deptree.Package{
		pkg("shop-api", "1.0.0", 48_200, 1, 2, 3, 4, 5, 6, 36),
		pkg("express", "4.19.2", 220_400, 7, 8, 9, 10, 11, 12, 13, 14),
		pkg("zod", "3.23.8", 640_100),
		pkg("pino", "9.1.0", 790_300, 15, 16, 17),
		pkg("jsonwebtoken", "9.0.2", 43_500, 18, 19, 20),
		pkg("pg", "8.11.5", 72_800, 21, 22, 23),
		pkg("dotenv", "16.4.5", 76_200),
		pkg("body-parser", "1.20.2", 58_900, 9, 10, 24, 25),
		pkg("cookie", "0.6.0", 21_300),
		pkg("debug", "2.6.9", 79_400, 26),
		pkg("qs", "6.11.0", 233_100, 27),
		pkg("send", "0.18.0", 49_600, 9, 28, 19),
		pkg("serve-static", "1.15.0", 25_100, 11),
		pkg("finalhandler", "1.2.0", 18_700, 9),
		pkg("accepts", "1.3.8", 16_200, 29),
		pkg("sonic-boom", "4.0.1", 46_800),
		pkg("thread-stream", "3.0.0", 120_500),
		pkg("pino-std-serializers", "7.0.0", 30_400),
		pkg("jws", "3.2.2", 22_600, 30, 38),
		pkg("ms", "2.1.3", 6_000),
		pkg("semver", "7.6.2", 92_700),
		pkg("pg-pool", "3.6.2", 23_900),
		pkg("pg-protocol", "1.6.1", 330_200),
		pkg("pg-types", "2.2.0", 37_100, 31, 32),
		pkg("raw-body", "2.5.2", 20_800),
		pkg("type-is", "1.6.18", 21_500, 29),
		pkg("ms", "2.0.0", 5_400),
		pkg("side-channel", "1.0.6", 30_900),
		pkg("mime", "1.6.0", 108_300),
		pkg("mime-types", "2.1.35", 18_100, 33),
		pkg("jwa", "1.4.1", 15_200, 40, 39, 38),
		pkg("postgres-array", "2.0.0", 5_200),
		pkg("postgres-date", "1.0.7", 7_100),
		pkg("mime-db", "1.52.0", 205_400),
		pkg("debug", "4.3.4", 42_100, 35),
		pkg("ms", "2.1.2", 5_900),
		pkg("socket.io", "4.7.5", 1_220_300, 14, 34, 37),
		pkg("base64id", "2.0.0", 4_800),
		pkg("safe-buffer", "5.2.1", 31_700),
		pkg("ecdsa-sig-formatter", "1.0.11", 17_400, 38),
		pkg("buffer-equal-constant-time", "1.0.1", 4_200),
	}
	state deptree.State
)

func view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Children(func() {
		deptree.View(c, &state, packages).Grow(1)
	})
}

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Dependencies",
			Width:   960,
			Height:  680,
			Content: ui.View(view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
