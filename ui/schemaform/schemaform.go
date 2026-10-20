// Package schemaform is a form built from a JSON Schema, as tools that
// call MCP servers and API consoles build one for a tool's input: a field
// for each property, in the order the schema writes them, under its title
// and above its description; a text input for a string, a switch for a
// boolean, a drop-down for an enum, a list with Add and Remove for an
// array, and a group for an object. A field shows what is wrong with it
// once it has been edited, and every field does after ShowErrors.
// Changed reports an edit.
//
//	schema, err := schemaform.Parse(tool.InputSchema)
//	...
//	schemaform.View(c, &app.form, schema, &app.args)
//	if ui.Button(c, "Call").Clicked() {
//		if len(schemaform.Validate(schema, app.args)) > 0 {
//			app.form.ShowErrors()
//		} else {
//			app.call(app.args)
//		}
//	}
package schemaform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// Schema is a JSON Schema, the keywords a form shows and checks.
type Schema struct {
	// Type is "object", "array", "string", "number", "integer" or
	// "boolean"; of a list of types, the first that is not "null".
	Type        string
	Title       string
	Description string
	// Properties are an object's, in the order the schema writes them.
	Properties []Property
	Required   []string
	Items      *Schema
	Enum       []any
	Default    any

	Minimum, Maximum     *float64
	MinLength, MaxLength *int
	MinItems, MaxItems   *int
	Pattern              string
}

// Property is a property of an object.
type Property struct {
	Name   string
	Schema *Schema
}

// Problem is what is wrong with the value at Path, as "labels[1]" or
// "assignee.login".
type Problem struct {
	Path string
	Msg  string
}

// Parse reads a JSON Schema.
func Parse(data []byte) (*Schema, error) {
	var r struct {
		Type        any
		Title       string
		Description string
		Properties  json.RawMessage
		Required    []string
		Items       json.RawMessage
		Enum        []any
		Default     any
		Minimum     *float64
		Maximum     *float64
		MinLength   *int
		MaxLength   *int
		MinItems    *int
		MaxItems    *int
		Pattern     string
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	s := &Schema{
		Title: r.Title, Description: r.Description, Required: r.Required, Enum: r.Enum, Default: r.Default,
		Minimum: r.Minimum, Maximum: r.Maximum, MinLength: r.MinLength, MaxLength: r.MaxLength,
		MinItems: r.MinItems, MaxItems: r.MaxItems, Pattern: r.Pattern,
	}
	switch t := r.Type.(type) {
	case string:
		s.Type = t
	case []any:
		for _, x := range t {
			if x, ok := x.(string); ok && x != "null" {
				s.Type = x
				break
			}
		}
	}
	if len(r.Properties) > 0 {
		if s.Type == "" {
			s.Type = "object"
		}
		// A map would lose the order, which the form keeps.
		dec := json.NewDecoder(bytes.NewReader(r.Properties))
		if _, err := dec.Token(); err != nil {
			return nil, err
		}
		for dec.More() {
			tok, err := dec.Token()
			if err != nil {
				return nil, err
			}
			var sub json.RawMessage
			if err := dec.Decode(&sub); err != nil {
				return nil, err
			}
			ps, err := Parse(sub)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", tok, err)
			}
			s.Properties = append(s.Properties, Property{fmt.Sprint(tok), ps})
		}
	}
	if len(r.Items) > 0 {
		items, err := Parse(r.Items)
		if err != nil {
			return nil, fmt.Errorf("items: %w", err)
		}
		s.Items = items
	}
	return s, nil
}

// Validate returns what is wrong with v, a value as encoding/json decodes
// one into an any, by schema.
func Validate(schema *Schema, v any) []Problem {
	var out []Problem
	check(schema, v, "", &out)
	return out
}

func join(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}

func check(s *Schema, v any, path string, out *[]Problem) {
	add := func(format string, a ...any) { *out = append(*out, Problem{path, fmt.Sprintf(format, a...)}) }
	if v == nil {
		return
	}
	if len(s.Enum) > 0 {
		if !slices.ContainsFunc(s.Enum, func(e any) bool { return fmt.Sprint(e) == fmt.Sprint(v) }) {
			add("Must be one of %v", s.Enum)
		}
		return
	}
	switch s.Type {
	case "object":
		m, ok := v.(map[string]any)
		if !ok {
			add("Must be an object")
			return
		}
		for _, p := range s.Properties {
			pv := m[p.Name]
			if pv == nil {
				if slices.Contains(s.Required, p.Name) {
					*out = append(*out, Problem{join(path, p.Name), "Required"})
				}
				continue
			}
			check(p.Schema, pv, join(path, p.Name), out)
		}
	case "array":
		a, ok := v.([]any)
		if !ok {
			add("Must be a list")
			return
		}
		if s.MinItems != nil && len(a) < *s.MinItems {
			add("Must have at least %d", *s.MinItems)
		}
		if s.MaxItems != nil && len(a) > *s.MaxItems {
			add("Must have at most %d", *s.MaxItems)
		}
		if s.Items != nil {
			for i, x := range a {
				if x == nil {
					*out = append(*out, Problem{fmt.Sprintf("%s[%d]", path, i), "Required"})
					continue
				}
				check(s.Items, x, fmt.Sprintf("%s[%d]", path, i), out)
			}
		}
	case "string":
		str, ok := v.(string)
		if !ok {
			add("Must be text")
			return
		}
		n := utf8.RuneCountInString(str)
		if s.MinLength != nil && n < *s.MinLength {
			add("Must be at least %d characters", *s.MinLength)
		}
		if s.MaxLength != nil && n > *s.MaxLength {
			add("Must be at most %d characters", *s.MaxLength)
		}
		if s.Pattern != "" {
			if re, err := regexp.Compile(s.Pattern); err == nil && !re.MatchString(str) {
				add("Must match %s", s.Pattern)
			}
		}
	case "number", "integer":
		f, ok := v.(float64)
		if !ok {
			add("Must be a number")
			return
		}
		if s.Type == "integer" && f != math.Trunc(f) {
			add("Must be a whole number")
		}
		if s.Minimum != nil && f < *s.Minimum {
			add("Must be at least %v", *s.Minimum)
		}
		if s.Maximum != nil && f > *s.Maximum {
			add("Must be at most %v", *s.Maximum)
		}
	case "boolean":
		if _, ok := v.(bool); !ok {
			add("Must be true or false")
		}
	}
}

// defaults returns the value a field of s starts with: its default, or
// for an object, one of its properties' defaults.
func defaults(s *Schema) any {
	if s.Default != nil {
		return clone(s.Default)
	}
	if s.Type == "object" {
		m := map[string]any{}
		for _, p := range s.Properties {
			if d := defaults(p.Schema); d != nil {
				m[p.Name] = d
			}
		}
		return m
	}
	return nil
}

func clone(v any) any {
	switch v := v.(type) {
	case map[string]any:
		m := make(map[string]any, len(v))
		for k, x := range v {
			m[k] = clone(x)
		}
		return m
	case []any:
		a := make([]any, len(v))
		for i, x := range v {
			a[i] = clone(x)
		}
		return a
	}
	return v
}

// State is what a form keeps: the fields edited, whose problems show, or
// whether all do. Keep it in the app's state, one for each form, and
// reset it to the zero State for a new value.
type State struct {
	touched map[string]bool
	all     bool
	// problems are those of this frame's value, by path.
	problems map[string]string
	changed  bool
}

// ShowErrors shows the problems of every field, as a form does once the
// user tries to send it.
func (s *State) ShowErrors() { s.all = true }

// View creates the form of *value by schema, an object's, starting it
// from the schema's defaults when it is nil.
func View(c *ui.Context, s *State, schema *Schema, value *any) ui.Element {
	if _, ok := (*value).(map[string]any); !ok {
		*value = defaults(schema)
	}
	if s.touched == nil {
		s.touched = map[string]bool{}
	}
	s.problems = map[string]string{}
	for _, p := range Validate(schema, *value) {
		if _, ok := s.problems[p.Path]; !ok {
			s.problems[p.Path] = p.Msg
		}
	}
	s.changed = false
	form := ui.Form(c, func() {
		s.object(c, "", schema, (*value).(map[string]any))
	})
	if s.changed {
		form.SetChanged()
	}
	return form
}

func (s *State) problem(path string) string {
	if !s.all && !s.touched[path] {
		return ""
	}
	return s.problems[path]
}

func (s *State) object(c *ui.Context, path string, sc *Schema, m map[string]any) {
	for _, p := range sc.Properties {
		name, ps := p.Name, p.Schema
		at := join(path, name)
		label := ps.Title
		if label == "" {
			label = name
		}
		if slices.Contains(sc.Required, name) {
			label += " *"
		}
		if ps.Type == "object" {
			sub, ok := m[name].(map[string]any)
			if !ok {
				sub = defaults(ps).(map[string]any)
				m[name] = sub
			}
			ui.Fieldset(c.Key(name), label, func() { s.object(c, at, ps, sub) })
			continue
		}
		f := ui.Field(c.Key(name), label, func() {
			s.input(c, at, ps, m[name], func(v any) {
				if v == nil {
					delete(m, name)
				} else {
					m[name] = v
				}
			})
		})
		if ps.Description != "" {
			f.Description(ps.Description)
		}
		if msg := s.problem(at); msg != "" {
			f.Error(msg)
		}
	}
}

// input shows the control of v, which set changes, nil to leave it out.
func (s *State) input(c *ui.Context, path string, sc *Schema, v any, set func(any)) {
	t := c.Theme()
	edit := func(v any) {
		set(v)
		s.touched[path], s.changed = true, true
	}
	switch {
	case len(sc.Enum) > 0:
		opts := make([]string, len(sc.Enum))
		for i, e := range sc.Enum {
			opts[i] = fmt.Sprint(e)
		}
		sel := ""
		if v != nil {
			sel = fmt.Sprint(v)
		}
		if ui.Select(c, &sel, opts).Changed() {
			edit(sc.Enum[slices.Index(opts, sel)])
		}
	case sc.Type == "boolean":
		on, _ := v.(bool)
		if ui.Switch(c, &on).Changed() {
			edit(on)
		}
	case sc.Type == "number" || sc.Type == "integer":
		// Text that is not a number stays in the value, as a string, for
		// the problem it is to show.
		text := ""
		switch v := v.(type) {
		case float64:
			text = strconv.FormatFloat(v, 'f', -1, 64)
		case string:
			text = v
		}
		in := ui.TextInput(c, &text).FontFeatures("tnum")
		if sc.Minimum != nil && sc.Maximum != nil {
			in.Placeholder(fmt.Sprintf("%v to %v", *sc.Minimum, *sc.Maximum))
		}
		if in.Changed() {
			if text == "" {
				edit(nil)
			} else if f, err := strconv.ParseFloat(text, 64); err == nil {
				edit(f)
			} else {
				edit(text)
			}
		}
	case sc.Type == "array" && sc.Items != nil:
		a, _ := v.([]any)
		ui.Column(c).Gap(t.Space(2)).Children(func() {
			for i := range a {
				at := fmt.Sprintf("%s[%d]", path, i)
				ui.Row(c.Key(i)).Gap(t.Space(2)).AlignItems(ui.Start).Children(func() {
					ui.Column(c).Grow(1).Gap(t.Space(1)).Children(func() {
						if sc.Items.Type == "object" {
							m, ok := a[i].(map[string]any)
							if !ok {
								m = defaults(sc.Items).(map[string]any)
								a[i] = m
							}
							ui.Form(c, func() { s.object(c, at, sc.Items, m) }).
								Padding(t.Space(3)).Radius(t.Space(2)).Border(1, t.Border)
							return
						}
						s.input(c, at, sc.Items, a[i], func(x any) { a[i] = x })
						if msg := s.problem(at); msg != "" {
							ui.Text(c, msg).FontSize(t.FontSize - 1).TextColor(t.Danger)
						}
					})
					if ui.Button(c, "Remove").Clicked() {
						edit(slices.Delete(slices.Clone(a), i, i+1))
					}
				})
			}
			if ui.Button(c, "Add").AlignSelf(ui.Start).Clicked() {
				edit(append(slices.Clone(a), defaults(sc.Items)))
			}
		})
	default:
		text, _ := v.(string)
		if ui.TextInput(c, &text).Changed() {
			if text == "" {
				edit(nil)
			} else {
				edit(text)
			}
		}
	}
}
