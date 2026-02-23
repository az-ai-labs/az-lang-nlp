package validate

import (
	"testing"
	"unicode/utf8"
)

func FuzzValidate(f *testing.F) {
	f.Add("kitab")
	f.Add("Bu kitab gözəldir.")
	f.Add("Bu ketab gözəldir.")
	f.Add("")
	f.Add("a")
	f.Add("123 456")
	f.Add("kitab  gözəl")
	f.Add("kitab ,gözəl")
	f.Add("kitab...")
	f.Add("kitab!!")
	f.Add("\xff\xfe")
	f.Add("\x00")
	f.Add("sosial-iqtisadi")
	f.Add("Bakı'nın")

	f.Fuzz(func(t *testing.T, text string) {
		a := Validate(text)
		b := Validate(text)

		// Determinism: two calls must produce identical results.
		if a.Score != b.Score {
			t.Errorf("non-deterministic score: %d vs %d", a.Score, b.Score)
		}
		if len(a.Issues) != len(b.Issues) {
			t.Errorf("non-deterministic issue count: %d vs %d", len(a.Issues), len(b.Issues))
		}

		// Invariant: score in [0, 100].
		if a.Score < 0 || a.Score > maxScore {
			t.Errorf("score %d out of [0, %d] range", a.Score, maxScore)
		}

		// Invariant: issue count capped.
		if len(a.Issues) > maxIssues {
			t.Errorf("issue count %d exceeds cap %d", len(a.Issues), maxIssues)
		}

		// Invariant: byte offsets are valid when input is valid UTF-8.
		if utf8.ValidString(text) {
			for i, issue := range a.Issues {
				if issue.Start < 0 || issue.End > len(text) || issue.Start > issue.End {
					t.Errorf("issue[%d]: invalid offsets Start=%d End=%d (len=%d)",
						i, issue.Start, issue.End, len(text))
					continue
				}
				if text[issue.Start:issue.End] != issue.Text {
					t.Errorf("issue[%d]: text[%d:%d] = %q, issue.Text = %q",
						i, issue.Start, issue.End, text[issue.Start:issue.End], issue.Text)
				}
			}
		}
	})
}
