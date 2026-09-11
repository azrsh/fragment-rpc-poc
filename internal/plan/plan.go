package plan

import "encoding/json"

type Binding struct {
	Variable string `json:"variable,omitempty"`
	Literal  any    `json:"literal,omitempty"`
}

// A null literal must remain distinguishable from a variable in generated metadata.
func (b Binding) MarshalJSON() ([]byte, error) {
	if b.Variable != "" {
		return json.Marshal(struct {
			Variable string `json:"variable"`
		}{b.Variable})
	}
	return json.Marshal(struct {
		Literal any `json:"literal"`
	}{b.Literal})
}

type Field struct {
	Name         string             `json:"name"`
	ResponseName string             `json:"responseName"`
	Coordinate   string             `json:"coordinate"`
	Type         string             `json:"type"`
	Required     bool               `json:"required"`
	List         bool               `json:"list"`
	Args         map[string]Binding `json:"args"`
	Fields       []Field            `json:"fields,omitempty"`
}

type Variable struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

type Operation struct {
	Name      string     `json:"name"`
	Variables []Variable `json:"variables"`
	Fields    []Field    `json:"fields"`
}
