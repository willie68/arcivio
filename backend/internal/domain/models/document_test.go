package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProperties(t *testing.T) {
	props := NewProperties()
	require.NotNil(t, props)
	assert.Len(t, props, 0)

	props["lang"] = *NewValue(ValueTypeText).WithString("de")
	props["count"] = *NewValue(ValueTypeInt).WithInt(3)
	assert.Equal(t, "de", props["lang"].Value)
	assert.Equal(t, int64(3), props["count"].Value)

	_, ok := props["missing"]
	assert.False(t, ok)

	other := NewProperties()
	assert.Len(t, other, 0)
}

func TestPropertiesJSON(t *testing.T) {
	props := NewProperties()
	props["flag"] = *NewValue(ValueTypeBool).WithBool(true)
	props["note"] = *NewValue(ValueTypeMultiline).WithString("line")

	raw, err := json.Marshal(props)
	require.NoError(t, err)

	var decoded Properties
	require.NoError(t, json.Unmarshal(raw, &decoded))
	require.Len(t, decoded, 2)
	assert.Equal(t, ValueTypeBool, decoded["flag"].Type)
	assert.Equal(t, true, decoded["flag"].Value)
	assert.Equal(t, "line", decoded["note"].Value)
}

func TestNewField(t *testing.T) {
	field := NewField("title")
	assert.Equal(t, "title", field.Name)
	require.NotNil(t, field.Value)
	assert.Len(t, field.Value, 0)
	require.NotNil(t, field.Properties)
	assert.Len(t, field.Properties, 0)

	field.Value = append(field.Value, NewValue(ValueTypeText).WithString("Rechnung"))
	field.Value = append(field.Value, NewValue(ValueTypeText).WithString("Invoice"))
	field.Properties["archival"] = *NewValue(ValueTypeBool).WithBool(true)

	require.Len(t, field.Value, 2)
	assert.Equal(t, "Rechnung", field.Value[0].Value)
	assert.Equal(t, true, field.Properties["archival"].Value)
}

func TestFieldsAdd(t *testing.T) {
	fields := NewFields()
	require.NotNil(t, fields)
	assert.Len(t, fields, 0)

	title := NewField("title")
	amount := NewField("amount")
	same := fields.Add(title)
	assert.Same(t, &fields, same)
	same.Add(amount)

	require.Len(t, fields, 2)
	assert.Equal(t, "title", fields[0].Name)
	assert.Equal(t, "amount", fields[1].Name)

	other := NewFields()
	assert.Len(t, other, 0)
}

func TestFieldAndFieldsJSON(t *testing.T) {
	field := NewField("attachment")
	field.Value = append(field.Value, NewValue(ValueTypeFile).WithFile("f1"))
	field.Properties["mandatory"] = *NewValue(ValueTypeBool).WithBool(true)

	fields := NewFields()
	fields.Add(field).Add(NewField("owner"))

	raw, err := json.Marshal(fields)
	require.NoError(t, err)

	var decoded Fields
	require.NoError(t, json.Unmarshal(raw, &decoded))
	require.Len(t, decoded, 2)
	assert.Equal(t, "attachment", decoded[0].Name)
	require.Len(t, decoded[0].Value, 1)
	assert.Equal(t, ValueTypeFile, decoded[0].Value[0].Type)
	assert.Equal(t, true, decoded[0].Properties["mandatory"].Value)
	assert.Equal(t, "owner", decoded[1].Name)
	assert.Empty(t, decoded[1].Value)
}

func TestNewDocumentUsesFieldsAndProperties(t *testing.T) {
	doc := NewDocument("d1")
	assert.Equal(t, "d1", doc.ID)
	assert.Equal(t, FormatVersion, doc.FormatVersion)
	require.NotNil(t, doc.Fields)
	require.NotNil(t, doc.Properties)
	assert.Len(t, doc.Fields, 0)
	assert.Len(t, doc.Properties, 0)

	doc.Fields.Add(NewField("title"))
	doc.Properties["source"] = *NewValue(ValueTypeText).WithString("inbox")
	require.Len(t, doc.Fields, 1)
	assert.Equal(t, "title", doc.Fields[0].Name)
	assert.Equal(t, "inbox", doc.Properties["source"].Value)
}
