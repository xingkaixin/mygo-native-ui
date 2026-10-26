// Disasm lists two x86-64 functions, a loop summing an array's positive
// numbers and a main that calls it, and steps through them as a debugger
// would. Hover a jump to light its arrow, double-click it to go to its
// target, and press Escape to come back.
//
//	go run ./examples/disasm
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/disasm"
)

func b(v ...byte) []byte { return v }

var code = []disasm.Inst{
	{Addr: 0x401000, Bytes: b(0x55), Op: "push", Args: "rbp", Label: "sum_positive"},
	{Addr: 0x401001, Bytes: b(0x48, 0x89, 0xe5), Op: "mov", Args: "rbp, rsp"},
	{Addr: 0x401004, Bytes: b(0x31, 0xc0), Op: "xor", Args: "eax, eax", Comment: "sum = 0"},
	{Addr: 0x401006, Bytes: b(0x48, 0x85, 0xf6), Op: "test", Args: "rsi, rsi"},
	{Addr: 0x401009, Bytes: b(0x74, 0x1d), Op: "je", Args: "0x401028", Target: 0x401028, Comment: "n == 0"},
	{Addr: 0x40100b, Bytes: b(0x31, 0xc9), Op: "xor", Args: "ecx, ecx", Comment: "i = 0"},
	{Addr: 0x40100d, Bytes: b(0x0f, 0x1f, 0x00), Op: "nop", Args: "dword ptr [rax]"},
	{Addr: 0x401010, Bytes: b(0x8b, 0x14, 0x8f), Op: "mov", Args: "edx, dword ptr [rdi+rcx*4]", Comment: "a[i]"},
	{Addr: 0x401013, Bytes: b(0x85, 0xd2), Op: "test", Args: "edx, edx"},
	{Addr: 0x401015, Bytes: b(0x7e, 0x02), Op: "jle", Args: "0x401019", Target: 0x401019, Comment: "skip if a[i] <= 0"},
	{Addr: 0x401017, Bytes: b(0x01, 0xd0), Op: "add", Args: "eax, edx"},
	{Addr: 0x401019, Bytes: b(0x48, 0xff, 0xc1), Op: "inc", Args: "rcx"},
	{Addr: 0x40101c, Bytes: b(0x48, 0x39, 0xf1), Op: "cmp", Args: "rcx, rsi"},
	{Addr: 0x40101f, Bytes: b(0x72, 0xef), Op: "jb", Args: "0x401010", Target: 0x401010, Comment: "i < n"},
	{Addr: 0x401021, Bytes: b(0x3d, 0xff, 0xff, 0xff, 0x7f), Op: "cmp", Args: "eax, 0x7fffffff"},
	{Addr: 0x401026, Bytes: b(0x74, 0x02), Op: "je", Args: "0x40102a", Target: 0x40102a, Comment: "saturated"},
	{Addr: 0x401028, Bytes: b(0x5d), Op: "pop", Args: "rbp"},
	{Addr: 0x401029, Bytes: b(0xc3), Op: "ret"},
	{Addr: 0x40102a, Bytes: b(0x0f, 0x0b), Op: "ud2"},
	{Addr: 0x401030, Bytes: b(0x55), Op: "push", Args: "rbp", Label: "main"},
	{Addr: 0x401031, Bytes: b(0x48, 0x89, 0xe5), Op: "mov", Args: "rbp, rsp"},
	{Addr: 0x401034, Bytes: b(0x48, 0x83, 0xec, 0x20), Op: "sub", Args: "rsp, 0x20"},
	{Addr: 0x401038, Bytes: b(0x48, 0x8d, 0x7d, 0xe0), Op: "lea", Args: "rdi, [rbp-0x20]", Comment: "int a[8]"},
	{Addr: 0x40103c, Bytes: b(0xbe, 0x08, 0x00, 0x00, 0x00), Op: "mov", Args: "esi, 8"},
	{Addr: 0x401041, Bytes: b(0xe8, 0xba, 0xff, 0xff, 0xff), Op: "call", Args: "0x401000 <sum_positive>", Target: 0x401000},
	{Addr: 0x401046, Bytes: b(0x85, 0xc0), Op: "test", Args: "eax, eax"},
	{Addr: 0x401048, Bytes: b(0x78, 0x0e), Op: "js", Args: "0x401058", Target: 0x401058},
	{Addr: 0x40104a, Bytes: b(0x89, 0xc6), Op: "mov", Args: "esi, eax"},
	{Addr: 0x40104c, Bytes: b(0x48, 0x8d, 0x3d, 0xad, 0x0f, 0x00, 0x00), Op: "lea", Args: "rdi, [rip+0xfad]", Comment: `"sum = %d\n"`},
	{Addr: 0x401053, Bytes: b(0xe8, 0x28, 0x00, 0x00, 0x00), Op: "call", Args: "0x401080 <printf>", Target: 0x401080},
	{Addr: 0x401058, Bytes: b(0x31, 0xc0), Op: "xor", Args: "eax, eax"},
	{Addr: 0x40105a, Bytes: b(0xc9), Op: "leave"},
	{Addr: 0x40105b, Bytes: b(0xc3), Op: "ret"},
}

// trace is the order the program runs its instructions in, from main,
// summing {3, -1, 4}.
var trace = []uint64{
	0x401030, 0x401031, 0x401034, 0x401038, 0x40103c, 0x401041,
	0x401000, 0x401001, 0x401004, 0x401006, 0x401009, 0x40100b, 0x40100d,
	0x401010, 0x401013, 0x401015, 0x401017, 0x401019, 0x40101c, 0x40101f,
	0x401010, 0x401013, 0x401015, 0x401019, 0x40101c, 0x40101f,
	0x401010, 0x401013, 0x401015, 0x401017, 0x401019, 0x40101c, 0x40101f,
	0x401021, 0x401026, 0x401028, 0x401029,
	0x401046, 0x401048, 0x40104a, 0x40104c, 0x401053, 0x401058, 0x40105a, 0x40105b,
}

type app struct {
	step int
	code disasm.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	a.code.PC = trace[a.step]
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Padding(t.Space(2), t.Space(3)).Gap(t.Space(2)).AlignItems(ui.Center).Background(t.Surface).Children(func() {
			if ui.Button(c, "Step").Tooltip("Step (F10)").Clicked() || c.Shortcut(0, ui.KeyF10) {
				a.step = (a.step + 1) % len(trace)
			}
			if ui.Button(c, "Restart").Clicked() {
				a.step = 0
			}
			ui.Text(c, fmt.Sprintf("rip = 0x%x", a.code.PC)).Font("monospace").TextColor(t.TextMuted)
		})
		ui.Divider(c)
		disasm.View(c, &a.code, code).Grow(1)
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Disassembly",
			Width:   760,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
