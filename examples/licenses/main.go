// Licenses lists the packages a made-up Go app is built with. Search by
// name or license, click a license's chip to keep only its packages, and
// click a package, or move with the arrow keys, to read its license.
//
//	go run ./examples/licenses
package main

import (
	"log"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/licenses"
)

const mit = `MIT License

Copyright (c) {year} {holder}

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.`

const bsd = `Copyright (c) {year} {holder}. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of the copyright holder nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.`

const apache = `Copyright {year} {holder}

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.`

const mpl = `This Source Code Form is subject to the terms of the Mozilla Public
License, v. 2.0. If a copy of the MPL was not distributed with this
file, You can obtain one at https://mozilla.org/MPL/2.0/.

Copyright (c) {year} {holder}`

const gpl = `Copyright (C) {year} {holder}

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.`

var texts = map[string]string{"MIT": mit, "BSD-3-Clause": bsd, "Apache-2.0": apache, "MPL-2.0": mpl, "GPL-3.0": gpl}

// pkg makes the package of path at version under license, whose text
// names the package's authors.
func pkg(path, version, license, year string) licenses.Package {
	name := path[strings.LastIndexByte(path, '/')+1:]
	text := strings.NewReplacer("{year}", year, "{holder}", "The "+name+" Authors").Replace(texts[license])
	return licenses.Package{Name: path, Version: version, License: license, URL: "https://" + path, Text: text}
}

var packages = []licenses.Package{
	pkg("github.com/egoist/mygo", "v0.7.1", "MIT", "2025"),
	pkg("golang.org/x/text", "v0.43.0", "BSD-3-Clause", "2009"),
	pkg("golang.org/x/image", "v0.46.0", "BSD-3-Clause", "2009"),
	pkg("golang.org/x/net", "v0.59.0", "BSD-3-Clause", "2009"),
	pkg("golang.org/x/sys", "v0.48.0", "BSD-3-Clause", "2009"),
	pkg("github.com/go-text/typesetting", "v0.3.5", "BSD-3-Clause", "2021"),
	pkg("github.com/ebitengine/purego", "v0.11.1", "Apache-2.0", "2022"),
	pkg("github.com/yuin/goldmark", "v1.8.2", "MIT", "2019"),
	pkg("github.com/alecthomas/chroma", "v2.20.0", "MIT", "2017"),
	pkg("github.com/google/uuid", "v1.6.0", "BSD-3-Clause", "2009"),
	pkg("github.com/spf13/cobra", "v1.10.1", "Apache-2.0", "2013"),
	pkg("github.com/spf13/pflag", "v1.0.10", "BSD-3-Clause", "2012"),
	pkg("github.com/fsnotify/fsnotify", "v1.9.0", "BSD-3-Clause", "2012"),
	pkg("github.com/dustin/go-humanize", "v1.0.1", "MIT", "2005"),
	pkg("github.com/mattn/go-sqlite3", "v1.14.32", "MIT", "2014"),
	pkg("github.com/klauspost/compress", "v1.18.0", "BSD-3-Clause", "2019"),
	pkg("github.com/hashicorp/go-multierror", "v1.1.1", "MPL-2.0", "2014"),
	pkg("github.com/hashicorp/golang-lru", "v2.0.7", "MPL-2.0", "2014"),
	pkg("github.com/cespare/xxhash", "v2.3.0", "MIT", "2016"),
	pkg("go.opentelemetry.io/otel", "v1.38.0", "Apache-2.0", "2019"),
	pkg("example.com/labelprint", "v0.4.2", "GPL-3.0", "2024"),
}

func view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(5)).Gap(t.Space(4)).Children(func() {
		ui.Text(c, "Acknowledgements").Bold().FontSize(t.FontSize + 4)
		licenses.View(c, &state, packages).Grow(1)
	})
}

var state licenses.State

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Acknowledgements",
			Width:   1000,
			Height:  680,
			Content: ui.View(view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
