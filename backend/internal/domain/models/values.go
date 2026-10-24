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

func (v *Value) WithBool(value bool) *Value {
	v.Value = value
	return v
}

func (v *Value) WithInt(value int64) *Value {
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

const (
	fvHash        = "hash"
	fvFileName    = "fileName"
	fvContentType = "contentType"
	fvSize        = "size"
)

type FileValue struct {
	Reference  string     `json:"reference"`
	Properties Properties `json:"properties"`
}

func NewFileValue(id string) *FileValue {
	return &FileValue{
		Reference:  fmt.Sprintf("blob:%s", id),
		Properties: NewProperties(),
	}
}

// WithHash sets the hash of the FileValue
func (f *FileValue) WithHash(hash string) *FileValue {
	v := NewValue(ValueTypeText).WithString(hash)
	f.AddProperty(fvHash, *v)
	return f
}

// Hash returns the hash of the FileValue
func (f *FileValue) Hash() string {
	v := f.GetProperty(fvHash)
	if v == nil {
		return ""
	}
	return v.Value.(string)
}

// WithFileName sets the file name of the FileValue
func (f *FileValue) WithFileName(fileName string) *FileValue {
	v := NewValue(ValueTypeText).WithString(fileName)
	f.AddProperty(fvFileName, *v)
	return f
}

// FileName returns the file name of the FileValue
func (f *FileValue) FileName() string {
	v := f.GetProperty(fvFileName)
	if v == nil {
		return ""
	}
	return v.Value.(string)
}

// WithContentType sets the content type of the FileValue
func (f *FileValue) WithContentType(contentType string) *FileValue {
	v := NewValue(ValueTypeText).WithString(contentType)
	f.AddProperty(fvContentType, *v)
	return f
}

// ContentType returns the content type of the FileValue
func (f *FileValue) ContentType() string {
	v := f.GetProperty(fvContentType)
	if v == nil {
		return ""
	}
	return v.Value.(string)
}

// WithSize sets the size of the FileValue
func (f *FileValue) WithSize(size int64) *FileValue {
	v := NewValue(ValueTypeInt).WithInt(int64(size))
	f.AddProperty(fvSize, *v)
	return f
}

// Size returns the size of the FileValue
func (f *FileValue) Size() int64 {
	v := f.GetProperty(fvSize)
	if v == nil {
		return 0
	}
	return asInt64(v.Value)
}

// AddProperty adds a property to the FileValue
func (f *FileValue) AddProperty(key string, value Value) *FileValue {
	f.Properties[key] = value
	return f
}

// GetProperty gets a property from the FileValue
func (f *FileValue) GetProperty(key string) *Value {
	v, ok := f.Properties[key]
	if !ok {
		return nil
	}
	return &v
}

// ParseFileValue parses a JSON string into a FileValue
func ParseFileValue(jsonstr string) *FileValue {
	var fileValue FileValue
	err := json.Unmarshal([]byte(jsonstr), &fileValue)
	if err != nil {
		return nil
	}
	return &fileValue
}

const (
	irvFilter = "filter"
	irvLabel  = "label"
)

// IntRefValue represents a value that references an internal object
type IntRefValue struct {
	Reference  string     `json:"reference"`
	Properties Properties `json:"properties"`
}

// NewIntRefValue creates a new IntRefValue with the given reference
func NewIntRefValue(ref string) *IntRefValue {
	return &IntRefValue{
		Reference:  ref,
		Properties: NewProperties(),
	}
}

// WithFilter sets the filter for the IntRefValue
func (i *IntRefValue) WithFilter(filter string) *IntRefValue {
	v := NewValue(ValueTypeText).WithString(filter)
	i.AddProperty(irvFilter, *v)
	return i
}

func (i *IntRefValue) Filter() string {
	v := i.GetProperty(irvFilter)
	if v == nil {
		return ""
	}
	s, _ := v.Value.(string)
	return s
}

// WithLabel sets the label for the IntRefValue
func (i *IntRefValue) WithLabel(label LabelText) *IntRefValue {
	v := NewValue(ValueTypeText).WithString(label.String())
	i.AddProperty(irvLabel, *v)
	return i
}

// Label returns the label of the IntRefValue
func (i *IntRefValue) Label() LabelText {
	v := i.GetProperty(irvLabel)
	if v == nil || v.Value == nil {
		return LabelText{}
	}
	switch t := v.Value.(type) {
	case LabelText:
		return t
	case *LabelText:
		if t == nil {
			return LabelText{}
		}
		return *t
	case string:
		var label LabelText
		if err := json.Unmarshal([]byte(t), &label); err != nil {
			return LabelText{}
		}
		return label
	default:
		return LabelText{}
	}
}

// AddProperty adds a property to the IntRefValue
func (i *IntRefValue) AddProperty(key string, value Value) *IntRefValue {
	i.Properties[key] = value
	return i
}

// GetProperty gets a property from the IntRefValue
func (i *IntRefValue) GetProperty(key string) *Value {
	v, ok := i.Properties[key]
	if !ok {
		return nil
	}
	return &v
}

// ParseIntRefValue parses a JSON string into an IntRefValue
func ParseIntRefValue(jsonstr string) *IntRefValue {
	var intRefValue IntRefValue
	err := json.Unmarshal([]byte(jsonstr), &intRefValue)
	if err != nil {
		return nil
	}
	return &intRefValue
}

func asInt64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case int32:
		return int64(n)
	case float64:
		return int64(n)
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return 0
		}
		return i
	default:
		return 0
	}
}
