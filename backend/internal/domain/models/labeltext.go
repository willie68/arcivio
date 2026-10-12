package models

import (
	"encoding/json"
	"strings"
)

// LabelText is a UI string in German and English.
type LabelText struct {
	De string `json:"de"`
	En string `json:"en"`
}

func (l *LabelText) String() string {
	json, err := json.Marshal(l)
	if err != nil {
		return ""
	}
	return string(json)
}

func TrimLabelText(t LabelText) LabelText {
	return LabelText{De: strings.TrimSpace(t.De), En: strings.TrimSpace(t.En)}
}
