package openapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Document is an OpenAPI document: its title and version, its servers,
// and its operations by tag.
type Document struct {
	title, version string
	servers        []string
	// untagged are the operations without a tag, and tags the others, a
	// tag for each, in the order the document names them.
	untagged []*operation
	tags     []*tag
	problems []string
}

// Problems returns what in the document could not be read, each with
// where it is, as "GET /notes response 404: $ref #/components/responses/Gone:
// not found". The rest of the document shows.
func (d *Document) Problems() []string { return d.problems }

// tag is a tag of operations, which the sidebar shows them under.
type tag struct {
	name string
	ops  []*operation
}

type operation struct {
	method, path         string
	summary, description string
	id                   string
	deprecated           bool
	params               []*param
	body                 *body
	responses            []*response
}

// key identifies op in the document, "GET /notes/{noteId}".
func (op *operation) key() string { return op.method + " " + op.path }

// param is a parameter of an operation, where it goes: "path", "query",
// "header" or "cookie".
type param struct {
	*field
	in string
}

type body struct {
	media, description string
	required           bool
	schema             *schema
	example            any
}

type response struct {
	status, description, media string
	schema                     *schema
}

// field is a row of a tree of a schema: a property, a parameter, or one of
// the schemas a value may match. Its description and example, when it has
// them, are those of where it is, over its schema's.
type field struct {
	name        string
	schema      *schema
	required    bool
	deprecated  bool
	description string
	example     any
}

// schema is a JSON Schema, as the reference shows it. Schemas named in the
// document's components are read once and shared, so that a schema may
// hold itself, as a note holds its replies; what walks them stops where a
// schema comes back.
type schema struct {
	// ref is the name of the component it is, "Note".
	ref         string
	types       []string
	format      string
	description string
	enum        []any
	example     any
	def         any
	props       []*field
	items       *schema
	// variants are the schemas a value may match, one of them or any.
	variants []*field
	choice   string
	// missing is a $ref that does not resolve.
	missing string
}

// kind returns the type of s other than null: "object", "array",
// "string", "integer", "number", "boolean", or "" for any.
func (s *schema) kind() string {
	for _, t := range s.types {
		if t != "null" {
			return t
		}
	}
	switch {
	case len(s.props) > 0:
		return "object"
	case s.items != nil:
		return "array"
	}
	return ""
}

// inner returns the schema of the items of s, through arrays of arrays,
// or s.
func (s *schema) inner() *schema {
	for s.kind() == "array" && s.items != nil {
		s = s.items
	}
	return s
}

// fields returns the rows under s: its properties, or the items', or the
// schemas a value may match.
func (s *schema) fields() []*field {
	in := s.inner()
	if len(in.props) > 0 {
		return in.props
	}
	return in.variants
}

// typeName returns the type of s as the reference writes it: "string",
// "Note", "Note[]", "string | null", "one of".
func (s *schema) typeName() string {
	if s.missing != "" {
		return s.missing
	}
	name := ""
	switch k := s.kind(); {
	case k == "array" && s.items != nil:
		name = s.items.typeName() + "[]"
	case s.ref != "" && (k == "object" || k == ""):
		name = s.ref
	case k == "" && s.choice != "":
		name = s.choice
	case k == "":
		name = "any"
	default:
		name = k
	}
	if slices.Contains(s.types, "null") {
		name += " | null"
	}
	return name
}

var methods = []string{"get", "put", "post", "delete", "options", "head", "patch", "trace"}

// Parse reads an OpenAPI 3.0 or 3.1 document in JSON; YAML is not read,
// convert it to JSON first. It fails on what is not JSON, saying the line
// and column where it stops being so, and on a document of another
// version. What fails within the document, as a $ref to nothing or to
// another file, Problems lists.
func Parse(data []byte) (*Document, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	if !bytes.HasPrefix(bytes.TrimSpace(data), []byte("{")) {
		return nil, errors.New("openapi: not a JSON object; YAML is not read, convert it to JSON")
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	v, err := decode(dec)
	if err != nil {
		at := dec.InputOffset()
		var syntax *json.SyntaxError
		if errors.As(err, &syntax) {
			// The offset is of the byte after the one at fault.
			at = max(0, syntax.Offset-1)
		}
		line, col := position(data, at)
		return nil, fmt.Errorf("openapi: line %d, column %d: %w", line, col, err)
	}
	if _, err := dec.Token(); err != io.EOF {
		line, col := position(data, dec.InputOffset())
		return nil, fmt.Errorf("openapi: line %d, column %d: more after the document", line, col)
	}
	root := v.(*object)
	if version, _ := root.get("openapi").(string); !strings.HasPrefix(version, "3.") {
		if root.get("swagger") != nil {
			return nil, errors.New("openapi: a Swagger 2.0 document; only OpenAPI 3.0 and 3.1 are read")
		}
		return nil, errors.New(`openapi: no "openapi": "3.x"; only OpenAPI 3.0 and 3.1 are read`)
	}
	p := &parser{root: root, schemas: map[string]*schema{}}
	return p.document(), nil
}

// position returns the line and column of the byte at offset in data,
// from 1.
func position(data []byte, offset int64) (line, col int) {
	before := data[:min(int(offset), len(data))]
	start := bytes.LastIndexByte(before, '\n') + 1
	return bytes.Count(before, []byte("\n")) + 1, utf8.RuneCount(before[start:]) + 1
}

// parser reads a document: its root, the schemas of its components read,
// by $ref, and what failed.
type parser struct {
	root     *object
	schemas  map[string]*schema
	problems []string
}

func (p *parser) problem(at string, err error) {
	p.problems = append(p.problems, at+": "+err.Error())
}

func (p *parser) document() *Document {
	d := &Document{}
	if info, ok := p.root.get("info").(*object); ok {
		d.title, d.version = info.str("title"), info.str("version")
	}
	for _, v := range list(p.root.get("servers")) {
		o, ok := v.(*object)
		if !ok {
			continue
		}
		u := o.str("url")
		if vars, ok := o.get("variables").(*object); ok {
			for i, name := range vars.keys {
				if vo, ok := vars.vals[i].(*object); ok {
					u = strings.ReplaceAll(u, "{"+name+"}", vo.str("default"))
				}
			}
		}
		d.servers = append(d.servers, u)
	}
	if len(d.servers) == 0 {
		d.servers = []string{"/"}
	}
	tags := map[string]*tag{}
	add := func(name string) *tag {
		if tags[name] == nil {
			tags[name] = &tag{name: name}
			d.tags = append(d.tags, tags[name])
		}
		return tags[name]
	}
	for _, v := range list(p.root.get("tags")) {
		if o, ok := v.(*object); ok && o.str("name") != "" {
			add(o.str("name"))
		}
	}
	paths, _ := p.root.get("paths").(*object)
	if paths == nil {
		paths = &object{}
	}
	for i, path := range paths.keys {
		item := p.deref(paths.vals[i], path)
		if item == nil {
			continue
		}
		shared := list(item.get("parameters"))
		for j, m := range item.keys {
			if !slices.Contains(methods, m) {
				continue
			}
			o, ok := item.vals[j].(*object)
			if !ok {
				p.problem(strings.ToUpper(m)+" "+path, errors.New("not an operation object"))
				continue
			}
			op := p.operation(strings.ToUpper(m), path, o, shared)
			names := strs(o.get("tags"))
			if len(names) == 0 {
				d.untagged = append(d.untagged, op)
			}
			for _, name := range names {
				t := add(name)
				t.ops = append(t.ops, op)
			}
		}
	}
	d.tags = slices.DeleteFunc(d.tags, func(t *tag) bool { return len(t.ops) == 0 })
	d.problems = p.problems
	return d
}

func (p *parser) operation(method, path string, o *object, shared []any) *operation {
	at := method + " " + path
	op := &operation{method: method, path: path, summary: o.str("summary"), description: o.str("description"),
		id: o.str("operationId"), deprecated: o.get("deprecated") == true}
	// An operation's parameters replace those of its path of the same name
	// and place.
	for _, v := range append(slices.Clone(shared), list(o.get("parameters"))...) {
		po := p.deref(v, at)
		if po == nil {
			continue
		}
		pa := p.param(po, at)
		if i := slices.IndexFunc(op.params, func(q *param) bool { return q.name == pa.name && q.in == pa.in }); i >= 0 {
			op.params[i] = pa
		} else {
			op.params = append(op.params, pa)
		}
	}
	if rb := p.deref(o.get("requestBody"), at+" request body"); rb != nil {
		b := &body{description: rb.str("description"), required: rb.get("required") == true}
		var mo *object
		b.media, mo = media(rb.get("content"))
		if mo != nil {
			b.schema = p.schema(mo.get("schema"), at+" request body")
			b.example = exampleOf(p, mo)
		}
		op.body = b
	}
	if rs, ok := o.get("responses").(*object); ok {
		for i, status := range rs.keys {
			ro := p.deref(rs.vals[i], at+" response "+status)
			if ro == nil {
				continue
			}
			r := &response{status: status, description: ro.str("description")}
			var mo *object
			r.media, mo = media(ro.get("content"))
			if mo != nil && mo.get("schema") != nil {
				r.schema = p.schema(mo.get("schema"), at+" response "+status)
			}
			op.responses = append(op.responses, r)
		}
	}
	return op
}

func (p *parser) param(o *object, at string) *param {
	in := o.str("in")
	sv := o.get("schema")
	if sv == nil {
		// A parameter may give its schema by media type instead.
		if _, mo := media(o.get("content")); mo != nil {
			sv = mo.get("schema")
		}
	}
	f := &field{name: o.str("name"), schema: p.schema(sv, at), required: o.get("required") == true || in == "path",
		deprecated: o.get("deprecated") == true, description: o.str("description"), example: exampleOf(p, o)}
	return &param{field: f, in: in}
}

// media returns the media type of content the reference shows, the first
// in JSON, or the first, and its object.
func media(v any) (string, *object) {
	content, ok := v.(*object)
	if !ok || len(content.keys) == 0 {
		return "", nil
	}
	i := max(0, slices.IndexFunc(content.keys, func(k string) bool { return strings.Contains(k, "json") }))
	mo, _ := content.vals[i].(*object)
	return content.keys[i], mo
}

// exampleOf returns the example of o, a parameter or media type: its
// example, or the value of the first of its examples.
func exampleOf(p *parser, o *object) any {
	if v := o.get("example"); v != nil {
		return v
	}
	if ex, ok := o.get("examples").(*object); ok && len(ex.keys) > 0 {
		if eo := p.deref(ex.vals[0], "example"); eo != nil {
			return eo.get("value")
		}
	}
	return nil
}

// deref returns the object v is, following its $ref, and the $ref of what
// that finds, or nil.
func (p *parser) deref(v any, at string) *object {
	for range 32 {
		o, ok := v.(*object)
		if !ok {
			return nil
		}
		ref, ok := o.get("$ref").(string)
		if !ok {
			return o
		}
		target, err := p.lookup(ref)
		if err != nil {
			p.problem(at, err)
			return nil
		}
		v = target
	}
	p.problem(at, errors.New("$ref: references that go round in a circle"))
	return nil
}

// lookup returns the value a local $ref points to, as "#/components/schemas/Note".
func (p *parser) lookup(ref string) (any, error) {
	ptr, ok := strings.CutPrefix(ref, "#")
	if !ok {
		return nil, fmt.Errorf("$ref %s: only references within the document are read", ref)
	}
	if u, err := url.PathUnescape(ptr); err == nil {
		ptr = u
	}
	var v any = p.root
	if ptr == "" {
		return v, nil
	}
	unescape := strings.NewReplacer("~1", "/", "~0", "~")
	for _, part := range strings.Split(strings.TrimPrefix(ptr, "/"), "/") {
		part = unescape.Replace(part)
		switch x := v.(type) {
		case *object:
			v = x.get(part)
		case []any:
			i, err := strconv.Atoi(part)
			if err != nil || i < 0 || i >= len(x) {
				return nil, fmt.Errorf("$ref %s: not found", ref)
			}
			v = x[i]
		default:
			v = nil
		}
		if v == nil {
			return nil, fmt.Errorf("$ref %s: not found", ref)
		}
	}
	return v, nil
}

// schema reads the schema v, from where at says.
func (p *parser) schema(v any, at string) *schema {
	o, ok := v.(*object)
	if !ok {
		return &schema{}
	}
	if ref, ok := o.get("$ref").(string); ok {
		return p.ref(ref, at)
	}
	s := &schema{}
	p.fill(s, o, at)
	return s
}

// ref returns the schema ref points to, read once: a schema that holds
// itself gets the one being read.
func (p *parser) ref(ref, at string) *schema {
	if s, ok := p.schemas[ref]; ok {
		return s
	}
	target, err := p.lookup(ref)
	if err != nil {
		p.problem(at, err)
		return &schema{missing: ref}
	}
	s := &schema{}
	p.schemas[ref] = s
	*s = *p.schema(target, ref)
	if name, ok := strings.CutPrefix(ref, "#/components/schemas/"); ok {
		s.ref = name
	}
	return s
}

func (p *parser) fill(s *schema, o *object, at string) {
	switch t := o.get("type").(type) {
	case string:
		s.types = []string{t}
	case []any:
		s.types = strs(t)
	}
	if o.get("nullable") == true && !slices.Contains(s.types, "null") {
		s.types = append(s.types, "null")
	}
	s.format, s.description = o.str("format"), o.str("description")
	s.enum, _ = o.get("enum").([]any)
	s.example, s.def = o.get("example"), o.get("default")
	if ex, ok := o.get("examples").([]any); ok && s.example == nil && len(ex) > 0 {
		s.example = ex[0]
	}
	required := strs(o.get("required"))
	if props, ok := o.get("properties").(*object); ok {
		for i, name := range props.keys {
			f := &field{name: name, schema: p.schema(props.vals[i], at), required: slices.Contains(required, name)}
			if po, ok := props.vals[i].(*object); ok {
				f.description, f.deprecated, f.example = po.str("description"), po.get("deprecated") == true, po.get("example")
			}
			s.props = append(s.props, f)
		}
	}
	if items := o.get("items"); items != nil {
		s.items = p.schema(items, at)
	}
	all := list(o.get("allOf"))
	for _, v := range all {
		x := p.schema(v, at)
		merge(s, x)
		// A $ref wrapped in allOf, to give it a description of its own,
		// keeps its name.
		if len(all) == 1 {
			s.ref = x.ref
		}
	}
	for _, k := range []string{"oneOf", "anyOf"} {
		for i, v := range list(o.get(k)) {
			x := p.schema(v, at)
			name := x.ref
			if name == "" {
				name = "option " + strconv.Itoa(i+1)
			}
			s.variants = append(s.variants, &field{name: name, schema: x})
			s.choice = map[string]string{"oneOf": "one of", "anyOf": "any of"}[k]
		}
	}
	// The properties allOf brings may be required here.
	for i, f := range s.props {
		if !f.required && slices.Contains(required, f.name) {
			r := *f
			r.required = true
			s.props[i] = &r
		}
	}
}

// merge adds to s what x says of a value that matches both, as allOf does.
func merge(s, x *schema) {
	if len(s.types) == 0 {
		s.types = x.types
	}
	if s.format == "" {
		s.format = x.format
	}
	if s.description == "" {
		s.description = x.description
	}
	if len(s.enum) == 0 {
		s.enum = x.enum
	}
	if s.items == nil {
		s.items = x.items
	}
	for _, f := range x.props {
		if !slices.ContainsFunc(s.props, func(g *field) bool { return g.name == f.name }) {
			s.props = append(s.props, f)
		}
	}
	if len(s.variants) == 0 {
		s.variants, s.choice = x.variants, x.choice
	}
}

// sample returns a value s describes, for a request's body: its example,
// default or first value allowed, or one made of its properties'.
func sample(s *schema, seen []*schema) any {
	switch {
	case s.example != nil:
		return s.example
	case s.def != nil:
		return s.def
	case len(s.enum) > 0:
		return s.enum[0]
	case slices.Contains(seen, s):
		return nil
	}
	seen = append(seen, s)
	switch {
	case len(s.props) > 0:
		o := &object{}
		for _, f := range s.props {
			v := f.example
			if v == nil {
				v = sample(f.schema, seen)
			}
			o.keys, o.vals = append(o.keys, f.name), append(o.vals, v)
		}
		return o
	case len(s.variants) > 0:
		return sample(s.variants[0].schema, seen)
	}
	switch s.kind() {
	case "array":
		return []any{sample(s.items, seen)}
	case "object":
		return &object{}
	case "string":
		return "string"
	case "integer", "number":
		return json.Number("0")
	case "boolean":
		return false
	}
	return nil
}

// object is a JSON object, its members in the order written.
type object struct {
	keys []string
	vals []any // string, json.Number, bool, nil, []any or *object
}

func (o *object) get(key string) any {
	if i := slices.Index(o.keys, key); i >= 0 {
		return o.vals[i]
	}
	return nil
}

func (o *object) str(key string) string {
	s, _ := o.get(key).(string)
	return s
}

func list(v any) []any {
	a, _ := v.([]any)
	return a
}

// strs returns the strings of the list v.
func strs(v any) []string {
	var out []string
	for _, x := range list(v) {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// decode reads the next value of dec.
func decode(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	d, ok := tok.(json.Delim)
	if !ok {
		return tok, nil
	}
	if d == '[' {
		a := []any{}
		for dec.More() {
			v, err := decode(dec)
			if err != nil {
				return nil, err
			}
			a = append(a, v)
		}
		_, err := dec.Token()
		return a, err
	}
	o := &object{}
	for dec.More() {
		k, err := dec.Token()
		if err != nil {
			return nil, err
		}
		v, err := decode(dec)
		if err != nil {
			return nil, err
		}
		o.keys, o.vals = append(o.keys, k.(string)), append(o.vals, v)
	}
	_, err = dec.Token()
	return o, err
}

// encode returns v as JSON on one line, its objects' members in order.
func encode(v any) string {
	var b strings.Builder
	write(&b, v)
	return b.String()
}

func write(b *strings.Builder, v any) {
	switch v := v.(type) {
	case *object:
		b.WriteByte('{')
		for i, k := range v.keys {
			if i > 0 {
				b.WriteString(", ")
			}
			name, _ := json.Marshal(k)
			b.Write(name)
			b.WriteString(": ")
			write(b, v.vals[i])
		}
		b.WriteByte('}')
	case []any:
		b.WriteByte('[')
		for i, x := range v {
			if i > 0 {
				b.WriteString(", ")
			}
			write(b, x)
		}
		b.WriteByte(']')
	default:
		s, _ := json.Marshal(v)
		b.Write(s)
	}
}

// indented returns v as JSON indented by two spaces.
func indented(v any) string {
	var b bytes.Buffer
	if err := json.Indent(&b, []byte(encode(v)), "", "  "); err != nil {
		return encode(v)
	}
	return b.String()
}

// plain returns v as encoding/json decodes into an any: maps for objects
// and float64 for numbers.
func plain(v any) any {
	switch v := v.(type) {
	case *object:
		m := make(map[string]any, len(v.keys))
		for i, k := range v.keys {
			m[k] = plain(v.vals[i])
		}
		return m
	case []any:
		a := make([]any, len(v))
		for i, x := range v {
			a[i] = plain(x)
		}
		return a
	case json.Number:
		f, _ := v.Float64()
		return f
	}
	return v
}
