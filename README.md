# Modern Azerbaijani NLP

[![CI](https://github.com/az-ai-labs/az-lang-nlp/actions/workflows/ci.yml/badge.svg)](https://github.com/az-ai-labs/az-lang-nlp/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/az-ai-labs/az-lang-nlp.svg)](https://pkg.go.dev/github.com/az-ai-labs/az-lang-nlp)
[![Go Version](https://img.shields.io/github/go-mod/go-version/az-ai-labs/az-lang-nlp)](https://github.com/az-ai-labs/az-lang-nlp/blob/main/go.mod)
[![License](https://img.shields.io/github/license/az-ai-labs/az-lang-nlp)](LICENSE)

NLP toolkit for Azerbaijani language. Pure Go, zero dependencies.

All packages are safe for concurrent use.

## Packages

| Package                          | Description                                             |
| -------------------------------- | ------------------------------------------------------- |
| [translit](#transliteration)     | Latin / Cyrillic script conversion                      |
| [tokenizer](#tokenizer)          | Word and sentence tokenization with byte offsets        |
| [morph](#morphological-analysis) | Stem and suffix chain decomposition                     |
| [numtext](#number-to-text)       | Number / text conversion ("123" &rarr; "yuz iyirmi uc") |
| [ner](#named-entity-recognition) | FIN, VOEN, phone, email, IBAN, plate, URL extraction    |
| [datetime](#datetime)            | Date/time parser ("5 mart 2026" &rarr; structured)      |
| [normalize](#text-normalization) | Diacritic restoration ("gozel" &rarr; "g&ouml;z&auml;l")            |
| [spell](#spell-checker)          | Spell checking (SymSpell algorithm)                     |
| [detect](#language-detection)    | Language detection (az/ru/en/tr)                        |
| [keywords](#keyword-extraction)  | Keyword extraction (TF-IDF / TextRank)                  |

## Install

```
go get github.com/az-ai-labs/az-lang-nlp
```

Requires Go 1.25.7 or later.

## Transliteration

Convert Azerbaijani text between Latin and Cyrillic scripts.

```go
translit.CyrillicToLatin("Азәрбајҹан")
// Azərbaycan

translit.LatinToCyrillic("Həyat gözəldir")
// Һәјат ҝөзәлдир
```

Contextual rules handle Cyrillic Г/г disambiguation automatically. Non-Azerbaijani characters (digits, punctuation, emoji) pass through unchanged.

## Tokenizer

Split Azerbaijani text into words and sentences with byte offsets.

```go
// Word tokenization
tokenizer.Words("Bakı'nın küçələri gözəldir.")
// [Bakı'nın küçələri gözəldir]

// Structured tokens with byte offsets
for _, t := range tokenizer.WordTokens("Salam, dünya!") {
    fmt.Printf("%s: %q\n", t.Type, t.Text)
}
// Word: "Salam"
// Punctuation: ","
// Space: " "
// Word: "dünya"
// Punctuation: "!"

// Sentence splitting
tokenizer.Sentences("Birinci cümlə. İkinci cümlə.")
// [Birinci cümlə.  İkinci cümlə.]
```

Handles URLs, emails, Azerbaijani abbreviations (Prof., Az.R.), thousand-separator dots (1.000.000), decimal commas (3,14), hyphens (sosial-iqtisadi), and apostrophe suffixes (Bakı'nın).

## Morphological Analysis

Decompose inflected Azerbaijani words into stem and suffix chain.

```go
// Extract stem from inflected word
morph.Stem("kitablarımızdan")
// kitab

// Full morphological analysis
for _, a := range morph.Analyze("kitablar") {
    fmt.Println(a)
}
// kitab[Plural:lar]
// kitabl[TenseAorist:ar]
// kitablar

// Batch stemming (pairs with tokenizer.Words)
morph.Stems([]string{"kitablarımızdan", "evlərdə", "gəlmişdir"})
// [kitab ev gəl]
```

Uses a table-driven morphotactic state machine with backtracking. Validates vowel harmony, consonant assimilation, and suffix ordering. Includes an embedded dictionary (~12K stems from Wiktionary) for stem validation.

## Number-to-Text

Convert between numbers and Azerbaijani text representations.

```go
// Cardinal number
numtext.Convert(123)
// yüz iyirmi üç

// Ordinal number with vowel-harmony suffix
numtext.ConvertOrdinal(5)
// beşinci

// Decimal: math mode
numtext.ConvertFloat("3.14", numtext.MathMode)
// üç tam yüzdə on dörd

// Decimal: digit-by-digit mode
numtext.ConvertFloat("3.14", numtext.DigitMode)
// üç vergül bir dörd

// Parse text back to number
n, _ := numtext.Parse("iki milyon üç yüz min doxsan beş")
fmt.Println(n)
// 2300095
```

Supports integers up to ±10^18, negative numbers, ordinals, and decimals with dot or comma separator. Parse is case-insensitive and accepts both canonical ("yüz") and explicit ("bir yüz") forms.

## Named Entity Recognition

Extract structured entities from Azerbaijani text: FIN, VOEN, phone numbers, emails, IBANs, license plates, and URLs.

```go
// Extract all entities with byte offsets
for _, e := range ner.Recognize("FIN: 5ARPXK2, tel +994501234567") {
    fmt.Printf("%s: %q (labeled=%v)\n", e.Type, e.Text, e.Labeled)
}
// FIN: "5ARPXK2" (labeled=true)
// Phone: "+994501234567" (labeled=false)

// Convenience functions return []string
ner.Phones("+994501234567 və 0551234567")
// [+994501234567 0551234567]

ner.Emails("info@gov.az")
// [info@gov.az]

ner.IBANs("AZ21NABZ00000000137010001944")
// [AZ21NABZ00000000137010001944]
```

FIN and VOEN patterns are ambiguous in isolation. When preceded by a keyword (e.g. "FIN:", "VOEN:"), `Entity.Labeled` is true, indicating higher confidence. Overlapping entities are resolved by preferring longer matches.

## Datetime

Parse Azerbaijani date and time expressions into structured values.

```go
// Parse a natural-language date
r, _ := datetime.Parse("5 mart 2026", time.Time{})
fmt.Println(r.Type, r.Time.Format("2006-01-02"))
// Date 2026-03-05

// Extract dates from running text
for _, r := range datetime.Extract("Görüş 15 yanvar 2026 saat 14:30-da olacaq", time.Time{}) {
    fmt.Printf("%s: %q -> %s\n", r.Type, r.Text, r.Time.Format("2006-01-02 15:04"))
}
// Date: "15 yanvar 2026" -> 2026-01-15 00:00
// Time: "14:30" -> ... 14:30
```

Handles natural text ("5 mart 2026"), numeric formats ("05.03.2026", "2026-03-05"), and relative expressions ("bu gun", "3 gun evvel", "kecen hefte"). Relative expressions resolve against a reference time.

## Text Normalization

Restore missing Azerbaijani diacritics in ASCII-degraded text.

```go
// Restore diacritics in a single word
normalize.NormalizeWord("gozel")
// gözəl

normalize.NormalizeWord("azerbaycan")
// azərbaycan

// Ambiguous words are left unchanged
normalize.NormalizeWord("seher")
// seher (could be səhər or şəhər)

// Full text normalization
normalize.Normalize("Bu gozel seherde yasayiram.")
// Bu gözəl seherde yasayiram.

// Case is preserved
normalize.NormalizeWord("GOZEL")
// GÖZƏL
```

Uses dictionary lookup against the morph package's ~12K stem dictionary to find unambiguous diacritic restorations. Words with multiple possible restorations or not found in the dictionary are returned unchanged. Handles hyphenated words and apostrophe suffixes. Input longer than 1 MiB is returned unchanged.

## Spell Checker

Check and correct spelling errors in Azerbaijani text using the SymSpell algorithm with morphology-aware validation.

```go
// Check if a word is correctly spelled
spell.IsCorrect("kitab")    // true
spell.IsCorrect("ketab")    // false
spell.IsCorrect("kitablar") // true (morphologically valid)
spell.IsCorrect("gozel")    // true (normalizable to gözəl)

// Get correction suggestions
suggestions := spell.Suggest("ketab", 2)
fmt.Println(suggestions[0].Term, suggestions[0].Distance)
// kitab 1

// Correct a single word (preserves case)
spell.CorrectWord("ketab")  // kitab
spell.CorrectWord("KETAB")  // KİTAB

// Correct all misspelled words in text
spell.Correct("Bu ketab gozeldir")
// Bu kitab gozeldir
```

Uses an embedded frequency dictionary (~86K entries from a 1.25 GB Azerbaijani corpus) with the SymSpell symmetric delete algorithm for sub-microsecond lookups. Validates words through frequency dictionary, morphological analysis, and diacritic normalization. Handles hyphenated words, apostrophe suffixes, and case preservation. Title-case unknown words are left unchanged to avoid over-correcting proper nouns.

## Language Detection

Identify the language of input text: Azerbaijani, Russian, English, or Turkish.

```go
// Detect language with confidence score
r := detect.Detect("Salam, necəsən? Bu gün hava çox gözəldir.")
fmt.Println(r.Lang, r.Script, r.Confidence)
// Azerbaijani Latn 0.95...

// ISO 639-1 code
detect.Lang("Hello, how are you doing today?")
// en

// Ranked results for all supported languages
for _, r := range detect.DetectAll("Привет, как дела?") {
    fmt.Printf("%s: %.2f\n", r.Lang, r.Confidence)
}
// Russian: 0.55
// Azerbaijani: 0.45
// English: 0.00
// Turkish: 0.00
```

Uses hybrid character-set scoring with trigram fallback for ambiguous cases (Azerbaijani vs Turkish). Supports Azerbaijani in both Latin and Cyrillic scripts. Input longer than 1 MiB is silently truncated.

## Keyword Extraction

Extract keywords from Azerbaijani text using TF-IDF or TextRank algorithms.

```go
// Structured: TF-IDF scored keywords
for _, kw := range keywords.ExtractTFIDF("Azərbaycan iqtisadiyyatı sürətlə inkişaf edir. Azərbaycan neft sektorunda liderdir.", 3) {
    fmt.Printf("%s (score=%.2f, count=%d)\n", kw.Stem, kw.Score, kw.Count)
}
// azərbaycan (score=1.41, count=2)
// sektor (score=1.25, count=1)
// lider (score=1.11, count=1)

// Structured: TextRank scored keywords
for _, kw := range keywords.ExtractTextRank("Azərbaycan iqtisadiyyatı sürətlə inkişaf edir", 3) {
    fmt.Printf("%s (score=%.2f)\n", kw.Stem, kw.Score)
}
// iqtisadiyyat (score=0.30)
// sürət (score=0.30)
// azərbaycan (score=0.20)

// Convenience: top 10 keyword stems via TextRank
keywords.Keywords("Azərbaycan iqtisadiyyatı sürətlə inkişaf edir")
// [iqtisadiyyat sürət azərbaycan inkişaf]
```

Integrates with `normalize` for diacritic restoration, `tokenizer` for word splitting, and `morph` for stemming. Inflected forms ("kitab", "kitablar", "kitabdan") group under a single stem. Stopwords (pronouns, conjunctions, particles, auxiliaries) are filtered after stemming. Input longer than 1 MiB returns nil.

## Planned

- **validate** — text validator (spelling + punctuation + layout)

## License

[MIT](LICENSE)
