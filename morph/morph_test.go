package morph

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Phonology helpers
// ---------------------------------------------------------------------------

func TestIsVowel(t *testing.T) {
	tests := []struct {
		name string
		r    rune
		want bool
	}{
		// -- Azerbaijani vowels --
		{"a", 'a', true},
		{"e", 'e', true},
		{"ə", 'ə', true},
		{"ı", 'ı', true},
		{"i", 'i', true},
		{"o", 'o', true},
		{"ö", 'ö', true},
		{"u", 'u', true},
		{"ü", 'ü', true},

		// -- Uppercase vowels --
		{"A", 'A', true},
		{"E", 'E', true},
		{"Ə", 'Ə', true},
		{"I", 'I', true},
		{"İ", 'İ', true},
		{"O", 'O', true},
		{"Ö", 'Ö', true},
		{"U", 'U', true},
		{"Ü", 'Ü', true},

		// -- Consonants --
		{"b", 'b', false},
		{"k", 'k', false},
		{"ş", 'ş', false},
		{"ç", 'ç', false},

		// -- Non-Azerbaijani --
		{"z", 'z', false},
		{"1", '1', false},
		{"space", ' ', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isVowel(tt.r); got != tt.want {
				t.Errorf("isVowel(%q) = %v, want %v", tt.r, got, tt.want)
			}
		})
	}
}

func TestIsBackVowel(t *testing.T) {
	tests := []struct {
		name string
		r    rune
		want bool
	}{
		// -- Back vowels --
		{"a", 'a', true},
		{"A", 'A', true},
		{"ı", 'ı', true},
		{"I", 'I', true},
		{"o", 'o', true},
		{"O", 'O', true},
		{"u", 'u', true},
		{"U", 'U', true},

		// -- Front vowels --
		{"e", 'e', false},
		{"E", 'E', false},
		{"ə", 'ə', false},
		{"Ə", 'Ə', false},
		{"i", 'i', false},
		{"İ", 'İ', false},
		{"ö", 'ö', false},
		{"Ö", 'Ö', false},
		{"ü", 'ü', false},
		{"Ü", 'Ü', false},

		// -- Consonants --
		{"b", 'b', false},
		{"k", 'k', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBackVowel(tt.r); got != tt.want {
				t.Errorf("isBackVowel(%q) = %v, want %v", tt.r, got, tt.want)
			}
		})
	}
}

func TestBackFrontMutualExclusivity(t *testing.T) {
	vowels := []rune{'a', 'e', 'ə', 'ı', 'i', 'o', 'ö', 'u', 'ü',
		'A', 'E', 'Ə', 'I', 'İ', 'O', 'Ö', 'U', 'Ü'}

	for _, v := range vowels {
		isBack := isBackVowel(v)
		isFront := frontVowels[v]

		if isBack && isFront {
			t.Errorf("vowel %q is both back and front", v)
		}
		if !isBack && !isFront {
			t.Errorf("vowel %q is neither back nor front", v)
		}
	}
}

func TestIsVoiceless(t *testing.T) {
	tests := []struct {
		name string
		r    rune
		want bool
	}{
		// -- Voiceless consonants --
		{"p", 'p', true},
		{"ç", 'ç', true},
		{"t", 't', true},
		{"k", 'k', true},
		{"q", 'q', true},
		{"f", 'f', true},
		{"s", 's', true},
		{"ş", 'ş', true},
		{"x", 'x', true},
		{"h", 'h', true},

		// -- Voiced consonants --
		{"b", 'b', false},
		{"c", 'c', false},
		{"d", 'd', false},
		{"g", 'g', false},
		{"ğ", 'ğ', false},
		{"l", 'l', false},
		{"m", 'm', false},
		{"n", 'n', false},
		{"r", 'r', false},
		{"v", 'v', false},
		{"y", 'y', false},
		{"z", 'z', false},

		// -- Vowels --
		{"a", 'a', false},
		{"e", 'e', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isVoiceless(tt.r); got != tt.want {
				t.Errorf("isVoiceless(%q) = %v, want %v", tt.r, got, tt.want)
			}
		})
	}
}

func TestLastVowel(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want rune
	}{
		{"kitab", "kitab", 'a'},
		{"gəl", "gəl", 'ə'},
		{"st", "st", 0},
		{"empty", "", 0},
		{"dünya", "dünya", 'a'},
		{"ürək", "ürək", 'ə'},
		{"su", "su", 'u'},
		{"consonant cluster", "strkt", 0},
		{"single vowel", "a", 'a'},
		{"vowel at start", "astr", 'a'},
		{"multiple vowels", "kitablar", 'a'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lastVowel(tt.s); got != tt.want {
				t.Errorf("lastVowel(%q) = %q, want %q", tt.s, got, tt.want)
			}
		})
	}
}

func TestIsValidStem(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{"kitab", "kitab", true},
		{"gəl", "gəl", true},
		{"su", "su", true},
		{"at", "at", true},
		{"a", "a", false},
		{"st", "st", false},
		{"empty", "", false},
		{"ab", "ab", true},
		{"qq", "qq", false},
		{"single consonant", "k", false},
		{"two consonants no vowel", "kk", false},
		{"three chars with vowel", "kar", true},
		{"three chars no vowel", "str", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidStem(tt.s); got != tt.want {
				t.Errorf("isValidStem(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Azerbaijani-specific casing
// ---------------------------------------------------------------------------

func TestAzLower(t *testing.T) {
	tests := []struct {
		name string
		r    rune
		want rune
	}{
		{"I to ı", 'I', 'ı'},
		{"İ to i", 'İ', 'i'},
		{"A to a", 'A', 'a'},
		{"Ə to ə", 'Ə', 'ə'},
		{"Ş to ş", 'Ş', 'ş'},
		{"already lowercase a", 'a', 'a'},
		{"already lowercase ı", 'ı', 'ı'},
		{"digit unchanged", '1', '1'},
		{"space unchanged", ' ', ' '},
		{"Ç to ç", 'Ç', 'ç'},
		{"Ğ to ğ", 'Ğ', 'ğ'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := azLower(tt.r); got != tt.want {
				t.Errorf("azLower(%q) = %q, want %q", tt.r, got, tt.want)
			}
		})
	}
}

func TestToLower(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"Bakı", "Bakı", "bakı"},
		{"KITAB", "KITAB", "kıtab"},
		{"İstanbul", "İstanbul", "istanbul"},
		{"Əli", "Əli", "əli"},
		{"hello", "hello", "hello"},
		{"empty", "", ""},
		{"123", "123", "123"},
		{"MixedCaseƏŞÇ", "MixedCaseƏŞÇ", "mixedcaseəşç"},
		{"DAĞLAR", "DAĞLAR", "dağlar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toLower(tt.input); got != tt.want {
				t.Errorf("toLower(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Vowel harmony
// ---------------------------------------------------------------------------

func TestFourWayTarget(t *testing.T) {
	tests := []struct {
		name string
		v    rune
		want rune
	}{
		{"a to ı", 'a', 'ı'},
		{"ı to ı", 'ı', 'ı'},
		{"o to u", 'o', 'u'},
		{"u to u", 'u', 'u'},
		{"e to i", 'e', 'i'},
		{"ə to i", 'ə', 'i'},
		{"i to i", 'i', 'i'},
		{"ö to ü", 'ö', 'ü'},
		{"ü to ü", 'ü', 'ü'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fourWayTarget(tt.v); got != tt.want {
				t.Errorf("fourWayTarget(%q) = %q, want %q", tt.v, got, tt.want)
			}
		})
	}
}

func TestMatchesBackFront(t *testing.T) {
	tests := []struct {
		name          string
		stemLastVowel rune
		suffixVowel   rune
		want          bool
	}{
		{"back stem, back suffix", 'a', 'a', true},
		{"back stem, back suffix ı", 'a', 'ı', true},
		{"back stem, front suffix", 'a', 'e', false},
		{"front stem, front suffix", 'e', 'e', true},
		{"front stem, front suffix i", 'e', 'i', true},
		{"front stem, back suffix", 'e', 'a', false},
		{"no stem vowel, any suffix", 0, 'a', true},
		{"no stem vowel, front suffix", 0, 'e', true},
		{"back o, back u", 'o', 'u', true},
		{"front ö, front ü", 'ö', 'ü', true},
		{"back u, front i", 'u', 'i', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesBackFront(tt.stemLastVowel, tt.suffixVowel); got != tt.want {
				t.Errorf("matchesBackFront(%q, %q) = %v, want %v",
					tt.stemLastVowel, tt.suffixVowel, got, tt.want)
			}
		})
	}
}

func TestMatchesFourWay(t *testing.T) {
	tests := []struct {
		name          string
		stemLastVowel rune
		suffixVowel   rune
		want          bool
	}{
		{"a stem, ı suffix", 'a', 'ı', true},
		{"a stem, i suffix", 'a', 'i', false},
		{"o stem, u suffix", 'o', 'u', true},
		{"o stem, ı suffix", 'o', 'ı', false},
		{"ə stem, i suffix", 'ə', 'i', true},
		{"ə stem, ü suffix", 'ə', 'ü', false},
		{"ö stem, ü suffix", 'ö', 'ü', true},
		{"ö stem, i suffix", 'ö', 'i', false},
		{"no stem vowel, any suffix", 0, 'ı', true},
		{"no stem vowel, i suffix", 0, 'i', true},
		{"ı stem, ı suffix", 'ı', 'ı', true},
		{"e stem, i suffix", 'e', 'i', true},
		{"u stem, u suffix", 'u', 'u', true},
		{"ü stem, ü suffix", 'ü', 'ü', true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesFourWay(tt.stemLastVowel, tt.suffixVowel); got != tt.want {
				t.Errorf("matchesFourWay(%q, %q) = %v, want %v",
					tt.stemLastVowel, tt.suffixVowel, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// MorphTag String and JSON
// ---------------------------------------------------------------------------

func TestMorphTagString(t *testing.T) {
	tests := []struct {
		tag  MorphTag
		want string
	}{
		{Plural, "Plural"},
		{Poss1Sg, "Poss1Sg"},
		{Poss2Sg, "Poss2Sg"},
		{Poss3Sg, "Poss3Sg"},
		{Poss1Pl, "Poss1Pl"},
		{Poss2Pl, "Poss2Pl"},
		{Poss3Pl, "Poss3Pl"},
		{CaseGen, "CaseGen"},
		{CaseDat, "CaseDat"},
		{CaseAcc, "CaseAcc"},
		{CaseLoc, "CaseLoc"},
		{CaseAbl, "CaseAbl"},
		{CaseIns, "CaseIns"},
		{DerivAgent, "DerivAgent"},
		{DerivAbstract, "DerivAbstract"},
		{DerivPriv, "DerivPriv"},
		{DerivPoss, "DerivPoss"},
		{DerivVerb, "DerivVerb"},
		{Copula, "Copula"},
		{VoicePass, "VoicePass"},
		{VoiceReflex, "VoiceReflex"},
		{VoiceRecip, "VoiceRecip"},
		{VoiceCaus, "VoiceCaus"},
		{Negation, "Negation"},
		{TensePastDef, "TensePastDef"},
		{TensePastIndef, "TensePastIndef"},
		{TensePresent, "TensePresent"},
		{TenseFuture, "TenseFuture"},
		{TenseAorist, "TenseAorist"},
		{MoodOblig, "MoodOblig"},
		{MoodCond, "MoodCond"},
		{MoodImper, "MoodImper"},
		{Participle, "Participle"},
		{ParticipleAdj, "ParticipleAdj"},
		{Gerund, "Gerund"},
		{Pers1Sg, "Pers1Sg"},
		{Pers2Sg, "Pers2Sg"},
		{Pers1Pl, "Pers1Pl"},
		{Pers2Pl, "Pers2Pl"},
		{Pers3, "Pers3"},
		{Question, "Question"},
		{MorphTag(-1), "MorphTag(-1)"},
		{MorphTag(9999), "MorphTag(9999)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.tag.String(); got != tt.want {
				t.Errorf("MorphTag(%d).String() = %q, want %q", int(tt.tag), got, tt.want)
			}
		})
	}
}

func TestMorphTagJSON(t *testing.T) {
	tags := []MorphTag{
		Plural, Poss1Sg, Poss2Sg, Poss3Sg, Poss1Pl, Poss2Pl, Poss3Pl,
		CaseGen, CaseDat, CaseAcc, CaseLoc, CaseAbl, CaseIns,
		DerivAgent, DerivAbstract, DerivPriv, DerivPoss, DerivVerb,
		Copula,
		VoicePass, VoiceReflex, VoiceRecip, VoiceCaus,
		Negation,
		TensePastDef, TensePastIndef, TensePresent, TenseFuture, TenseAorist,
		MoodOblig, MoodCond, MoodImper,
		Participle, ParticipleAdj, Gerund,
		Pers1Sg, Pers2Sg, Pers1Pl, Pers2Pl, Pers3,
		Question,
	}

	for _, tag := range tags {
		t.Run(tag.String(), func(t *testing.T) {
			// Marshal
			data, err := json.Marshal(tag)
			if err != nil {
				t.Fatalf("Marshal(%v) failed: %v", tag, err)
			}
			expectedJSON := `"` + tag.String() + `"`
			if string(data) != expectedJSON {
				t.Errorf("Marshal(%v) = %s, want %s", tag, data, expectedJSON)
			}

			// Unmarshal
			var got MorphTag
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal(%s) failed: %v", data, err)
			}
			if got != tag {
				t.Errorf("Unmarshal(%s) = %v, want %v", data, got, tag)
			}
		})
	}
}

func TestMorphTagJSONUnknown(t *testing.T) {
	var tag MorphTag
	err := json.Unmarshal([]byte(`"UnknownTag"`), &tag)
	if err == nil {
		t.Error("expected error for unknown tag, got nil")
	}
	if !strings.Contains(err.Error(), "unknown morph tag") {
		t.Errorf("expected 'unknown morph tag' error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Analysis String
// ---------------------------------------------------------------------------

func TestAnalysisString(t *testing.T) {
	tests := []struct {
		name string
		a    Analysis
		want string
	}{
		{
			"bare stem",
			Analysis{Stem: "kitab"},
			"kitab",
		},
		{
			"stem with one morpheme",
			Analysis{
				Stem: "kitab",
				Morphemes: []Morpheme{
					{Surface: "lar", Tag: Plural},
				},
			},
			"kitab[Plural:lar]",
		},
		{
			"stem with two morphemes",
			Analysis{
				Stem: "kitab",
				Morphemes: []Morpheme{
					{Surface: "lar", Tag: Plural},
					{Surface: "dan", Tag: CaseAbl},
				},
			},
			"kitab[Plural:lar|CaseAbl:dan]",
		},
		{
			"stem with three morphemes",
			Analysis{
				Stem: "kitab",
				Morphemes: []Morpheme{
					{Surface: "lar", Tag: Plural},
					{Surface: "ımız", Tag: Poss1Pl},
					{Surface: "dan", Tag: CaseAbl},
				},
			},
			"kitab[Plural:lar|Poss1Pl:ımız|CaseAbl:dan]",
		},
		{
			"empty stem no morphemes",
			Analysis{Stem: ""},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.String(); got != tt.want {
				t.Errorf("Analysis.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Suffix table completeness
// ---------------------------------------------------------------------------

func TestSuffixTableCompleteness(t *testing.T) {
	if len(suffixRules) != 40 {
		t.Errorf("suffixRules has %d entries, want 40", len(suffixRules))
	}

	// Check all surfaces are lowercase
	for i, rule := range suffixRules {
		for j, surf := range rule.surfaces {
			if toLower(surf) != surf {
				t.Errorf("rule %d surface %d %q is not lowercase", i, j, surf)
			}
		}
	}

	// Check surfaces are sorted by rune count (longest first) within each rule
	for i, rule := range suffixRules {
		for j := 1; j < len(rule.surfaces); j++ {
			prevLen := len([]rune(rule.surfaces[j-1]))
			currLen := len([]rune(rule.surfaces[j]))
			if currLen > prevLen {
				t.Errorf("rule %d surfaces not sorted longest first (by rune count): %q (%d runes) before %q (%d runes)",
					i, rule.surfaces[j-1], prevLen, rule.surfaces[j], currLen)
			}
		}
	}

	// Verify expected tags are present
	expectedTags := map[MorphTag]bool{
		Plural: true, Poss1Sg: true, Poss2Sg: true, Poss3Sg: true,
		Poss1Pl: true, Poss2Pl: true, Poss3Pl: true,
		CaseGen: true, CaseDat: true, CaseAcc: true, CaseLoc: true,
		CaseAbl: true, CaseIns: true,
		DerivAgent: true, DerivAbstract: true, DerivPriv: true,
		DerivPoss: true, DerivVerb: true,
		Copula:  true,
		VoicePass: true, VoiceReflex: true, VoiceRecip: true, VoiceCaus: true,
		Negation: true,
		TensePastDef: true, TensePastIndef: true, TensePresent: true,
		TenseFuture: true, TenseAorist: true,
		MoodOblig: true, MoodCond: true,
		Participle: true, Gerund: true,
		Pers1Sg: true, Pers2Sg: true, Pers1Pl: true, Pers2Pl: true, Pers3: true,
		Question: true,
	}

	foundTags := make(map[MorphTag]bool)
	for _, rule := range suffixRules {
		foundTags[rule.tag] = true
	}

	for tag := range expectedTags {
		if !foundTags[tag] {
			t.Errorf("expected tag %v not found in suffixRules", tag)
		}
	}
}

// ---------------------------------------------------------------------------
// Stem (expanded from Phase 1 stubs, renamed from TestStemStubs)
// ---------------------------------------------------------------------------

func TestStem(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// -- Kept from Phase 1 stubs --
		{"empty", "", ""},
		{"too long", strings.Repeat("a", 257), strings.Repeat("a", 257)},
		{"apostrophe single", "Bakı'nın", "Bakı"},
		{"apostrophe right", "Bakı\u2019nın", "Bakı"},
		{"apostrophe modifier", "Bakı\u02BCnın", "Bakı"},
		{"hyphen compound", "sosial-iqtisadi", "sosial-iqtisadi"},
		{"multiple hyphens", "a-b-c", "a-b-c"},
		{"apostrophe at start", "'test", "'test"},
		{"apostrophe at end", "test'", "test"},

		// -- Bare words that should NOT be stemmed --
		{"bare kitab", "kitab", "kitab"},
		{"bare gəl", "gəl", "gəl"},
		{"bare su", "su", "su"},

		// -- Single suffix: plural --
		{"plural kitablar", "kitablar", "kitab"},
		{"plural evlər", "evlər", "ev"},
		{"plural uşaqlar", "uşaqlar", "uşaq"},

		// -- Multi-suffix chains --
		{"multi kitablarımızdan", "kitablarımızdan", "kitab"},
		{"multi evlərdə", "evlərdə", "ev"},

		// -- Verb forms --
		{"verb gəlmişdir", "gəlmişdir", "gəl"},
		{"verb gəldim", "gəldim", "gəl"},
		{"verb yazdılar", "yazdılar", "yaz"},

		// -- Copula on noun --
		// Dictionary-aware ranking picks müəllim (known stem) over the
		// deeper but incorrect müəl (DerivPoss:li + Poss1Sg:m + Copula:dir).
		{"copula müəllimdir", "müəllimdir", "müəllim"},

		// -- Derivational: agent --
		{"deriv kitabçı", "kitabçı", "kitab"},
		// -- Derivational chain: agent + abstract --
		{"deriv kitabçılıq", "kitabçılıq", "kitab"},
		// -- Derivational chain: agent + abstract + locative --
		// kitabçılıqda now correctly parses as kitabçılıq + da (CaseLoc)
		// after fixing d/t assimilation to allow -da after q.
		{"deriv+case kitabçılıqda", "kitabçılıqda", "kitab"},

		// -- Case preservation --
		{"case Kitablar", "Kitablar", "Kitab"},
		{"case KITABLAR", "KITABLAR", "KITAB"},

		// -- Possessive --
		{"poss1sg kitabım", "kitabım", "kitab"},
		{"poss2sg kitabın", "kitabın", "kitab"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Stem(tt.input)
			if got != tt.want {
				t.Errorf("Stem(%q) = %q, want %q\n  analyses: %v",
					tt.input, got, tt.want, Analyze(tt.input))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Analyze (expanded from Phase 1 stubs, renamed from TestAnalyzeStubs)
// ---------------------------------------------------------------------------

func TestAnalyze(t *testing.T) {
	t.Run("empty returns nil", func(t *testing.T) {
		got := Analyze("")
		if got != nil {
			t.Errorf("Analyze(%q) = %v, want nil", "", got)
		}
	})

	t.Run("bare word has stem-only analysis", func(t *testing.T) {
		got := Analyze("kitab")
		if len(got) == 0 {
			t.Fatal("Analyze(\"kitab\") returned empty")
		}
		found := false
		for _, a := range got {
			if a.Stem == "kitab" && len(a.Morphemes) == 0 {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Analyze(\"kitab\") missing bare-stem analysis; got %v", got)
		}
	})

	t.Run("kitablar has plural analysis", func(t *testing.T) {
		got := Analyze("kitablar")
		if !hasAnalysis(got, "kitab", []MorphTag{Plural}) {
			t.Errorf("Analyze(\"kitablar\") missing kitab[Plural]; got %v", got)
		}
	})

	t.Run("too long returns original as stem", func(t *testing.T) {
		long := strings.Repeat("a", 257)
		got := Analyze(long)
		if len(got) != 1 || got[0].Stem != long {
			t.Errorf("Analyze(too_long) = %v, want [{Stem: too_long}]", got)
		}
	})
}

// ---------------------------------------------------------------------------
// Stems (expanded from Phase 1 stubs, renamed from TestStemsStubs)
// ---------------------------------------------------------------------------

func TestStems(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{"nil input", nil, nil},
		{"empty slice", []string{}, []string{}},
		{"single word", []string{"kitab"}, []string{"kitab"}},
		{"multiple words", []string{"kitab", "gəl", "su"}, []string{"kitab", "gəl", "su"}},
		{"apostrophe handling", []string{"Bakı'nın", "kitab"}, []string{"Bakı", "kitab"}},
		{"stemming words", []string{"kitablar", "evlər"}, []string{"kitab", "ev"}},
		{"batch stemming", []string{"kitablarımızdan", "evlərdə", "gəlmişdir"}, []string{"kitab", "ev", "gəl"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Stems(tt.input)
			if tt.want == nil {
				if got != nil {
					t.Errorf("Stems(%v) = %v, want nil", tt.input, got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("Stems(%v): got %d stems, want %d", tt.input, len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("stem %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Phase 2: Suffix category analysis tests
// ---------------------------------------------------------------------------

// hasAnalysis checks whether results contain at least one analysis with the
// given stem (case-insensitive) and all expected tags appearing in order
// (subsequence match).
func hasAnalysis(results []Analysis, stem string, tags []MorphTag) bool {
	for _, a := range results {
		if toLower(a.Stem) != toLower(stem) {
			continue
		}
		if containsTags(a.Morphemes, tags) {
			return true
		}
	}
	return false
}

// containsTags checks whether morphemes contain all tags in order
// (subsequence match, not exact match).
func containsTags(morphemes []Morpheme, tags []MorphTag) bool {
	ti := 0
	for _, m := range morphemes {
		if ti < len(tags) && m.Tag == tags[ti] {
			ti++
		}
	}
	return ti == len(tags)
}

func TestAnalyzeSuffixCategories(t *testing.T) {
	tests := []struct {
		word string
		stem string
		tags []MorphTag
	}{
		// Plural
		{"kitablar", "kitab", []MorphTag{Plural}},
		{"evlər", "ev", []MorphTag{Plural}},

		// Possessive 1sg
		{"kitabım", "kitab", []MorphTag{Poss1Sg}},

		// Possessive 2sg (also matches CaseGen, but Poss2Sg must exist)
		{"kitabın", "kitab", []MorphTag{Poss2Sg}},

		// Multi-suffix: possessive 1pl + ablative
		{"kitabımızdan", "kitab", []MorphTag{Poss1Pl, CaseAbl}},

		// Case locative
		{"evdə", "ev", []MorphTag{CaseLoc}},
		{"kitabçılıqda", "kitabçılıq", []MorphTag{CaseLoc}},
		{"otaqda", "otaq", []MorphTag{CaseLoc}},

		// Case ablative
		{"evdən", "ev", []MorphTag{CaseAbl}},

		// Case dative
		{"evə", "ev", []MorphTag{CaseDat}},

		// Derivational agent
		{"kitabçı", "kitab", []MorphTag{DerivAgent}},

		// Derivational agent + abstract
		{"kitabçılıq", "kitab", []MorphTag{DerivAgent, DerivAbstract}},

		// Verb: past indefinite + copula
		{"gəlmişdir", "gəl", []MorphTag{TensePastIndef, Copula}},

		// Verb: past definite + person 3
		{"yazdılar", "yaz", []MorphTag{TensePastDef, Pers3}},

		// Verb: past definite + person 1sg
		{"gəldim", "gəl", []MorphTag{TensePastDef, Pers1Sg}},

		// Plural + possessive 1pl + ablative
		{"kitablarımızdan", "kitab", []MorphTag{Plural, Poss1Pl, CaseAbl}},

		// Plural + locative
		{"evlərdə", "ev", []MorphTag{Plural, CaseLoc}},

		// Copula on noun (the analysis set must contain this parse
		// even though Stem() picks a deeper analysis)
		{"müəllimdir", "müəllim", []MorphTag{Copula}},

		// Present tense conjugation (extended person suffixes)
		{"bilirəm", "bil", []MorphTag{TensePresent, Pers1Sg}},
		{"bilirsən", "bil", []MorphTag{TensePresent, Pers2Sg}},
		{"bilirik", "bil", []MorphTag{TensePresent, Pers1Pl}},
		{"bilirsiniz", "bil", []MorphTag{TensePresent, Pers2Pl}},
		{"bilirlər", "bil", []MorphTag{TensePresent, Pers3}},

		// Past definite conjugation (existing short person suffixes)
		{"yazdıq", "yaz", []MorphTag{TensePastDef, Pers1Pl}},

		// Aorist conjugation
		{"yazaram", "yaz", []MorphTag{TenseAorist, Pers1Sg}},
	}

	for _, tt := range tests {
		t.Run(tt.word, func(t *testing.T) {
			results := Analyze(tt.word)
			if !hasAnalysis(results, tt.stem, tt.tags) {
				strs := make([]string, len(results))
				for i, a := range results {
					strs[i] = a.String()
				}
				t.Errorf("Analyze(%q) missing analysis with stem=%q tags=%v\n  got: %v",
					tt.word, tt.stem, tt.tags, strs)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Consonant assimilation (d/t alternation)
// ---------------------------------------------------------------------------

func TestConsonantAssimilation(t *testing.T) {
	// b is voiced, so locative uses -da form
	t.Run("kitabda voiced b uses da", func(t *testing.T) {
		got := Stem("kitabda")
		if toLower(got) != "kitab" {
			t.Errorf("Stem(\"kitabda\") = %q, want \"kitab\"", got)
		}
	})

	// c is voiced in Azerbaijani, so locative uses -da form
	t.Run("ağacda voiced c uses da", func(t *testing.T) {
		got := Stem("ağacda")
		if toLower(got) != "ağac" {
			t.Errorf("Stem(\"ağacda\") = %q, want \"ağac\"", got)
		}
	})

	// t after voiced b should NOT produce a locative parse
	t.Run("kitabta t after voiced b rejected", func(t *testing.T) {
		results := Analyze("kitabta")
		for _, a := range results {
			for _, m := range a.Morphemes {
				if m.Tag == CaseLoc && m.Surface == "ta" && toLower(a.Stem) == "kitab" {
					t.Errorf("kitabta should not parse as kitab+ta (t after voiced b), got: %s", a)
				}
			}
		}
	})

	// t after voiceless k is the correct form
	t.Run("çiçəktə t after voiceless k valid", func(t *testing.T) {
		results := Analyze("çiçəktə")
		if !hasAnalysis(results, "çiçək", []MorphTag{CaseLoc}) {
			strs := make([]string, len(results))
			for i, a := range results {
				strs[i] = a.String()
			}
			t.Errorf("çiçəktə should have CaseLoc parse with stem çiçək, got: %v", strs)
		}
	})

	// d after voiceless k should NOT produce a locative parse
	t.Run("çiçəkdə d after voiceless k rejected", func(t *testing.T) {
		results := Analyze("çiçəkdə")
		for _, a := range results {
			for _, m := range a.Morphemes {
				if m.Tag == CaseLoc && m.Surface == "də" && toLower(a.Stem) == "çiçək" {
					t.Errorf("çiçəkdə should not parse as çiçək+də (d after voiceless k), got: %s", a)
				}
			}
		}
	})
}

// ---------------------------------------------------------------------------
// k/q softening restoration
// ---------------------------------------------------------------------------

func TestKQSoftening(t *testing.T) {
	tests := []struct {
		word     string
		restored string
		desc     string
	}{
		// k -> y before vowel suffix, FSM tries restoring k
		{"ürəyi", "ürək", "k restored from y in ürək"},
		// q -> ğ before vowel suffix, FSM tries restoring q
		{"uşağı", "uşaq", "q restored from ğ in uşaq"},
		// k -> y restored
		{"çiçəyi", "çiçək", "k restored from y in çiçək"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			results := Analyze(tt.word)
			found := false
			for _, a := range results {
				if toLower(a.Stem) == toLower(tt.restored) {
					found = true
					break
				}
			}
			if !found {
				strs := make([]string, len(results))
				for i, a := range results {
					strs[i] = a.String()
				}
				t.Errorf("Analyze(%q) missing restored stem %q; got %v",
					tt.word, tt.restored, strs)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Case preservation
// ---------------------------------------------------------------------------

func TestCasePreservation(t *testing.T) {
	tests := []struct {
		word string
		stem string
	}{
		{"Kitablar", "Kitab"},
		{"KITABLAR", "KITAB"},
	}

	for _, tt := range tests {
		t.Run(tt.word, func(t *testing.T) {
			got := Stem(tt.word)
			if got != tt.stem {
				t.Errorf("Stem(%q) = %q, want %q (case must be preserved)",
					tt.word, got, tt.stem)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Over-stemming regression
// ---------------------------------------------------------------------------

func TestOverStemmingRegression(t *testing.T) {
	tests := []struct {
		word string
		want string
	}{
		{"ana", "ana"},
		{"baba", "baba"},
		{"gecə", "gecə"},
		{"sevgi", "sevgi"},
		{"alma", "alma"},
	}

	for _, tt := range tests {
		t.Run(tt.word, func(t *testing.T) {
			if got := Stem(tt.word); got != tt.want {
				t.Errorf("Stem(%q) = %q, want %q", tt.word, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// False positive elimination
// ---------------------------------------------------------------------------

func TestPersonSuffixFalsePositives(t *testing.T) {
	// These words must not be over-stemmed by person suffix markers.
	tests := []struct {
		word string
		desc string
	}{
		{"dəniz", "should not strip -niz as Pers2Pl"},
		{"gün", "should not strip -n as Pers2Sg"},
		{"ürək", "should not strip -k as Pers1Pl"},
	}

	for _, tt := range tests {
		t.Run(tt.word, func(t *testing.T) {
			if got := Stem(tt.word); got != tt.word {
				t.Errorf("Stem(%q) = %q, want %q (%s)",
					tt.word, got, tt.word, tt.desc)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Under-stemming regression
// ---------------------------------------------------------------------------

func TestUnderStemmingRegression(t *testing.T) {
	tests := []struct {
		word string
		stem string
	}{
		{"kitablarımızdan", "kitab"},
		{"evlərdə", "ev"},
		{"gəlmişdir", "gəl"},
		{"gəldim", "gəl"},
		{"yazdılar", "yaz"},
		{"uşaqlar", "uşaq"},
	}

	for _, tt := range tests {
		t.Run(tt.word, func(t *testing.T) {
			got := Stem(tt.word)
			if toLower(got) != toLower(tt.stem) {
				t.Errorf("Stem(%q) = %q, want %q (under-stemming regression)",
					tt.word, got, tt.stem)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Reconstruction invariant: stem + morpheme surfaces == original word
// ---------------------------------------------------------------------------

func verifyInvariants(t *testing.T, word string, a Analysis) {
	t.Helper()

	suffixes := ""
	for _, m := range a.Morphemes {
		suffixes += m.Surface
	}

	// Direct reconstruction: stem + suffixes.
	direct := a.Stem + suffixes
	if toLower(direct) == toLower(word) {
		return
	}

	// Try k/q softening: when the analyzer restores k/q at the stem
	// boundary but the original word has y/ğ, reconstruction needs to
	// reverse the restoration.
	stemRunes := []rune(a.Stem)
	if len(stemRunes) > 0 && len(a.Morphemes) > 0 {
		firstSuffRunes := []rune(a.Morphemes[0].Surface)
		if len(firstSuffRunes) > 0 && isVowel(firstSuffRunes[0]) {
			var softened string
			switch azLower(stemRunes[len(stemRunes)-1]) {
			case 'k':
				softened = string(stemRunes[:len(stemRunes)-1]) + "y" + suffixes
			case 'q':
				softened = string(stemRunes[:len(stemRunes)-1]) + "ğ" + suffixes
			}
			if softened != "" && toLower(softened) == toLower(word) {
				return
			}
		}
	}

	// Try k/q softening at suffix boundaries: when a suffix ends in k/q
	// and the next suffix starts with a vowel, k→y or q→ğ.
	if len(a.Morphemes) > 1 {
		reconstructed := a.Stem
		for i, m := range a.Morphemes {
			surfRunes := []rune(m.Surface)
			if i > 0 && len(surfRunes) > 0 && isVowel(surfRunes[0]) {
				// Check if previous suffix ended in k or q
				prevRunes := []rune(a.Morphemes[i-1].Surface)
				if len(prevRunes) > 0 {
					lastRune := azLower(prevRunes[len(prevRunes)-1])
					if lastRune == 'k' {
						// Replace k with y
						reconstructed = reconstructed[:len(reconstructed)-1] + "y"
					} else if lastRune == 'q' {
						// Replace q with ğ
						reconstructed = reconstructed[:len(reconstructed)-1] + "ğ"
					}
				}
			}
			reconstructed += m.Surface
		}
		if toLower(reconstructed) == toLower(word) {
			return
		}
	}

	t.Errorf("invariant violation: %q reconstructed as %q from %s",
		word, direct, a)
}

func TestVerifyInvariants(t *testing.T) {
	words := []string{
		"kitab", "kitablar", "kitablarımızdan",
		"evlər", "evlərdə", "evdən", "evə",
		"gəlmişdir", "gəldim", "yazdılar",
		"kitabçı", "kitabçılıq", "kitabçılıqda",
		"kitabım", "kitabın",
		"ürəyi", "uşağı", "çiçəyi",
		"kitabda", "ağacda", "otaqda",
		"uşaqlar",
		"Kitablar", "KITABLAR",
		"müəllimdir",
		"bilirəm", "bilirsən", "bilirik", "bilirsiniz", "bilirlər",
	}

	for _, w := range words {
		t.Run(w, func(t *testing.T) {
			results := Analyze(w)
			for _, a := range results {
				verifyInvariants(t, w, a)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkStem(b *testing.B) {
	for b.Loop() {
		Stem("kitablarımızdan")
	}
}

func BenchmarkStemShort(b *testing.B) {
	for b.Loop() {
		Stem("evdə")
	}
}

func BenchmarkStemBare(b *testing.B) {
	for b.Loop() {
		Stem("kitab")
	}
}

func BenchmarkAnalyze(b *testing.B) {
	for b.Loop() {
		Analyze("kitablarımızdan")
	}
}

func BenchmarkStems(b *testing.B) {
	words := []string{"kitablarımızdan", "evlərdə", "gəlmişdir", "yazdılar", "müəllimdir"}
	for b.Loop() {
		Stems(words)
	}
}

// ---------------------------------------------------------------------------
// Golden tests
// ---------------------------------------------------------------------------

type goldenEntry struct {
	Word      string     `json:"word"`
	Stem      string     `json:"stem"`
	Morphemes []Morpheme `json:"morphemes"`
}

var updateGolden = flag.Bool("update", false, "update golden test file")

func TestGolden(t *testing.T) {
	data, err := os.ReadFile("../data/golden/morph.json")
	if err != nil {
		t.Skipf("golden file not found: %v", err)
	}
	var entries []goldenEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("parse golden file: %v", err)
	}

	if *updateGolden {
		for i := range entries {
			entries[i].Stem = Stem(entries[i].Word)
			results := Analyze(entries[i].Word)
			// Pick the first analysis with morphemes, or leave empty
			entries[i].Morphemes = nil
			for _, a := range results {
				if len(a.Morphemes) > 0 {
					entries[i].Morphemes = a.Morphemes
					break
				}
			}
		}
		out, _ := json.MarshalIndent(entries, "", "  ")
		if err := os.WriteFile("../data/golden/morph.json", out, 0644); err != nil {
			t.Fatalf("write golden file: %v", err)
		}
		t.Log("golden file updated")
		return
	}

	for _, e := range entries {
		t.Run(e.Word, func(t *testing.T) {
			gotStem := Stem(e.Word)
			if gotStem != e.Stem {
				t.Errorf("Stem(%q) = %q, want %q", e.Word, gotStem, e.Stem)
			}
			for _, a := range Analyze(e.Word) {
				verifyInvariants(t, e.Word, a)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Fuzz tests
// ---------------------------------------------------------------------------

func FuzzStem(f *testing.F) {
	f.Add("kitablar")
	f.Add("gəlmişdir")
	f.Add("evlərdə")
	f.Add("")
	f.Add("a")
	f.Add("test-test")
	f.Add("Bakı'nın")
	f.Fuzz(func(t *testing.T, word string) {
		result := Stem(word)
		if word != "" && result == "" {
			t.Errorf("Stem(%q) returned empty for non-empty input", word)
		}
	})
}

func FuzzAnalyze(f *testing.F) {
	f.Add("kitablar")
	f.Add("gəlmişdir")
	f.Add("")
	f.Add("a")
	f.Fuzz(func(t *testing.T, word string) {
		results := Analyze(word)
		if word == "" {
			if results != nil {
				t.Errorf("Analyze(%q) = %v, want nil", word, results)
			}
			return
		}
		if len(results) == 0 {
			t.Errorf("Analyze(%q) returned empty", word)
		}
		// Every analysis stem must be non-empty
		for _, a := range results {
			if a.Stem == "" {
				t.Errorf("Analyze(%q) produced empty stem", word)
			}
			// Verify reconstruction invariant
			verifyInvariants(t, word, a)
		}
	})
}

// ---------------------------------------------------------------------------
// Examples
// ---------------------------------------------------------------------------

func ExampleStem() {
	fmt.Println(Stem("kitablarımızdan"))
	fmt.Println(Stem("gəlmişdir"))
	fmt.Println(Stem("evlərdə"))
	// Output:
	// kitab
	// gəl
	// ev
}

func ExampleAnalyze() {
	for _, a := range Analyze("kitablar") {
		fmt.Println(a)
	}
	// Output:
	// kitab[Plural:lar]
	// kitabl[TenseAorist:ar]
	// kitablar
}

func ExampleStems() {
	words := []string{"kitablarımızdan", "evlərdə", "gəlmişdir"}
	fmt.Println(Stems(words))
	// Output:
	// [kitab ev gəl]
}
