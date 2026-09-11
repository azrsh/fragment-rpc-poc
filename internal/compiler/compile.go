package compiler

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/azrsh/fragment-colocation-with-grpc/internal/plan"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"github.com/vektah/gqlparser/v2/parser"
	"github.com/vektah/gqlparser/v2/validator"
)

type NativeSource struct {
	Language    string `json:"language"`
	Declaration string `json:"declaration"`
	Kind        string `json:"kind"`
	PackageName string `json:"packageName,omitempty"`
}
type SourceFile struct {
	Name   string        `json:"name"`
	Body   string        `json:"body"`
	Line   int           `json:"line,omitempty"`
	Column int           `json:"column,omitempty"`
	Native *NativeSource `json:"native,omitempty"`
}
type LockedField struct {
	Number    int    `json:"number"`
	Signature string `json:"signature"`
}
type LockedMessage struct {
	Fields  map[string]LockedField `json:"fields"`
	Retired []string               `json:"retired"`
}
type ContractLock struct {
	Version  int                       `json:"version"`
	Messages map[string]*LockedMessage `json:"messages"`
}
type Fragment struct {
	Name   string
	Type   string
	Fields []plan.Field
	Native *NativeSource
}
type Output struct {
	Proto         string
	Operations    []plan.Operation
	Lock          ContractLock
	FragmentTypes string
	Fragments     []Fragment
}

var scalarTypes = map[string]string{"ID": "string", "String": "string", "Int": "int32", "Float": "double", "Boolean": "bool"}
var camelName = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)
var queryName = regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)

func upper(s string) string { return strings.ToUpper(s[:1]) + s[1:] }

func diagnostic(err error) error {
	var list gqlerror.List
	switch e := err.(type) {
	case gqlerror.List:
		list = e
	case *gqlerror.Error:
		list = gqlerror.List{e}
	default:
		return err
	}
	var messages []string
	for _, e := range list {
		file, _ := e.Extensions["file"].(string)
		if len(e.Locations) > 0 {
			messages = append(messages, fmt.Sprintf("%s:%d:%d: %s", file, e.Locations[0].Line, e.Locations[0].Column, e.Message))
		} else {
			messages = append(messages, e.Message)
		}
	}
	return fmt.Errorf("%s", strings.Join(messages, "\n"))
}

func parseSource(s SourceFile) (*ast.QueryDocument, error) {
	line, column := max(1, s.Line), max(1, s.Column)
	body := s.Body
	if s.Native != nil && s.Native.Language == "swift" {
		body = strings.ReplaceAll(body, "\n", "\n"+strings.Repeat(" ", column-1))
	}
	body = strings.Repeat("\n", line-1) + strings.Repeat(" ", column-1) + body
	d, err := parser.ParseQuery(&ast.Source{Name: s.Name, Input: body})
	if err != nil {
		return nil, diagnostic(err)
	}
	if s.Native != nil {
		valid := len(d.Operations)+len(d.Fragments) == 1
		if s.Native.Kind == "fragment" {
			valid = valid && len(d.Fragments) == 1
		} else {
			valid = valid && len(d.Operations) == 1 && d.Operations[0].Operation == ast.Query
		}
		if !valid {
			return nil, fmt.Errorf("%s: %s must declare exactly one %s", s.Name, s.Native.Declaration, s.Native.Kind)
		}
	}
	return d, nil
}

func Compile(sdl string, sources []SourceFile, previous *ContractLock) (*Output, error) {
	schema, err := gqlparser.LoadSchema(&ast.Source{Name: "schema.graphql", Input: sdl})
	if err != nil {
		return nil, diagnostic(err)
	}
	doc := &ast.QueryDocument{}
	owners := map[*ast.FragmentDefinition]*NativeSource{}
	for _, s := range sources {
		d, err := parseSource(s)
		if err != nil {
			return nil, err
		}
		doc.Operations = append(doc.Operations, d.Operations...)
		doc.Fragments = append(doc.Fragments, d.Fragments...)
		for _, f := range d.Fragments {
			owners[f] = s.Native
		}
	}
	if errs := validator.Validate(schema, doc); len(errs) > 0 {
		return nil, diagnostic(errs)
	}
	checkDirectives := func(ds ast.DirectiveList) error {
		if len(ds) > 0 {
			return fmt.Errorf("Directive @%s is not supported by this PoC", ds[0].Name)
		}
		return nil
	}
	var checkSet func(ast.SelectionSet) error
	checkSet = func(set ast.SelectionSet) error {
		for _, sel := range set {
			var ds ast.DirectiveList
			var child ast.SelectionSet
			switch n := sel.(type) {
			case *ast.Field:
				ds = n.Directives
				child = n.SelectionSet
			case *ast.FragmentSpread:
				ds = n.Directives
			case *ast.InlineFragment:
				ds = n.Directives
				child = n.SelectionSet
			}
			if err := checkDirectives(ds); err != nil {
				return err
			}
			if err := checkSet(child); err != nil {
				return err
			}
		}
		return nil
	}
	for _, o := range doc.Operations {
		if err := checkDirectives(o.Directives); err != nil {
			return nil, err
		}
		for _, v := range o.VariableDefinitions {
			if err := checkDirectives(v.Directives); err != nil {
				return nil, err
			}
		}
		if err := checkSet(o.SelectionSet); err != nil {
			return nil, err
		}
	}
	for _, f := range doc.Fragments {
		if len(f.VariableDefinition) > 0 {
			return nil, fmt.Errorf("Fragment variables are not supported")
		}
		if err := checkDirectives(f.Directives); err != nil {
			return nil, err
		}
		if err := checkSet(f.SelectionSet); err != nil {
			return nil, err
		}
	}
	var fieldsFor func(*ast.Definition, ast.SelectionSet) ([]plan.Field, error)
	fieldsFor = func(typ *ast.Definition, set ast.SelectionSet) ([]plan.Field, error) {
		merged := map[string][]*ast.Field{}
		var collect func(ast.SelectionSet) error
		collect = func(set ast.SelectionSet) error {
			for _, sel := range set {
				switch n := sel.(type) {
				case *ast.Field:
					if !camelName.MatchString(n.Alias) {
						return fmt.Errorf("Field/alias '%s' must use lowerCamelCase", n.Alias)
					}
					merged[n.Alias] = append(merged[n.Alias], n)
				case *ast.FragmentSpread:
					f := doc.Fragments.ForName(n.Name)
					if f.TypeCondition != typ.Name {
						return fmt.Errorf("Polymorphic fragments are not supported")
					}
					if err := collect(f.SelectionSet); err != nil {
						return err
					}
				case *ast.InlineFragment:
					if n.TypeCondition != "" && n.TypeCondition != typ.Name {
						return fmt.Errorf("Polymorphic fragments are not supported")
					}
					if err := collect(n.SelectionSet); err != nil {
						return err
					}
				}
			}
			return nil
		}
		if err := collect(set); err != nil {
			return nil, err
		}
		keys := make([]string, 0, len(merged))
		for k := range merged {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fields := make([]plan.Field, 0, len(keys))
		for _, key := range keys {
			nodes := merged[key]
			first := nodes[0]
			def := typ.Fields.ForName(first.Name)
			if def == nil || strings.HasPrefix(first.Name, "__") {
				return nil, fmt.Errorf("Introspection field %s is not supported", first.Name)
			}
			for _, a := range def.Arguments {
				if a.DefaultValue != nil {
					return nil, fmt.Errorf("Schema argument defaults are not supported: %s.%s", typ.Name, first.Name)
				}
			}
			t := def.Type
			list := t.Elem != nil
			if list && (!t.NonNull || !t.Elem.NonNull || t.Elem.Elem != nil) {
				return nil, fmt.Errorf("Only non-null lists with non-null, non-list items are supported: %s", t.String())
			}
			f := plan.Field{Name: first.Name, ResponseName: key, Coordinate: typ.Name + "." + first.Name, Type: t.Name(), Required: t.NonNull, List: list, Args: map[string]plan.Binding{}}
			for _, a := range first.Arguments {
				v := a.Value
				b := plan.Binding{}
				switch v.Kind {
				case ast.Variable:
					b.Variable = v.Raw
				case ast.StringValue, ast.BlockValue, ast.IntValue, ast.FloatValue, ast.BooleanValue, ast.NullValue:
					b.Literal, err = v.Value(nil)
					if err != nil {
						return nil, err
					}
				default:
					return nil, fmt.Errorf("Only scalar arguments are supported: %s", f.Coordinate)
				}
				f.Args[a.Name] = b
			}
			named := schema.Types[t.Name()]
			if named.Kind == ast.Object {
				var children ast.SelectionSet
				for _, n := range nodes {
					children = append(children, n.SelectionSet...)
				}
				f.Fields, err = fieldsFor(named, children)
				if err != nil {
					return nil, err
				}
			} else if scalarTypes[t.Name()] == "" {
				return nil, fmt.Errorf("Unsupported output type %s", t.Name())
			}
			fields = append(fields, f)
		}
		return fields, nil
	}
	output := &Output{Operations: []plan.Operation{}, Fragments: []Fragment{}}
	for _, o := range doc.Operations {
		if o.Operation != ast.Query || !queryName.MatchString(o.Name) {
			return nil, fmt.Errorf("Only named queries with PascalCase names are supported")
		}
		vars := []plan.Variable{}
		for _, v := range o.VariableDefinitions {
			if !camelName.MatchString(v.Variable) {
				return nil, fmt.Errorf("Variable '%s' must use lowerCamelCase", v.Variable)
			}
			if v.DefaultValue != nil {
				return nil, fmt.Errorf("Variable defaults are not supported: $%s", v.Variable)
			}
			if v.Type.Elem != nil || scalarTypes[v.Type.Name()] == "" {
				return nil, fmt.Errorf("Only scalar variables are supported: $%s", v.Variable)
			}
			vars = append(vars, plan.Variable{Name: v.Variable, Type: v.Type.Name(), Required: v.Type.NonNull})
		}
		fields, err := fieldsFor(schema.Query, o.SelectionSet)
		if err != nil {
			return nil, err
		}
		output.Operations = append(output.Operations, plan.Operation{Name: o.Name, Variables: vars, Fields: fields})
	}
	sort.Slice(output.Operations, func(i, j int) bool { return output.Operations[i].Name < output.Operations[j].Name })
	if len(output.Operations) == 0 {
		return nil, fmt.Errorf("At least one named query is required")
	}
	output.Lock = ContractLock{Version: 1, Messages: map[string]*LockedMessage{}}
	if previous != nil {
		if previous.Version != 1 {
			return nil, fmt.Errorf("Unsupported contract lock version")
		}
		data, err := json.Marshal(previous)
		if err != nil {
			return nil, err
		}
		if err = json.Unmarshal(data, &output.Lock); err != nil {
			return nil, err
		}
	}
	if output.Lock.Messages == nil {
		return nil, fmt.Errorf("Invalid contract lock messages")
	}
	var messages []string
	type protoField struct{ name, typ, signature, modifier string }
	emit := func(name string, fields []protoField) error {
		entry := output.Lock.Messages[name]
		if entry == nil {
			entry = &LockedMessage{Fields: map[string]LockedField{}, Retired: []string{}}
		}
		active := map[string]bool{}
		for _, f := range fields {
			active[f.name] = true
		}
		next := 1
		for n, f := range entry.Fields {
			next = max(next, f.Number+1)
			if !active[n] && !slices.Contains(entry.Retired, n) {
				entry.Retired = append(entry.Retired, n)
			}
		}
		var lines []string
		for _, f := range fields {
			if slices.Contains(entry.Retired, f.name) {
				return fmt.Errorf("Retired field cannot be reused: %s.%s", name, f.name)
			}
			old, ok := entry.Fields[f.name]
			if ok && old.Signature != f.signature {
				return fmt.Errorf("Breaking type change: %s.%s (%s → %s)", name, f.name, old.Signature, f.signature)
			}
			if !ok {
				if next == 19000 {
					next = 20000
				}
				old = LockedField{Number: next, Signature: f.signature}
				entry.Fields[f.name] = old
				next++
			}
			lines = append(lines, fmt.Sprintf("  %s%s %s = %d;", f.modifier, f.typ, f.name, old.Number))
		}
		sort.Strings(entry.Retired)
		if len(entry.Retired) > 0 {
			numbers := []int{}
			names := []string{}
			for _, n := range entry.Retired {
				numbers = append(numbers, entry.Fields[n].Number)
				names = append(names, strconv.Quote(n))
			}
			sort.Ints(numbers)
			nums := []string{}
			for _, n := range numbers {
				nums = append(nums, strconv.Itoa(n))
			}
			lines = append(lines, "  reserved "+strings.Join(nums, ", ")+";", "  reserved "+strings.Join(names, ", ")+";")
		}
		output.Lock.Messages[name] = entry
		messages = append(messages, "message "+name+" {\n"+strings.Join(lines, "\n")+"\n}")
		return nil
	}
	signature := func(typ string, list, required bool) string {
		if list {
			typ += "[]"
		}
		if required {
			return typ + "!"
		}
		return typ + "?"
	}
	var response func(string, []plan.Field) error
	response = func(name string, fields []plan.Field) error {
		fs := []protoField{}
		for _, f := range fields {
			typ := scalarTypes[f.Type]
			modifier := ""
			if f.Fields != nil {
				typ = name + "_" + upper(f.ResponseName)
				if err := response(typ, f.Fields); err != nil {
					return err
				}
			}
			if f.List {
				modifier = "repeated "
			} else if f.Fields == nil && !f.Required {
				modifier = "optional "
			}
			fs = append(fs, protoField{f.ResponseName, typ, signature(f.Type, f.List, f.Required), modifier})
		}
		return emit(name, fs)
	}
	for _, o := range output.Operations {
		fs := []protoField{}
		for _, v := range o.Variables {
			fs = append(fs, protoField{v.Name, scalarTypes[v.Type], signature(v.Type, false, v.Required), "optional "})
		}
		if err := emit(o.Name+"Request", fs); err != nil {
			return nil, err
		}
		if err := response(o.Name+"Response", o.Fields); err != nil {
			return nil, err
		}
	}
	lines := []string{"// Generated by npm run generate. Edit colocated .graphql sources instead.", `syntax = "proto3";`, `package app.v1;`, "", "service AppService {"}
	for _, o := range output.Operations {
		lines = append(lines, fmt.Sprintf("  rpc %s(%sRequest) returns (%sResponse);", o.Name, o.Name, o.Name))
	}
	lines = append(lines, "}", "")
	lines = append(lines, messages...)
	output.Proto = strings.Join(lines, "\n") + "\n"
	sort.Slice(doc.Fragments, func(i, j int) bool { return doc.Fragments[i].Name < doc.Fragments[j].Name })
	for _, f := range doc.Fragments {
		typ := schema.Types[f.TypeCondition]
		if typ.Kind != ast.Object {
			return nil, fmt.Errorf("Only concrete object fragments are supported: %s", f.Name)
		}
		fields, err := fieldsFor(typ, f.SelectionSet)
		if err != nil {
			return nil, err
		}
		output.Fragments = append(output.Fragments, Fragment{Name: f.Name, Type: typ.Name, Fields: fields, Native: owners[f]})
		output.FragmentTypes += "export type " + f.Name + " = " + tsShape(fields) + ";\n"
	}
	return output, nil
}

func tsShape(fields []plan.Field) string {
	parts := []string{}
	for _, f := range fields {
		typ := "string"
		if f.Type == "Boolean" {
			typ = "boolean"
		} else if f.Type == "Int" || f.Type == "Float" {
			typ = "number"
		}
		if f.Fields != nil {
			typ = tsShape(f.Fields)
		}
		optional := ""
		if !f.Required || (f.Fields != nil && !f.List) {
			optional = "?"
		}
		if f.List {
			typ += "[]"
		}
		parts = append(parts, f.ResponseName+optional+": "+typ+";")
	}
	return "{ " + strings.Join(parts, " ") + " }"
}
