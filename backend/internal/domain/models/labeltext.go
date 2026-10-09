package models

import "strings"

// LabelText is a UI string in German and English.
type LabelText struct {
	De string `json:"de"`
	En string `json:"en"`
}

func TrimLabelText(t LabelText) LabelText {
	return LabelText{De: strings.TrimSpace(t.De), En: strings.TrimSpace(t.En)}
}
