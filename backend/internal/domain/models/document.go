package models

const (
	FormatVersion = 1
)

type Document struct {
	ID            string     `json:"id"`
	FormatVersion int        `json:"formatVersion"`
	Fields        Fields     `json:"fields"`
	Properties    Properties `json:"properties"`
}

func NewDocument(id string) *Document {
	return &Document{
		ID:            id,
		FormatVersion: FormatVersion,
		Fields:        NewFields(),
		Properties:    NewProperties(),
	}
}

type Fields []*Field

func NewFields() Fields {
	return make(Fields, 0)
}

func (f *Fields) Add(field *Field) *Fields {
	*f = append(*f, field)
	return f
}

type Field struct {
	Name       string     `json:"name"`
	Value      []*Value   `json:"value"`
	Properties Properties `json:"properties"`
}

func NewField(name string) *Field {
	return &Field{
		Name:       name,
		Value:      NewValues(),
		Properties: NewProperties(),
	}
}

type Properties map[string]Value

func NewProperties() Properties {
	return make(Properties)
}
