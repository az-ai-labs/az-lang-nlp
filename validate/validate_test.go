package validate

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// TestValidate
// ---------------------------------------------------------------------------

func TestValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantScore int
		wantCount int // expected number of issues (-1 = don't check)
	}{
		{
			name:      "empty string",
			input:     "",
			wantScore: 100,
			wantCount: 0,
		},
		{
			name:      "oversized input",
			input:     strings.Repeat("a", maxInputBytes+1),
			wantScore: 100,
			wantCount: 0,
		},
		{
			name:      "clean Azerbaijani text",
			input:     "Bu kitab çox gözəldir.",
			wantScore: 100,
			wantCount: 0,
		},
		{
			name:      "only numbers",
			input:     "123 456 789",
			wantScore: 100,
			wantCount: 0,
		},
		{
			name:      "single punctuation character",
			input:     ".",
			wantScore: 100,
			wantCount: 0,
		},
		{
			name:      "single letter",
			input:     "a",
			wantScore: 100,
			wantCount: 0,
		},
		{
			name:      "ellipsis not flagged",
			input:     "Bu kitab...",
			wantScore: 100,
			wantCount: 0,
		},
		{
			name:      "digit-containing token not spell-checked",
			input:     "3-cü sinif",
			wantScore: 100,
			wantCount: 0,
		},
		{
			name:      "title-case unknown word skipped",
			input:     "Qərbi gözəldir.",
			wantScore: 100,
			wantCount: 0,
		},
		{
			name:      "known word with diacritics",
			input:     "dəniz sahili",
			wantScore: 100,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Validate(tt.input)

			if got.Score != tt.wantScore {
				t.Errorf("Validate(%q).Score = %d, want %d", tt.input, got.Score, tt.wantScore)
			}

			if tt.wantCount >= 0 && len(got.Issues) != tt.wantCount {
				t.Errorf("Validate(%q): got %d issues, want %d\n  issues: %v",
					tt.input, len(got.Issues), tt.wantCount, got.Issues)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestValidateSpelling
// ---------------------------------------------------------------------------

func TestValidateSpelling(t *testing.T) {
	t.Parallel()

	report := Validate("Bu ketab gözəldir.")
	if report.Score >= 100 {
		t.Fatalf("expected score < 100 for misspelled text, got %d", report.Score)
	}

	found := false
	for _, issue := range report.Issues {
		if issue.Type == Spelling && issue.Text == "ketab" {
			found = true
			if issue.Severity != Error {
				t.Errorf("spelling issue severity = %v, want Error", issue.Severity)
			}
			if issue.Suggestion == "" {
				t.Error("spelling issue has no suggestion")
			}
			// Verify byte offsets.
			src := "Bu ketab gözəldir."
			if src[issue.Start:issue.End] != issue.Text {
				t.Errorf("offset mismatch: text[%d:%d] = %q, issue.Text = %q",
					issue.Start, issue.End, src[issue.Start:issue.End], issue.Text)
			}
			break
		}
	}
	if !found {
		t.Errorf("expected spelling issue for 'ketab', got issues: %v", report.Issues)
	}
}

// ---------------------------------------------------------------------------
// TestValidatePunctuation
// ---------------------------------------------------------------------------

func TestValidatePunctuation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantMsg string
		wantSev Severity
	}{
		{
			name:    "space before comma",
			input:   "kitab , gözəl",
			wantMsg: "space before punctuation",
			wantSev: Warning,
		},
		{
			name:    "space before period",
			input:   "kitab .",
			wantMsg: "space before punctuation",
			wantSev: Warning,
		},
		{
			name:    "missing space after period",
			input:   "kitab.gözəl",
			wantMsg: "missing space after punctuation",
			wantSev: Warning,
		},
		{
			name:    "double space",
			input:   "kitab  gözəl",
			wantMsg: "multiple spaces",
			wantSev: Warning,
		},
		{
			name:    "repeated punctuation",
			input:   "kitab!!",
			wantMsg: "repeated punctuation",
			wantSev: Info,
		},
		{
			name:    "two dots flagged",
			input:   "kitab..",
			wantMsg: "repeated punctuation",
			wantSev: Info,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			report := Validate(tt.input)

			found := false
			for _, issue := range report.Issues {
				if issue.Type == Punctuation && issue.Message == tt.wantMsg {
					found = true
					if issue.Severity != tt.wantSev {
						t.Errorf("severity = %v, want %v", issue.Severity, tt.wantSev)
					}
					break
				}
			}
			if !found {
				t.Errorf("expected punctuation issue %q, got issues: %v", tt.wantMsg, report.Issues)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestValidateLayout
// ---------------------------------------------------------------------------

func TestValidateLayout(t *testing.T) {
	t.Parallel()

	// "kitаb" with Cyrillic 'а' (U+0430) instead of Latin 'a' (U+0061)
	// in a Latin-dominant text.
	input := "Bu kit\u0430b g\u00f6z\u0259ldir."
	report := Validate(input)

	found := false
	for _, issue := range report.Issues {
		if issue.Type == Layout {
			found = true
			if issue.Severity != Error {
				t.Errorf("layout issue severity = %v, want Error", issue.Severity)
			}
			if issue.Suggestion == "" {
				t.Error("layout issue has no suggestion")
			}
			break
		}
	}
	if !found {
		t.Errorf("expected layout issue for homoglyph, got issues: %v", report.Issues)
	}
}

// ---------------------------------------------------------------------------
// TestValidateMixedScript
// ---------------------------------------------------------------------------

func TestValidateMixedScript(t *testing.T) {
	t.Parallel()

	// "Москва" in a Latin-dominant text contains non-homoglyph Cyrillic chars.
	input := "Bu kitab \u041c\u043e\u0441\u043a\u0432\u0430 g\u00f6z\u0259ldir."
	report := Validate(input)

	found := false
	for _, issue := range report.Issues {
		if issue.Type == MixedScript {
			found = true
			if issue.Severity != Info {
				t.Errorf("mixed script severity = %v, want Info", issue.Severity)
			}
			break
		}
	}
	if !found {
		t.Errorf("expected mixed-script issue, got issues: %v", report.Issues)
	}
}

// TestNoDoubleReporting verifies all-homoglyph tokens are flagged as Layout
// only, not also as MixedScript.
func TestNoDoubleReporting(t *testing.T) {
	t.Parallel()

	// "cор" using Cyrillic 'с' (U+0441) and 'о' (U+043E) and 'р' (U+0440)
	// -- all three are homoglyphs of Latin c, o, p.
	input := "Bu kitab \u0441\u043e\u0440 g\u00f6z\u0259ldir."
	report := Validate(input)

	layoutCount := 0
	mixedCount := 0
	for _, issue := range report.Issues {
		if issue.Type == Layout {
			layoutCount++
		}
		if issue.Type == MixedScript {
			mixedCount++
		}
	}
	if layoutCount == 0 {
		t.Error("expected at least one layout issue for all-homoglyph token")
	}
	if mixedCount > 0 {
		t.Error("all-homoglyph token should not be reported as mixed-script")
	}
}

// ---------------------------------------------------------------------------
// TestValidateScoreFloor
// ---------------------------------------------------------------------------

func TestValidateScoreFloor(t *testing.T) {
	t.Parallel()

	// Generate enough spelling errors to push score below 0.
	words := []string{}
	for i := range 15 {
		words = append(words, fmt.Sprintf("xyzqw%d", i))
	}
	input := strings.Join(words, " ")
	report := Validate(input)

	if report.Score < 0 {
		t.Errorf("score = %d, want >= 0 (floor)", report.Score)
	}
}

// ---------------------------------------------------------------------------
// TestValidateIssueCap
// ---------------------------------------------------------------------------

func TestValidateIssueCap(t *testing.T) {
	t.Parallel()

	report := Validate(strings.Repeat("xyzabc ", maxIssues+100))

	if len(report.Issues) > maxIssues {
		t.Errorf("got %d issues, want <= %d", len(report.Issues), maxIssues)
	}
}

// ---------------------------------------------------------------------------
// TestValidateIssueSorting
// ---------------------------------------------------------------------------

func TestValidateIssueSorting(t *testing.T) {
	t.Parallel()

	report := Validate("Bu ketab ,gözəldir.")
	if len(report.Issues) < 2 {
		t.Skip("not enough issues to test sorting")
	}

	for i := 1; i < len(report.Issues); i++ {
		a, b := report.Issues[i-1], report.Issues[i]
		if a.Start > b.Start {
			t.Errorf("issues not sorted by offset: [%d].Start=%d > [%d].Start=%d",
				i-1, a.Start, i, b.Start)
		}
		if a.Start == b.Start && a.Severity < b.Severity {
			t.Errorf("same-offset issues not sorted by severity desc: [%d].Severity=%v < [%d].Severity=%v",
				i-1, a.Severity, i, b.Severity)
		}
	}
}

// ---------------------------------------------------------------------------
// TestIsValid
// ---------------------------------------------------------------------------

func TestIsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "empty string",
			input: "",
			want:  true,
		},
		{
			name:  "clean text",
			input: "Bu kitab gözəldir.",
			want:  true,
		},
		{
			name:  "spelling error makes invalid",
			input: "Bu ketab gözəldir.",
			want:  false,
		},
		{
			name:  "punctuation warning only is still valid",
			input: "kitab  gözəl",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := IsValid(tt.input)
			if got != tt.want {
				t.Errorf("IsValid(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestIssueTypeJSON
// ---------------------------------------------------------------------------

func TestIssueTypeJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		val  IssueType
		name string
	}{
		{Spelling, "spelling"},
		{Punctuation, "punctuation"},
		{Layout, "layout"},
		{MixedScript, "mixed_script"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tt.val)
			if err != nil {
				t.Fatalf("MarshalJSON: %v", err)
			}
			want := fmt.Sprintf("%q", tt.name)
			if string(data) != want {
				t.Errorf("MarshalJSON(%v) = %s, want %s", tt.val, data, want)
			}

			var got IssueType
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("UnmarshalJSON: %v", err)
			}
			if got != tt.val {
				t.Errorf("round-trip: got %v, want %v", got, tt.val)
			}
		})
	}
}

func TestIssueTypeUnmarshalUnknown(t *testing.T) {
	t.Parallel()
	var it IssueType
	err := json.Unmarshal([]byte(`"bogus"`), &it)
	if err == nil {
		t.Error("expected error for unknown issue type, got nil")
	}
}

// ---------------------------------------------------------------------------
// TestSeverityJSON
// ---------------------------------------------------------------------------

func TestSeverityJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		val  Severity
		name string
	}{
		{Info, "info"},
		{Warning, "warning"},
		{Error, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tt.val)
			if err != nil {
				t.Fatalf("MarshalJSON: %v", err)
			}
			want := fmt.Sprintf("%q", tt.name)
			if string(data) != want {
				t.Errorf("MarshalJSON(%v) = %s, want %s", tt.val, data, want)
			}

			var got Severity
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("UnmarshalJSON: %v", err)
			}
			if got != tt.val {
				t.Errorf("round-trip: got %v, want %v", got, tt.val)
			}
		})
	}
}

func TestSeverityUnmarshalUnknown(t *testing.T) {
	t.Parallel()
	var s Severity
	err := json.Unmarshal([]byte(`"bogus"`), &s)
	if err == nil {
		t.Error("expected error for unknown severity, got nil")
	}
}

// ---------------------------------------------------------------------------
// TestReportJSON
// ---------------------------------------------------------------------------

func TestReportJSON(t *testing.T) {
	t.Parallel()

	report := Report{
		Score: 90,
		Issues: []Issue{
			{
				Text:       "ketab",
				Start:      3,
				End:        8,
				Type:       Spelling,
				Severity:   Error,
				Message:    "unknown word",
				Suggestion: "kitab",
			},
		},
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got Report
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.Score != report.Score {
		t.Errorf("Score = %d, want %d", got.Score, report.Score)
	}
	if len(got.Issues) != len(report.Issues) {
		t.Fatalf("Issues len = %d, want %d", len(got.Issues), len(report.Issues))
	}
	if got.Issues[0].Type != Spelling {
		t.Errorf("Issues[0].Type = %v, want Spelling", got.Issues[0].Type)
	}
	if got.Issues[0].Severity != Error {
		t.Errorf("Issues[0].Severity = %v, want Error", got.Issues[0].Severity)
	}
}

// ---------------------------------------------------------------------------
// TestConcurrentSafety
// ---------------------------------------------------------------------------

func TestConcurrentSafety(t *testing.T) {
	t.Parallel()

	inputs := []string{
		"Bu kitab gözəldir.",
		"Bu ketab gözəldir.",
		"kitab  gözəl",
		"",
		"123 456",
	}

	const numGoroutines = 100
	done := make(chan bool, numGoroutines)

	for i := range numGoroutines {
		go func(id int) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("goroutine %d panicked: %v", id, r)
				}
				done <- true
			}()

			for j := range 100 {
				input := inputs[j%len(inputs)]
				_ = Validate(input)
				_ = IsValid(input)
			}
		}(i)
	}

	for range numGoroutines {
		<-done
	}
}

// ---------------------------------------------------------------------------
// TestMalformedUTF8
// ---------------------------------------------------------------------------

func TestMalformedUTF8(t *testing.T) {
	t.Parallel()

	inputs := []string{
		"\xff\xfe",
		"kitab\xC3",
		"kitab\xC0\x80",
	}

	for _, input := range inputs {
		t.Run("", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panicked on %q: %v", input, r)
				}
			}()
			_ = Validate(input)
		})
	}
}

// TestNullBytes verifies no panic on null byte inputs.
func TestNullBytes(t *testing.T) {
	t.Parallel()

	inputs := []string{
		"\x00",
		"\x00kitab",
		"kitab\x00lar",
	}

	for _, input := range inputs {
		t.Run("", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panicked on %q: %v", input, r)
				}
			}()
			_ = Validate(input)
		})
	}
}

// ---------------------------------------------------------------------------
// TestCalculateScore
// ---------------------------------------------------------------------------

func TestCalculateScore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		issues []Issue
		want   int
	}{
		{
			name:   "no issues",
			issues: nil,
			want:   100,
		},
		{
			name:   "one error",
			issues: []Issue{{Severity: Error}},
			want:   90,
		},
		{
			name:   "one warning",
			issues: []Issue{{Severity: Warning}},
			want:   97,
		},
		{
			name:   "one info",
			issues: []Issue{{Severity: Info}},
			want:   99,
		},
		{
			name: "mixed",
			issues: []Issue{
				{Severity: Error},
				{Severity: Warning},
				{Severity: Info},
			},
			want: 86, // 100 - 10 - 3 - 1
		},
		{
			name: "floors at zero",
			issues: func() []Issue {
				out := make([]Issue, 11)
				for i := range out {
					out[i].Severity = Error
				}
				return out
			}(),
			want: 0, // 100 - 110 = -10 → 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := calculateScore(tt.issues)
			if got != tt.want {
				t.Errorf("calculateScore() = %d, want %d", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkValidate(b *testing.B) {
	text := "Bu kitab çox gözəldir. Azərbaycan dili gözəl dildir."
	b.SetBytes(int64(len(text)))
	for b.Loop() {
		Validate(text)
	}
}

func BenchmarkIsValid(b *testing.B) {
	text := "Bu kitab çox gözəldir."
	for b.Loop() {
		IsValid(text)
	}
}

// ---------------------------------------------------------------------------
// Examples
// ---------------------------------------------------------------------------

func ExampleValidate() {
	report := Validate("Bu kitab gözəldir.")
	fmt.Println(report.Score)
	// Output:
	// 100
}

func ExampleIsValid() {
	fmt.Println(IsValid("Bu kitab gözəldir."))
	fmt.Println(IsValid("Bu ketab gözəldir."))
	// Output:
	// true
	// false
}
