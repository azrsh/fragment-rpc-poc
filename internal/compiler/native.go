package compiler

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/azrsh/fragment-colocation-with-grpc/internal/plan"
)

type NativeField struct {
	Name       string                  `json:"name"`
	Coordinate string                  `json:"coordinate"`
	Args       map[string]plan.Binding `json:"args"`
	Type       string                  `json:"type"`
	Optional   bool                    `json:"optional"`
	List       bool                    `json:"list"`
	Object     bool                    `json:"object"`
}
type NativeModel struct {
	Name       string        `json:"name"`
	SchemaType string        `json:"schemaType"`
	Fields     []NativeField `json:"fields"`
	Language   string        `json:"language"`
	Proto      bool          `json:"proto"`
}
type NativeConversion struct {
	From NativeModel `json:"from"`
	To   NativeModel `json:"to"`
}
type NativeDeclaration struct {
	NativeSource
	Name      string `json:"name"`
	Document  string `json:"document"`
	Operation string `json:"operation"`
}
type NativeOutput struct {
	Models       []NativeModel       `json:"models"`
	Conversions  []NativeConversion  `json:"conversions"`
	Declarations []NativeDeclaration `json:"declarations"`
}

func NativeModels(output *Output, sources []SourceFile) (*NativeOutput, error) {
	native := &NativeOutput{Models: []NativeModel{}, Conversions: []NativeConversion{}, Declarations: []NativeDeclaration{}}
	names := map[string]bool{}
	for _, s := range sources {
		if s.Native == nil {
			continue
		}
		d, err := parseSource(s)
		if err != nil {
			return nil, err
		}
		name := s.Native.Declaration + "Data"
		key := s.Native.Language + ":" + name
		if names[key] {
			return nil, fmt.Errorf("Duplicate native generated type %s", name)
		}
		names[key] = true
		operation := ""
		if len(d.Operations) > 0 {
			operation = d.Operations[0].Name
		} else {
			operation = d.Fragments[0].Name
		}
		native.Declarations = append(native.Declarations, NativeDeclaration{NativeSource: *s.Native, Name: name, Document: s.Body, Operation: operation})
	}
	var add func(string, string, []plan.Field, string, bool)
	add = func(name, schemaType string, fields []plan.Field, language string, proto bool) {
		mapped := []NativeField{}
		for _, f := range fields {
			typ := f.Type
			if f.Fields != nil {
				typ = name + "_" + upper(f.ResponseName)
				add(typ, f.Type, f.Fields, language, proto)
			}
			mapped = append(mapped, NativeField{Name: f.ResponseName, Coordinate: f.Coordinate, Args: f.Args, Type: typ, Optional: !f.Required, List: f.List, Object: f.Fields != nil})
		}
		native.Models = append(native.Models, NativeModel{Name: name, SchemaType: schemaType, Fields: mapped, Language: language, Proto: proto})
	}
	for _, f := range output.Fragments {
		if f.Native != nil {
			add(f.Native.Declaration+"Data", f.Type, f.Fields, f.Native.Language, false)
		}
	}
	for _, language := range []string{"swift", "kotlin"} {
		for _, o := range output.Operations {
			for _, d := range native.Declarations {
				if d.Language == language && d.Operation == o.Name {
					add(o.Name+"Response", "Query", o.Fields, language, true)
					break
				}
			}
		}
	}
	find := func(name, language string) NativeModel {
		for _, m := range native.Models {
			if m.Name == name && m.Language == language {
				return m
			}
		}
		return NativeModel{}
	}
	var contains func(NativeModel, NativeModel) bool
	contains = func(from, to NativeModel) bool {
		if from.SchemaType != to.SchemaType {
			return false
		}
		for _, field := range to.Fields {
			var f *NativeField
			for i := range from.Fields {
				if from.Fields[i].Name == field.Name {
					f = &from.Fields[i]
					break
				}
			}
			if f == nil || f.Coordinate != field.Coordinate || !reflect.DeepEqual(f.Args, field.Args) || f.Optional != field.Optional || f.List != field.List || f.Object != field.Object {
				return false
			}
			if field.Object {
				if !contains(find(f.Type, from.Language), find(field.Type, to.Language)) {
					return false
				}
			} else if f.Type != field.Type {
				return false
			}
		}
		return true
	}
	for _, to := range native.Models {
		if to.Proto {
			continue
		}
		for _, from := range native.Models {
			if from.Language == to.Language && from.Name != to.Name && contains(from, to) {
				native.Conversions = append(native.Conversions, NativeConversion{From: from, To: to})
			}
		}
	}
	return native, nil
}

func SwiftModels(native *NativeOutput) string {
	scalars := map[string]string{"ID": "String", "String": "String", "Int": "Int32", "Float": "Double", "Boolean": "Bool"}
	output := []string{"// Generated from Swift GraphQL declarations.\n"}
	for _, d := range native.Declarations {
		if d.Language == "swift" && d.Kind == "query" {
			output = append(output, "typealias "+d.Name+" = App_V1_"+d.Operation+"Response")
		}
	}
	for _, m := range native.Models {
		if m.Language != "swift" || m.Proto {
			continue
		}
		fields := []string{}
		for _, f := range m.Fields {
			typ := f.Type
			if !f.Object {
				typ = scalars[f.Type]
			}
			if f.List {
				typ = "[" + typ + "]"
			}
			if f.Optional {
				typ += "?"
			}
			fields = append(fields, "    let "+f.Name+": "+typ)
		}
		output = append(output, "struct "+m.Name+": Equatable {\n"+strings.Join(fields, "\n")+"\n}")
	}
	for _, c := range native.Conversions {
		from, to := c.From, c.To
		if from.Language != "swift" {
			continue
		}
		name := from.Name
		if from.Proto {
			name = "App_V1_" + name
		}
		fields := []string{}
		for _, f := range to.Fields {
			property := f.Name
			if from.Proto && strings.HasSuffix(property, "Url") {
				property = strings.TrimSuffix(property, "Url") + "URL"
			}
			value := "self." + property
			if f.Object {
				if f.List {
					value += ".map { $0.as" + f.Type + "() }"
				} else {
					if f.Optional && !from.Proto {
						value += "?"
					}
					value += ".as" + f.Type + "()"
				}
			}
			if f.Optional && from.Proto {
				value = "self.has" + upper(property) + " ? " + value + " : nil"
			}
			fields = append(fields, "            "+f.Name+": "+value)
		}
		output = append(output, "extension "+name+" {\n    func as"+to.Name+"() -> "+to.Name+" {\n        "+to.Name+"(\n"+strings.Join(fields, ",\n")+"\n        )\n    }\n}")
	}
	return strings.Join(output, "\n\n") + "\n"
}
