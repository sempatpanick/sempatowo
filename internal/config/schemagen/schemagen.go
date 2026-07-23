// Package schemagen builds a JSON Schema for the config file from the Go source
// that defines it.
//
// A config file cannot carry comments — JSON has none, and the program rewrites
// the file anyway — so the documentation for each key has to live somewhere the
// editor can find it. Deriving the schema from the AST rather than from
// `desc:"…"` struct tags keeps one source of truth: the doc comments in
// config.go are already written, already good, and already what you would want
// to see on hover.
//
// It lives outside package config so that go/parser stays out of the bot binary;
// only the generator command and the freshness test import it.
package schemagen

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"reflect"
	"strings"
)

// Root is the type the schema describes.
const Root = "Settings"

// Generate parses the Go package in srcDir and returns the JSON Schema for Root.
func Generate(srcDir string) ([]byte, error) {
	g, err := parseDir(srcDir)
	if err != nil {
		return nil, err
	}

	root, ok := g.object(Root)
	if !ok {
		return nil, fmt.Errorf("schemagen: type %s not found in %s", Root, srcDir)
	}
	root["$schema"] = "http://json-schema.org/draft-07/schema#"
	root["title"] = "sempatowo account config"

	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

type generator struct {
	structs map[string]*ast.StructType
	docs    map[string]string
}

func parseDir(dir string) (*generator, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	g := &generator{
		structs: map[string]*ast.StructType{},
		docs:    map[string]string{},
	}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok || gd.Tok != token.TYPE {
					continue
				}
				for _, spec := range gd.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					st, ok := ts.Type.(*ast.StructType)
					if !ok {
						continue
					}
					g.structs[ts.Name.Name] = st
					// A single-spec declaration carries its comment on the
					// GenDecl, a grouped one on the TypeSpec.
					doc := docText(ts.Doc)
					if doc == "" {
						doc = docText(gd.Doc)
					}
					g.docs[ts.Name.Name] = doc
				}
			}
		}
	}
	return g, nil
}

// object renders a named struct type as a JSON Schema object.
func (g *generator) object(name string) (map[string]any, bool) {
	st, ok := g.structs[name]
	if !ok {
		return nil, false
	}

	props := map[string]any{}
	g.collect(st, props)

	out := map[string]any{
		"type":       "object",
		"properties": props,
		// The point of the schema in an editor is the red squiggle under a
		// misspelled key, which is what this line buys.
		"additionalProperties": false,
	}
	if doc := g.docs[name]; doc != "" {
		out["description"] = doc
	}
	return out, true
}

// collect adds st's fields to props, following the same rules encoding/json
// does: an anonymous field with a tag nests under that name, an anonymous field
// without one merges its fields into the parent.
func (g *generator) collect(st *ast.StructType, props map[string]any) {
	for _, field := range st.Fields.List {
		name, ok := jsonName(field)
		if !ok {
			continue
		}

		if len(field.Names) == 0 && name == "" {
			if embedded, ok := g.structs[typeName(field.Type)]; ok {
				g.collect(embedded, props)
			}
			continue
		}
		if name == "" {
			continue
		}

		schema := g.fieldSchema(field.Type)
		if doc := docText(field.Doc); doc != "" {
			schema["description"] = doc
		} else if doc := docText(field.Comment); doc != "" {
			schema["description"] = doc
		}
		props[name] = schema
	}
}

func (g *generator) fieldSchema(expr ast.Expr) map[string]any {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return g.fieldSchema(t.X)
	case *ast.Ident:
		switch t.Name {
		case "string":
			return map[string]any{"type": "string"}
		case "bool":
			return map[string]any{"type": "boolean"}
		case "int", "int64":
			return map[string]any{"type": "integer"}
		case "float64", "float32":
			return map[string]any{"type": "number"}
		case "Duration":
			return durationSchema()
		case "Range":
			return rangeSchema()
		}
		if obj, ok := g.object(t.Name); ok {
			return obj
		}
	}
	// Anything unrecognised is left unconstrained rather than guessed at: a
	// wrong constraint would reject a valid file.
	return map[string]any{}
}

func durationSchema() map[string]any {
	return map[string]any{
		"type":        "string",
		"pattern":     `^-?([0-9]+(\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$`,
		"description": `A duration with its unit, as Go's time.ParseDuration accepts: "15s", "5m", "1m30s". A bare number is rejected rather than guessed at.`,
	}
}

func rangeSchema() map[string]any {
	return map[string]any{
		"description": `An inclusive wait range. Either a bare duration ("30s") for a fixed wait, or {"min": "…", "max": "…"} to jitter between two.`,
		"oneOf": []any{
			durationSchema(),
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"min": durationSchema(),
					"max": durationSchema(),
				},
				"additionalProperties": false,
			},
		},
	}
}

// jsonName returns the field's key in the file. The second result is false when
// the field is not serialised at all.
func jsonName(field *ast.Field) (string, bool) {
	tag := ""
	if field.Tag != nil {
		// The literal includes its backquotes, which StructTag does not want.
		tag = reflect.StructTag(strings.Trim(field.Tag.Value, "`")).Get("json")
	}
	name, _, _ := strings.Cut(tag, ",")

	if name == "-" {
		return "", false
	}
	if name != "" {
		return name, true
	}
	if len(field.Names) == 0 {
		// Embedded and untagged: the caller merges it.
		return "", true
	}
	if !field.Names[0].IsExported() {
		return "", false
	}
	return field.Names[0].Name, true
}

func typeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return typeName(t.X)
	}
	return ""
}

// docText flattens a comment group into one line. Tooltips render it as a
// paragraph, so the source's hard wrapping has to come out.
func docText(g *ast.CommentGroup) string {
	if g == nil {
		return ""
	}
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(g.Text()), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, " ")
}
