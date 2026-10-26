// Tokenizer splits what is typed into tokens as it changes, with a toy
// tokenizer standing in for a model's: words with the space before them,
// long words in pieces, numbers by three digits, a token for each Chinese
// character, and emoji as their bytes.
//
//	go run ./examples/tokenizer
package main

import (
	"hash/fnv"
	"log"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/tokenizer"
)

var pieces = regexp.MustCompile(`'(?:s|t|re|ve|m|ll|d)| ?\p{L}+| ?\p{N}{1,3}| ?[^\s\p{L}\p{N}]+|\s+`)

// tokenize splits text as a byte-level BPE tokenizer might.
func tokenize(text string) []tokenizer.Token {
	var out []tokenizer.Token
	add := func(s string) {
		h := fnv.New32a()
		h.Write([]byte(s))
		out = append(out, tokenizer.Token{Text: s, ID: int(h.Sum32() % 100000)})
	}
	for _, p := range pieces.FindAllString(text, -1) {
		word := strings.TrimPrefix(p, " ")
		r, _ := utf8.DecodeRuneInString(word)
		switch {
		case unicode.Is(unicode.Han, r):
			if word != p {
				add(" ")
			}
			for _, r := range word {
				add(string(r))
			}
		case unicode.IsLetter(r) && utf8.RuneCountInString(word) > 6:
			runes := []rune(p)
			for len(runes) > 0 {
				n := min(4, len(runes))
				add(string(runes[:n]))
				runes = runes[n:]
			}
		case r >= 0x2000 && !unicode.IsLetter(r) && !unicode.IsPunct(r):
			for i := range len(p) {
				add(p[i : i+1])
			}
		default:
			add(p)
		}
	}
	return out
}

type app struct {
	text   string
	tokens tokenizer.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Padding(20).Gap(16).Children(func() {
			ui.TextArea(c, &a.text).Lines(4, 8).Label("Text")
			tokenizer.View(c, &a.tokens, tokenize(a.text))
		})
	})
}

func main() {
	a := &app{text: "Tokenizers split text into pieces a model reads, often whole words with " +
		"the space before them, and uncommonly long words like internationalization in parts.\n" +
		"Numbers go by three digits: 1234567.\n分词器把中文切成单字。 Emoji become bytes 🫖."}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Tokenizer",
			Width:   760,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
