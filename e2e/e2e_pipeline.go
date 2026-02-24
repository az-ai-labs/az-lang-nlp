//go:build ignore

// e2e_pipeline exercises all 13 NLP modules in a single run and writes
// structured results to data/e2e_pipeline.log.
// Run from the project root:
//
//	go run scripts/e2e_pipeline.go
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/az-ai-labs/az-lang-nlp/chunker"
	"github.com/az-ai-labs/az-lang-nlp/datetime"
	"github.com/az-ai-labs/az-lang-nlp/detect"
	"github.com/az-ai-labs/az-lang-nlp/keywords"
	"github.com/az-ai-labs/az-lang-nlp/morph"
	"github.com/az-ai-labs/az-lang-nlp/ner"
	"github.com/az-ai-labs/az-lang-nlp/normalize"
	"github.com/az-ai-labs/az-lang-nlp/numtext"
	"github.com/az-ai-labs/az-lang-nlp/sentiment"
	"github.com/az-ai-labs/az-lang-nlp/spell"
	"github.com/az-ai-labs/az-lang-nlp/tokenizer"
	"github.com/az-ai-labs/az-lang-nlp/translit"
	"github.com/az-ai-labs/az-lang-nlp/validate"
)

// ---------- constants ----------

const (
	logPath       = "data/e2e_pipeline.log"
	moduleCount   = 13
	maxDetailLen  = 200
	concWorkers   = 8
	concIter      = 100
	separator     = "=========================================================="
	suiteCount    = 16
	goldenDir     = "data/golden"
	truncMaxRunes = 80
)

// ---------- test corpus ----------

const textPositiveAz = `Azərbaycan gözəl bir ölkədir. İnsanlar çox mehribandır və mədəniyyəti zəngindir. Bu torpaqda yaşamaq xoşbəxtlikdir.`

const textNegativeAz = `Bu xidmət çox pisdir. Heç bir keyfiyyət yoxdur, hər şey korlanmışdır. Məyus oldum və narazıyam.`

const textWithEntities = `Əlaqə: +994501234567, email: info@example.com. FIN: 1A2B3C4, VOEN: 1234567890.`

const textWithNumbersAndDates = `Konfrans 5 mart 2026 tarixində keçiriləcək. Tədbirdə 123 nəfər iştirak edəcək.`

const textCyrillicAz = `Азәрбајҹан Республикасы Ҹәнуби Гафгазда јерләшән мүстәгил дөвләтдир. Бакы онун пајтахтыдыр.`

const textRussian = `Москва является столицей Российской Федерации. Это крупнейший город страны с богатой историей и культурой.`

const textEnglish = `The quick brown fox jumps over the lazy dog. This sentence contains every letter of the English alphabet.`

const textTurkish = `İstanbul, Türkiye'nin en kalabalık şehridir. Boğaz köprüsü Avrupa ve Asya kıtalarını birbirine bağlar.`

const textDegraded = `Azerbaycan gozel bir olkedir. Insanlar cox mehribandir ve medeniyyeti zengindir.`

const textKeywords = `Neft sənayesi Azərbaycanın iqtisadiyyatında mühüm rol oynayır. Neft hasilatı və neft emalı ölkənin əsas gəlir mənbəyidir. Sənaye sahəsində neft sektoru aparıcı mövqe tutur.`

const textBroken = `Bu  bir sianq cumlesidur ,noqte yoxdu  ve  bosluglar   var.Duzgun deyil`

const textForChunker = `Birinci paraqraf birinci cümlədir. Birinci paraqraf ikinci cümlədir. Birinci paraqraf üçüncü cümlədir.

İkinci paraqraf birinci cümlədir. İkinci paraqraf ikinci cümlədir. İkinci paraqraf üçüncü cümlədir.

Üçüncü paraqraf birinci cümlədir. Üçüncü paraqraf ikinci cümlədir. Üçüncü paraqraf üçüncü cümlədir.`

// ---------- types ----------

type testResult struct {
	name     string
	module   string
	passed   bool
	duration time.Duration
	detail   string
}

type moduleReport struct {
	name     string
	tests    int
	passed   int
	failed   int
	duration time.Duration
}

// ---------- helpers ----------

func pass(module, name string, start time.Time) testResult {
	return testResult{name: name, module: module, passed: true, duration: time.Since(start)}
}

func fail(module, name, detail string, start time.Time) testResult {
	return testResult{name: name, module: module, passed: false, duration: time.Since(start), detail: truncate(detail, maxDetailLen)}
}

func truncate(s string, maxRunes int) string {
	n := 0
	for i := range s {
		n++
		if n > maxRunes {
			return s[:i] + "..."
		}
	}
	return s
}

func safeRun(module, name string, fn func() testResult) (r testResult) {
	defer func() {
		if p := recover(); p != nil {
			r = fail(module, name, fmt.Sprintf("PANIC: %v", p), time.Now())
		}
	}()
	return fn()
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func hasCyrillicRune(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Cyrillic, r) {
			return true
		}
	}
	return false
}

func hasLetterRune(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

// ---------- test suites ----------

func testTranslit() []testResult {
	const mod = "translit"
	var results []testResult

	results = append(results, safeRun(mod, "cyrillic_to_latin_basic", func() testResult {
		start := time.Now()
		out := translit.CyrillicToLatin(textCyrillicAz)
		if out == "" {
			return fail(mod, "cyrillic_to_latin_basic", "result is empty", start)
		}
		if hasCyrillicRune(out) {
			return fail(mod, "cyrillic_to_latin_basic", fmt.Sprintf("Cyrillic runes remain: %s", truncate(out, truncMaxRunes)), start)
		}
		return pass(mod, "cyrillic_to_latin_basic", start)
	}))

	results = append(results, safeRun(mod, "latin_to_cyrillic_basic", func() testResult {
		start := time.Now()
		out := translit.LatinToCyrillic(textPositiveAz)
		if out == "" {
			return fail(mod, "latin_to_cyrillic_basic", "result is empty", start)
		}
		if !hasCyrillicRune(out) {
			return fail(mod, "latin_to_cyrillic_basic", "no Cyrillic runes in output", start)
		}
		return pass(mod, "latin_to_cyrillic_basic", start)
	}))

	results = append(results, safeRun(mod, "roundtrip_lat_cyr_lat", func() testResult {
		start := time.Now()
		cyr := translit.LatinToCyrillic(textPositiveAz)
		back := translit.CyrillicToLatin(cyr)
		if back != textPositiveAz {
			return fail(mod, "roundtrip_lat_cyr_lat",
				fmt.Sprintf("expect: %s\nactual: %s", truncate(textPositiveAz, truncMaxRunes), truncate(back, truncMaxRunes)), start)
		}
		return pass(mod, "roundtrip_lat_cyr_lat", start)
	}))

	results = append(results, safeRun(mod, "roundtrip_cyr_lat_cyr", func() testResult {
		start := time.Now()
		lat := translit.CyrillicToLatin(textCyrillicAz)
		back := translit.LatinToCyrillic(lat)
		// Lossy: soft/hard signs removed. Check letter count preserved.
		origLetters := 0
		for _, r := range textCyrillicAz {
			if unicode.IsLetter(r) {
				origLetters++
			}
		}
		backLetters := 0
		for _, r := range back {
			if unicode.IsLetter(r) {
				backLetters++
			}
		}
		if backLetters == 0 {
			return fail(mod, "roundtrip_cyr_lat_cyr", "back-converted text has no letters", start)
		}
		return pass(mod, "roundtrip_cyr_lat_cyr", start)
	}))

	return results
}

func testTokenizer() []testResult {
	const mod = "tokenizer"
	var results []testResult

	results = append(results, safeRun(mod, "word_tokens_reconstruction", func() testResult {
		start := time.Now()
		tokens := tokenizer.WordTokens(textPositiveAz)
		var sb strings.Builder
		for _, t := range tokens {
			sb.WriteString(t.Text)
		}
		if sb.String() != textPositiveAz {
			return fail(mod, "word_tokens_reconstruction", "concatenated tokens != original", start)
		}
		return pass(mod, "word_tokens_reconstruction", start)
	}))

	results = append(results, safeRun(mod, "word_tokens_offset_invariant", func() testResult {
		start := time.Now()
		tokens := tokenizer.WordTokens(textPositiveAz)
		for _, t := range tokens {
			slice := textPositiveAz[t.Start:t.End]
			if slice != t.Text {
				return fail(mod, "word_tokens_offset_invariant",
					fmt.Sprintf("text[%d:%d]=%q != token.Text=%q", t.Start, t.End, slice, t.Text), start)
			}
		}
		return pass(mod, "word_tokens_offset_invariant", start)
	}))

	results = append(results, safeRun(mod, "sentence_tokens_offset_invariant", func() testResult {
		start := time.Now()
		tokens := tokenizer.SentenceTokens(textForChunker)
		for _, t := range tokens {
			slice := textForChunker[t.Start:t.End]
			if slice != t.Text {
				return fail(mod, "sentence_tokens_offset_invariant",
					fmt.Sprintf("text[%d:%d]=%q != token.Text=%q", t.Start, t.End, slice, t.Text), start)
			}
		}
		return pass(mod, "sentence_tokens_offset_invariant", start)
	}))

	results = append(results, safeRun(mod, "words_nonempty", func() testResult {
		start := time.Now()
		words := tokenizer.Words(textPositiveAz)
		if len(words) == 0 {
			return fail(mod, "words_nonempty", "Words() returned 0 words", start)
		}
		for _, w := range words {
			if !hasLetterRune(w) {
				return fail(mod, "words_nonempty", fmt.Sprintf("word %q has no letters", w), start)
			}
		}
		return pass(mod, "words_nonempty", start)
	}))

	results = append(results, safeRun(mod, "sentences_count", func() testResult {
		start := time.Now()
		sents := tokenizer.Sentences(textForChunker)
		if len(sents) < 6 {
			return fail(mod, "sentences_count", fmt.Sprintf("expected >=6 sentences, got %d", len(sents)), start)
		}
		return pass(mod, "sentences_count", start)
	}))

	return results
}

func testSpell() []testResult {
	const mod = "spell"
	var results []testResult

	results = append(results, safeRun(mod, "is_correct_known_words", func() testResult {
		start := time.Now()
		for _, w := range []string{"gözəl", "kitab"} {
			if !spell.IsCorrect(w) {
				return fail(mod, "is_correct_known_words", fmt.Sprintf("IsCorrect(%q) == false", w), start)
			}
		}
		return pass(mod, "is_correct_known_words", start)
	}))

	results = append(results, safeRun(mod, "is_correct_misspelled", func() testResult {
		start := time.Now()
		if spell.IsCorrect("gözal") {
			return fail(mod, "is_correct_misspelled", `IsCorrect("gözal") == true`, start)
		}
		return pass(mod, "is_correct_misspelled", start)
	}))

	results = append(results, safeRun(mod, "correct_text", func() testResult {
		start := time.Now()
		out := spell.Correct("gözal")
		if !strings.Contains(out, "gözəl") {
			return fail(mod, "correct_text", fmt.Sprintf("Correct(\"gözal\") = %q, want contains \"gözəl\"", out), start)
		}
		return pass(mod, "correct_text", start)
	}))

	return results
}

func testMorph() []testResult {
	const mod = "morph"
	var results []testResult

	results = append(results, safeRun(mod, "stem_inflected", func() testResult {
		start := time.Now()
		cases := []struct {
			input, want string
		}{
			{"kitablar", "kitab"},
			{"evlərdən", "ev"},
		}
		for _, c := range cases {
			got := morph.Stem(c.input)
			if got != c.want {
				return fail(mod, "stem_inflected", fmt.Sprintf("Stem(%q)=%q, want %q", c.input, got, c.want), start)
			}
		}
		return pass(mod, "stem_inflected", start)
	}))

	results = append(results, safeRun(mod, "analyze_morphemes", func() testResult {
		start := time.Now()
		analyses := morph.Analyze("kitablardan")
		if len(analyses) == 0 {
			return fail(mod, "analyze_morphemes", `Analyze("kitablardan") returned 0 analyses`, start)
		}
		if len(analyses[0].Morphemes) == 0 {
			return fail(mod, "analyze_morphemes", "first analysis has 0 morphemes", start)
		}
		return pass(mod, "analyze_morphemes", start)
	}))

	results = append(results, safeRun(mod, "stems_batch", func() testResult {
		start := time.Now()
		words := tokenizer.Words(textPositiveAz)
		stems := morph.Stems(words)
		if len(stems) != len(words) {
			return fail(mod, "stems_batch",
				fmt.Sprintf("Stems len=%d, Words len=%d", len(stems), len(words)), start)
		}
		return pass(mod, "stems_batch", start)
	}))

	results = append(results, safeRun(mod, "stem_identity", func() testResult {
		start := time.Now()
		got := morph.Stem("kitab")
		if got != "kitab" {
			return fail(mod, "stem_identity", fmt.Sprintf("Stem(\"kitab\")=%q, want \"kitab\"", got), start)
		}
		return pass(mod, "stem_identity", start)
	}))

	return results
}

func testNumtext() []testResult {
	const mod = "numtext"
	var results []testResult

	results = append(results, safeRun(mod, "convert_basic", func() testResult {
		start := time.Now()
		cases := []struct {
			n    int64
			want string
		}{
			{0, "sıfır"},
			{123, "yüz iyirmi üç"},
		}
		for _, c := range cases {
			got := numtext.Convert(c.n)
			if got != c.want {
				return fail(mod, "convert_basic", fmt.Sprintf("Convert(%d)=%q, want %q", c.n, got, c.want), start)
			}
		}
		return pass(mod, "convert_basic", start)
	}))

	results = append(results, safeRun(mod, "convert_ordinal", func() testResult {
		start := time.Now()
		got := numtext.ConvertOrdinal(3)
		if got != "üçüncü" {
			return fail(mod, "convert_ordinal", fmt.Sprintf("ConvertOrdinal(3)=%q, want \"üçüncü\"", got), start)
		}
		return pass(mod, "convert_ordinal", start)
	}))

	results = append(results, safeRun(mod, "parse_basic", func() testResult {
		start := time.Now()
		got, err := numtext.Parse("yüz iyirmi üç")
		if err != nil {
			return fail(mod, "parse_basic", fmt.Sprintf("Parse error: %v", err), start)
		}
		if got != 123 {
			return fail(mod, "parse_basic", fmt.Sprintf("Parse(\"yüz iyirmi üç\")=%d, want 123", got), start)
		}
		return pass(mod, "parse_basic", start)
	}))

	results = append(results, safeRun(mod, "roundtrip", func() testResult {
		start := time.Now()
		for _, n := range []int64{0, 1, 42, 123, 1000, 999999} {
			text := numtext.Convert(n)
			back, err := numtext.Parse(text)
			if err != nil {
				return fail(mod, "roundtrip", fmt.Sprintf("Parse(Convert(%d)) error: %v", n, err), start)
			}
			if back != n {
				return fail(mod, "roundtrip", fmt.Sprintf("Parse(Convert(%d))=%d", n, back), start)
			}
		}
		return pass(mod, "roundtrip", start)
	}))

	return results
}

func testDatetime() []testResult {
	const mod = "datetime"
	var results []testResult
	ref := time.Date(2026, time.February, 25, 12, 0, 0, 0, time.UTC)

	results = append(results, safeRun(mod, "parse_date", func() testResult {
		start := time.Now()
		r, err := datetime.Parse("5 mart 2026", ref)
		if err != nil {
			return fail(mod, "parse_date", fmt.Sprintf("Parse error: %v", err), start)
		}
		if r.Time.Month() != time.March || r.Time.Day() != 5 || r.Time.Year() != 2026 {
			return fail(mod, "parse_date",
				fmt.Sprintf("got %v, want 2026-03-05", r.Time.Format("2006-01-02")), start)
		}
		return pass(mod, "parse_date", start)
	}))

	results = append(results, safeRun(mod, "extract_offsets", func() testResult {
		start := time.Now()
		rs := datetime.Extract(textWithNumbersAndDates, ref)
		if len(rs) == 0 {
			return fail(mod, "extract_offsets", "Extract returned 0 results", start)
		}
		for _, r := range rs {
			if r.Start < 0 || r.End > len(textWithNumbersAndDates) || r.Start >= r.End {
				return fail(mod, "extract_offsets",
					fmt.Sprintf("invalid offset [%d:%d] for text len %d", r.Start, r.End, len(textWithNumbersAndDates)), start)
			}
		}
		return pass(mod, "extract_offsets", start)
	}))

	results = append(results, safeRun(mod, "parse_relative", func() testResult {
		start := time.Now()
		r, err := datetime.Parse("bu gün", ref)
		if err != nil {
			return fail(mod, "parse_relative", fmt.Sprintf("Parse error: %v", err), start)
		}
		if r.Time.Day() != ref.Day() || r.Time.Month() != ref.Month() {
			return fail(mod, "parse_relative",
				fmt.Sprintf("got %v, want same day as ref %v", r.Time.Format("2006-01-02"), ref.Format("2006-01-02")), start)
		}
		return pass(mod, "parse_relative", start)
	}))

	return results
}

func testNER() []testResult {
	const mod = "ner"
	var results []testResult

	entities := ner.Recognize(textWithEntities)

	findEntity := func(typ ner.EntityType) *ner.Entity {
		for i := range entities {
			if entities[i].Type == typ {
				return &entities[i]
			}
		}
		return nil
	}

	results = append(results, safeRun(mod, "recognize_phone", func() testResult {
		start := time.Now()
		e := findEntity(ner.Phone)
		if e == nil {
			return fail(mod, "recognize_phone", "no Phone entity found", start)
		}
		if !strings.Contains(e.Text, "994501234567") {
			return fail(mod, "recognize_phone", fmt.Sprintf("Phone text=%q", e.Text), start)
		}
		return pass(mod, "recognize_phone", start)
	}))

	results = append(results, safeRun(mod, "recognize_email", func() testResult {
		start := time.Now()
		e := findEntity(ner.Email)
		if e == nil {
			return fail(mod, "recognize_email", "no Email entity found", start)
		}
		if e.Text != "info@example.com" {
			return fail(mod, "recognize_email", fmt.Sprintf("Email text=%q", e.Text), start)
		}
		return pass(mod, "recognize_email", start)
	}))

	results = append(results, safeRun(mod, "recognize_fin", func() testResult {
		start := time.Now()
		e := findEntity(ner.FIN)
		if e == nil {
			return fail(mod, "recognize_fin", "no FIN entity found", start)
		}
		if e.Text != "1A2B3C4" {
			return fail(mod, "recognize_fin", fmt.Sprintf("FIN text=%q", e.Text), start)
		}
		return pass(mod, "recognize_fin", start)
	}))

	results = append(results, safeRun(mod, "recognize_voen", func() testResult {
		start := time.Now()
		e := findEntity(ner.VOEN)
		if e == nil {
			return fail(mod, "recognize_voen", "no VOEN entity found", start)
		}
		if e.Text != "1234567890" {
			return fail(mod, "recognize_voen", fmt.Sprintf("VOEN text=%q", e.Text), start)
		}
		return pass(mod, "recognize_voen", start)
	}))

	results = append(results, safeRun(mod, "offset_invariant", func() testResult {
		start := time.Now()
		for _, e := range entities {
			slice := textWithEntities[e.Start:e.End]
			if slice != e.Text {
				return fail(mod, "offset_invariant",
					fmt.Sprintf("text[%d:%d]=%q != entity.Text=%q", e.Start, e.End, slice, e.Text), start)
			}
		}
		return pass(mod, "offset_invariant", start)
	}))

	return results
}

func testDetect() []testResult {
	const mod = "detect"
	var results []testResult

	results = append(results, safeRun(mod, "az_latin", func() testResult {
		start := time.Now()
		r := detect.Detect(textPositiveAz)
		if r.Lang != detect.Azerbaijani {
			return fail(mod, "az_latin", fmt.Sprintf("Lang=%v, want Azerbaijani", r.Lang), start)
		}
		if r.Script != detect.ScriptLatn {
			return fail(mod, "az_latin", fmt.Sprintf("Script=%v, want Latin", r.Script), start)
		}
		return pass(mod, "az_latin", start)
	}))

	results = append(results, safeRun(mod, "az_cyrillic", func() testResult {
		start := time.Now()
		r := detect.Detect(textCyrillicAz)
		if r.Lang != detect.Azerbaijani {
			return fail(mod, "az_cyrillic", fmt.Sprintf("Lang=%v, want Azerbaijani", r.Lang), start)
		}
		if r.Script != detect.ScriptCyrl {
			return fail(mod, "az_cyrillic", fmt.Sprintf("Script=%v, want Cyrillic", r.Script), start)
		}
		return pass(mod, "az_cyrillic", start)
	}))

	results = append(results, safeRun(mod, "russian", func() testResult {
		start := time.Now()
		lang := detect.Lang(textRussian)
		if lang != "ru" {
			return fail(mod, "russian", fmt.Sprintf("Lang=%q, want \"ru\"", lang), start)
		}
		return pass(mod, "russian", start)
	}))

	results = append(results, safeRun(mod, "english", func() testResult {
		start := time.Now()
		lang := detect.Lang(textEnglish)
		if lang != "en" {
			return fail(mod, "english", fmt.Sprintf("Lang=%q, want \"en\"", lang), start)
		}
		return pass(mod, "english", start)
	}))

	results = append(results, safeRun(mod, "turkish", func() testResult {
		start := time.Now()
		lang := detect.Lang(textTurkish)
		if lang != "tr" {
			return fail(mod, "turkish", fmt.Sprintf("Lang=%q, want \"tr\"", lang), start)
		}
		return pass(mod, "turkish", start)
	}))

	return results
}

func testNormalize() []testResult {
	const mod = "normalize"
	var results []testResult

	results = append(results, safeRun(mod, "normalize_degraded", func() testResult {
		start := time.Now()
		out := normalize.Normalize(textDegraded)
		if !containsAny(out, "gözəl", "ölkə") {
			return fail(mod, "normalize_degraded",
				fmt.Sprintf("normalized text does not contain expected diacritics: %s", truncate(out, truncMaxRunes)), start)
		}
		return pass(mod, "normalize_degraded", start)
	}))

	results = append(results, safeRun(mod, "normalize_word", func() testResult {
		start := time.Now()
		got := normalize.NormalizeWord("gozel")
		if got != "gözəl" {
			return fail(mod, "normalize_word", fmt.Sprintf("NormalizeWord(\"gozel\")=%q, want \"gözəl\"", got), start)
		}
		return pass(mod, "normalize_word", start)
	}))

	results = append(results, safeRun(mod, "idempotent", func() testResult {
		start := time.Now()
		out := normalize.Normalize(textPositiveAz)
		if out != textPositiveAz {
			return fail(mod, "idempotent", "Normalize changed already-correct text", start)
		}
		return pass(mod, "idempotent", start)
	}))

	return results
}

func testKeywords() []testResult {
	const mod = "keywords"
	var results []testResult

	results = append(results, safeRun(mod, "tfidf", func() testResult {
		start := time.Now()
		kws := keywords.ExtractTFIDF(textKeywords, 5)
		if len(kws) == 0 || len(kws) > 5 {
			return fail(mod, "tfidf", fmt.Sprintf("ExtractTFIDF returned %d keywords, want 1-5", len(kws)), start)
		}
		found := false
		for _, kw := range kws {
			if strings.Contains(kw.Stem, "neft") {
				found = true
				break
			}
		}
		if !found {
			stems := make([]string, len(kws))
			for i, kw := range kws {
				stems[i] = kw.Stem
			}
			return fail(mod, "tfidf", fmt.Sprintf("no neft-related stem in %v", stems), start)
		}
		return pass(mod, "tfidf", start)
	}))

	results = append(results, safeRun(mod, "textrank", func() testResult {
		start := time.Now()
		kws := keywords.ExtractTextRank(textKeywords, 5)
		if len(kws) == 0 {
			return fail(mod, "textrank", "ExtractTextRank returned 0 keywords", start)
		}
		return pass(mod, "textrank", start)
	}))

	results = append(results, safeRun(mod, "convenience", func() testResult {
		start := time.Now()
		kws := keywords.Keywords(textKeywords)
		if len(kws) == 0 {
			return fail(mod, "convenience", "Keywords returned empty slice", start)
		}
		return pass(mod, "convenience", start)
	}))

	return results
}

func testValidate() []testResult {
	const mod = "validate"
	var results []testResult

	results = append(results, safeRun(mod, "well_formed_high_score", func() testResult {
		start := time.Now()
		report := validate.Validate(textPositiveAz)
		if report.Score < 80 {
			return fail(mod, "well_formed_high_score",
				fmt.Sprintf("Score=%d, want >=80 (issues=%d)", report.Score, len(report.Issues)), start)
		}
		return pass(mod, "well_formed_high_score", start)
	}))

	results = append(results, safeRun(mod, "broken_has_issues", func() testResult {
		start := time.Now()
		report := validate.Validate(textBroken)
		if report.Score >= 80 {
			return fail(mod, "broken_has_issues",
				fmt.Sprintf("Score=%d, want <80 for broken text", report.Score), start)
		}
		if len(report.Issues) == 0 {
			return fail(mod, "broken_has_issues", "no issues found in broken text", start)
		}
		return pass(mod, "broken_has_issues", start)
	}))

	results = append(results, safeRun(mod, "is_valid", func() testResult {
		start := time.Now()
		if !validate.IsValid(textPositiveAz) {
			return fail(mod, "is_valid", "IsValid returned false for well-formed text", start)
		}
		return pass(mod, "is_valid", start)
	}))

	return results
}

func testSentiment() []testResult {
	const mod = "sentiment"
	var results []testResult

	results = append(results, safeRun(mod, "positive", func() testResult {
		start := time.Now()
		r := sentiment.Analyze(textPositiveAz)
		if r.Sentiment != sentiment.Positive {
			return fail(mod, "positive", fmt.Sprintf("Sentiment=%v, want Positive (score=%.2f)", r.Sentiment, r.Score), start)
		}
		if r.Score <= 0 {
			return fail(mod, "positive", fmt.Sprintf("Score=%.2f, want >0", r.Score), start)
		}
		return pass(mod, "positive", start)
	}))

	results = append(results, safeRun(mod, "negative", func() testResult {
		start := time.Now()
		r := sentiment.Analyze(textNegativeAz)
		if r.Sentiment != sentiment.Negative {
			return fail(mod, "negative", fmt.Sprintf("Sentiment=%v, want Negative (score=%.2f)", r.Sentiment, r.Score), start)
		}
		if r.Score >= 0 {
			return fail(mod, "negative", fmt.Sprintf("Score=%.2f, want <0", r.Score), start)
		}
		return pass(mod, "negative", start)
	}))

	results = append(results, safeRun(mod, "score_sign", func() testResult {
		start := time.Now()
		s := sentiment.Score(textPositiveAz)
		if s <= 0 {
			return fail(mod, "score_sign", fmt.Sprintf("Score=%.2f, want >0", s), start)
		}
		return pass(mod, "score_sign", start)
	}))

	results = append(results, safeRun(mod, "is_positive", func() testResult {
		start := time.Now()
		if !sentiment.IsPositive(textPositiveAz) {
			return fail(mod, "is_positive", "IsPositive(positive text) == false", start)
		}
		if sentiment.IsPositive(textNegativeAz) {
			return fail(mod, "is_positive", "IsPositive(negative text) == true", start)
		}
		return pass(mod, "is_positive", start)
	}))

	return results
}

func testChunker() []testResult {
	const mod = "chunker"
	var results []testResult

	results = append(results, safeRun(mod, "by_size_offsets", func() testResult {
		start := time.Now()
		chunks := chunker.BySize(textForChunker, 100, 10)
		if len(chunks) <= 1 {
			return fail(mod, "by_size_offsets", fmt.Sprintf("BySize returned %d chunks, want >1", len(chunks)), start)
		}
		for _, c := range chunks {
			slice := textForChunker[c.Start:c.End]
			if slice != c.Text {
				return fail(mod, "by_size_offsets",
					fmt.Sprintf("text[%d:%d] != chunk.Text for chunk %d", c.Start, c.End, c.Index), start)
			}
		}
		return pass(mod, "by_size_offsets", start)
	}))

	results = append(results, safeRun(mod, "by_sentence", func() testResult {
		start := time.Now()
		chunks := chunker.BySentence(textForChunker, 200, 20)
		if len(chunks) < 1 {
			return fail(mod, "by_sentence", "BySentence returned 0 chunks", start)
		}
		for _, c := range chunks {
			slice := textForChunker[c.Start:c.End]
			if slice != c.Text {
				return fail(mod, "by_sentence",
					fmt.Sprintf("text[%d:%d] != chunk.Text for chunk %d", c.Start, c.End, c.Index), start)
			}
		}
		return pass(mod, "by_sentence", start)
	}))

	results = append(results, safeRun(mod, "recursive", func() testResult {
		start := time.Now()
		chunks := chunker.Recursive(textForChunker, 150, 15)
		if len(chunks) < 1 {
			return fail(mod, "recursive", "Recursive returned 0 chunks", start)
		}
		for _, c := range chunks {
			slice := textForChunker[c.Start:c.End]
			if slice != c.Text {
				return fail(mod, "recursive",
					fmt.Sprintf("text[%d:%d] != chunk.Text for chunk %d", c.Start, c.End, c.Index), start)
			}
		}
		return pass(mod, "recursive", start)
	}))

	results = append(results, safeRun(mod, "full_coverage", func() testResult {
		start := time.Now()
		chunks := chunker.BySize(textForChunker, 100, 0)
		var sb strings.Builder
		for _, c := range chunks {
			sb.WriteString(c.Text)
		}
		if sb.String() != textForChunker {
			return fail(mod, "full_coverage", "concatenated chunks != original text", start)
		}
		return pass(mod, "full_coverage", start)
	}))

	return results
}

func testPipeline() []testResult {
	const mod = "pipeline"
	var results []testResult

	results = append(results, safeRun(mod, "normalize_tokenize_morph_sentiment", func() testResult {
		start := time.Now()
		normalized := normalize.Normalize(textDegraded)
		if normalized == "" {
			return fail(mod, "normalize_tokenize_morph_sentiment", "normalize returned empty", start)
		}
		words := tokenizer.Words(normalized)
		if len(words) == 0 {
			return fail(mod, "normalize_tokenize_morph_sentiment", "tokenizer returned 0 words", start)
		}
		stems := morph.Stems(words)
		if len(stems) == 0 {
			return fail(mod, "normalize_tokenize_morph_sentiment", "morph returned 0 stems", start)
		}
		r := sentiment.Analyze(normalized)
		if r.Total == 0 {
			return fail(mod, "normalize_tokenize_morph_sentiment", "sentiment analyzed 0 words", start)
		}
		return pass(mod, "normalize_tokenize_morph_sentiment", start)
	}))

	results = append(results, safeRun(mod, "translit_detect_tokenize", func() testResult {
		start := time.Now()
		latin := translit.CyrillicToLatin(textCyrillicAz)
		if latin == "" {
			return fail(mod, "translit_detect_tokenize", "CyrillicToLatin returned empty", start)
		}
		lang := detect.Lang(latin)
		if lang != "az" {
			return fail(mod, "translit_detect_tokenize", fmt.Sprintf("detect Lang=%q, want \"az\"", lang), start)
		}
		tokens := tokenizer.WordTokens(latin)
		var sb strings.Builder
		for _, t := range tokens {
			sb.WriteString(t.Text)
		}
		if sb.String() != latin {
			return fail(mod, "translit_detect_tokenize", "token reconstruction failed after translit", start)
		}
		return pass(mod, "translit_detect_tokenize", start)
	}))

	return results
}

func testConcurrent() []testResult {
	const mod = "concurrent"
	var results []testResult

	results = append(results, safeRun(mod, "all_modules_8_goroutines_x100", func() testResult {
		start := time.Now()
		ref := time.Date(2026, time.February, 25, 12, 0, 0, 0, time.UTC)
		var panics atomic.Int64
		var wg sync.WaitGroup

		for range concWorkers {
			wg.Go(func() {
				for range concIter {
					func() {
						defer func() {
							if p := recover(); p != nil {
								panics.Add(1)
							}
						}()
						translit.CyrillicToLatin(textCyrillicAz)
						translit.LatinToCyrillic(textPositiveAz)
						tokenizer.WordTokens(textPositiveAz)
						tokenizer.Words(textPositiveAz)
						tokenizer.Sentences(textForChunker)
						spell.IsCorrect("gözəl")
						morph.Stem("kitablar")
						numtext.Convert(123)
						_, _ = numtext.Parse("yüz")
						_, _ = datetime.Parse("bu gün", ref)
						ner.Recognize(textWithEntities)
						detect.Detect(textPositiveAz)
						normalize.NormalizeWord("gozel")
						keywords.Keywords(textKeywords)
						validate.IsValid(textPositiveAz)
						sentiment.Analyze(textPositiveAz)
						chunker.BySize(textForChunker, 100, 10)
					}()
				}
			})
		}
		wg.Wait()

		if n := panics.Load(); n > 0 {
			return fail(mod, "all_modules_8_goroutines_x100",
				fmt.Sprintf("%d panics detected across goroutines", n), start)
		}
		return pass(mod, "all_modules_8_goroutines_x100", start)
	}))

	return results
}

// ---------- corpus helpers ----------

// goldenEntry represents one entry from a golden JSON test file.
type goldenEntry struct {
	Input string `json:"input"`
	Text  string `json:"text"`
}

// loadGoldenCorpus reads all golden JSON files and returns concatenated input texts.
func loadGoldenCorpus() (string, int, error) {
	files, err := filepath.Glob(filepath.Join(goldenDir, "*.json"))
	if err != nil {
		return "", 0, err
	}
	if len(files) == 0 {
		return "", 0, fmt.Errorf("no golden files found in %s", goldenDir)
	}

	var texts []string
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			return "", 0, fmt.Errorf("reading %s: %w", f, err)
		}
		var entries []goldenEntry
		if err := json.Unmarshal(raw, &entries); err != nil {
			continue // skip non-array golden files
		}
		for _, e := range entries {
			inp := e.Input
			if inp == "" {
				inp = e.Text
			}
			if inp != "" {
				texts = append(texts, inp)
			}
		}
	}
	corpus := strings.Join(texts, "\n\n")
	return corpus, len(texts), nil
}

func testCorpus() []testResult {
	const mod = "corpus"
	var results []testResult

	corpus, inputCount, err := loadGoldenCorpus()
	if err != nil {
		results = append(results, fail(mod, "load_golden_corpus", fmt.Sprintf("error: %v", err), time.Now()))
		return results
	}

	results = append(results, safeRun(mod, "load_golden_corpus", func() testResult {
		start := time.Now()
		if inputCount == 0 {
			return fail(mod, "load_golden_corpus", "no inputs found", start)
		}
		log.Printf("  corpus: %d inputs, %d bytes", inputCount, len(corpus))
		return pass(mod, "load_golden_corpus", start)
	}))

	results = append(results, safeRun(mod, "tokenize_full_corpus", func() testResult {
		start := time.Now()
		tokens := tokenizer.WordTokens(corpus)
		if len(tokens) == 0 {
			return fail(mod, "tokenize_full_corpus", "WordTokens returned 0 tokens", start)
		}
		var sb strings.Builder
		for _, t := range tokens {
			sb.WriteString(t.Text)
		}
		if sb.String() != corpus {
			return fail(mod, "tokenize_full_corpus", "reconstruction failed on full corpus", start)
		}
		return pass(mod, "tokenize_full_corpus", start)
	}))

	results = append(results, safeRun(mod, "morph_full_corpus", func() testResult {
		start := time.Now()
		words := tokenizer.Words(corpus)
		stems := morph.Stems(words)
		if len(stems) != len(words) {
			return fail(mod, "morph_full_corpus",
				fmt.Sprintf("Stems len=%d != Words len=%d", len(stems), len(words)), start)
		}
		empty := 0
		for _, s := range stems {
			if s == "" {
				empty++
			}
		}
		if empty > len(stems)/2 {
			return fail(mod, "morph_full_corpus",
				fmt.Sprintf("%d/%d stems are empty", empty, len(stems)), start)
		}
		return pass(mod, "morph_full_corpus", start)
	}))

	results = append(results, safeRun(mod, "detect_full_corpus", func() testResult {
		start := time.Now()
		r := detect.Detect(corpus)
		if r.Lang != detect.Azerbaijani {
			return fail(mod, "detect_full_corpus",
				fmt.Sprintf("Lang=%v, want Azerbaijani", r.Lang), start)
		}
		return pass(mod, "detect_full_corpus", start)
	}))

	results = append(results, safeRun(mod, "sentiment_full_corpus", func() testResult {
		start := time.Now()
		r := sentiment.Analyze(corpus)
		if r.Total == 0 {
			return fail(mod, "sentiment_full_corpus", "sentiment analyzed 0 words", start)
		}
		return pass(mod, "sentiment_full_corpus", start)
	}))

	results = append(results, safeRun(mod, "keywords_full_corpus", func() testResult {
		start := time.Now()
		kws := keywords.Keywords(corpus)
		if len(kws) == 0 {
			return fail(mod, "keywords_full_corpus", "Keywords returned 0 results", start)
		}
		return pass(mod, "keywords_full_corpus", start)
	}))

	results = append(results, safeRun(mod, "chunker_full_corpus", func() testResult {
		start := time.Now()
		chunks := chunker.Recursive(corpus, 200, 20)
		if len(chunks) == 0 {
			return fail(mod, "chunker_full_corpus", "Recursive returned 0 chunks", start)
		}
		for _, c := range chunks {
			if c.Start < 0 || c.End > len(corpus) || c.Start > c.End {
				return fail(mod, "chunker_full_corpus",
					fmt.Sprintf("invalid offset [%d:%d]", c.Start, c.End), start)
			}
			if corpus[c.Start:c.End] != c.Text {
				return fail(mod, "chunker_full_corpus",
					fmt.Sprintf("offset invariant broken at chunk %d", c.Index), start)
			}
		}
		return pass(mod, "chunker_full_corpus", start)
	}))

	results = append(results, safeRun(mod, "ner_full_corpus", func() testResult {
		start := time.Now()
		entities := ner.Recognize(corpus)
		for _, e := range entities {
			if e.Start < 0 || e.End > len(corpus) || e.Start >= e.End {
				return fail(mod, "ner_full_corpus",
					fmt.Sprintf("invalid offset [%d:%d]", e.Start, e.End), start)
			}
			if corpus[e.Start:e.End] != e.Text {
				return fail(mod, "ner_full_corpus",
					fmt.Sprintf("offset invariant broken for %s entity", e.Type), start)
			}
		}
		return pass(mod, "ner_full_corpus", start)
	}))

	results = append(results, safeRun(mod, "validate_full_corpus", func() testResult {
		start := time.Now()
		report := validate.Validate(corpus)
		if report.Score < 0 || report.Score > 100 {
			return fail(mod, "validate_full_corpus",
				fmt.Sprintf("Score=%d out of range [0,100]", report.Score), start)
		}
		return pass(mod, "validate_full_corpus", start)
	}))

	results = append(results, safeRun(mod, "normalize_full_corpus", func() testResult {
		start := time.Now()
		out := normalize.Normalize(corpus)
		if out == "" {
			return fail(mod, "normalize_full_corpus", "Normalize returned empty", start)
		}
		if len(out) < len(corpus)/2 {
			return fail(mod, "normalize_full_corpus",
				fmt.Sprintf("output suspiciously short: %d vs %d bytes", len(out), len(corpus)), start)
		}
		return pass(mod, "normalize_full_corpus", start)
	}))

	return results
}

// ---------- orchestration ----------

func runAllSuites() []testResult {
	suites := []func() []testResult{
		testTranslit,
		testTokenizer,
		testSpell,
		testMorph,
		testNumtext,
		testDatetime,
		testNER,
		testDetect,
		testNormalize,
		testKeywords,
		testValidate,
		testSentiment,
		testChunker,
		testPipeline,
		testConcurrent,
		testCorpus,
	}

	var all []testResult
	for _, suite := range suites {
		all = append(all, suite()...)
	}
	return all
}

func buildReports(results []testResult) []moduleReport {
	order := make(map[string]int)
	var reports []moduleReport

	for _, r := range results {
		idx, exists := order[r.module]
		if !exists {
			idx = len(reports)
			order[r.module] = idx
			reports = append(reports, moduleReport{name: r.module})
		}
		reports[idx].tests++
		reports[idx].duration += r.duration
		if r.passed {
			reports[idx].passed++
		} else {
			reports[idx].failed++
		}
	}
	return reports
}

func writeLog(path string, results []testResult) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	bw := bufio.NewWriter(f)

	now := time.Now().UTC().Format(time.RFC3339)
	goVer := runtime.Version()
	platform := runtime.GOOS + "/" + runtime.GOARCH

	fmt.Fprintln(bw, separator)
	fmt.Fprintln(bw, "  az-lang-nlp E2E Pipeline Test")
	fmt.Fprintf(bw, "  Timestamp: %s\n", now)
	fmt.Fprintf(bw, "  Go: %s  OS: %s\n", goVer, platform)
	fmt.Fprintf(bw, "  Modules: %d\n", moduleCount)
	fmt.Fprintln(bw, separator)
	fmt.Fprintln(bw)

	reports := buildReports(results)
	var totalDuration time.Duration
	for _, rep := range reports {
		totalDuration += rep.duration
	}

	// Per-module sections.
	for _, rep := range reports {
		fmt.Fprintf(bw, "[%s] %d tests | %d passed | %d failed | %s\n",
			rep.name, rep.tests, rep.passed, rep.failed, rep.duration.Round(time.Microsecond))
		for _, r := range results {
			if r.module != rep.name {
				continue
			}
			status := "PASS"
			if !r.passed {
				status = "FAIL"
			}
			fmt.Fprintf(bw, "  %-6s %-45s %s\n", status, r.name, r.duration.Round(time.Microsecond))
		}
		fmt.Fprintln(bw)
	}

	// Failures section.
	var failures []testResult
	for _, r := range results {
		if !r.passed {
			failures = append(failures, r)
		}
	}
	if len(failures) > 0 {
		fmt.Fprintln(bw, "--- FAILURES ---")
		for _, r := range failures {
			fmt.Fprintf(bw, "  FAIL  [%s] %-40s %s\n", r.module, r.name, r.duration.Round(time.Microsecond))
			if r.detail != "" {
				for line := range strings.SplitSeq(r.detail, "\n") {
					fmt.Fprintf(bw, "        %s\n", line)
				}
			}
		}
		fmt.Fprintln(bw)
	}

	// Summary.
	totalTests := len(results)
	totalPassed := 0
	totalFailed := 0
	for _, r := range results {
		if r.passed {
			totalPassed++
		} else {
			totalFailed++
		}
	}

	fmt.Fprintln(bw, separator)
	fmt.Fprintf(bw, "  SUMMARY: %d tests | %d passed | %d failed | %s\n",
		totalTests, totalPassed, totalFailed, totalDuration.Round(time.Microsecond))
	fmt.Fprintln(bw, separator)

	if err := bw.Flush(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func printSummary(results []testResult) {
	reports := buildReports(results)
	totalPassed := 0
	totalFailed := 0
	var totalDuration time.Duration

	for _, rep := range reports {
		totalPassed += rep.passed
		totalFailed += rep.failed
		totalDuration += rep.duration

		status := "OK"
		if rep.failed > 0 {
			status = "FAIL"
		}
		log.Printf("  %-12s %d/%d %s", rep.name, rep.passed, rep.tests, status)
	}

	log.Printf("")
	log.Printf("  %d tests | %d passed | %d failed | %s",
		len(results), totalPassed, totalFailed, totalDuration.Round(time.Microsecond))

	for _, r := range results {
		if !r.passed {
			log.Printf("  FAIL [%s] %s: %s", r.module, r.name, r.detail)
		}
	}
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("[e2e] ")

	log.Printf("starting E2E pipeline test (%d modules, %d suites)", moduleCount, suiteCount)
	totalStart := time.Now()

	results := runAllSuites()

	log.Printf("completed in %s", time.Since(totalStart).Round(time.Microsecond))
	log.Printf("")

	printSummary(results)

	if err := writeLog(logPath, results); err != nil {
		log.Fatalf("cannot write log: %v", err)
	}
	log.Printf("log written to %s", logPath)

	for _, r := range results {
		if !r.passed {
			os.Exit(1)
		}
	}
}
