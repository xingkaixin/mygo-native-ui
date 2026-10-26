// Package jwt shows a JSON Web Token decoded, as jwt.io does: the token
// with its header, payload and signature each in a color, whether it has
// expired or is not valid yet, and the header's and the payload's members
// in those colors, the registered ones named, as "Expires", and times
// written as dates and as how long before or after now they are. What is
// wrong with a token that cannot be decoded shows in their place. The
// view does not verify the signature: it takes no key.
//
//	ui.TextArea(c, &app.token)
//	jwt.View(c, app.token)
package jwt

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
)

// member is a name and its value in the header or the payload, in the
// order the token has them.
type member struct {
	name  string
	value any
}

type token struct {
	parts           []string
	header, payload []member
}

// parse decodes s, or says what is wrong with it.
func parse(s string) (token, string) {
	parts := strings.Split(strings.TrimSpace(s), ".")
	tk := token{parts: parts}
	if len(parts) != 3 {
		return tk, fmt.Sprintf("A JWT has three parts separated by dots; this has %d.", len(parts))
	}
	var problem string
	if tk.header, problem = members(parts[0]); problem != "" {
		return tk, "The header " + problem
	}
	if tk.payload, problem = members(parts[1]); problem != "" {
		return tk, "The payload " + problem
	}
	return tk, ""
}

// members decodes a part of a token, a JSON object in base64url, or says
// what is wrong with it.
func members(part string) ([]member, string) {
	data, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(part, "="))
	if err != nil {
		return nil, "is not base64url."
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return nil, "is not a JSON object."
	}
	var ms []member
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, "is not a JSON object."
		}
		var v any
		if err := dec.Decode(&v); err != nil {
			return nil, "is not a JSON object."
		}
		ms = append(ms, member{tok.(string), v})
	}
	return ms, ""
}

func (tk token) time(name string) (time.Time, bool) {
	for _, m := range tk.payload {
		if n, ok := m.value.(json.Number); ok && m.name == name {
			if f, err := n.Float64(); err == nil {
				return time.Unix(0, int64(f*1e9)), true
			}
		}
	}
	return time.Time{}, false
}

var names = map[string]string{
	"alg": "Algorithm",
	"typ": "Type",
	"cty": "Content type",
	"kid": "Key ID",
	"iss": "Issuer",
	"sub": "Subject",
	"aud": "Audience",
	"exp": "Expires",
	"nbf": "Not before",
	"iat": "Issued at",
	"jti": "JWT ID",
}

var times = map[string]bool{"exp": true, "nbf": true, "iat": true}

// colors returns the colors of the header, the payload and the signature.
func colors(t *ui.Theme) [3]ui.Color {
	if t.Dark {
		return [3]ui.Color{ui.Hex("#FF6B81"), ui.Hex("#C792EA"), ui.Hex("#4FC3F7")}
	}
	return [3]ui.Color{ui.Hex("#D6204E"), ui.Hex("#8E3FCF"), ui.Hex("#0B84C6")}
}

// View creates the view of the token s.
func View(c *ui.Context, s string) ui.Element {
	t := c.Theme()
	now := c.Now()
	locale := c.Region().FormatLocale(c.Locale())
	tk, problem := parse(s)
	color := colors(t)
	return ui.Column(c).Gap(t.Space(5)).Children(func() {
		var spans []ui.Span
		for i, p := range tk.parts {
			if i > 0 {
				spans = append(spans, ui.Span{Text: ".", Color: t.TextMuted})
			}
			spans = append(spans, ui.Span{Text: p, Color: color[min(i, 2)]})
		}
		ui.RichText(c, spans...).Font("monospace").FontSize(t.FontSize-1).Selectable().
			Padding(t.Space(3)).Radius(t.Space(2)).Background(t.Surface).Border(1, t.Border).Label("Token")
		if problem != "" {
			if strings.TrimSpace(s) != "" {
				banner(c, t.Danger, problem)
			}
			return
		}
		exp, hasExp := tk.time("exp")
		nbf, hasNbf := tk.time("nbf")
		switch {
		case hasExp && !now.Before(exp):
			banner(c, t.Danger, "Expired "+relative(exp.Sub(now)))
		case hasNbf && now.Before(nbf):
			banner(c, t.Warning, "Becomes valid "+relative(nbf.Sub(now)))
		case hasExp:
			banner(c, t.Success, "Expires "+relative(exp.Sub(now)))
		}
		c.After(time.Second)
		section(c, "Header", color[0], tk.header, now, locale)
		section(c, "Payload", color[1], tk.payload, now, locale)
		ui.Column(c).Gap(t.Space(1.5)).Children(func() {
			ui.Text(c, "Signature").Bold().TextColor(color[2])
			ui.Text(c, "Not verified: checking it takes the issuer's key.").TextColor(t.TextMuted)
		})
	})
}

// banner shows text on a tint of color.
func banner(c *ui.Context, color ui.Color, text string) {
	t := c.Theme()
	ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Padding(t.Space(2), t.Space(3)).Radius(t.Space(2)).
		Background(color.Alpha(0.1)).Role(ui.RoleStatus).Children(func() {
		ui.Box(c).Size(t.Space(2), t.Space(2)).Radius(t.Space(1)).Background(color)
		ui.Text(c, text).TextColor(color).Bold()
	})
}

// section shows the members ms of the part titled title.
func section(c *ui.Context, title string, color ui.Color, ms []member, now time.Time, locale string) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(1.5)).Children(func() {
		ui.Text(c, title).Bold().TextColor(color)
		ui.Grid(c).ColumnTracks(ui.FitContent(), ui.Fr(1)).GapX(t.Space(4)).GapY(t.Space(2)).Padding(t.Space(3)).
			Radius(t.Space(2)).Border(1, t.Border).Label(title).Children(func() {
			for i, m := range ms {
				ui.Column(c.Key(i)).Children(func() {
					ui.Text(c, m.name).Font("monospace").TextColor(color)
					if n, ok := names[m.name]; ok {
						ui.Text(c, n).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
					}
				})
				ui.Column(c.Key(-1 - i)).Children(func() {
					ui.Text(c, text(m.value)).Font("monospace").Selectable()
					if n, ok := m.value.(json.Number); ok && times[m.name] {
						if f, err := n.Float64(); err == nil {
							at := time.Unix(0, int64(f*1e9))
							ui.Text(c, i18n.FormatDate(locale, at, i18n.DateOptions{Skeleton: "yMMMdjms"})+" · "+relative(at.Sub(now))).
								FontSize(t.FontSize - 2).TextColor(t.TextMuted)
						}
					}
				})
			}
		})
	})
}

// text writes v as JSON, strings without their quotes when they are a
// member's whole value.
func text(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}

// relative writes how long d is from now, rounded: "in 3 days", "2
// hours ago".
func relative(d time.Duration) string {
	past := d < 0
	if past {
		d = -d
	}
	if d < time.Minute {
		if past {
			return "just now"
		}
		return "in less than a minute"
	}
	n, unit := math.Round(d.Minutes()), "minute"
	for _, u := range []struct {
		per  float64
		name string
	}{{60, "hour"}, {24, "day"}, {30, "month"}, {12, "year"}} {
		if n < u.per {
			break
		}
		n, unit = math.Round(n/u.per), u.name
	}
	s := strconv.Itoa(int(n)) + " " + unit
	if n != 1 {
		s += "s"
	}
	if past {
		return s + " ago"
	}
	return "in " + s
}
