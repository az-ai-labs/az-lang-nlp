package numtext

import "testing"

// FuzzConvert verifies that Convert never panics for any int64 input.
func FuzzConvert(f *testing.F) {
	f.Add(int64(0))
	f.Add(int64(1))
	f.Add(int64(-1))
	f.Add(int64(100))
	f.Add(int64(1000))
	f.Add(int64(1_000_000))
	f.Add(int64(1_000_000_000_000_000_000))
	f.Add(int64(-1_000_000_000_000_000_000))
	f.Add(int64(9223372036854775807))  // math.MaxInt64
	f.Add(int64(-9223372036854775808)) // math.MinInt64

	f.Fuzz(func(t *testing.T, n int64) {
		// Must not panic.
		_ = Convert(n)
		_ = ConvertOrdinal(n)
	})
}

// FuzzParse verifies that Parse never panics for any string input.
func FuzzParse(f *testing.F) {
	f.Add("")
	f.Add("sıfır")
	f.Add("bir")
	f.Add("yüz iyirmi üç")
	f.Add("mənfi beş")
	f.Add("hello world")
	f.Add("\xff\xfe")
	f.Add("bir milyon iki yüz min")
	f.Add(string([]byte{0x00}))

	f.Fuzz(func(t *testing.T, s string) {
		// Must not panic.
		_, _ = Parse(s)
	})
}

// FuzzRoundTrip verifies that Parse(Convert(n)) == n for all valid n.
func FuzzRoundTrip(f *testing.F) {
	f.Add(int64(0))
	f.Add(int64(1))
	f.Add(int64(-1))
	f.Add(int64(42))
	f.Add(int64(123))
	f.Add(int64(1000))
	f.Add(int64(2300095))
	f.Add(int64(1_000_000_000_000_000_000))
	f.Add(int64(-999_999_999_999_999_999))

	f.Fuzz(func(t *testing.T, n int64) {
		text := Convert(n)
		if text == "" {
			return // out of range, skip
		}
		got, err := Parse(text)
		if err != nil {
			t.Errorf("Parse(Convert(%d)) = %q, error: %v", n, text, err)
		}
		if got != n {
			t.Errorf("Parse(Convert(%d)) = %d, want %d (text: %q)", n, got, n, text)
		}
	})
}

// FuzzConvertFloat verifies that ConvertFloat never panics for any string input.
func FuzzConvertFloat(f *testing.F) {
	f.Add("")
	f.Add("3.14")
	f.Add("0.5")
	f.Add("-2.5")
	f.Add("abc")
	f.Add("3,14")
	f.Add(".5")
	f.Add("3.")
	f.Add("3.14.15")
	f.Add("\xff\xfe")
	f.Add("999999999999999999.999999999999999999")

	f.Fuzz(func(t *testing.T, s string) {
		// Must not panic in either mode.
		_ = ConvertFloat(s, MathMode)
		_ = ConvertFloat(s, DigitMode)
	})
}
