// Package datetime parses Azerbaijani natural-language date and time
// expressions into structured time values.
//
// The package recognizes three result types: dates, times, and combined
// date-time expressions. It handles natural text ("5 mart 2026"),
// numeric formats ("05.03.2026", "2026-03-05"), and relative expressions
// ("bu gün", "3 gün əvvəl", "keçən həftə").
//
// Two API layers are provided:
//
//   - Extract returns []Result with byte offsets for scanning running text.
//   - Parse returns a single Result for isolated date/time expressions.
//
// Relative and partial expressions are resolved against a reference time.
// When ref is the zero value, time.Now().UTC() is used. All returned times
// use the location from the reference time (UTC by default).
//
// All functions are safe for concurrent use by multiple goroutines.
package datetime

import (
	"encoding/json"
	"fmt"
	"time"
)

// Type classifies the kind of date/time expression that was parsed.
type Type int

const (
	TypeDate     Type = iota // Only date components (year, month, day)
	TypeTime                 // Only time components (hour, minute, second)
	TypeDateTime             // Both date and time components
	TypeDuration             // A time duration (e.g. "2 saat 30 dəqiqə")
)

// typeNames maps Type values to their string names.
var typeNames = [...]string{
	TypeDate:     "Date",
	TypeTime:     "Time",
	TypeDateTime: "DateTime",
	TypeDuration: "Duration",
}

// typeFromName maps string names back to Type values.
var typeFromName = map[string]Type{
	"Date":     TypeDate,
	"Time":     TypeTime,
	"DateTime": TypeDateTime,
	"Duration": TypeDuration,
}

// String returns the name of the type.
func (t Type) String() string {
	if int(t) >= 0 && int(t) < len(typeNames) {
		return typeNames[t]
	}
	return fmt.Sprintf("Type(%d)", int(t))
}

// MarshalJSON encodes the type as a JSON string (e.g. "Date").
func (t Type) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

// UnmarshalJSON decodes a JSON string (e.g. "Date") into a Type.
func (t *Type) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	tt, ok := typeFromName[s]
	if !ok {
		const maxErrLen = 50
		if len(s) > maxErrLen {
			s = s[:maxErrLen] + "..."
		}
		return fmt.Errorf("datetime: unknown type: %q", s)
	}
	*t = tt
	return nil
}

// Components is a bitmask indicating which date/time fields were explicitly
// present in the input (vs. inferred from the reference time).
type Components uint8

const (
	HasYear Components = 1 << iota
	HasMonth
	HasDay
	HasHour
	HasMinute
	HasSecond
)

// String returns a debug representation of the components bitmask.
func (c Components) String() string {
	var parts []byte
	if c&HasYear != 0 {
		parts = append(parts, 'Y')
	}
	if c&HasMonth != 0 {
		parts = append(parts, 'M')
	}
	if c&HasDay != 0 {
		parts = append(parts, 'D')
	}
	if c&HasHour != 0 {
		parts = append(parts, 'h')
	}
	if c&HasMinute != 0 {
		parts = append(parts, 'm')
	}
	if c&HasSecond != 0 {
		parts = append(parts, 's')
	}
	if len(parts) == 0 {
		return "none"
	}
	return string(parts)
}

// Result represents a parsed date/time expression with its position in the source text.
type Result struct {
	Text     string        `json:"text"`               // The matched substring
	Start    int           `json:"start"`              // Byte offset in the original string (inclusive)
	End      int           `json:"end"`                // Byte offset in the original string (exclusive)
	Type     Type          `json:"type"`               // Classification of the expression
	Time     time.Time     `json:"time"`               // Resolved point in time
	Duration time.Duration `json:"duration,omitempty"` // Populated when Type == TypeDuration
	Explicit Components    `json:"explicit"`           // Which components came from input vs. ref
}

// String returns a debug representation, e.g. Date("5 mart 2026")[3:15].
func (r Result) String() string {
	return fmt.Sprintf("%s(%q)[%d:%d]", r.Type, r.Text, r.Start, r.End)
}

// Extract finds all date/time spans in s, resolved against ref.
// Returns nil for empty or oversized input.
// When ref is the zero value, time.Now() is used.
func Extract(s string, ref time.Time) []Result {
	if s == "" || len(s) > maxInputBytes {
		return nil
	}
	if ref.IsZero() {
		ref = time.Now().UTC()
	}
	return extract(s, ref)
}

// Parse parses a single date/time expression from s.
// Returns a descriptive error for empty, unrecognized, or invalid input.
// When ref is the zero value, time.Now() is used.
func Parse(s string, ref time.Time) (Result, error) {
	if s == "" {
		return Result{}, fmt.Errorf("datetime: empty input")
	}
	if len(s) > maxInputBytes {
		return Result{}, fmt.Errorf("datetime: input exceeds %d bytes", maxInputBytes)
	}
	if ref.IsZero() {
		ref = time.Now().UTC()
	}
	results := extract(s, ref)
	if len(results) == 0 {
		return Result{}, fmt.Errorf("datetime: unrecognized input")
	}
	return results[0], nil
}
