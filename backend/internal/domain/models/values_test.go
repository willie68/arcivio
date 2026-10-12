package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToValueType(t *testing.T) {
	assert.Equal(t, ValueType(ValueTypeBool), ToValueType(ValueTypeBool))
	assert.Equal(t, ValueType(ValueTypeInt), ToValueType(ValueTypeInt))
	assert.Equal(t, ValueType(ValueTypeDecimal), ToValueType(ValueTypeDecimal))
	assert.Equal(t, ValueType(ValueTypeDouble), ToValueType(ValueTypeDouble))
	assert.Equal(t, ValueType(ValueTypeText), ToValueType(ValueTypeText))
	assert.Equal(t, ValueType(ValueTypeMultiline), ToValueType(ValueTypeMultiline))
	assert.Equal(t, ValueType(ValueTypeDateTime), ToValueType(ValueTypeDateTime))
	assert.Equal(t, ValueType(ValueTypeFile), ToValueType(ValueTypeFile))
	assert.Equal(t, ValueType(ValueTypeIntRef), ToValueType(ValueTypeIntRef))
	assert.Equal(t, ValueType(""), ToValueType("unknown"))
	assert.Equal(t, ValueType(""), ToValueType(""))
	assert.Equal(t, ValueType(""), ToValueType(TypeIntRef(ValueTypeIntRefUser)))
}

func TestTypeIntRef(t *testing.T) {
	assert.Equal(t, "intref:user", TypeIntRef(ValueTypeIntRefUser))
	assert.Equal(t, "intref:group", TypeIntRef(ValueTypeIntRefGroup))
	assert.Equal(t, "", TypeIntRef("role"))
	assert.Equal(t, "", TypeIntRef(""))
}

func TestNewValues(t *testing.T) {
	values := NewValues()
	require.NotNil(t, values)
	assert.Len(t, values, 0)

	values = append(values, NewValue(ValueTypeText).WithString("a"))
	values = append(values, NewValue(ValueTypeInt).WithInt(7))
	require.Len(t, values, 2)
	assert.Equal(t, "a", values[0].Value)
	assert.Equal(t, int64(7), values[1].Value)

	other := NewValues()
	assert.Len(t, other, 0)

	raw, err := json.Marshal(values)
	require.NoError(t, err)
	var decoded Values
	require.NoError(t, json.Unmarshal(raw, &decoded))
	require.Len(t, decoded, 2)
	assert.Equal(t, ValueTypeText, decoded[0].Type)
	assert.Equal(t, "a", decoded[0].Value)
	assert.Equal(t, ValueTypeInt, decoded[1].Type)
	assert.Equal(t, float64(7), decoded[1].Value)
}

func TestValueBuilders(t *testing.T) {
	text := NewValue(ValueTypeText)
	assert.Equal(t, ValueTypeText, text.Type)
	assert.Nil(t, text.Value)
	assert.Same(t, text, text.WithString("hello"))
	assert.Equal(t, "hello", text.Value)

	num := NewValue(ValueTypeDouble).WithNumber(1.5)
	assert.Equal(t, 1.5, num.Value)

	flag := NewValue(ValueTypeBool).WithBool(true)
	assert.Equal(t, true, flag.Value)

	count := NewValue(ValueTypeInt).WithInt(42)
	assert.Equal(t, int64(42), count.Value)

	file := NewValue(ValueTypeFile).WithFile("abc")
	fv, ok := file.Value.(*FileValue)
	require.True(t, ok)
	assert.Equal(t, "blob:abc", fv.Reference)

	ref := NewValue(ValueTypeIntRef).WithIntRef("user:u1")
	iv, ok := ref.Value.(*IntRefValue)
	require.True(t, ok)
	assert.Equal(t, "user:u1", iv.Reference)
}

func TestValueJSONOmitsEmpty(t *testing.T) {
	raw, err := json.Marshal(NewValue(ValueTypeText))
	require.NoError(t, err)
	assert.JSONEq(t, `{"type":"text"}`, string(raw))

	raw, err = json.Marshal(NewValue(ValueTypeBool).WithBool(false))
	require.NoError(t, err)
	assert.JSONEq(t, `{"type":"bool","value":false}`, string(raw))
}

func TestFileValueFluentAndGetters(t *testing.T) {
	fv := NewFileValue("blob-1")
	assert.Equal(t, "blob:blob-1", fv.Reference)
	require.NotNil(t, fv.Properties)
	assert.Empty(t, fv.Hash())
	assert.Empty(t, fv.FileName())
	assert.Empty(t, fv.ContentType())
	assert.Equal(t, int64(0), fv.Size())
	assert.Nil(t, fv.GetProperty("missing"))

	same := fv.WithHash("deadbeef").WithFileName("scan.pdf").WithContentType("application/pdf").WithSize(1024)
	assert.Same(t, fv, same)
	assert.Equal(t, "deadbeef", fv.Hash())
	assert.Equal(t, "scan.pdf", fv.FileName())
	assert.Equal(t, "application/pdf", fv.ContentType())
	assert.Equal(t, int64(1024), fv.Size())

	fv.WithHash("cafe").WithSize(0)
	assert.Equal(t, "cafe", fv.Hash())
	assert.Equal(t, int64(0), fv.Size())

	custom := NewValue(ValueTypeText).WithString("de")
	assert.Same(t, fv, fv.AddProperty("lang", *custom))
	got := fv.GetProperty("lang")
	require.NotNil(t, got)
	assert.Equal(t, "de", got.Value)
}

func TestFileValueParseJSON(t *testing.T) {
	src := NewFileValue("abc").
		WithHash("deadbeef").
		WithFileName("scan.pdf").
		WithContentType("application/pdf").
		WithSize(42).
		AddProperty("origin", *NewValue(ValueTypeText).WithString("scan"))
	raw, err := json.Marshal(src)
	require.NoError(t, err)

	parsed := ParseFileValue(string(raw))
	require.NotNil(t, parsed)
	assert.Equal(t, "blob:abc", parsed.Reference)
	assert.Equal(t, "deadbeef", parsed.Hash())
	assert.Equal(t, "scan.pdf", parsed.FileName())
	assert.Equal(t, "application/pdf", parsed.ContentType())
	assert.Equal(t, int64(42), parsed.Size())
	origin := parsed.GetProperty("origin")
	require.NotNil(t, origin)
	assert.Equal(t, "scan", origin.Value)

	fromLiteral := ParseFileValue(`{
		"reference":"blob:xyz",
		"properties":{
			"hash":{"type":"text","value":"aa"},
			"fileName":{"type":"text","value":"a.bin"},
			"contentType":{"type":"text","value":"application/octet-stream"},
			"size":{"type":"int","value":7}
		}
	}`)
	require.NotNil(t, fromLiteral)
	assert.Equal(t, "blob:xyz", fromLiteral.Reference)
	assert.Equal(t, "aa", fromLiteral.Hash())
	assert.Equal(t, "a.bin", fromLiteral.FileName())
	assert.Equal(t, "application/octet-stream", fromLiteral.ContentType())
	assert.Equal(t, int64(7), fromLiteral.Size())
}

func TestFileValueParseRejectsInvalidAndLegacyTopLevel(t *testing.T) {
	assert.Nil(t, ParseFileValue(""))
	assert.Nil(t, ParseFileValue("{"))
	assert.Nil(t, ParseFileValue("[]"))

	legacy := ParseFileValue(`{"reference":"blob:x","hash":"deadbeef","fileName":"a.pdf","contentLength":9}`)
	require.NotNil(t, legacy)
	assert.Equal(t, "blob:x", legacy.Reference)
	assert.Empty(t, legacy.Hash())
	assert.Empty(t, legacy.FileName())
	assert.Equal(t, int64(0), legacy.Size())
}

func TestIntRefValueFluentAndGetters(t *testing.T) {
	iv := NewIntRefValue("user:u1")
	assert.Equal(t, "user:u1", iv.Reference)
	assert.Empty(t, iv.Filter())
	assert.Equal(t, LabelText{}, iv.Label())
	require.NotNil(t, iv.Properties)
	assert.Nil(t, iv.GetProperty("missing"))

	label := LabelText{De: "Nutzer", En: "User"}
	same := iv.WithFilter("active").WithLabel(label)
	assert.Same(t, iv, same)
	assert.Equal(t, "active", iv.Filter())
	assert.Equal(t, label, iv.Label())

	prop := NewValue(ValueTypeText).WithString("admin")
	assert.Same(t, iv, iv.AddProperty("role", *prop))
	got := iv.GetProperty("role")
	require.NotNil(t, got)
	assert.Equal(t, "admin", got.Value)
}

func TestIntRefValueParseJSON(t *testing.T) {
	src := NewIntRefValue("group:g1").
		WithFilter("enabled").
		WithLabel(LabelText{De: "Gruppe", En: "Group"}).
		AddProperty("kind", *NewValue(ValueTypeText).WithString("acl"))
	raw, err := json.Marshal(src)
	require.NoError(t, err)

	parsed := ParseIntRefValue(string(raw))
	require.NotNil(t, parsed)
	assert.Equal(t, "group:g1", parsed.Reference)
	assert.Equal(t, "enabled", parsed.Filter())
	assert.Equal(t, "Gruppe", parsed.Label().De)
	assert.Equal(t, "Group", parsed.Label().En)
	kind := parsed.GetProperty("kind")
	require.NotNil(t, kind)
	assert.Equal(t, "acl", kind.Value)

	assert.Nil(t, ParseIntRefValue(""))
	assert.Nil(t, ParseIntRefValue("{"))
	assert.Nil(t, ParseIntRefValue("true"))
}
