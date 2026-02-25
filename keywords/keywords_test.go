package keywords

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// TestExtractTFIDF
// ---------------------------------------------------------------------------

func TestExtractTFIDF(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		topN     int
		wantNil  bool
		wantTop  string // expected stem in first position (if non-empty)
		wantMaxN int    // max expected result count (0 = don't check)
	}{
		{
			name:    "empty string",
			input:   "",
			topN:    5,
			wantNil: true,
		},
		{
			name:    "oversized input returns nil",
			input:   strings.Repeat("kitab ", maxInputBytes/6+1),
			topN:    5,
			wantNil: true,
		},
		{
			name:    "all stopwords",
			input:   "və bu da o ki",
			topN:    5,
			wantNil: true,
		},
		{
			name:     "single meaningful word",
			input:    "kitab",
			topN:     5,
			wantMaxN: 1,
			wantTop:  "kitab",
		},
		{
			name:    "known input has expected top keyword",
			input:   "Azərbaycan iqtisadiyyatı sürətlə inkişaf edir. Azərbaycan regionun ən böyük iqtisadiyyatına malikdir. Azərbaycan iqtisadiyyatı neft sektoruna əsaslanır.",
			topN:    3,
			wantTop: "iqtisadiyyat",
		},
		{
			name:     "topN zero uses default",
			input:    "kitab kitab kitab",
			topN:     0,
			wantMaxN: 1,
		},
		{
			name:     "topN negative uses default",
			input:    "kitab kitab kitab",
			topN:     -1,
			wantMaxN: 1,
		},
		{
			name:    "ASCII-degraded input normalized",
			input:   "gozel gozel gozel",
			topN:    5,
			wantTop: "gözəl",
		},
		{
			name:     "inflected forms group under stem",
			input:    "kitab kitablar kitabdan kitabların",
			topN:     5,
			wantTop:  "kitab",
			wantMaxN: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ExtractTFIDF(tt.input, tt.topN)

			if tt.wantNil {
				if got != nil {
					t.Errorf("ExtractTFIDF() = %v, want nil", got)
				}
				return
			}

			if len(got) == 0 {
				t.Fatal("ExtractTFIDF() returned nil, want non-nil")
			}

			if tt.wantTop != "" && got[0].Stem != tt.wantTop {
				t.Errorf("ExtractTFIDF()[0].Stem = %q, want %q", got[0].Stem, tt.wantTop)
			}

			if tt.wantMaxN > 0 && len(got) > tt.wantMaxN {
				t.Errorf("ExtractTFIDF() returned %d results, want <= %d", len(got), tt.wantMaxN)
			}

			for i, kw := range got {
				if kw.Score <= 0 {
					t.Errorf("ExtractTFIDF()[%d].Score = %f, want > 0", i, kw.Score)
				}
				if kw.Count <= 0 {
					t.Errorf("ExtractTFIDF()[%d].Count = %d, want > 0", i, kw.Count)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestExtractTextRank
// ---------------------------------------------------------------------------

func TestExtractTextRank(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		topN     int
		wantNil  bool
		wantTop  string
		wantMaxN int
	}{
		{
			name:    "empty string",
			input:   "",
			topN:    5,
			wantNil: true,
		},
		{
			name:    "oversized input returns nil",
			input:   strings.Repeat("kitab ", maxInputBytes/6+1),
			topN:    5,
			wantNil: true,
		},
		{
			name:    "all stopwords",
			input:   "və bu da o ki",
			topN:    5,
			wantNil: true,
		},
		{
			name:     "single meaningful word",
			input:    "kitab",
			topN:     5,
			wantMaxN: 1,
		},
		{
			name: "known input returns keywords",
			input: "Azərbaycan iqtisadiyyatı sürətlə inkişaf edir. " +
				"Azərbaycan regionun ən böyük iqtisadiyyatına malikdir.",
			topN:    3,
			wantNil: false,
		},
		{
			name:     "sparse graph single word repeated",
			input:    "kitab kitab kitab",
			topN:     5,
			wantMaxN: 1,
		},
		{
			name:     "topN larger than available",
			input:    "kitab ev",
			topN:     100,
			wantMaxN: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ExtractTextRank(tt.input, tt.topN)

			if tt.wantNil {
				if got != nil {
					t.Errorf("ExtractTextRank() = %v, want nil", got)
				}
				return
			}

			if len(got) == 0 {
				t.Fatal("ExtractTextRank() returned nil, want non-nil")
			}

			if tt.wantTop != "" && got[0].Stem != tt.wantTop {
				t.Errorf("ExtractTextRank()[0].Stem = %q, want %q", got[0].Stem, tt.wantTop)
			}

			if tt.wantMaxN > 0 && len(got) > tt.wantMaxN {
				t.Errorf("ExtractTextRank() returned %d results, want <= %d", len(got), tt.wantMaxN)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestKeywords
// ---------------------------------------------------------------------------

func TestKeywords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantNil bool
	}{
		{
			name:    "empty string returns nil",
			input:   "",
			wantNil: true,
		},
		{
			name:    "all stopwords returns nil",
			input:   "və bu da o",
			wantNil: true,
		},
		{
			name:    "meaningful text returns keywords",
			input:   "Azərbaycan iqtisadiyyatı sürətlə inkişaf edir",
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Keywords(tt.input)

			if tt.wantNil {
				if got != nil {
					t.Errorf("Keywords() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatal("Keywords() = nil, want non-nil")
			}

			for i, s := range got {
				if s == "" {
					t.Errorf("Keywords()[%d] is empty", i)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestDeterminism
// ---------------------------------------------------------------------------

func TestDeterminism(t *testing.T) {
	t.Parallel()

	input := "kitab ev uşaq məktəb dərs müəllim şagird sinif"
	for range 10 {
		a := ExtractTFIDF(input, 10)
		b := ExtractTFIDF(input, 10)
		if !slices.Equal(a, b) {
			t.Fatalf("non-deterministic TF-IDF:\n  a = %v\n  b = %v", a, b)
		}

		c := ExtractTextRank(input, 10)
		d := ExtractTextRank(input, 10)
		if !slices.Equal(c, d) {
			t.Fatalf("non-deterministic TextRank:\n  c = %v\n  d = %v", c, d)
		}
	}
}

// TestDeterministicTieBreaking verifies that stems with equal scores
// are ordered lexicographically.
func TestDeterministicTieBreaking(t *testing.T) {
	t.Parallel()

	// Two words appearing exactly once each should tie on TF, so
	// tie-breaking should produce lexicographic order.
	input := "zebra alma"
	got := ExtractTFIDF(input, 10)
	if len(got) >= 2 {
		for i := 1; i < len(got); i++ {
			if got[i-1].Score == got[i].Score && got[i-1].Stem > got[i].Stem {
				t.Errorf("tie-break violated: %q (score=%.6f) before %q (score=%.6f)",
					got[i-1].Stem, got[i-1].Score, got[i].Stem, got[i].Score)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// TestMaxInputBytes
// ---------------------------------------------------------------------------

func TestMaxInputBytes(t *testing.T) {
	oversized := strings.Repeat("a", maxInputBytes+1)

	if got := ExtractTFIDF(oversized, 5); got != nil {
		t.Errorf("ExtractTFIDF on oversized input: got %v, want nil", got)
	}
	if got := ExtractTextRank(oversized, 5); got != nil {
		t.Errorf("ExtractTextRank on oversized input: got %v, want nil", got)
	}
	if got := Keywords(oversized); got != nil {
		t.Errorf("Keywords on oversized input: got %v, want nil", got)
	}
}

// ---------------------------------------------------------------------------
// TestConcurrentSafety
// ---------------------------------------------------------------------------

func TestConcurrentSafety(t *testing.T) {
	inputs := []string{
		"Azərbaycan iqtisadiyyatı inkişaf edir",
		"kitab kitablar kitabdan",
		"gözəl ev böyük bağ",
		"dərs müəllim şagird",
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

			for j := range 50 {
				input := inputs[j%len(inputs)]
				_ = ExtractTFIDF(input, 5)
				_ = ExtractTextRank(input, 5)
				_ = Keywords(input)
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
	tests := []struct {
		name  string
		input string
	}{
		{"invalid byte sequence", "kitab\xFF\xFElar"},
		{"truncated multibyte", "kitab\xC3"},
		{"overlong encoding", "kitab\xC0\x80"},
		{"null bytes", "kitab\x00lar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panicked on %q: %v", tt.name, r)
				}
			}()

			_ = ExtractTFIDF(tt.input, 5)
			_ = ExtractTextRank(tt.input, 5)
			_ = Keywords(tt.input)
		})
	}
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

var benchText = "Azərbaycan Respublikası Cənubi Qafqazda yerləşən müstəqil dövlətdir. " +
	"Ölkənin paytaxtı və ən böyük şəhəri Bakıdır. Azərbaycan ərazisi Böyük " +
	"Qafqaz dağları, Kür-Araz ovalığı və Kiçik Qafqaz dağlarını əhatə edir. " +
	"Xəzər dənizi sahilində yerləşən Bakı şəhəri ölkənin iqtisadi və mədəni " +
	"mərkəzidir. Azərbaycan neft və qaz ehtiyatları ilə zəngindir. Neft sənayesi " +
	"ölkə iqtisadiyyatının əsasını təşkil edir. Azərbaycan müstəqilliyini 1991-ci " +
	"ildə bərpa etmişdir. Ölkənin rəsmi dili Azərbaycan dilidir. Azərbaycan " +
	"əhalisi təxminən on milyon nəfərdir. Ölkədə müxtəlif milli azlıqlar yaşayır. " +
	"Azərbaycan Birləşmiş Millətlər Təşkilatının üzvüdür. Ölkə Avropa Şurasının " +
	"da üzvüdür. Azərbaycan iqtisadiyyatı son illərdə sürətlə inkişaf etmişdir. " +
	"Kənd təsərrüfatı ölkənin mühüm sahələrindən biridir. Pambıq, tütün və " +
	"üzüm əsas kənd təsərrüfatı məhsullarıdır. Azərbaycan mədəniyyəti zəngin " +
	"tarixə malikdir. Muğam Azərbaycanın ənənəvi musiqi janrıdır. Novruz bayramı " +
	"ölkənin ən mühüm bayramlarından biridir. Azərbaycan xalçaçılığı dünyada " +
	"məşhurdur. Ölkənin mətbəxi müxtəlif və dadlı yeməkləri ilə tanınır."

func BenchmarkExtractTFIDF(b *testing.B) {
	b.SetBytes(int64(len(benchText)))
	for b.Loop() {
		ExtractTFIDF(benchText, 10)
	}
}

func BenchmarkExtractTextRank(b *testing.B) {
	b.SetBytes(int64(len(benchText)))
	for b.Loop() {
		ExtractTextRank(benchText, 10)
	}
}

// ---------------------------------------------------------------------------
// Examples
// ---------------------------------------------------------------------------

func ExampleExtractTFIDF() {
	kws := ExtractTFIDF("Azərbaycan iqtisadiyyatı sürətlə inkişaf edir. Azərbaycan neft sektorunda liderdir.", 3)
	for _, kw := range kws {
		fmt.Printf("%s (count=%d)\n", kw.Stem, kw.Count)
	}
	// Output:
	// azərbaycan (count=2)
	// sektor (count=1)
	// lider (count=1)
}

func ExampleKeywords() {
	kws := Keywords("Azərbaycan iqtisadiyyatı sürətlə inkişaf edir")
	fmt.Println(kws)
	// Output:
	// [iqtisadiyyat sürət azərbaycan inkişaf]
}

func ExampleExtractTextRank() {
	kws := ExtractTextRank("Azərbaycan iqtisadiyyatı sürətlə inkişaf edir. Azərbaycan neft sektorunda liderdir.", 3)
	for _, kw := range kws {
		fmt.Printf("%s (count=%d)\n", kw.Stem, kw.Count)
	}
	// Output:
	// azərbaycan (count=2)
	// neft (count=1)
	// inkişaf (count=1)
}
