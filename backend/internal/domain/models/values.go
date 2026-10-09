package models

import (
	"encoding/json"
	"fmt"
)

type ValueType string

const (
	ValueTypeBool      = "bool"
	ValueTypeInt       = "int"
	ValueTypeDecimal   = "decimal"
	ValueTypeDouble    = "double"
	ValueTypeText      = "text"
	ValueTypeMultiline = "multiline"
	ValueTypeDateTime  = "datetime"
	ValueTypeFile      = "file"
	ValueTypeIntRef    = "intref"

	ValueTypeIntRefUser  = "user"
	ValueTypeIntRefGroup = "group"
)

func ToValueType(value string) ValueType {
	switch value {
	case ValueTypeBool, ValueTypeInt, ValueTypeDecimal, ValueTypeDouble, ValueTypeText, ValueTypeMultiline, ValueTypeDateTime, ValueTypeFile, ValueTypeIntRef:
		return ValueType(value)
	default:
		return ""
	}
}

func TypeIntRef(ref string) string {
	switch ref {
	case ValueTypeIntRefUser, ValueTypeIntRefGroup:
		return fmt.Sprintf("intref:%s", ref)
	default:
		return ""
	}
}

type Values []*Value

func NewValues() Values {
	return make(Values, 0)
}

type Value struct {
	Type  string `json:"type"`
	Value any    `json:"value,omitempty"`
}

func NewValue(typ string) *Value {
	return &Value{
		Type:  typ,
		Value: nil,
	}
}

func (v *Value) WithString(value string) *Value {
	v.Value = value
	return v
}

func (v *Value) WithNumber(value float64) *Value {
	v.Value = value
	return v
}

func (v *Value) WithFile(id string) *Value {
	v.Value = NewFileValue(id)
	return v
}

func (v *Value) WithIntRef(ref string) *Value {
	v.Value = NewIntRefValue(ref)
	return v
}

type FileValue struct {
	Reference   string     `json:"reference"`
	Hash        string     `json:"hash"`
	FileName    string     `json:"fileName"`
	ContentType string     `json:"contentType"`
	Size        int        `json:"contentLength"`
	Properties  Properties `json:"properties"`
}

func NewFileValue(id string) *FileValue {
	return &FileValue{
		Reference:  fmt.Sprintf("blob:%s", id),
		Properties: NewProperties(),
	}
}

func (f *FileValue) WithHash(hash string) *FileValue {
	f.Hash = hash
	return f
}

func (f *FileValue) WithFileName(fileName string) *FileValue {
	f.FileName = fileName
	return f
}

func (f *FileValue) WithContentType(contentType string) *FileValue {
	f.ContentType = contentType
	return f
}

func (f *FileValue) WithSize(size int) *FileValue {
	f.Size = size
	return f
}

func (f *FileValue) AddProperty(key string, value Value) *FileValue {
	f.Properties[key] = value
	return f
}

func (f *FileValue) GetProperty(key string) *Value {
	v, ok := f.Properties[key]
	if !ok {
		return nil
	}
	return &v
}

func ParseFileValue(jsonstr string) *FileValue {
	var fileValue FileValue
	err := json.Unmarshal([]byte(jsonstr), &fileValue)
	if err != nil {
		return nil
	}
	return &fileValue
}

type IntRefValue struct {
	Reference  string     `json:"reference"`
	Filter     string     `json:"filter"`
	Label      LabelText  `json:"label"`
	Properties Properties `json:"properties"`
}

func NewIntRefValue(ref string) *IntRefValue {
	return &IntRefValue{
		Reference:  ref,
		Filter:     "",
		Label:      LabelText{},
		Properties: NewProperties(),
	}
}

func (i *IntRefValue) WithFilter(filter string) *IntRefValue {
	i.Filter = filter
	return i
}

func (i *IntRefValue) WithLabel(label LabelText) *IntRefValue {
	i.Label = label
	return i
}

func (i *IntRefValue) AddProperty(key string, value Value) *IntRefValue {
	i.Properties[key] = value
	return i
}

func (i *IntRefValue) GetProperty(key string) *Value {
	v, ok := i.Properties[key]
	if !ok {
		return nil
	}
	return &v
}

func ParseIntRefValue(jsonstr string) *IntRefValue {
	var intRefValue IntRefValue
	err := json.Unmarshal([]byte(jsonstr), &intRefValue)
	if err != nil {
		return nil
	}
	return &intRefValue
}
