// Package detect identifies the natural language of input text.
//
// Four languages are supported: Azerbaijani (Latin and Cyrillic scripts),
// Russian, English, and Turkish. Detection uses a hybrid approach: character-set
// scoring as the primary path with a short trigram fallback for ambiguous cases
// (Azerbaijani vs Turkish when no schwa ə is present).
//
// Two API layers are provided:
//
//   - Structured: Detect returns a Result with language, script, and confidence.
//     DetectAll returns all four languages ranked by confidence.
//   - Convenience: Lang returns the ISO 639-1 code as a string.
//
// Input longer than 1 MiB is silently truncated (rune-safe). Input with fewer
// than 10 letter runes returns the zero Result (Lang: Unknown).
//
// All functions are safe for concurrent use by multiple goroutines.
package detect

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"unicode"
	"unicode/utf8"
)

// Language identifies a natural language.
type Language int

const (
	Unknown     Language = iota // zero value, no detection performed
	Azerbaijani                 // Azerbaijani (Latin or Cyrillic script)
	Russian                     // Russian (Cyrillic script)
	English                     // English (Latin script)
	Turkish                     // Turkish (Latin script)
)

// languageNames maps Language values to their string names.
var languageNames = [...]string{
	Unknown:     "Unknown",
	Azerbaijani: "Azerbaijani",
	Russian:     "Russian",
	English:     "English",
	Turkish:     "Turkish",
}

// languageFromName maps string names back to Language values.
var languageFromName = map[string]Language{
	"Unknown":     Unknown,
	"Azerbaijani": Azerbaijani,
	"Russian":     Russian,
	"English":     English,
	"Turkish":     Turkish,
}

// languageCodes maps Language values to ISO 639-1 codes.
var languageCodes = [...]string{
	Unknown:     "",
	Azerbaijani: "az",
	Russian:     "ru",
	English:     "en",
	Turkish:     "tr",
}

// String returns the name of the language.
func (l Language) String() string {
	if int(l) >= 0 && int(l) < len(languageNames) {
		return languageNames[l]
	}
	return fmt.Sprintf("Language(%d)", int(l))
}

// MarshalJSON encodes the language as a JSON string (e.g. "Azerbaijani").
func (l Language) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.String())
}

// UnmarshalJSON decodes a JSON string (e.g. "Azerbaijani") into a Language.
func (l *Language) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	lang, ok := languageFromName[s]
	if !ok {
		return fmt.Errorf("detect: unknown language: %q", s)
	}
	*l = lang
	return nil
}

// Script identifies the writing system of a detected language.
type Script int

const (
	ScriptUnknown Script = iota // zero value or not applicable
	ScriptLatn                  // ISO 15924: Latin
	ScriptCyrl                  // ISO 15924: Cyrillic
)

// scriptNames maps Script values to their ISO 15924 string codes.
var scriptNames = [...]string{
	ScriptUnknown: "",
	ScriptLatn:    "Latn",
	ScriptCyrl:    "Cyrl",
}

// scriptFromName maps ISO 15924 string codes back to Script values.
var scriptFromName = map[string]Script{
	"":     ScriptUnknown,
	"Latn": ScriptLatn,
	"Cyrl": ScriptCyrl,
}

// String returns the ISO 15924 code of the script, or "" for ScriptUnknown.
func (s Script) String() string {
	if int(s) >= 0 && int(s) < len(scriptNames) {
		return scriptNames[s]
	}
	return fmt.Sprintf("Script(%d)", int(s))
}

// MarshalJSON encodes the script as a JSON string (e.g. "Latn").
func (s Script) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON decodes a JSON string (e.g. "Latn") into a Script.
func (s *Script) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	sc, ok := scriptFromName[str]
	if !ok {
		return fmt.Errorf("detect: unknown script: %q", str)
	}
	*s = sc
	return nil
}

// Result holds the outcome of a language detection.
//
// Confidence is a sum-normalized score in [0.0, 1.0]. All four language scores
// are divided by their total, so Confidence reflects the relative strength of
// the detection within this input, not an absolute probability.
type Result struct {
	Lang       Language `json:"lang"`
	Script     Script   `json:"script"`
	Confidence float64  `json:"confidence"`
}

const (
	maxInputBytes = 1 << 20 // 1 MiB — inputs longer than this are truncated
	minLetters    = 10      // minimum letter count for meaningful detection
)

// Scoring weights for the hybrid detection algorithm.
const (
	// cyrillicAzBias is the prior probability for Azerbaijani when no
	// discriminating Cyrillic characters are found.
	cyrillicAzBias = 0.45

	// cyrillicRuBias is the prior probability for Russian when no
	// discriminating Cyrillic characters are found.
	cyrillicRuBias = 0.55

	// schwaMultiplier amplifies the schwa count because schwa is the
	// single strongest discriminator between Azerbaijani and Turkish Latin.
	schwaMultiplier = 10.0

	// sharedTurkicDampener reduces the shared-character score to keep it
	// subordinate to the schwa score in the clear Azerbaijani path.
	sharedTurkicDampener = 0.1

	// xqBoostPerChar is the per-character score bonus for x/q letters,
	// which are common in Azerbaijani but rare in Turkish.
	xqBoostPerChar = 0.5

	// englishTurkicDampener suppresses the English score when Turkic
	// markers are present in the text.
	englishTurkicDampener = 0.05
)

// Detect identifies the most likely language of s.
// Returns the zero Result when detection is not possible (empty input, too
// few letters, or input that does not resemble a supported language).
func Detect(s string) Result {
	results := DetectAll(s)
	if len(results) == 0 {
		return Result{}
	}
	return results[0]
}

// Lang returns the ISO 639-1 code of the most likely language of s
// (e.g. "az", "ru", "en", "tr"), or "" when detection is not possible.
func Lang(s string) string {
	r := Detect(s)
	if r.Lang == Unknown {
		return ""
	}
	return languageCodes[r.Lang]
}

// DetectAll returns all four supported languages ranked by descending
// confidence, or nil when detection is not possible.
func DetectAll(s string) []Result {
	if s == "" {
		return nil
	}

	// Truncate to maxInputBytes rune-safely.
	if len(s) > maxInputBytes {
		pos := maxInputBytes
		for pos > 0 && !utf8.RuneStart(s[pos]) {
			pos--
		}
		s = s[:pos]
	}

	// Single-pass character classification.
	//
	// Cyrillic unique to Azerbaijani: ә/Ә ғ/Ғ ҹ/Ҹ ҝ/Ҝ ө/Ө ү/Ү һ/Һ ј/Ј
	// Cyrillic unique to Russian:     ы/Ы э/Э щ/Щ
	// Latin unique to Azerbaijani:    ə/Ə (schwa — strongest discriminator)
	// Latin shared Turkish/Azerbaijani: ğ/Ğ ş/Ş ç/Ç ö/Ö ü/Ü ı/İ
	// Latin Azerbaijani signal:        x/X q/Q (common in az, rare in tr)
	var (
		totalLetters       int
		cyrillicLetters    int
		latinLetters       int
		asciiLetters       int
		azLatinUniqueCount int
		azCyrUniqueCount   int
		ruUniqueCount      int
		trAzSharedCount    int
		xqCount            int
	)

	for _, r := range s {
		if !unicode.IsLetter(r) {
			continue
		}
		totalLetters++

		if isCyrillic(r) {
			cyrillicLetters++
			switch r {
			case 'ә', 'Ә', 'ғ', 'Ғ', 'ҹ', 'Ҹ', 'ҝ', 'Ҝ', 'ө', 'Ө', 'ү', 'Ү', 'һ', 'Һ', 'ј', 'Ј':
				azCyrUniqueCount++
			case 'ы', 'Ы', 'э', 'Э', 'щ', 'Щ':
				ruUniqueCount++
			}
		} else {
			latinLetters++
			switch r {
			case 'ə', 'Ə':
				azLatinUniqueCount++
			case 'ğ', 'Ğ', 'ş', 'Ş', 'ç', 'Ç', 'ö', 'Ö', 'ü', 'Ü', 'ı', 'İ':
				trAzSharedCount++
			case 'x', 'X', 'q', 'Q':
				xqCount++
			}
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				asciiLetters++
			}
		}
	}

	if totalLetters < minLetters {
		return nil
	}

	isCyrillicDominant := cyrillicLetters > latinLetters

	// Raw scores for each language. Scores are non-negative floats; they are
	// normalized to sum to 1.0 before building the Result slice.
	var azScore, ruScore, enScore, trScore float64
	var azScript Script

	if isCyrillicDominant {
		azScript = ScriptCyrl
		azScore = float64(azCyrUniqueCount)
		ruScore = float64(ruUniqueCount)

		// No discriminating characters found — apply a slight Russian bias
		// because Russian is more common in Cyrillic contexts.
		if azScore == 0 && ruScore == 0 {
			azScore = cyrillicAzBias
			ruScore = cyrillicRuBias
		}
		// English and Turkish do not use Cyrillic.
		enScore = 0
		trScore = 0
	} else {
		azScript = ScriptLatn

		if azLatinUniqueCount > 0 {
			// Schwa (ə/Ə) is exclusive to Azerbaijani Latin — strong signal.
			azScore = float64(azLatinUniqueCount) * schwaMultiplier
			trScore = float64(trAzSharedCount) * sharedTurkicDampener
		} else if trAzSharedCount > 0 {
			// Shared Turkic special characters present but no schwa — ambiguous.
			// Use trigram cosine similarity to break the tie.
			inputTrigrams := extractTrigrams(s)
			azTrigram := trigramCosine(inputTrigrams, azLatnTrigrams, azLatnTrigramNorm)
			trTrigram := trigramCosine(inputTrigrams, trTrigrams, trTrigramNorm)

			// x/q letters are a secondary Azerbaijani signal.
			xqBoost := float64(xqCount) * xqBoostPerChar

			azScore = azTrigram + xqBoost
			trScore = trTrigram
		}
		// English and Turkish without Turkic markers are not covered by the
		// branches above. azScore and trScore default to 0.0 in that case.

		// English score: high when text is mostly ASCII with no Turkic markers.
		if trAzSharedCount == 0 && azLatinUniqueCount == 0 {
			enScore = float64(asciiLetters) / float64(totalLetters)
		} else {
			// Turkic markers are present — dampen English score strongly.
			enScore = float64(asciiLetters) / float64(totalLetters) * englishTurkicDampener
		}

		ruScore = 0
	}

	// Normalize scores so they sum to 1.0.
	total := azScore + ruScore + enScore + trScore
	if total == 0 {
		return nil
	}

	results := []Result{
		{Lang: Azerbaijani, Script: azScript, Confidence: azScore / total},
		{Lang: Russian, Script: ScriptCyrl, Confidence: ruScore / total},
		{Lang: English, Script: ScriptLatn, Confidence: enScore / total},
		{Lang: Turkish, Script: ScriptLatn, Confidence: trScore / total},
	}

	slices.SortStableFunc(results, func(a, b Result) int {
		return cmp.Compare(b.Confidence, a.Confidence)
	})

	return results
}
