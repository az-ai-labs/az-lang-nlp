package ner

import "testing"

func FuzzRecognize(f *testing.F) {
	f.Add("FIN: 5ARPXK2")
	f.Add("+994501234567")
	f.Add("info@example.com")
	f.Add("AZ21NABZ00000000137010001944")
	f.Add("10-AA-123")
	f.Add("https://gov.az")
	f.Add("VOEN: 1234567890")
	f.Add("")
	f.Add("\xff\xfe")
	f.Add("FIN FIN FIN FIN FIN")
	f.Add("+994 50 123 45 67 və 050 123 45 67")

	f.Fuzz(func(t *testing.T, s string) {
		entities := Recognize(s)
		for _, e := range entities {
			if e.Start < 0 || e.End > len(s) || e.Start > e.End {
				t.Fatalf("invalid offsets: start=%d end=%d len=%d", e.Start, e.End, len(s))
			}
			if s[e.Start:e.End] != e.Text {
				t.Fatalf("invariant broken: s[%d:%d]=%q != Text=%q",
					e.Start, e.End, s[e.Start:e.End], e.Text)
			}
		}
	})
}
