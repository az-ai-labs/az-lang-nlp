// Package ner extracts named entities from Azerbaijani text using rule-based
// pattern matching.
//
// The package recognizes seven entity types: FIN (personal ID), VOEN (tax ID),
// Phone, Email, IBAN, LicensePlate, and URL. Each entity is returned with byte
// offsets satisfying the invariant s[e.Start:e.End] == e.Text.
//
// Two API layers are provided:
//
//   - Structured: Recognize returns []Entity with offsets, type, and labeling info.
//   - Convenience: Phones, Emails, etc. return []string for common use cases.
//
// FIN and VOEN patterns are ambiguous in isolation. When preceded by a keyword
// (e.g. "FIN:" or "VOEN:"), the Entity.Labeled field is set to true, indicating
// higher confidence. Standalone matches have Labeled=false.
//
// All functions are safe for concurrent use by multiple goroutines.
package ner

import (
	"encoding/json"
	"fmt"
)

// EntityType classifies a recognized entity.
type EntityType int

const (
	FIN          EntityType = iota // Personal identification number (7 alphanumeric chars, no I/O)
	VOEN                           // Tax identification number (10 digits)
	Phone                          // Phone number (+994... or 0XX...)
	Email                          // Email address
	IBAN                           // International bank account number (AZ prefix, 28 chars)
	LicensePlate                   // Azerbaijani vehicle license plate (XX-YY-ZZZ)
	URL                            // HTTP or HTTPS URL
)

// entityTypeNames maps EntityType values to their string names.
var entityTypeNames = [...]string{
	FIN:          "FIN",
	VOEN:         "VOEN",
	Phone:        "Phone",
	Email:        "Email",
	IBAN:         "IBAN",
	LicensePlate: "LicensePlate",
	URL:          "URL",
}

// entityTypeFromName maps string names back to EntityType values.
var entityTypeFromName = map[string]EntityType{
	"FIN":          FIN,
	"VOEN":         VOEN,
	"Phone":        Phone,
	"Email":        Email,
	"IBAN":         IBAN,
	"LicensePlate": LicensePlate,
	"URL":          URL,
}

// String returns the name of the entity type.
func (t EntityType) String() string {
	if int(t) >= 0 && int(t) < len(entityTypeNames) {
		return entityTypeNames[t]
	}
	return fmt.Sprintf("EntityType(%d)", int(t))
}

// MarshalJSON encodes the entity type as a JSON string (e.g. "Phone").
func (t EntityType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

// UnmarshalJSON decodes a JSON string (e.g. "Phone") into an EntityType.
func (t *EntityType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	et, ok := entityTypeFromName[s]
	if !ok {
		const maxErrLen = 50
		if len(s) > maxErrLen {
			s = s[:maxErrLen] + "..."
		}
		return fmt.Errorf("unknown entity type: %q", s)
	}
	*t = et
	return nil
}

// Entity represents a recognized named entity with its position in the source text.
type Entity struct {
	Text    string     `json:"text"`    // The matched text
	Start   int        `json:"start"`   // Byte offset in the original string (inclusive)
	End     int        `json:"end"`     // Byte offset in the original string (exclusive)
	Type    EntityType `json:"type"`    // Classification of the entity
	Labeled bool       `json:"labeled"` // True if preceded by a keyword (e.g. "FIN:", "VOEN:")
}

// String returns a debug representation, e.g. Phone("0501234567")[5:15].
func (e Entity) String() string {
	label := ""
	if e.Labeled {
		label = ",labeled"
	}
	return fmt.Sprintf("%s(%q)[%d:%d%s]", e.Type, e.Text, e.Start, e.End, label)
}

// maxInputBytes is the maximum input length Recognize will process.
// Inputs exceeding this are returned with no results.
const maxInputBytes = 1 << 20 // 1 MiB

// Recognize extracts all named entities from the input string.
// Returns entities sorted by Start offset. When entities overlap,
// the longer (more specific) match wins; if equal length, labeled wins.
func Recognize(s string) []Entity {
	if s == "" || len(s) > maxInputBytes {
		return nil
	}
	return recognize(s)
}

// Phones returns all phone number texts found in s.
func Phones(s string) []string {
	return filterTexts(Recognize(s), Phone)
}

// Emails returns all email address texts found in s.
func Emails(s string) []string {
	return filterTexts(Recognize(s), Email)
}

// FINs returns all FIN (personal ID) texts found in s.
func FINs(s string) []string {
	return filterTexts(Recognize(s), FIN)
}

// VOENs returns all VOEN (tax ID) texts found in s.
func VOENs(s string) []string {
	return filterTexts(Recognize(s), VOEN)
}

// IBANs returns all IBAN texts found in s.
func IBANs(s string) []string {
	return filterTexts(Recognize(s), IBAN)
}

// LicensePlates returns all license plate texts found in s.
func LicensePlates(s string) []string {
	return filterTexts(Recognize(s), LicensePlate)
}

// URLs returns all URL texts found in s.
func URLs(s string) []string {
	return filterTexts(Recognize(s), URL)
}

// filterTexts returns the Text field of entities matching the given type.
func filterTexts(entities []Entity, typ EntityType) []string {
	var out []string
	for _, e := range entities {
		if e.Type == typ {
			out = append(out, e.Text)
		}
	}
	return out
}
