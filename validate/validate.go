// Package validate provides text quality validation for Azerbaijani text.
//
// The validator checks four categories of issues:
//
//   - Spelling: misspelled words detected via [spell.IsCorrect] with
//     suggestions from [spell.Suggest]. Title-case unknown words are
//     skipped as likely proper nouns.
//   - Punctuation: spacing errors (space before comma, missing space
//     after period, double spaces) and repeated punctuation.
//   - Layout: visually confusable homoglyph characters from the wrong
//     script (e.g. Cyrillic 'а' U+0430 in a Latin-dominant text).
//   - Mixed script: tokens entirely in a different script from the
//     document's dominant script.
//
// Two API layers are provided:
//
//   - Structured: [Validate] returns a [Report] with a quality score
//     (0–100) and a positioned issue list sorted by byte offset.
//   - Convenience: [IsValid] returns true when no error-severity issues
//     exist.
//
// The quality score starts at 100 and deducts points per issue:
// error −10, warning −3, info −1, with a floor of 0. Score deductions
// are absolute, not normalized by text length.
//
// All functions are safe for concurrent use by multiple goroutines.
//
// Known limitations:
//
//   - Spelling checks require Latin-script Azerbaijani text. Cyrillic
//     Azerbaijani input skips the spelling check (layout and mixed-script
//     checks still run).
//   - Compound word splitting is not supported (inherited from spell).
//   - Grammar checking (word order, agreement) is not performed.
//   - Arabic script is not supported.
//   - Title-case heuristic may skip genuine misspellings that happen to
//     be capitalized.
//   - Only homoglyph detection is performed, not full keyboard layout
//     mapping.
//   - Bracket/quote matching is not supported (v2).
package validate

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/az-ai-labs/az-lang-nlp/detect"
	"github.com/az-ai-labs/az-lang-nlp/tokenizer"
)

// IssueType classifies a validation issue.
type IssueType int

const (
	Spelling    IssueType = iota // misspelled word
	Punctuation                  // punctuation error
	Layout                       // wrong keyboard layout (homoglyph)
	MixedScript                  // mixed script usage
)

// issueTypeNames maps IssueType values to their string names.
var issueTypeNames = [...]string{
	Spelling:    "spelling",
	Punctuation: "punctuation",
	Layout:      "layout",
	MixedScript: "mixed_script",
}

// issueTypeFromName maps string names back to IssueType values.
var issueTypeFromName = map[string]IssueType{
	"spelling":     Spelling,
	"punctuation":  Punctuation,
	"layout":       Layout,
	"mixed_script": MixedScript,
}

// String returns the name of the issue type.
func (t IssueType) String() string {
	if int(t) >= 0 && int(t) < len(issueTypeNames) {
		return issueTypeNames[t]
	}
	return fmt.Sprintf("IssueType(%d)", int(t))
}

// MarshalJSON encodes the issue type as a JSON string (e.g. "spelling").
func (t IssueType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

// UnmarshalJSON decodes a JSON string (e.g. "spelling") into an IssueType.
func (t *IssueType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	it, ok := issueTypeFromName[s]
	if !ok {
		return fmt.Errorf("validate: unknown issue type: %q", s)
	}
	*t = it
	return nil
}

// Severity indicates the severity of a validation issue.
// Higher numeric values mean higher severity.
type Severity int

const (
	Info    Severity = iota // informational
	Warning                 // should fix
	Error                   // must fix
)

// severityNames maps Severity values to their string names.
var severityNames = [...]string{
	Info:    "info",
	Warning: "warning",
	Error:   "error",
}

// severityFromName maps string names back to Severity values.
var severityFromName = map[string]Severity{
	"info":    Info,
	"warning": Warning,
	"error":   Error,
}

// String returns the name of the severity.
func (s Severity) String() string {
	if int(s) >= 0 && int(s) < len(severityNames) {
		return severityNames[s]
	}
	return fmt.Sprintf("Severity(%d)", int(s))
}

// MarshalJSON encodes the severity as a JSON string (e.g. "error").
func (s Severity) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON decodes a JSON string (e.g. "error") into a Severity.
func (s *Severity) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	sv, ok := severityFromName[str]
	if !ok {
		return fmt.Errorf("validate: unknown severity: %q", str)
	}
	*s = sv
	return nil
}

// Issue represents a single validation finding with position information.
type Issue struct {
	Text       string    `json:"text"`
	Start      int       `json:"start"` // byte offset, inclusive
	End        int       `json:"end"`   // byte offset, exclusive
	Type       IssueType `json:"type"`
	Severity   Severity  `json:"severity"`
	Message    string    `json:"message"`
	Suggestion string    `json:"suggestion"` // empty if no fix available
}

// Report contains the validation result: a quality score and issue list.
type Report struct {
	Score  int     `json:"score"`  // 0-100, higher is better
	Issues []Issue `json:"issues"` // sorted by byte offset, then severity desc
}

const (
	maxInputBytes       = 1 << 20 // 1 MiB limit (same as spell, normalize, keywords)
	maxIssues           = 1000    // cap total issues to prevent memory exhaustion
	deductError         = 10      // score penalty per error-severity issue
	deductWarning       = 3       // score penalty per warning-severity issue
	deductInfo          = 1       // score penalty per info-severity issue
	maxScore            = 100     // starting score (no issues)
	minDetectConfidence = 0.5     // minimum detect confidence for layout/mixed-script checks
)

// Validate checks text for quality issues.
// Returns a Report with a quality score (0-100) and positioned issues.
// All checks run: spelling, punctuation, layout (homoglyphs), mixed script.
// Empty or oversized (>1 MiB) input returns Report{Score: 100, Issues: nil}.
// Safe for concurrent use.
func Validate(text string) Report {
	if text == "" || len(text) > maxInputBytes {
		return Report{Score: maxScore}
	}

	tokens := tokenizer.WordTokens(text)
	if len(tokens) == 0 {
		return Report{Score: maxScore}
	}

	detection := detect.Detect(text)

	var issues []Issue
	issues = appendSpellingIssues(issues, tokens, detection)
	issues = appendPunctuationIssues(issues, tokens)
	issues = appendLayoutIssues(issues, tokens, detection)
	issues = appendMixedScriptIssues(issues, tokens, detection)

	// Cap total issues.
	if len(issues) > maxIssues {
		issues = issues[:maxIssues]
	}

	// Sort by byte offset ascending, then severity descending (Error first).
	slices.SortFunc(issues, func(a, b Issue) int {
		if a.Start != b.Start {
			if a.Start < b.Start {
				return -1
			}
			return 1
		}
		// Higher severity first (Error=2 > Warning=1 > Info=0).
		if a.Severity != b.Severity {
			if a.Severity > b.Severity {
				return -1
			}
			return 1
		}
		return 0
	})

	return Report{
		Score:  calculateScore(issues),
		Issues: issues,
	}
}

// IsValid reports whether text has no error-severity issues.
// Returns true for empty or oversized input (no issues found).
// More efficient than checking Validate().Score: stops at the first
// error-severity issue without sorting or scoring.
// Safe for concurrent use.
func IsValid(text string) bool {
	if text == "" || len(text) > maxInputBytes {
		return true
	}

	tokens := tokenizer.WordTokens(text)
	if len(tokens) == 0 {
		return true
	}

	detection := detect.Detect(text)

	// Run each check and return false as soon as any error is found.
	for _, check := range []func([]Issue, []tokenizer.Token, detect.Result) []Issue{
		appendSpellingIssues,
		appendLayoutIssues,
	} {
		for _, issue := range check(nil, tokens, detection) {
			if issue.Severity == Error {
				return false
			}
		}
	}

	return true
}

// calculateScore computes the quality score from the issue list.
// Starts at 100, deducts per issue by severity, floors at 0.
func calculateScore(issues []Issue) int {
	score := maxScore
	for _, issue := range issues {
		switch issue.Severity {
		case Error:
			score -= deductError
		case Warning:
			score -= deductWarning
		case Info:
			score -= deductInfo
		}
	}
	if score < 0 {
		score = 0
	}
	return score
}
